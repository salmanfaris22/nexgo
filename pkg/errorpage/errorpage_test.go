package errorpage

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	cfg := Config{PagesDir: "/pages", DevMode: true}
	ep := New(cfg)
	if ep == nil {
		t.Fatal("expected ErrorPages")
	}
}

func TestLoadAll(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "blog"), 0755)

	os.WriteFile(filepath.Join(dir, "404.html"), []byte("<h1>404</h1>"), 0644)
	os.WriteFile(filepath.Join(dir, "500.html"), []byte("<h1>500</h1>"), 0644)
	os.WriteFile(filepath.Join(dir, "blog", "error.html"), []byte("<h1>Blog Error</h1>"), 0644)
	os.WriteFile(filepath.Join(dir, "blog", "not-found.html"), []byte("<h1>Blog Not Found</h1>"), 0644)

	ep := New(Config{PagesDir: dir, DevMode: true})
	err := ep.LoadAll(template.FuncMap{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRenderError_Default(t *testing.T) {
	dir := t.TempDir()
	ep := New(Config{PagesDir: dir, DevMode: true})
	ep.LoadAll(template.FuncMap{})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/missing", nil)
	ep.RenderError(w, r, 404, nil)

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "404") {
		t.Error("expected 404 in body")
	}
}

func TestRenderError_CustomPage(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "404.html"), []byte("Custom 404 Page"), 0644)

	ep := New(Config{PagesDir: dir, DevMode: true})
	ep.LoadAll(template.FuncMap{})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/missing", nil)
	ep.RenderError(w, r, 404, nil)

	if !strings.Contains(w.Body.String(), "Custom 404 Page") {
		t.Error("expected custom 404 page")
	}
}

func TestRenderError_DirectorySpecific(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "blog"), 0755)
	os.WriteFile(filepath.Join(dir, "blog", "error.html"), []byte("Blog Error"), 0644)

	ep := New(Config{PagesDir: dir, DevMode: true})
	ep.LoadAll(template.FuncMap{})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/blog/some-post", nil)
	ep.RenderError(w, r, 500, nil)

	if !strings.Contains(w.Body.String(), "Blog Error") {
		t.Error("expected directory-specific error page")
	}
}

func TestRenderNotFound_Default(t *testing.T) {
	dir := t.TempDir()
	ep := New(Config{PagesDir: dir, DevMode: true})
	ep.LoadAll(template.FuncMap{})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/missing", nil)
	ep.RenderNotFound(w, r)

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestRenderNotFound_CustomPage(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "404.html"), []byte("Custom 404"), 0644)

	ep := New(Config{PagesDir: dir, DevMode: true})
	ep.LoadAll(template.FuncMap{})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/missing", nil)
	ep.RenderNotFound(w, r)

	if !strings.Contains(w.Body.String(), "Custom 404") {
		t.Error("expected custom 404 page")
	}
}

func TestRenderNotFound_DirectorySpecific(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "blog"), 0755)
	os.WriteFile(filepath.Join(dir, "blog", "not-found.html"), []byte("Blog 404"), 0644)

	ep := New(Config{PagesDir: dir, DevMode: true})
	ep.LoadAll(template.FuncMap{})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/blog/missing", nil)
	ep.RenderNotFound(w, r)

	if !strings.Contains(w.Body.String(), "Blog 404") {
		t.Error("expected directory-specific not-found page")
	}
}

func TestMiddleware(t *testing.T) {
	dir := t.TempDir()
	ep := New(Config{PagesDir: dir, DevMode: true})
	ep.LoadAll(template.FuncMap{})

	mw := ep.Middleware()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	mw(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})(w, r)

	if w.Code != 500 {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestDefaultErrorPage(t *testing.T) {
	data := ErrorData{
		Status:     500,
		StatusText: "Internal Server Error",
		Message:    "Something went wrong",
		Path:       "/test",
		DevMode:    true,
	}

	html := defaultErrorPage(data)
	if !strings.Contains(html, "500") {
		t.Error("expected 500 in error page")
	}
	if !strings.Contains(html, "Something went wrong") {
		t.Error("expected error message")
	}
}

func TestPathToDir(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/blog/post", "blog"},
		{"/blog/post/comments", "blog/post"},
		{"/about", "about"},
		{"/", ""},
	}

	for _, tt := range tests {
		got := pathToDir(tt.path)
		if got != tt.want {
			t.Errorf("pathToDir(%s) = %s, want %s", tt.path, got, tt.want)
		}
	}
}

func TestErrorData(t *testing.T) {
	data := ErrorData{
		Status:     403,
		StatusText: "Forbidden",
		Message:    "Access denied",
		Path:       "/admin",
		DevMode:    false,
	}

	if data.Status != 403 {
		t.Errorf("expected 403, got %d", data.Status)
	}
}
