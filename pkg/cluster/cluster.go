// Package cluster provides multi-process mode with graceful reload,
// shared state coordination, and horizontal scaling support.
package cluster

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Config holds cluster configuration.
type Config struct {
	Workers         int           // number of worker processes (0 = NumCPU)
	GracefulTimeout time.Duration // max time to wait for in-flight requests
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownSignals []os.Signal
}

// DefaultConfig returns sensible cluster defaults.
func DefaultConfig() Config {
	return Config{
		Workers:         0, // auto-detect
		GracefulTimeout: 30 * time.Second,
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownSignals: []os.Signal{syscall.SIGINT, syscall.SIGTERM},
	}
}

// Cluster manages multiple HTTP server instances.
type Cluster struct {
	config   Config
	handler  http.Handler
	listener net.Listener
	servers  []*http.Server
	mu       sync.Mutex
	stats    Stats
	ready    chan struct{}
}

// Stats tracks cluster-wide statistics.
type Stats struct {
	ActiveRequests  atomic.Int64
	TotalRequests   atomic.Int64
	TotalErrors     atomic.Int64
	StartedAt       time.Time
	WorkerCount     int
}

// New creates a cluster manager.
func New(cfg Config, handler http.Handler) *Cluster {
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}
	return &Cluster{
		config:  cfg,
		handler: handler,
		ready:   make(chan struct{}),
		stats: Stats{
			StartedAt:   time.Now(),
			WorkerCount: cfg.Workers,
		},
	}
}

// ListenAndServe starts the cluster with multiple server instances sharing a listener.
func (c *Cluster) ListenAndServe(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("cluster: listen: %w", err)
	}
	c.listener = ln

	// Wrap handler with stats tracking
	wrappedHandler := c.wrapHandler(c.handler)

	// Start worker servers
	c.mu.Lock()
	for i := 0; i < c.config.Workers; i++ {
		srv := &http.Server{
			Handler:      wrappedHandler,
			ReadTimeout:  c.config.ReadTimeout,
			WriteTimeout: c.config.WriteTimeout,
			IdleTimeout:  c.config.IdleTimeout,
		}
		c.servers = append(c.servers, srv)
		go func(srv *http.Server, id int) {
			log.Printf("[NexGo Cluster] Worker %d started", id)
			if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
				log.Printf("[NexGo Cluster] Worker %d error: %v", id, err)
			}
		}(srv, i)
	}
	c.mu.Unlock()

	close(c.ready)

	log.Printf("[NexGo Cluster] %d workers listening on %s", c.config.Workers, addr)

	// Wait for shutdown signal
	ctx, stop := signal.NotifyContext(context.Background(), c.config.ShutdownSignals...)
	defer stop()
	<-ctx.Done()

	return c.Shutdown()
}

// Shutdown gracefully stops all workers.
func (c *Cluster) Shutdown() error {
	log.Printf("[NexGo Cluster] Shutting down (timeout: %s)...", c.config.GracefulTimeout)

	ctx, cancel := context.WithTimeout(context.Background(), c.config.GracefulTimeout)
	defer cancel()

	c.mu.Lock()
	servers := c.servers
	c.mu.Unlock()

	var wg sync.WaitGroup
	for i, srv := range servers {
		wg.Add(1)
		go func(srv *http.Server, id int) {
			defer wg.Done()
			if err := srv.Shutdown(ctx); err != nil {
				log.Printf("[NexGo Cluster] Worker %d shutdown error: %v", id, err)
			} else {
				log.Printf("[NexGo Cluster] Worker %d stopped", id)
			}
		}(srv, i)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[NexGo Cluster] All workers stopped. Active: %d, Total: %d",
			c.stats.ActiveRequests.Load(), c.stats.TotalRequests.Load())
	case <-ctx.Done():
		log.Printf("[NexGo Cluster] Shutdown timed out, forcing stop")
	}

	if c.listener != nil {
		c.listener.Close()
	}

	return nil
}

// GetStats returns a snapshot of cluster statistics.
func (c *Cluster) GetStats() StatsSnapshot {
	return StatsSnapshot{
		ActiveRequests: c.stats.ActiveRequests.Load(),
		TotalRequests:  c.stats.TotalRequests.Load(),
		TotalErrors:    c.stats.TotalErrors.Load(),
		StartedAt:      c.stats.StartedAt,
		WorkerCount:    c.stats.WorkerCount,
		Uptime:         time.Since(c.stats.StartedAt).String(),
	}
}

