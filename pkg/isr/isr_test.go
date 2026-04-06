package isr

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestISRNew(t *testing.T) {
	i := New(60 * time.Second)
	if i.revalidate != 60*time.Second {
		t.Errorf("expected 60s, got %v", i.revalidate)
	}
}

func TestISRServe(t *testing.T) {
	i := New(1 * time.Second)

	callCount := 0
	generate := func() (int, http.Header, []byte, error) {
		callCount++
		return 200, http.Header{"X-Custom": []string{"val"}}, []byte("content"), nil
	}

	// First request - MISS
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("GET", "/page", nil)
	i.Serve(w1, r1, generate)

	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
	if w1.Body.String() != "content" {
		t.Errorf("expected 'content', got %s", w1.Body.String())
	}
	if w1.Header().Get("X-ISR") != "MISS" {
		t.Errorf("expected MISS, got %s", w1.Header().Get("X-ISR"))
	}

	// Second request - HIT (within revalidate window)
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/page", nil)
	i.Serve(w2, r2, generate)

	if callCount != 1 {
		t.Error("expected no new call (cached)")
	}
	if w2.Header().Get("X-ISR") != "STALE" {
		t.Errorf("expected STALE, got %s", w2.Header().Get("X-ISR"))
	}
}

func TestISRRevalidate(t *testing.T) {
	i := New(60 * time.Second)

	callCount := 0
	generate := func() (int, http.Header, []byte, error) {
		callCount++
		return 200, nil, []byte("v1"), nil
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/page", nil)
	i.Serve(w, r, generate)

	// Trigger background revalidation
	generate2 := func() (int, http.Header, []byte, error) {
		callCount++
		return 200, nil, []byte("v2"), nil
	}
	i.Revalidate("/page", generate2)

	// Wait for background regen
	time.Sleep(50 * time.Millisecond)

	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestISRPurge(t *testing.T) {
	i := New(60 * time.Second)

	generate := func() (int, http.Header, []byte, error) {
		return 200, nil, []byte("data"), nil
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/page", nil)
	i.Serve(w, r, generate)

	if !i.IsCached("/page") {
		t.Error("expected page to be cached")
	}

	i.Purge("/page")
	if i.IsCached("/page") {
		t.Error("expected page to be purged")
	}
}

func TestISRPurgeAll(t *testing.T) {
	i := New(60 * time.Second)

	generate := func() (int, http.Header, []byte, error) {
		return 200, nil, []byte("data"), nil
	}

	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("GET", "/page1", nil)
	i.Serve(w1, r1, generate)

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/page2", nil)
	i.Serve(w2, r2, generate)

	i.PurgeAll()
	if i.IsCached("/page1") || i.IsCached("/page2") {
		t.Error("expected all pages purged")
	}
}

func TestISRAge(t *testing.T) {
	i := New(60 * time.Second)

	generate := func() (int, http.Header, []byte, error) {
		return 200, nil, []byte("data"), nil
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/page", nil)
	i.Serve(w, r, generate)

	age := i.Age("/page")
	if age == 0 {
		t.Error("expected non-zero age")
	}

	ageMissing := i.Age("/missing")
	if ageMissing != 0 {
		t.Error("expected 0 age for missing page")
	}
}

func TestISRGenerateError(t *testing.T) {
	i := New(60 * time.Second)

	generate := func() (int, http.Header, []byte, error) {
		return 0, nil, nil, nil
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/error", nil)
	i.Serve(w, r, generate)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestGlobalISR(t *testing.T) {
	generate := func() (int, http.Header, []byte, error) {
		return 200, nil, []byte("global"), nil
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/global-test", nil)
	Serve(w, r, generate)

	if w.Body.String() != "global" {
		t.Errorf("expected 'global', got %s", w.Body.String())
	}

	Purge("/global-test")
}

func TestSetRevalidate(t *testing.T) {
	SetRevalidate(30 * time.Second)
	if globalISR.revalidate != 30*time.Second {
		t.Errorf("expected 30s, got %v", globalISR.revalidate)
	}
}
