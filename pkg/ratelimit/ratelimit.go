// Package ratelimit provides token bucket and sliding window rate limiting middleware.
package ratelimit

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Limiter tracks rate limits per key (typically IP address).
type Limiter struct {
	mu       sync.Mutex
	visitors map[string]*bucket
	rate     int           // requests per window
	window   time.Duration // time window
	burst    int           // max burst
}

type bucket struct {
	tokens    float64
	lastCheck time.Time
}

// New creates a rate limiter. rate is requests per window, burst is max burst size.
func New(rate int, window time.Duration, burst int) *Limiter {
	if burst < rate {
		burst = rate
	}
	l := &Limiter{
		visitors: make(map[string]*bucket),
		rate:     rate,
		window:   window,
		burst:    burst,
	}
	go l.cleanup()
	return l
}

// Allow checks if a request from the given key should be allowed.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	b, exists := l.visitors[key]
	if !exists {
		b = &bucket{
			tokens:    float64(l.burst - 1),
			lastCheck: time.Now(),
		}
		l.visitors[key] = b
		return true
	}

	elapsed := time.Since(b.lastCheck).Seconds()
	b.lastCheck = time.Now()

	// Refill tokens
	refillRate := float64(l.rate) / l.window.Seconds()
	b.tokens += elapsed * refillRate
	if b.tokens > float64(l.burst) {
		b.tokens = float64(l.burst)
	}

	if b.tokens < 1 {
		return false
	}

	b.tokens--
	return true
}

// Remaining returns remaining requests for a key.
func (l *Limiter) Remaining(key string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, exists := l.visitors[key]
	if !exists {
		return l.burst
	}
	return int(b.tokens)
}

// Reset clears rate limit state for a key.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	delete(l.visitors, key)
	l.mu.Unlock()
}

func (l *Limiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		cutoff := time.Now().Add(-l.window * 2)
		for key, b := range l.visitors {
			if b.lastCheck.Before(cutoff) {
				delete(l.visitors, key)
			}
		}
		l.mu.Unlock()
	}
}

// Middleware returns HTTP middleware that rate limits by client IP.
func (l *Limiter) Middleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := clientIP(r)
			if !l.Allow(key) {
				w.Header().Set("Retry-After", strconv.Itoa(int(l.window.Seconds())))
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(l.rate))
				w.Header().Set("X-RateLimit-Remaining", "0")
				http.Error(w, `{"error":"rate limit exceeded","status":429}`, http.StatusTooManyRequests)
				return
			}
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(l.rate))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(l.Remaining(key)))
			next(w, r)
		}
	}
}

// KeyFunc allows custom key extraction.
func (l *Limiter) MiddlewareWithKey(keyFn func(*http.Request) string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			if !l.Allow(key) {
				w.Header().Set("Retry-After", strconv.Itoa(int(l.window.Seconds())))
				http.Error(w, `{"error":"rate limit exceeded","status":429}`, http.StatusTooManyRequests)
				return
			}
			next(w, r)
		}
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	idx := strings.LastIndex(r.RemoteAddr, ":")
	if idx > 0 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}

// --- Preset rate limiters ---

// PerMinute creates a limiter allowing n requests per minute.
func PerMinute(n int) *Limiter {
	return New(n, time.Minute, n*2)
}

// PerSecond creates a limiter allowing n requests per second.
func PerSecond(n int) *Limiter {
	return New(n, time.Second, n*3)
}

// PerHour creates a limiter allowing n requests per hour.
func PerHour(n int) *Limiter {
	return New(n, time.Hour, n*2)
}

// --- Sliding Window Rate Limiter ---

// SlidingWindow implements a sliding window counter rate limiter.
type SlidingWindow struct {
	mu      sync.Mutex
	windows map[string]*windowEntry
	limit   int
	window  time.Duration
}

type windowEntry struct {
	current  int
	previous int
	start    time.Time
}

// NewSlidingWindow creates a sliding window rate limiter.
func NewSlidingWindow(limit int, window time.Duration) *SlidingWindow {
	sw := &SlidingWindow{
		windows: make(map[string]*windowEntry),
		limit:   limit,
		window:  window,
	}
	go sw.cleanup()
	return sw
}

// Allow checks if a request should be allowed.
func (sw *SlidingWindow) Allow(key string) bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	e, exists := sw.windows[key]
	if !exists {
		sw.windows[key] = &windowEntry{current: 1, start: now}
		return true
	}

	elapsed := now.Sub(e.start)
	if elapsed >= sw.window {
		e.previous = e.current
		e.current = 0
		e.start = now
		elapsed = 0
	}

	// Calculate weighted count
	weight := 1.0 - (float64(elapsed) / float64(sw.window))
	count := float64(e.previous)*weight + float64(e.current)

	if count >= float64(sw.limit) {
		return false
	}

	e.current++
	return true
}

// Middleware returns HTTP middleware using sliding window.
func (sw *SlidingWindow) Middleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := clientIP(r)
			if !sw.Allow(key) {
				w.Header().Set("Retry-After", strconv.Itoa(int(sw.window.Seconds())))
				http.Error(w, `{"error":"rate limit exceeded","status":429}`, http.StatusTooManyRequests)
				return
			}
			next(w, r)
		}
	}
}

func (sw *SlidingWindow) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		sw.mu.Lock()
		cutoff := time.Now().Add(-sw.window * 3)
		for key, e := range sw.windows {
			if e.start.Before(cutoff) {
				delete(sw.windows, key)
			}
		}
		sw.mu.Unlock()
	}
}
