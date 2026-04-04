// Package metrics provides Prometheus-compatible metrics and observability.
package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Counter is a monotonically increasing counter.
type Counter struct {
	name  string
	value atomic.Int64
	labels map[string]string
}

// NewCounter creates a named counter.
func NewCounter(name string) *Counter {
	c := &Counter{name: name, labels: make(map[string]string)}
	registry.register(c)
	return c
}

// Inc increments the counter by 1.
func (c *Counter) Inc() { c.value.Add(1) }

// Add increments the counter by n.
func (c *Counter) Add(n int64) { c.value.Add(n) }

// Value returns the current count.
func (c *Counter) Value() int64 { return c.value.Load() }

// WithLabel returns a labeled counter.
func (c *Counter) WithLabel(key, value string) *Counter {
	newLabels := make(map[string]string, len(c.labels)+1)
	for k, v := range c.labels {
		newLabels[k] = v
	}
	newLabels[key] = value
	nc := &Counter{
		name:   c.name,
		labels: newLabels,
	}
	registry.register(nc)
	return nc
}

// Gauge is a value that can go up and down.
type Gauge struct {
	name   string
	value  atomic.Int64
	labels map[string]string
}

// NewGauge creates a named gauge.
func NewGauge(name string) *Gauge {
	g := &Gauge{name: name, labels: make(map[string]string)}
	registry.registerGauge(g)
	return g
}

// Set sets the gauge value.
func (g *Gauge) Set(v int64) { g.value.Store(v) }

// Inc increments the gauge by 1.
func (g *Gauge) Inc() { g.value.Add(1) }

// Dec decrements the gauge by 1.
func (g *Gauge) Dec() { g.value.Add(-1) }

// Value returns the current value.
func (g *Gauge) Value() int64 { return g.value.Load() }

// Histogram tracks the distribution of observed values.
type Histogram struct {
	name    string
	mu      sync.Mutex
	count   int64
	sum     float64
	buckets []float64
	counts  []int64
	labels  map[string]string
}

// NewHistogram creates a histogram with the given bucket boundaries.
func NewHistogram(name string, buckets []float64) *Histogram {
	sort.Float64s(buckets)
	h := &Histogram{
		name:    name,
		buckets: buckets,
		counts:  make([]int64, len(buckets)+1), // +1 for +Inf
		labels:  make(map[string]string),
	}
	registry.registerHistogram(h)
	return h
}

// DefaultHTTPBuckets are common HTTP latency buckets in seconds.
var DefaultHTTPBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// Observe records a value in the histogram.
func (h *Histogram) Observe(v float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.count++
	h.sum += v
	for i, bound := range h.buckets {
		if v <= bound {
			h.counts[i]++
		}
	}
	h.counts[len(h.buckets)]++ // +Inf
}

// ObserveDuration records a duration in seconds.
func (h *Histogram) ObserveDuration(start time.Time) {
	h.Observe(time.Since(start).Seconds())
}

// Summary returns count, sum, and avg.
func (h *Histogram) Summary() (count int64, sum, avg float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	count = h.count
	sum = h.sum
	if count > 0 {
		avg = sum / float64(count)
	}
	return
}

// --- Timer ---

// Timer measures elapsed time.
type Timer struct {
	histogram *Histogram
	start     time.Time
}

// NewTimer starts a timer that records to a histogram.
func NewTimer(h *Histogram) *Timer {
	return &Timer{histogram: h, start: time.Now()}
}

// Stop records the elapsed time.
func (t *Timer) Stop() time.Duration {
	d := time.Since(t.start)
	t.histogram.Observe(d.Seconds())
	return d
}

// --- Registry ---

type metricsRegistry struct {
	mu         sync.RWMutex
	counters   []*Counter
	gauges     []*Gauge
	histograms []*Histogram
}

var registry = &metricsRegistry{}

func (r *metricsRegistry) register(c *Counter) {
	r.mu.Lock()
	r.counters = append(r.counters, c)
	r.mu.Unlock()
}

