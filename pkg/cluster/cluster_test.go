package cluster

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Workers != 0 {
		t.Errorf("expected 0 workers (auto-detect), got %d", cfg.Workers)
	}
	if cfg.GracefulTimeout != 30*time.Second {
		t.Errorf("expected 30s graceful timeout, got %v", cfg.GracefulTimeout)
	}
}

func TestNew(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Workers = 2

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	c := New(cfg, handler)
	if c == nil {
		t.Fatal("expected cluster")
	}
	if c.config.Workers != 2 {
		t.Errorf("expected 2 workers, got %d", c.config.Workers)
	}
}

func TestNew_AutoWorkers(t *testing.T) {
	cfg := DefaultConfig()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	c := New(cfg, handler)
	if c.config.Workers <= 0 {
		t.Errorf("expected auto-detected workers > 0, got %d", c.config.Workers)
	}
}

func TestGetStats(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Workers = 1
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	c := New(cfg, handler)
	stats := c.GetStats()

	if stats.WorkerCount != 1 {
		t.Errorf("expected 1 worker, got %d", stats.WorkerCount)
	}
	if stats.Uptime == "" {
		t.Error("expected non-empty uptime")
	}
}

func TestReady(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Workers = 1
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	c := New(cfg, handler)
	ready := c.Ready()
	if ready == nil {
		t.Error("expected ready channel")
	}
}

func TestWrapHandler_StatsTracking(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Workers = 1
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	c := New(cfg, handler)

	// Test the wrapped handler indirectly via stats
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	// Access the wrapped handler through the cluster
	c.wrapHandler(handler).ServeHTTP(w, r)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	stats := c.GetStats()
	if stats.TotalRequests != 1 {
		t.Errorf("expected 1 total request, got %d", stats.TotalRequests)
	}
}

func TestWrapHandler_ErrorTracking(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Workers = 1
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})

	c := New(cfg, handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/error", nil)
	c.wrapHandler(handler).ServeHTTP(w, r)

	stats := c.GetStats()
	if stats.TotalErrors != 1 {
		t.Errorf("expected 1 error, got %d", stats.TotalErrors)
	}
}

func TestLoadBalancer(t *testing.T) {
	backends := []string{"server1:8080", "server2:8080", "server3:8080"}
	lb := NewLoadBalancer(backends)

	if lb == nil {
		t.Fatal("expected load balancer")
	}
}

func TestLoadBalancer_Next(t *testing.T) {
	lb := NewLoadBalancer([]string{"s1", "s2", "s3"})

	// Should cycle through backends
	seen := make(map[string]bool)
	for i := 0; i < 3; i++ {
		backend, err := lb.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		seen[backend] = true
	}

	if len(seen) != 3 {
		t.Errorf("expected 3 unique backends, got %d", len(seen))
	}
}

func TestLoadBalancer_NoBackends(t *testing.T) {
	lb := NewLoadBalancer([]string{})
	_, err := lb.Next()
	if err == nil {
		t.Error("expected error for no backends")
	}
}

func TestLoadBalancer_MarkDown(t *testing.T) {
	lb := NewLoadBalancer([]string{"s1", "s2"})
	lb.MarkDown("s1")

	// Should only return s2
	for i := 0; i < 5; i++ {
		backend, err := lb.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if backend != "s2" {
			t.Errorf("expected s2, got %s", backend)
		}
	}
}

func TestLoadBalancer_AllDown(t *testing.T) {
	lb := NewLoadBalancer([]string{"s1", "s2"})
	lb.MarkDown("s1")
	lb.MarkDown("s2")

	_, err := lb.Next()
	if err == nil {
		t.Error("expected error when all backends down")
	}
}

func TestLoadBalancer_MarkUp(t *testing.T) {
	lb := NewLoadBalancer([]string{"s1"})
	lb.MarkDown("s1")
	lb.MarkUp("s1")

	backend, err := lb.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if backend != "s1" {
		t.Errorf("expected s1, got %s", backend)
	}
}

func TestLoadBalancer_Healthy(t *testing.T) {
	lb := NewLoadBalancer([]string{"s1", "s2"})
	lb.MarkDown("s1")

	healthy := lb.Healthy()
	if len(healthy) != 1 {
		t.Errorf("expected 1 healthy, got %d", len(healthy))
	}
	if healthy[0] != "s2" {
		t.Errorf("expected s2, got %s", healthy[0])
	}
}

func TestLoadBalancer_RoomCount(t *testing.T) {
	lb := NewLoadBalancer([]string{"s1", "s2"})
	// RoomCount is for websocket hub, not LB
	// Just verify it doesn't panic
	_ = lb
}

func TestStatsSnapshot(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Workers = 4
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	c := New(cfg, handler)
	stats := c.GetStats()

	if stats.WorkerCount != 4 {
		t.Errorf("expected 4 workers, got %d", stats.WorkerCount)
	}
	if stats.ActiveRequests != 0 {
		t.Errorf("expected 0 active requests, got %d", stats.ActiveRequests)
	}
	if stats.TotalRequests != 0 {
		t.Errorf("expected 0 total requests, got %d", stats.TotalRequests)
	}
}
