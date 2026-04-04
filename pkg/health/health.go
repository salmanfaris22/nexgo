// Package health provides health check and readiness endpoints.
package health

import (
	"encoding/json"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// Status represents a health check status.
type Status string

const (
	StatusUp   Status = "up"
	StatusDown Status = "down"
)

// Check is a function that verifies a dependency is healthy.
type Check func() error

// Response is the JSON health check response.
type Response struct {
	Status    Status            `json:"status"`
	Uptime    string            `json:"uptime"`
	Timestamp string            `json:"timestamp"`
	Version   string            `json:"version,omitempty"`
	Checks    map[string]string `json:"checks,omitempty"`
	System    *SystemInfo       `json:"system,omitempty"`
}

// SystemInfo contains runtime system information.
type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	NumGoroutine int    `json:"goroutines"`
	NumCPU       int    `json:"cpus"`
	MemAllocMB   uint64 `json:"mem_alloc_mb"`
	MemSysMB     uint64 `json:"mem_sys_mb"`
}

// Health manages health check endpoints.
type Health struct {
	mu        sync.RWMutex
	checks    map[string]Check
	startTime time.Time
	version   string
}

// New creates a health check manager.
func New(version string) *Health {
	return &Health{
		checks:    make(map[string]Check),
		startTime: time.Now(),
		version:   version,
	}
}

// AddCheck registers a named health check.
func (h *Health) AddCheck(name string, check Check) {
	h.mu.Lock()
	h.checks[name] = check
	h.mu.Unlock()
}

// Handler returns an HTTP handler for the /health endpoint.
func (h *Health) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := h.run(false)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache, no-store")

		status := http.StatusOK
		if resp.Status == StatusDown {
			status = http.StatusServiceUnavailable
		}
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(resp)
	}
}

// ReadyHandler returns an HTTP handler for the /ready endpoint.
// Returns 200 if all checks pass, 503 otherwise.
func (h *Health) ReadyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := h.run(true)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache, no-store")

		status := http.StatusOK
		if resp.Status == StatusDown {
			status = http.StatusServiceUnavailable
		}
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(resp)
	}
}

// LiveHandler returns a simple liveness probe (always 200 if server is running).
func (h *Health) LiveHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache, no-store")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"uptime": time.Since(h.startTime).Round(time.Second).String(),
		})
	}
}

func (h *Health) run(includeSystem bool) Response {
	h.mu.RLock()
	defer h.mu.RUnlock()

	resp := Response{
		Status:    StatusUp,
		Uptime:    time.Since(h.startTime).Round(time.Second).String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   h.version,
		Checks:    make(map[string]string),
	}

	for name, check := range h.checks {
		if err := check(); err != nil {
			resp.Checks[name] = "down: " + err.Error()
			resp.Status = StatusDown
		} else {
			resp.Checks[name] = "up"
		}
	}

	if includeSystem {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		resp.System = &SystemInfo{
			GoVersion:    runtime.Version(),
			NumGoroutine: runtime.NumGoroutine(),
			NumCPU:       runtime.NumCPU(),
			MemAllocMB:   m.Alloc / 1024 / 1024,
			MemSysMB:     m.Sys / 1024 / 1024,
		}
	}

	return resp
}

// RegisterEndpoints registers /health, /ready, and /live on a mux.
func (h *Health) RegisterEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.Handler())
	mux.HandleFunc("/ready", h.ReadyHandler())
	mux.HandleFunc("/live", h.LiveHandler())
}
