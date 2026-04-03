package router

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// RouteType defines how a page is rendered
type RouteType int

const (
	RouteTypePage RouteType = iota // HTML template page
	RouteTypeAPI                   // Go API handler
)

// Route priority levels (higher = matched first)
const (
	PriorityStatic   = 100
	PriorityDynamic  = 50
	PriorityCatchAll = 10
)

// Route represents a single route in the app
type Route struct {
	Pattern  string
	Regex    *regexp.Regexp
	Params   []string
	FilePath string
	Type     RouteType
	Handler  http.HandlerFunc
	CatchAll bool
	Priority int
}

// Middleware is a standard HTTP middleware
type Middleware func(http.HandlerFunc) http.HandlerFunc

// APIHandlerRegistry maps API route paths to Go handlers
var (
	apiHandlerRegistry = make(map[string]http.HandlerFunc)
	apiRegMu           sync.RWMutex
)

// RegisterAPI allows pages/api/*.go files to register handlers
func RegisterAPI(pattern string, handler http.HandlerFunc) {
	apiRegMu.Lock()
	defer apiRegMu.Unlock()
	apiHandlerRegistry[pattern] = handler
}

// Router manages all application routes
type Router struct {
	mu         sync.RWMutex
	routes     []*Route
	pagesDir   string
	notFound   http.HandlerFunc
	errorPage  http.HandlerFunc
	middleware []Middleware
}

// New creates a new Router scanning the given pages directory
func New(pagesDir string) *Router {
	r := &Router{
		pagesDir: pagesDir,
		notFound: defaultNotFound,
	}
	return r
}

// Scan walks the pages directory and builds the route table
func (r *Router) Scan() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes = nil

	err := filepath.WalkDir(r.pagesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, _ := filepath.Rel(r.pagesDir, path)
		route, err := r.fileToRoute(rel, path)
		if err != nil {
			// Log route parsing errors instead of silently ignoring
			fmt.Printf("[NexGo] Route error for %s: %v\n", rel, err)
			return nil
		}
		if route == nil {
			return nil
		}

		r.routes = append(r.routes, route)
		return nil
	})

	if err != nil {
		return err
	}

	// Sort routes by priority (most specific first)
	sort.Slice(r.routes, func(i, j int) bool {
		return r.routes[i].Priority > r.routes[j].Priority
	})

	return nil
}

// BindAPIHandlers associates routes with registered API handlers
func (r *Router) BindAPIHandlers() {
	r.mu.Lock()
	defer r.mu.Unlock()
	apiRegMu.RLock()
	defer apiRegMu.RUnlock()
	for _, route := range r.routes {
		if route.Type == RouteTypeAPI {
			if h, ok := apiHandlerRegistry[route.Pattern]; ok {
				route.Handler = h
			} else {
				normalized := route.Pattern
				if strings.HasSuffix(normalized, "/") {
					normalized = normalized[:len(normalized)-1]
				}
				if h, ok := apiHandlerRegistry[normalized]; ok {
					route.Handler = h
				}
			}
		}
	}
}