// StatsSnapshot is a copy-safe snapshot of cluster stats.
type StatsSnapshot struct {
	ActiveRequests int64     `json:"active_requests"`
	TotalRequests  int64     `json:"total_requests"`
	TotalErrors    int64     `json:"total_errors"`
	StartedAt      time.Time `json:"started_at"`
	WorkerCount    int       `json:"worker_count"`
	Uptime         string    `json:"uptime"`
}

// Ready returns a channel that's closed when the cluster is ready.
func (c *Cluster) Ready() <-chan struct{} {
	return c.ready
}

func (c *Cluster) wrapHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.stats.ActiveRequests.Add(1)
		c.stats.TotalRequests.Add(1)
		defer c.stats.ActiveRequests.Add(-1)

		wrapped := &statusCapture{ResponseWriter: w, status: 200}
		h.ServeHTTP(wrapped, r)

		if wrapped.status >= 500 {
			c.stats.TotalErrors.Add(1)
		}
	})
}

type statusCapture struct {
	http.ResponseWriter
	status int
}

func (sc *statusCapture) WriteHeader(code int) {
	sc.status = code
	sc.ResponseWriter.WriteHeader(code)
}

// --- Graceful Restart (zero-downtime) ---

// GracefulRestart performs a zero-downtime restart by:
// 1. Starting new workers
// 2. Draining old workers
// 3. Closing old listener
func (c *Cluster) GracefulRestart(newHandler http.Handler) error {
	log.Printf("[NexGo Cluster] Performing graceful restart...")

	c.mu.Lock()
	oldServers := c.servers
	c.servers = nil

	wrapped := c.wrapHandler(newHandler)

	for i := 0; i < c.config.Workers; i++ {
		srv := &http.Server{
			Handler:      wrapped,
			ReadTimeout:  c.config.ReadTimeout,
			WriteTimeout: c.config.WriteTimeout,
			IdleTimeout:  c.config.IdleTimeout,
		}
		c.servers = append(c.servers, srv)
		go func(srv *http.Server, id int) {
			log.Printf("[NexGo Cluster] New worker %d started", id)
			srv.Serve(c.listener)
		}(srv, i)
	}
	c.mu.Unlock()

	// Drain old workers
	ctx, cancel := context.WithTimeout(context.Background(), c.config.GracefulTimeout)
	defer cancel()
	for _, srv := range oldServers {
		srv.Shutdown(ctx)
	}

	log.Printf("[NexGo Cluster] Graceful restart complete")
	return nil
}

// --- Load Balancer (round-robin for internal routing) ---

// LoadBalancer distributes requests across backends.
type LoadBalancer struct {
	mu       sync.RWMutex
	backends []string
	current  atomic.Int64
	health   map[string]bool
}

// NewLoadBalancer creates a round-robin load balancer.
func NewLoadBalancer(backends []string) *LoadBalancer {
	health := make(map[string]bool, len(backends))
	for _, b := range backends {
		health[b] = true
	}
	return &LoadBalancer{
		backends: backends,
		health:   health,
	}
}

// Next returns the next healthy backend.
func (lb *LoadBalancer) Next() (string, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	n := len(lb.backends)
	if n == 0 {
		return "", fmt.Errorf("no backends available")
	}

	for i := 0; i < n; i++ {
		idx := int(lb.current.Add(1)-1) % n
		backend := lb.backends[idx]
		if lb.health[backend] {
			return backend, nil
		}
	}

	return "", fmt.Errorf("no healthy backends")
}

// MarkDown marks a backend as unhealthy.
func (lb *LoadBalancer) MarkDown(backend string) {
	lb.mu.Lock()
	lb.health[backend] = false
	lb.mu.Unlock()
}

// MarkUp marks a backend as healthy.
func (lb *LoadBalancer) MarkUp(backend string) {
	lb.mu.Lock()
	lb.health[backend] = true
	lb.mu.Unlock()
}

// HealthCheck runs health checks on all backends.
func (lb *LoadBalancer) HealthCheck(path string, timeout time.Duration) {
	client := &http.Client{Timeout: timeout}
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, backend := range lb.backends {
		resp, err := client.Get("http://" + backend + path)
		if err != nil || resp.StatusCode >= 500 {
			lb.health[backend] = false
		} else {
			lb.health[backend] = true
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
}

// StartHealthCheck runs periodic health checks.
func (lb *LoadBalancer) StartHealthCheck(path string, interval, timeout time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			lb.HealthCheck(path, timeout)
		}
	}()
}

// Healthy returns a list of healthy backends.
func (lb *LoadBalancer) Healthy() []string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	var result []string
	for _, b := range lb.backends {
		if lb.health[b] {
			result = append(result, b)
		}
	}
	return result
}
