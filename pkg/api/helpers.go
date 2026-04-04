// Package api provides helpers for writing NexGo API route handlers.
package api

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/salmanfaris22/nexgo/v2/pkg/cache"
)

// JSON writes a JSON response with status 200
func JSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[NexGo/api] JSON encode error: %v", err)
	}
}

// JSONStatus writes a JSON response with a custom status code
func JSONStatus(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[NexGo/api] JSON encode error: %v", err)
	}
}

// Error writes a JSON error response
func Error(w http.ResponseWriter, status int, message string) {
	JSONStatus(w, status, map[string]interface{}{
		"error":  message,
		"status": status,
	})
}

// BadRequest sends a 400 JSON error
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, message)
}

// NotFound sends a 404 JSON error
func NotFound(w http.ResponseWriter, message string) {
	Error(w, http.StatusNotFound, message)
}

// Unauthorized sends a 401 JSON error
func Unauthorized(w http.ResponseWriter) {
	Error(w, http.StatusUnauthorized, "Unauthorized")
}

// Forbidden sends a 403 JSON error
func Forbidden(w http.ResponseWriter) {
	Error(w, http.StatusForbidden, "Forbidden")
}

// InternalError sends a 500 JSON error and logs the cause
func InternalError(w http.ResponseWriter, err error) {
	log.Printf("[NexGo/api] Internal error: %v", err)
	Error(w, http.StatusInternalServerError, "Internal server error")
}

// Decode reads and decodes a JSON request body into v.
func Decode(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		BadRequest(w, "Invalid JSON: "+err.Error())
		return false
	}
	return true
}

// MethodGuard checks that the request uses one of the allowed methods.
func MethodGuard(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, m := range methods {
		if r.Method == m {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(methods, ", "))
	Error(w, http.StatusMethodNotAllowed, "Method "+r.Method+" not allowed")
	return false
}

// Methods is a map of HTTP method to handler
type Methods map[string]http.HandlerFunc

// Route dispatches to different handlers based on HTTP method.
func Route(w http.ResponseWriter, r *http.Request, methods Methods) {
	if h, ok := methods[r.Method]; ok {
		h(w, r)
		return
	}
	allowed := make([]string, 0, len(methods))
	for m := range methods {
		allowed = append(allowed, m)
	}
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	Error(w, http.StatusMethodNotAllowed, "Method "+r.Method+" not allowed")
}

// Paginate extracts page and limit from query params with defaults.
func Paginate(r *http.Request) (page, limit int) {
	page = queryInt(r, "page", 1)
	limit = queryInt(r, "limit", 20)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return page, limit
}

func queryInt(r *http.Request, key string, def int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return def
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return def
	}
	return n
}

// ---------------------------------------------------------------------------
// HTMX helpers
// ---------------------------------------------------------------------------

// IsHTMX returns true if the request comes from HTMX
func IsHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// HTMXHeader sets an HTMX response header (e.g. "HX-Trigger", "HX-Redirect")
func HTMXHeader(w http.ResponseWriter, key, value string) {
	w.Header().Set(key, value)
}

// HTMXHTML sends an HTML fragment response for HTMX
func HTMXHTML(w http.ResponseWriter, html string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

// HTMXTrigger sets the HX-Trigger header to fire a client-side event
func HTMXTrigger(w http.ResponseWriter, event string) {
	w.Header().Set("HX-Trigger", event)
}

// ---------------------------------------------------------------------------
// State management
// ---------------------------------------------------------------------------

// State is a thread-safe key-value store for application state
type State struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// NewState creates a new empty State
func NewState() *State {
	return &State{data: make(map[string]interface{})}
}

// Set stores a value
func (s *State) Set(key string, value interface{}) {
	s.mu.Lock()
	s.data[key] = value
	s.mu.Unlock()
}

// Get retrieves a value (returns nil if not found)
func (s *State) Get(key string) interface{} {
	s.mu.RLock()
	v := s.data[key]
	s.mu.RUnlock()
	return v
}

// Delete removes a key
func (s *State) Delete(key string) {
	s.mu.Lock()
	delete(s.data, key)
	s.mu.Unlock()
}

// All returns a copy of all state data
func (s *State) All() map[string]interface{} {
	s.mu.RLock()
	out := make(map[string]interface{}, len(s.data))
	for k, v := range s.data {
		out[k] = v
	}
	s.mu.RUnlock()
	return out
}

// Global state instance (shared across all handlers)
var globalState = NewState()

// SetState stores a value in the global state
func SetState(key string, value interface{}) {
	globalState.Set(key, value)
}

// GetState retrieves a value from the global state
func GetState(key string) interface{} {
	return globalState.Get(key)
}

// DeleteState removes a key from the global state
func DeleteState(key string) {
	globalState.Delete(key)
}

// ---------------------------------------------------------------------------
// HTML helpers
// ---------------------------------------------------------------------------

// Escape sanitizes HTML output
func Escape(s string) string {
	return template.HTMLEscapeString(s)
}

// HTML sends raw HTML response
func HTML(w http.ResponseWriter, html string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

// ---------------------------------------------------------------------------
// Cache helpers
// ---------------------------------------------------------------------------

// Cache wraps cache.Middleware for easy use as http middleware.
func Cache(ttl ...time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	return cache.CacheMiddleware(ttl...)
}

// CacheSet stores a response in the global cache.
func CacheSet(key string, statusCode int, headers http.Header, body []byte) {
	cache.CacheSet(key, statusCode, headers, body)
}

// CacheGet retrieves a cached response.
func CacheGet(key string) (int, http.Header, []byte, bool) {
	return cache.CacheGet(key)
}

// CacheDelete removes a key from the global cache.
func CacheDelete(key string) {
	cache.CacheDelete(key)
}

// CacheClear clears the entire global cache.
func CacheClear() {
	cache.CacheClear()
}
