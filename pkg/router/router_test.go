package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func setupPagesDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "api"), 0755)
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>Home</h1>"), 0644)
	os.WriteFile(filepath.Join(dir, "about.html"), []byte("<h1>About</h1>"), 0644)
	os.WriteFile(filepath.Join(dir, "api", "hello.go"), []byte("package api"), 0644)
	return dir
}

func TestNew(t *testing.T) {
	r := New("/pages")
	if r.pagesDir != "/pages" {
		t.Errorf("expected /pages, got %s", r.pagesDir)
	}
}

func TestScan(t *testing.T) {
	dir := setupPagesDir(t)
	r := New(dir)
	err := r.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	routes := r.GetRoutes()
	if len(routes) < 2 {
		t.Errorf("expected at least 2 routes, got %d", len(routes))
	}
}

func TestScan_NonExistentDir(t *testing.T) {
	r := New("/nonexistent")
	err := r.Scan()
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestFileToRoute(t *testing.T) {
	dir := setupPagesDir(t)
	r := New(dir)

	tests := []struct {
		rel         string
		wantPattern string
		wantType    RouteType
	}{
		{"index.html", "/", RouteTypePage},
		{"about.html", "/about", RouteTypePage},
		{"api/hello.go", "/api/hello", RouteTypeAPI},
	}

	for _, tt := range tests {
		abs := filepath.Join(dir, tt.rel)
		route, err := r.fileToRoute(tt.rel, abs)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", tt.rel, err)
		}
		if route.Pattern != tt.wantPattern {
			t.Errorf("%s: expected pattern %s, got %s", tt.rel, tt.wantPattern, route.Pattern)
		}
		if route.Type != tt.wantType {
			t.Errorf("%s: expected type %d, got %d", tt.rel, tt.wantType, route.Type)
		}
	}
}

func TestDynamicRoutes(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "blog"), 0755)
	os.WriteFile(filepath.Join(dir, "blog", "[slug].html"), []byte("post"), 0644)

	r := New(dir)
	r.Scan()

	route, params := r.Match("/blog/my-post")
	if route == nil {
		t.Fatal("expected route match")
	}
	if params["slug"] != "my-post" {
		t.Errorf("expected slug=my-post, got %s", params["slug"])
	}
}

func TestCatchAllRoute(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "[...path].html"), []byte("catchall"), 0644)

	r := New(dir)
	r.Scan()

	route, params := r.Match("/docs/api/auth")
	if route == nil {
		t.Fatal("expected route match")
	}
	if params["path"] != "docs/api/auth" {
		t.Errorf("expected path=docs/api/auth, got %s", params["path"])
	}
}

func TestMatch_NoRoute(t *testing.T) {
	dir := setupPagesDir(t)
	r := New(dir)
	r.Scan()

	route, params := r.Match("/nonexistent")
	if route != nil {
		t.Error("expected nil route")
	}
	if params != nil {
		t.Error("expected nil params")
	}
}

func TestMatch_EmptyPath(t *testing.T) {
	dir := setupPagesDir(t)
	r := New(dir)
	r.Scan()

	route, _ := r.Match("")
	if route == nil {
		t.Error("expected route match for empty path")
	}
}

func TestServeHTTP(t *testing.T) {
	dir := setupPagesDir(t)
	r := New(dir)
	r.Scan()

	req := httptest.NewRequest("GET", "/about", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("X-NexGo-Route") != "/about" {
		t.Errorf("expected X-NexGo-Route=/about, got %s", w.Header().Get("X-NexGo-Route"))
	}
}

func TestServeHTTP_NotFound(t *testing.T) {
	dir := setupPagesDir(t)
	r := New(dir)
	r.Scan()

	req := httptest.NewRequest("GET", "/missing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDefaultNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	defaultNotFound(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestMiddleware(t *testing.T) {
	r := New("/pages")
	called := false
	r.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			called = true
			next(w, r)
		}
	})
	if len(r.middleware) != 1 {
		t.Error("expected middleware to be added")
	}
	_ = called
}

func TestSetNotFound(t *testing.T) {
	r := New("/pages")
	r.SetNotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
	})
	req := httptest.NewRequest("GET", "/missing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusGone {
		t.Errorf("expected 410, got %d", w.Code)
	}
}

func TestPriority(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("home"), 0644)
	os.MkdirAll(filepath.Join(dir, "blog"), 0755)
	os.WriteFile(filepath.Join(dir, "blog", "[slug].html"), []byte("post"), 0644)
	os.WriteFile(filepath.Join(dir, "[...rest].html"), []byte("catchall"), 0644)

	r := New(dir)
	r.Scan()
	routes := r.GetRoutes()

	// Static should be first (highest priority)
	if len(routes) < 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}
	if routes[0].Priority != PriorityStatic {
		t.Errorf("expected first route to have static priority %d, got %d", PriorityStatic, routes[0].Priority)
	}
}
