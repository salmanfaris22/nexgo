// Package actions provides server actions — form-based server mutations
// without manual API route creation (like Next.js Server Actions).
package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// Action is a server-side function triggered by form submission.
type Action func(ctx context.Context, formData FormData) (*Result, error)

// FormData wraps parsed form data.
type FormData struct {
	Values map[string][]string
}

// Get returns the first value for a key.
func (f FormData) Get(key string) string {
	if vals, ok := f.Values[key]; ok && len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// GetAll returns all values for a key.
func (f FormData) GetAll(key string) []string {
	return f.Values[key]
}

// Has checks if a key exists.
func (f FormData) Has(key string) bool {
	_, ok := f.Values[key]
	return ok
}

// Result is the outcome of a server action.
type Result struct {
	Success  bool                   `json:"success"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Errors   map[string]string      `json:"errors,omitempty"`
	Redirect string                 `json:"redirect,omitempty"`
	Message  string                 `json:"message,omitempty"`
}

// OK creates a successful result.
func OK(data map[string]interface{}) *Result {
	return &Result{Success: true, Data: data}
}

// Fail creates a failed result with validation errors.
func Fail(errors map[string]string) *Result {
	return &Result{Success: false, Errors: errors}
}

// RedirectTo creates a result that redirects.
func RedirectTo(url string) *Result {
	return &Result{Success: true, Redirect: url}
}

// WithMessage creates a result with a message.
func WithMessage(msg string) *Result {
	return &Result{Success: true, Message: msg}
}

// Registry manages server actions.
type Registry struct {
	mu      sync.RWMutex
	actions map[string]Action
}

// NewRegistry creates an action registry.
func NewRegistry() *Registry {
	return &Registry{actions: make(map[string]Action)}
}

// Register adds a named server action.
func (r *Registry) Register(name string, action Action) {
	r.mu.Lock()
	r.actions[name] = action
	r.mu.Unlock()
}

// Handler returns an HTTP handler that dispatches server actions.
// Actions are identified by a hidden form field "_action" or the URL path.
func (r *Registry) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		// Parse form data
		if err := req.ParseForm(); err != nil {
			http.Error(w, `{"error":"invalid form data"}`, http.StatusBadRequest)
			return
		}

		// Determine action name
		actionName := req.FormValue("_action")
		if actionName == "" {
			// Use URL path: /_nexgo/action/{name}
			actionName = strings.TrimPrefix(req.URL.Path, "/_nexgo/action/")
		}

		r.mu.RLock()
		action, ok := r.actions[actionName]
		r.mu.RUnlock()

		if !ok {
			http.Error(w, fmt.Sprintf(`{"error":"action %q not found"}`, actionName), http.StatusNotFound)
			return
		}

		formData := FormData{Values: req.Form}
		result, err := action(req.Context(), formData)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		// Handle redirect
		if result.Redirect != "" {
			// If AJAX request, return JSON with redirect URL
			if req.Header.Get("X-Requested-With") == "XMLHttpRequest" ||
				req.Header.Get("Accept") == "application/json" {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(result)
				return
			}
			http.Redirect(w, req, result.Redirect, http.StatusSeeOther)
			return
		}

		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		if !result.Success {
			w.WriteHeader(http.StatusUnprocessableEntity)
		}
		json.NewEncoder(w).Encode(result)
	}
}

// Middleware returns middleware that handles action form submissions.
// It intercepts POST requests with "_action" field and routes them to the registry.
func (r *Registry) Middleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodPost {
				// Check for action field without consuming the body
				contentType := req.Header.Get("Content-Type")
				if strings.Contains(contentType, "application/x-www-form-urlencoded") ||
					strings.Contains(contentType, "multipart/form-data") {
					if err := req.ParseForm(); err == nil {
						if actionName := req.FormValue("_action"); actionName != "" {
							r.mu.RLock()
							action, ok := r.actions[actionName]
							r.mu.RUnlock()
							if ok {
								formData := FormData{Values: req.Form}
								result, err := action(req.Context(), formData)
								if err != nil {
									http.Error(w, err.Error(), http.StatusInternalServerError)
									return
								}
								if result.Redirect != "" {
									http.Redirect(w, req, result.Redirect, http.StatusSeeOther)
									return
								}
								w.Header().Set("Content-Type", "application/json")
								json.NewEncoder(w).Encode(result)
								return
							}
						}
					}
				}
			}
			next(w, req)
		}
	}
}

// TemplateHelpers returns template functions for server actions.
func TemplateHelpers() map[string]interface{} {
	return map[string]interface{}{
		"actionField": func(name string) string {
			return fmt.Sprintf(`<input type="hidden" name="_action" value="%s">`, name)
		},
		"actionURL": func(name string) string {
			return "/_nexgo/action/" + name
		},
	}
}

// --- Validation helpers ---

// Validator helps validate form data.
type Validator struct {
	errors map[string]string
}

// NewValidator creates a form validator.
func NewValidator() *Validator {
	return &Validator{errors: make(map[string]string)}
}

// Required checks that a field is not empty.
func (v *Validator) Required(data FormData, field, message string) {
	if strings.TrimSpace(data.Get(field)) == "" {
		v.errors[field] = message
	}
}

// MinLength checks minimum string length.
func (v *Validator) MinLength(data FormData, field string, min int, message string) {
	if len(data.Get(field)) < min {
		v.errors[field] = message
	}
}

// MaxLength checks maximum string length.
func (v *Validator) MaxLength(data FormData, field string, max int, message string) {
	if len(data.Get(field)) > max {
		v.errors[field] = message
	}
}

// Email validates an email format (basic check).
func (v *Validator) Email(data FormData, field, message string) {
	val := data.Get(field)
	if !strings.Contains(val, "@") || !strings.Contains(val, ".") {
		v.errors[field] = message
	}
}

// Custom adds a custom validation.
func (v *Validator) Custom(field string, valid bool, message string) {
	if !valid {
		v.errors[field] = message
	}
}

// IsValid returns true if there are no errors.
func (v *Validator) IsValid() bool {
	return len(v.errors) == 0
}

// Errors returns the validation errors.
func (v *Validator) Errors() map[string]string {
	return v.errors
}

// Result returns a fail Result if invalid.
func (v *Validator) Result() *Result {
	if v.IsValid() {
		return nil
	}
	return Fail(v.errors)
}
