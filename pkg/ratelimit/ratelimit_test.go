package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTokenBucketAllow(t *testing.T) {
	l := New(5, time.Second, 5)

	for i := 0; i < 5; i++ {
		if !l.Allow("test-ip") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	if l.Allow("test-ip") {
		t.Error("6th request should be denied")
	}
}

func TestTokenBucketRefill(t *testing.T) {
	l := New(100, time.Second, 100)

	// Exhaust tokens
	for i := 0; i < 100; i++ {
		l.Allow("refill-ip")
	}

	if l.Allow("refill-ip") {
		t.Error("should be denied after exhausting tokens")
	}

	// Wait for refill
	time.Sleep(50 * time.Millisecond)

	if !l.Allow("refill-ip") {
		t.Error("should be allowed after refill time")
	}
}

func TestDifferentKeysIndependent(t *testing.T) {
	l := New(2, time.Second, 2)

	l.Allow("ip-a")
	l.Allow("ip-a")

	if l.Allow("ip-a") {
		t.Error("ip-a should be limited")
	}

	if !l.Allow("ip-b") {
		t.Error("ip-b should not be limited")
	}
}

func TestReset(t *testing.T) {
	l := New(2, time.Second, 2)

	l.Allow("reset-ip")
	l.Allow("reset-ip")

	l.Reset("reset-ip")

	if !l.Allow("reset-ip") {
		t.Error("should be allowed after reset")
	}
}

func TestMiddleware(t *testing.T) {
	l := New(2, time.Second, 2)
	handler := l.Middleware()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		w := httptest.NewRecorder()
		handler(w, req)
		if w.Code != 200 {
			t.Errorf("request %d: got %d, want 200", i+1, w.Code)
		}
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("3rd request: got %d, want %d", w.Code, http.StatusTooManyRequests)
	}

	if w.Header().Get("Retry-After") == "" {
		t.Error("Retry-After header should be set")
	}
}

func TestPerMinutePreset(t *testing.T) {
	l := PerMinute(10)
	for i := 0; i < 10; i++ {
		if !l.Allow("test") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}
}

func TestSlidingWindow(t *testing.T) {
	sw := NewSlidingWindow(3, time.Second)

	for i := 0; i < 3; i++ {
		if !sw.Allow("test") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	if sw.Allow("test") {
		t.Error("4th request should be denied")
	}
}

func TestSlidingWindowMiddleware(t *testing.T) {
	sw := NewSlidingWindow(1, time.Second)
	handler := sw.Middleware()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "5.6.7.8:5678"
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != 200 {
		t.Errorf("first request: got %d, want 200", w.Code)
	}

	w = httptest.NewRecorder()
	handler(w, req)
	if w.Code != 429 {
		t.Errorf("second request: got %d, want 429", w.Code)
	}
}