// fileToRoute converts a file path to a Route
func (r *Router) fileToRoute(rel, abs string) (*Route, error) {
	ext := filepath.Ext(rel)
	name := strings.TrimSuffix(rel, ext)

	// Normalize separators
	name = filepath.ToSlash(name)

	var routeType RouteType
	switch ext {
	case ".html", ".gohtml", ".tmpl":
		routeType = RouteTypePage
	case ".go":
		routeType = RouteTypeAPI
	default:
		return nil, nil
	}

	// Build URL pattern from file path
	pattern := "/" + name

	if routeType == RouteTypeAPI {
		pattern = "/" + name
		if !strings.HasPrefix(pattern, "/api") {
			pattern = "/api" + pattern
		}
		pattern = strings.ReplaceAll(pattern, "//", "/")
		if pattern == "/api" {
			pattern = "/api/"
		}
	}

	// Handle index routes
	pattern = strings.TrimSuffix(pattern, "/index")
	if pattern == "" {
		pattern = "/"
	}

	// Parse dynamic segments: [param] and [...catchall]
	params := []string{}
	catchAll := false
	priority := PriorityStatic

	// Convert [param] and [...param] to regex groups
	regexStr := pattern
	segments := strings.Split(pattern, "/")
	for i, seg := range segments {
		if strings.HasPrefix(seg, "[...") && strings.HasSuffix(seg, "]") {
			paramName := seg[4 : len(seg)-1]
			params = append(params, paramName)
			segments[i] = "(.+)"
			catchAll = true
			priority = PriorityCatchAll
		} else if strings.HasPrefix(seg, "[") && strings.HasSuffix(seg, "]") {
			paramName := seg[1 : len(seg)-1]
			params = append(params, paramName)
			segments[i] = "([^/]+)"
			priority = PriorityDynamic
		}
	}
	regexStr = "^" + strings.Join(segments, "/") + "$"

	re, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid route regex for %s: %w", pattern, err)
	}

	route := &Route{
		Pattern:  pattern,
		Regex:    re,
		Params:   params,
		FilePath: abs,
		Type:     routeType,
		CatchAll: catchAll,
		Priority: priority,
	}

	apiRegMu.RLock()
	if h, ok := apiHandlerRegistry[pattern]; ok {
		route.Handler = h
	}
	apiRegMu.RUnlock()

	return route, nil
}

// Match finds the best matching route for a request path
func (r *Router) Match(urlPath string) (*Route, map[string]string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Normalize path
	if urlPath == "" {
		urlPath = "/"
	}

	for _, route := range r.routes {
		matches := route.Regex.FindStringSubmatch(urlPath)
		if matches == nil {
			continue
		}

		params := make(map[string]string)
		for i, name := range route.Params {
			if i+1 < len(matches) {
				params[name] = matches[i+1]
			}
		}
		return route, params
	}

	return nil, nil
}

// Use adds a global middleware
func (r *Router) Use(mw Middleware) {
	r.middleware = append(r.middleware, mw)
}

// SetNotFound sets custom 404 handler
func (r *Router) SetNotFound(h http.HandlerFunc) {
	r.notFound = h
}

// GetRoutes returns all registered routes (for dev tools / debugging)
func (r *Router) GetRoutes() []*Route {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*Route, len(r.routes))
	copy(result, r.routes)
	return result
}

// applyMiddleware wraps handler with middleware chain
func (r *Router) applyMiddleware(h http.HandlerFunc, extras []Middleware) http.HandlerFunc {
	all := append(r.middleware, extras...)
	for i := len(all) - 1; i >= 0; i-- {
		h = all[i](h)
	}
	return h
}

// ServeHTTP implements http.Handler - this is the main request dispatcher
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path

	route, params := r.Match(path)
	if route == nil {
		r.notFound(w, req)
		return
	}

	// Inject params into request context
	ctx := WithParams(req.Context(), params)
	req = req.WithContext(ctx)

	if route.Handler != nil {
		handler := r.applyMiddleware(route.Handler, nil)
		handler(w, req)
	} else {
		// Will be handled by renderer
		w.Header().Set("X-NexGo-Route", route.Pattern)
		w.Header().Set("X-NexGo-File", route.FilePath)
	}
}

func defaultNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>404 - Not Found | NexGo</title>
<style>
  body { font-family: system-ui; display:flex; align-items:center; justify-content:center; 
         min-height:100vh; margin:0; background:#0a0a0a; color:#fff; }
  .box { text-align:center; }
  h1 { font-size:8rem; margin:0; background:linear-gradient(135deg,#00d2ff,#7b2ff7); 
       -webkit-background-clip:text; -webkit-text-fill-color:transparent; }
  p { color:#888; font-size:1.2rem; }
  a { color:#00d2ff; text-decoration:none; }
</style>
</head>
<body>
  <div class="box">
    <h1>404</h1>
    <p>Page not found — <a href="/">Go home</a></p>
    <small style="color:#444">NexGo Framework</small>
  </div>
</body>
</html>`)
}
