package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCounter(t *testing.T) {
	c := &Counter{name: "test_counter", labels: make(map[string]string)}
	c.Inc()
	c.Inc()
	c.Add(5)

	if c.Value() != 7 {
		t.Errorf("Value = %d, want 7", c.Value())
	}
}

func TestGauge(t *testing.T) {
	g := &Gauge{name: "test_gauge", labels: make(map[string]string)}
	g.Set(10)
	if g.Value() != 10 {
		t.Errorf("Value = %d, want 10", g.Value())
	}
	g.Inc()
	if g.Value() != 11 {
		t.Errorf("After Inc = %d, want 11", g.Value())
	}
	g.Dec()
	if g.Value() != 10 {
		t.Errorf("After Dec = %d, want 10", g.Value())
	}
}

func TestHistogram(t *testing.T) {
	h := &Histogram{
		name:    "test_hist",
		buckets: []float64{0.1, 0.5, 1.0},
		counts:  make([]int64, 4),
		labels:  make(map[string]string),
	}
	h.Observe(0.05)
	h.Observe(0.3)
	h.Observe(0.8)
	h.Observe(2.0)

	count, sum, avg := h.Summary()
	if count != 4 {
		t.Errorf("count = %d, want 4", count)
	}
	if sum != 3.15 {
		t.Errorf("sum = %f, want 3.15", sum)
	}
	if avg < 0.78 || avg > 0.79 {
		t.Errorf("avg = %f, want ~0.7875", avg)
	}
}

func TestPrometheusHandler(t *testing.T) {
	// Note: uses global registry so test is not isolated
	// In a real project you'd use a per-test registry
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	Handler()(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", ct)
	}
}

func TestJSONMetricsHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/metrics/json", nil)
	w := httptest.NewRecorder()
	JSONHandler()(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestHTTPMiddleware(t *testing.T) {
	handler := HTTPMiddleware()(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if w.Body.String() != "OK" {
		t.Errorf("body = %q, want OK", w.Body.String())
	}
}
