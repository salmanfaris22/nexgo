// Package isr provides Incremental Static Regeneration — pages are cached and
// rebuilt in the background after a configurable revalidation interval.
package isr

import (
	"net/http"
	"sync"
	"time"
)

// entry tracks a cached page and its revalidation state.
type entry struct {
	statusCode   int
	headers      http.Header
	body         []byte
	createdAt    time.Time
	revalidating bool
}

// ISR manages incrementally regenerated pages.
type ISR struct {
	mu         sync.RWMutex
	pages      map[string]*entry
	revalidate time.Duration
}

// New creates an ISR manager with the given revalidation interval.
func New(revalidate time.Duration) *ISR {
	return &ISR{
		pages:      make(map[string]*entry),
		revalidate: revalidate,
	}
}

// Serve checks the cache and either serves the cached version or triggers
// a background rebuild. The generate function is called when the cache is
// stale (first request or after revalidate interval).
func (i *ISR) Serve(w http.ResponseWriter, r *http.Request, generate func() (int, http.Header, []byte, error)) {
	key := r.URL.Path

	i.mu.RLock()
	e, exists := i.pages[key]
	i.mu.RUnlock()

	// Serve cached content if fresh
	if exists && time.Since(e.createdAt) < i.revalidate {
		i.writeCached(w, e)
		return
	}

	// First request or expired — generate synchronously with lock to prevent duplicates
	i.mu.Lock()
	// Check again under lock — another goroutine may have just regenerated
	e2, exists2 := i.pages[key]
	if exists2 && time.Since(e2.createdAt) < i.revalidate {
		i.mu.Unlock()
		i.writeCached(w, e2)
		return
	}
	// Mark as generating to prevent other concurrent requests from also generating
	i.pages[key] = &entry{
		statusCode:   200,
		revalidating: true,
		createdAt:    time.Now(),
	}
	i.mu.Unlock()

	statusCode, headers, body, err := generate()
	if err != nil {
		i.mu.Lock()
		delete(i.pages, key)
		i.mu.Unlock()
		http.Error(w, "ISR generate error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	i.mu.Lock()
	i.pages[key] = &entry{
		statusCode:   statusCode,
		headers:      headers,
		body:         body,
		createdAt:    time.Now(),
		revalidating: false,
	}
	i.mu.Unlock()
	i.writeResponse(w, statusCode, headers, body, "MISS")
}

// Revalidate triggers a background rebuild of a specific path.
func (i *ISR) Revalidate(path string, generate func() (int, http.Header, []byte, error)) {
	i.mu.Lock()
	e, exists := i.pages[path]
	if exists && e.revalidating {
		i.mu.Unlock()
		return
	}
	if exists {
		e.revalidating = true
	}
	i.mu.Unlock()

	go func() {
		statusCode, headers, body, err := generate()
		if err != nil {
			return
		}
		i.mu.Lock()
		i.pages[path] = &entry{
			statusCode:   statusCode,
			headers:      headers,
			body:         body,
			createdAt:    time.Now(),
			revalidating: false,
		}
		i.mu.Unlock()
	}()
}

// Purge removes a page from the ISR cache.
func (i *ISR) Purge(path string) {
	i.mu.Lock()
	delete(i.pages, path)
	i.mu.Unlock()
}

// PurgeAll clears the entire ISR cache.
func (i *ISR) PurgeAll() {
	i.mu.Lock()
	i.pages = make(map[string]*entry)
	i.mu.Unlock()
}

// IsCached returns true if a path has a cached entry.
func (i *ISR) IsCached(path string) bool {
	i.mu.RLock()
	_, ok := i.pages[path]
	i.mu.RUnlock()
	return ok
}

// Age returns how long ago a page was cached.
func (i *ISR) Age(path string) time.Duration {
	i.mu.RLock()
	e, ok := i.pages[path]
	i.mu.RUnlock()
	if !ok {
		return 0
	}
	return time.Since(e.createdAt)
}

func (i *ISR) writeCached(w http.ResponseWriter, e *entry) {
	for k, vals := range e.headers {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.Header().Set("X-ISR", "STALE")
	w.WriteHeader(e.statusCode)
	if _, err := w.Write(e.body); err != nil {
		// Connection likely closed — ignore
	}
}

func (i *ISR) writeResponse(w http.ResponseWriter, code int, headers http.Header, body []byte, cacheStatus string) {
	for k, vals := range headers {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.Header().Set("X-ISR", cacheStatus)
	w.WriteHeader(code)
	if _, err := w.Write(body); err != nil {
		// Connection likely closed — ignore
	}
}

// Global ISR instance.
var globalISR = New(60 * time.Second)

// Serve wraps the global ISR Serve.
func Serve(w http.ResponseWriter, r *http.Request, generate func() (int, http.Header, []byte, error)) {
	globalISR.Serve(w, r, generate)
}

// Revalidate wraps the global ISR Revalidate.
func Revalidate(path string, generate func() (int, http.Header, []byte, error)) {
	globalISR.Revalidate(path, generate)
}

// Purge wraps the global ISR Purge.
func Purge(path string) {
	globalISR.Purge(path)
}

// SetRevalidate changes the global revalidation interval.
func SetRevalidate(d time.Duration) {
	globalISR.revalidate = d
}
