// Package errorpage provides custom error pages, error boundaries,
// and not-found page conventions for NexGo.
package errorpage

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Config holds error page configuration.
type Config struct {
	PagesDir string // root pages directory
	DevMode  bool
}

// ErrorPages manages custom error page rendering.
type ErrorPages struct {
	mu        sync.RWMutex
	config    Config
	templates map[int]*template.Template // status code -> template
	notFound  map[string]*template.Template // directory path -> not-found template
	errors    map[string]*template.Template // directory path -> error boundary template
}

// New creates an error page manager.
func New(cfg Config) *ErrorPages {
	return &ErrorPages{
		config:    cfg,
		templates: make(map[int]*template.Template),
		notFound:  make(map[string]*template.Template),
		errors:    make(map[string]*template.Template),
	}
}

// LoadAll scans for error page templates in the pages directory.
// Conventions:
//   - pages/404.html   -> custom 404 page
//   - pages/500.html   -> custom 500 page
//   - pages/{dir}/error.html -> error boundary for that directory
//   - pages/{dir}/not-found.html -> not-found page for that directory
func (ep *ErrorPages) LoadAll(funcMap template.FuncMap) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	pagesDir := ep.config.PagesDir

	// Load status code pages (404.html, 500.html, etc.)
	statusPages := map[int]string{
		404: "404.html",
		500: "500.html",
		403: "403.html",
		401: "401.html",
	}
	for code, file := range statusPages {
		path := filepath.Join(pagesDir, file)
		if tmpl, err := loadTemplate(path, funcMap); err == nil {
			ep.templates[code] = tmpl
		}
	}

	// Walk directories for error.html and not-found.html
	filepath.WalkDir(pagesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		name := d.Name()
		dir, _ := filepath.Rel(pagesDir, filepath.Dir(path))
		dir = filepath.ToSlash(dir)

		switch name {
		case "error.html":
			if tmpl, err := loadTemplate(path, funcMap); err == nil {
				ep.errors[dir] = tmpl
			}
		case "not-found.html":
			if tmpl, err := loadTemplate(path, funcMap); err == nil {
				ep.notFound[dir] = tmpl
			}
		}
		return nil
	})

	return nil
}

// RenderError renders the appropriate error page.
func (ep *ErrorPages) RenderError(w http.ResponseWriter, r *http.Request, status int, err error) {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	data := ErrorData{
		Status:     status,
		StatusText: http.StatusText(status),
		Message:    err.Error(),
		Path:       r.URL.Path,
		DevMode:    ep.config.DevMode,
	}

	// Check for directory-specific error boundary
	dir := pathToDir(r.URL.Path)
	for dir != "" {
		if tmpl, ok := ep.errors[dir]; ok {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(status)
			tmpl.Execute(w, data)
			return
		}
		// Walk up
		if idx := strings.LastIndex(dir, "/"); idx >= 0 {
			dir = dir[:idx]
		} else {
			break
		}
	}

	// Check for status-specific page
	if tmpl, ok := ep.templates[status]; ok {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		tmpl.Execute(w, data)
		return
	}

	// Default error page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprint(w, defaultErrorPage(data))
}

// RenderNotFound renders a 404 page, checking for directory-specific not-found.html.
func (ep *ErrorPages) RenderNotFound(w http.ResponseWriter, r *http.Request) {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	data := ErrorData{
		Status:     404,
		StatusText: "Not Found",
		Message:    "The page you're looking for doesn't exist.",
		Path:       r.URL.Path,
		DevMode:    ep.config.DevMode,
	}

	// Check directory-specific not-found
	dir := pathToDir(r.URL.Path)
	for dir != "" {
		if tmpl, ok := ep.notFound[dir]; ok {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(404)
			tmpl.Execute(w, data)
			return
		}
		if idx := strings.LastIndex(dir, "/"); idx >= 0 {
			dir = dir[:idx]
		} else {
			break
		}
	}

	// Fall back to global 404
	if tmpl, ok := ep.templates[404]; ok {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(404)
		tmpl.Execute(w, data)
		return
	}

	// Built-in default
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(404)
	fmt.Fprint(w, defaultErrorPage(data))
}

// ErrorData is passed to error page templates.
type ErrorData struct {
	Status     int
	StatusText string
	Message    string
	Path       string
	DevMode    bool
}

func loadTemplate(path string, funcMap template.FuncMap) (*template.Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New(filepath.Base(path)).Funcs(funcMap).Parse(string(data))
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}

func pathToDir(urlPath string) string {
	dir := strings.TrimPrefix(urlPath, "/")
	if idx := strings.LastIndex(dir, "/"); idx >= 0 {
		return dir[:idx]
	}
	return dir
}

func defaultErrorPage(data ErrorData) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <title>%d %s | NexGo</title>
  <style>
    *{box-sizing:border-box;margin:0;padding:0}
    body{font-family:system-ui,-apple-system,sans-serif;background:#0a0a0a;color:#e0e0e0;display:flex;align-items:center;justify-content:center;min-height:100vh;padding:2rem}
    .container{max-width:500px;text-align:center}
    .code{font-size:8rem;font-weight:900;line-height:1;background:linear-gradient(135deg,#ff4757,#ff6b81);-webkit-background-clip:text;-webkit-text-fill-color:transparent}
    .text{font-size:1.3rem;color:#888;margin:1rem 0 2rem}
    .msg{background:#111;border:1px solid #222;border-radius:8px;padding:1rem;font-size:.9rem;color:#999;margin-bottom:2rem}
    a{color:#00d2ff;text-decoration:none}
    a:hover{text-decoration:underline}
  </style>
</head>
<body>
  <div class="container">
    <div class="code">%d</div>
    <div class="text">%s</div>
    <div class="msg">%s</div>
    <a href="/">Go Home</a>
  </div>
</body>
</html>`, data.Status, data.StatusText, data.Status, data.StatusText, template.HTMLEscapeString(data.Message))
}

// Middleware returns HTTP middleware that catches panics and renders error pages.
func (ep *ErrorPages) Middleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					ep.RenderError(w, r, 500, fmt.Errorf("%v", err))
				}
			}()
			next(w, r)
		}
	}
}
