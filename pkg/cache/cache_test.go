package cache

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCacheNew(t *testing.T) {
	c := New(5 * time.Minute)
	defer c.Stop()
	if c.defaultTTL != 5*time.Minute {
		t.Errorf("expected 5m TTL, got %v", c.defaultTTL)
	}
}

func TestCacheSetGet(t *testing.T) {
	c := New(5 * time.Minute)
	defer c.Stop()

	c.Set("key1", 200, http.Header{"X-Test": []string{"val"}}, []byte("body"))
	code, headers, body, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if code != 200 {
		t.Errorf("expected 200, got %d", code)
	}
	if headers.Get("X-Test") != "val" {
		t.Errorf("expected X-Test=val, got %s", headers.Get("X-Test"))
	}
	if string(body) != "body" {
		t.Errorf("expected 'body', got %s", string(body))
	}
}

func TestCacheSetWithTTL(t *testing.T) {
	c := New(5 * time.Minute)
	defer c.Stop()

	c.SetWithTTL("key", 200, nil, []byte("data"), 50*time.Millisecond)
	_, _, _, ok := c.Get("key")
	if !ok {
		t.Error("expected cache hit immediately")
	}

	time.Sleep(100 * time.Millisecond)
	_, _, _, ok = c.Get("key")
	if ok {
		t.Error("expected cache miss after TTL expired")
	}
}

func TestCacheDelete(t *testing.T) {
	c := New(5 * time.Minute)
	defer c.Stop()

	c.Set("key", 200, nil, []byte("data"))
	c.Delete("key")
	_, _, _, ok := c.Get("key")
	if ok {
		t.Error("expected cache miss after delete")
	}
}

func TestCacheDeletePrefix(t *testing.T) {
	c := New(5 * time.Minute)
	defer c.Stop()

	c.Set("api/users", 200, nil, []byte("users"))
	c.Set("api/posts", 200, nil, []byte("posts"))
	c.Set("home", 200, nil, []byte("home"))

	c.DeletePrefix("api/")
	_, _, _, ok1 := c.Get("api/users")
	_, _, _, ok2 := c.Get("api/posts")
	_, _, _, ok3 := c.Get("home")

	if ok1 {
		t.Error("expected api/users deleted")
	}
	if ok2 {
		t.Error("expected api/posts deleted")
	}
	if !ok3 {
		t.Error("expected home to still exist")
	}
}

func TestCacheClear(t *testing.T) {
	c := New(5 * time.Minute)
	defer c.Stop()

	c.Set("a", 200, nil, []byte("a"))
	c.Set("b", 200, nil, []byte("b"))
	c.Clear()

	if c.Len() != 0 {
		t.Errorf("expected 0 items after clear, got %d", c.Len())
	}
}

func TestCacheLen(t *testing.T) {
	c := New(5 * time.Minute)
	defer c.Stop()

	if c.Len() != 0 {
		t.Error("expected 0 items initially")
	}
	c.Set("a", 200, nil, []byte("a"))
	c.Set("b", 200, nil, []byte("b"))
	if c.Len() != 2 {
		t.Errorf("expected 2 items, got %d", c.Len())
	}
}

func TestCacheKey(t *testing.T) {
	r1 := httptest.NewRequest("GET", "/test", nil)
	r2 := httptest.NewRequest("GET", "/test", nil)
	r3 := httptest.NewRequest("GET", "/other", nil)

	k1 := Key(r1)
	k2 := Key(r2)
	k3 := Key(r3)

	if k1 != k2 {
		t.Error("same URL should produce same key")
	}
	if k1 == k3 {
		t.Error("different URLs should produce different keys")
	}
}

func TestCacheMiddleware(t *testing.T) {
	c := New(5 * time.Minute)
	defer c.Stop()

	callCount := 0
	handler := Middleware(c, 5*time.Minute)(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(200)
		w.Write([]byte("response"))
	})

	// First request - MISS
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("GET", "/test", nil)
	handler(w1, r1)
	if callCount != 1 {
		t.Errorf("expected handler called once, got %d", callCount)
	}
	if w1.Header().Get("X-Cache") != "MISS" {
		t.Errorf("expected MISS, got %s", w1.Header().Get("X-Cache"))
	}

	// Second request - HIT
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/test", nil)
	handler(w2, r2)
	if callCount != 1 {
		t.Error("expected handler NOT called again (cache hit)")
	}
	if w2.Header().Get("X-Cache") != "HIT" {
		t.Errorf("expected HIT, got %s", w2.Header().Get("X-Cache"))
	}
}

func TestCacheMiddlewareNonGet(t *testing.T) {
	c := New(5 * time.Minute)
	defer c.Stop()

	callCount := 0
	handler := Middleware(c)(func(w http.ResponseWriter, r *http.Request) {
		callCount++
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	handler(w, r)
	if callCount != 1 {
		t.Error("expected handler to be called for POST")
	}
}

func TestGlobalCacheHelpers(t *testing.T) {
	CacheClear()
	CacheSet("gkey", 200, nil, []byte("data"))
	code, _, body, ok := CacheGet("gkey")
	if !ok || code != 200 || string(body) != "data" {
		t.Error("expected cached data")
	}
	CacheDelete("gkey")
	_, _, _, ok = CacheGet("gkey")
	if ok {
		t.Error("expected miss after delete")
	}
}

func TestSetGlobalTTL(t *testing.T) {
	SetGlobalTTL(10 * time.Minute)
	if globalCache.defaultTTL != 10*time.Minute {
		t.Errorf("expected 10m TTL, got %v", globalCache.defaultTTL)
	}
}
