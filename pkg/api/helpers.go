// Package api provides helpers for writing NexGo API route handlers.
package api

import (
	"encoding/json"
	"log"
	"net/http"
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
// Returns false and writes a 400 if parsing fails.
func Decode(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<20) // 4MB limit
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		BadRequest(w, "Invalid JSON: "+err.Error())
		return false
	}
	return true
}

// MethodGuard checks that the request uses one of the allowed methods.
// Returns false and writes 405 if the method is not allowed.
func MethodGuard(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, m := range methods {
		if r.Method == m {
			return true
		}
	}
	w.Header().Set("Allow", joinMethods(methods))
	Error(w, http.StatusMethodNotAllowed, "Method "+r.Method+" not allowed")
	return false
}

// Route dispatches to different handlers based on HTTP method.
//
//	api.Route(w, r, api.Methods{
//	    "GET":    listUsers,
//	    "POST":   createUser,
//	})
type Methods map[string]http.HandlerFunc

func Route(w http.ResponseWriter, r *http.Request, methods Methods) {
	if h, ok := methods[r.Method]; ok {
		h(w, r)
		return
	}
	allowed := make([]string, 0, len(methods))
	for m := range methods {
		allowed = append(allowed, m)
	}
	w.Header().Set("Allow", joinMethods(allowed))
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
	var n int
	if _, err := parseint(val, &n); err != nil {
		return def
	}
	return n
}

func parseint(s string, out *int) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, &parseError{s}
		}
		n = n*10 + int(c-'0')
	}
	if out != nil {
		*out = n
	}
	return n, nil
}

type parseError struct{ s string }

func (e *parseError) Error() string { return "not an integer: " + e.s }

func joinMethods(methods []string) string {
	result := ""
	for i, m := range methods {
		if i > 0 {
			result += ", "
		}
		result += m
	}
	return result
}

func Handler(w http.ResponseWriter, r *http.Request) {

	// Optional: only allow GET
	if !MethodGuard(w, r, http.MethodGet) {
		return
	}

	value := r.URL.Query().Get("q")

	// ✅ use your helper
	JSON(w, map[string]string{
		"value": value,
	})
}
