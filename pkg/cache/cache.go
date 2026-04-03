// Package cache provides in-memory response caching with TTL and invalidation.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

// entry holds a cached response.
type entry struct {
	statusCode int
	headers    http.Header
	body       []byte
	expiresAt  time.Time
}

// Cache is a thread-safe in-memory response cache.
type Cache struct {
	mu         sync.RWMutex
	items      map[string]*entry
	defaultTTL time.Duration
	stopCh     chan struct{}
}

// New creates a Cache with the given default TTL.
// Call Stop() to release the cleanup goroutine when done.
func New(ttl time.Duration) *Cache {
	c := &Cache{
		items:      make(map[string]*entry),
		defaultTTL: ttl,
		stopCh:     make(chan struct{}),
	}
	go c.cleanupLoop()
	return c
}

// Stop releases the cleanup goroutine.
func (c *Cache) Stop() {
	close(c.stopCh)
}

// Get returns a cached response if it exists and is not expired.
func (c *Cache) Get(key string) (statusCode int, headers http.Header, body []byte, ok bool) {
	c.mu.RLock()
	e, exists := c.items[key]
	c.mu.RUnlock()
	if !exists || time.Now().After(e.expiresAt) {
		return 0, nil, nil, false
	}
	return e.statusCode, e.headers, e.body, true
}

// Set stores a response in the cache with the default TTL.
func (c *Cache) Set(key string, statusCode int, headers http.Header, body []byte) {
	c.SetWithTTL(key, statusCode, headers, body, c.defaultTTL)
}

// SetWithTTL stores a response with a custom TTL.
func (c *Cache) SetWithTTL(key string, statusCode int, headers http.Header, body []byte, ttl time.Duration) {
	c.mu.Lock()
	c.items[key] = &entry{
		statusCode: statusCode,
		headers:    headers,
		body:       body,
		expiresAt:  time.Now().Add(ttl),
	}
	c.mu.Unlock()
}

// Delete removes a key from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

// DeletePrefix removes all keys starting with prefix.
func (c *Cache) DeletePrefix(prefix string) {
	c.mu.Lock()
	for k := range c.items {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
}

// Clear removes all cached entries.
func (c *Cache) Clear() {
	c.mu.Lock()
	c.items = make(map[string]*entry)
	c.mu.Unlock()
}

// Len returns the number of cached entries.
func (c *Cache) Len() int {
	c.mu.RLock()
	n := len(c.items)
	c.mu.RUnlock()
	return n
}

// Key generates a cache key from an HTTP request.
func Key(r *http.Request) string {
	h := sha256.Sum256([]byte(r.URL.String()))
	return hex.EncodeToString(h[:])
}

// Middleware returns an http.Handler that caches GET responses.
func Middleware(c *Cache, ttl ...time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next(w, r)
				return
			}
			key := Key(r)
			t := c.defaultTTL
			if len(ttl) > 0 {
				t = ttl[0]
			}
			if code, headers, body, ok := c.Get(key); ok {
				for k, vals := range headers {
					for _, v := range vals {
						w.Header().Add(k, v)
					}
				}
				w.Header().Set("X-Cache", "HIT")
				w.WriteHeader(code)
				if _, err := w.Write(body); err != nil {
					// Connection likely closed
				}
				return
			}
			rec := &responseRecorder{ResponseWriter: w, body: &syncBuffer{}}
			next(rec, r)
			c.SetWithTTL(key, rec.statusCode, rec.headers, rec.body.Bytes(), t)
			w.Header().Set("X-Cache", "MISS")
		}
	}
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	headers    http.Header
	body       *syncBuffer
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func (r *responseRecorder) Header() http.Header {
	if r.headers == nil {
		r.headers = make(http.Header)
		for k, v := range r.ResponseWriter.Header() {
			r.headers[k] = v
		}
	}
	return r.ResponseWriter.Header()
}

type syncBuffer struct {
	mu   sync.Mutex
	data []byte
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	b.data = append(b.data, p...)
	b.mu.Unlock()
	return len(p), nil
}

func (b *syncBuffer) Bytes() []byte {
	b.mu.Lock()
	out := make([]byte, len(b.data))
	copy(out, b.data)
	b.mu.Unlock()
	return out
}

func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for k, e := range c.items {
				if now.After(e.expiresAt) {
					delete(c.items, k)
				}
			}
			c.mu.Unlock()
		case <-c.stopCh:
			return
		}
	}
}

// Global cache instance for convenience.
var globalCache = New(5 * time.Minute)

// SetGlobalTTL changes the default TTL for the global cache.
func SetGlobalTTL(ttl time.Duration) {
	globalCache.defaultTTL = ttl
}

// CacheGet wraps Cache.Get on the global cache.
func CacheGet(key string) (int, http.Header, []byte, bool) {
	return globalCache.Get(key)
}

// CacheSet wraps Cache.Set on the global cache.
func CacheSet(key string, statusCode int, headers http.Header, body []byte) {
	globalCache.Set(key, statusCode, headers, body)
}

// CacheDelete wraps Cache.Delete on the global cache.
func CacheDelete(key string) {
	globalCache.Delete(key)
}

// CacheClear wraps Cache.Clear on the global cache.
func CacheClear() {
	globalCache.Clear()
}

// CacheMiddleware wraps Cache.Middleware on the global cache.
func CacheMiddleware(ttl ...time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	return Middleware(globalCache, ttl...)
}