func (r *metricsRegistry) registerGauge(g *Gauge) {
	r.mu.Lock()
	r.gauges = append(r.gauges, g)
	r.mu.Unlock()
}

func (r *metricsRegistry) registerHistogram(h *Histogram) {
	r.mu.Lock()
	r.histograms = append(r.histograms, h)
	r.mu.Unlock()
}

// Handler returns an HTTP handler that exports metrics in Prometheus text format.
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		registry.mu.RLock()
		defer registry.mu.RUnlock()

		var sb strings.Builder

		for _, c := range registry.counters {
			labels := formatLabels(c.labels)
			sb.WriteString(fmt.Sprintf("# TYPE %s counter\n", c.name))
			sb.WriteString(fmt.Sprintf("%s%s %d\n", c.name, labels, c.Value()))
		}

		for _, g := range registry.gauges {
			labels := formatLabels(g.labels)
			sb.WriteString(fmt.Sprintf("# TYPE %s gauge\n", g.name))
			sb.WriteString(fmt.Sprintf("%s%s %d\n", g.name, labels, g.Value()))
		}

		for _, h := range registry.histograms {
			h.mu.Lock()
			sb.WriteString(fmt.Sprintf("# TYPE %s histogram\n", h.name))
			for i, bound := range h.buckets {
				sb.WriteString(fmt.Sprintf("%s_bucket{le=\"%.3f\"} %d\n", h.name, bound, h.counts[i]))
			}
			sb.WriteString(fmt.Sprintf("%s_bucket{le=\"+Inf\"} %d\n", h.name, h.counts[len(h.buckets)]))
			sb.WriteString(fmt.Sprintf("%s_sum %f\n", h.name, h.sum))
			sb.WriteString(fmt.Sprintf("%s_count %d\n", h.name, h.count))
			h.mu.Unlock()
		}

		w.Write([]byte(sb.String()))
	}
}

// JSONHandler returns metrics as JSON.
func JSONHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		registry.mu.RLock()
		defer registry.mu.RUnlock()

		data := map[string]interface{}{}
		for _, c := range registry.counters {
			data[c.name] = c.Value()
		}
		for _, g := range registry.gauges {
			data[g.name] = g.Value()
		}
		for _, h := range registry.histograms {
			count, sum, avg := h.Summary()
			data[h.name] = map[string]interface{}{
				"count": count,
				"sum":   sum,
				"avg":   avg,
			}
		}

		json.NewEncoder(w).Encode(data)
	}
}

func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	parts := make([]string, 0, len(labels))
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, k, v))
	}
	sort.Strings(parts)
	return "{" + strings.Join(parts, ",") + "}"
}

// --- Built-in HTTP metrics ---

var (
	httpRequestsTotal   = NewCounter("http_requests_total")
	httpRequestDuration = NewHistogram("http_request_duration_seconds", DefaultHTTPBuckets)
	httpActiveRequests  = NewGauge("http_active_requests")
	httpResponseSize    = NewHistogram("http_response_size_bytes", []float64{100, 1000, 10000, 100000, 1000000})
)

// HTTPMiddleware returns middleware that collects HTTP metrics.
func HTTPMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			httpActiveRequests.Inc()
			defer httpActiveRequests.Dec()

			start := time.Now()
			wrapped := &metricsWriter{ResponseWriter: w, status: 200}
			next(wrapped, r)

			httpRequestsTotal.Inc()
			httpRequestDuration.ObserveDuration(start)
			httpResponseSize.Observe(float64(wrapped.bytes))
		}
	}
}

type metricsWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (mw *metricsWriter) WriteHeader(code int) {
	mw.status = code
	mw.ResponseWriter.WriteHeader(code)
}

func (mw *metricsWriter) Write(b []byte) (int, error) {
	n, err := mw.ResponseWriter.Write(b)
	mw.bytes += n
	return n, err
}
