package builder

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/salmanfaris22/nexgo/v2/pkg/config"
)

func TestNew(t *testing.T) {
	cfg := &config.NexGoConfig{
		RootDir:       t.TempDir(),
		PagesDir:      "pages",
		LayoutsDir:    "layouts",
		StaticDir:     "static",
		OutputDir:     ".nexgo/out",
		ComponentsDir: "components",
		IslandsDir:    "islands",
	}
	os.MkdirAll(filepath.Join(cfg.RootDir, "pages"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "layouts"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "static"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "components"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "islands"), 0755)

	b := New(cfg)
	if b == nil {
		t.Fatal("expected builder")
	}
	if b.Renderer() == nil {
		t.Error("expected renderer")
	}
}

func TestBuild_EmptyPages(t *testing.T) {
	cfg := &config.NexGoConfig{
		RootDir:       t.TempDir(),
		PagesDir:      "pages",
		LayoutsDir:    "layouts",
		StaticDir:     "static",
		OutputDir:     ".nexgo/out",
		ComponentsDir: "components",
		IslandsDir:    "islands",
	}
	os.MkdirAll(filepath.Join(cfg.RootDir, "pages"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "layouts"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "static"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "components"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "islands"), 0755)

	b := New(cfg)
	result, err := b.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PagesBuilt != 0 {
		t.Errorf("expected 0 pages, got %d", result.PagesBuilt)
	}
	if result.Duration == 0 {
		t.Error("expected non-zero duration")
	}
}

func TestBuild_WithPage(t *testing.T) {
	cfg := &config.NexGoConfig{
		RootDir:       t.TempDir(),
		PagesDir:      "pages",
		LayoutsDir:    "layouts",
		StaticDir:     "static",
		OutputDir:     ".nexgo/out",
		ComponentsDir: "components",
		IslandsDir:    "islands",
		ProjectName:   "test-app",
	}
	os.MkdirAll(filepath.Join(cfg.RootDir, "pages"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "layouts"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "static"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "components"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "islands"), 0755)

	os.WriteFile(filepath.Join(cfg.RootDir, "pages", "index.html"), []byte("<h1>Hello</h1>"), 0644)

	b := New(cfg)
	result, err := b.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PagesBuilt != 1 {
		t.Errorf("expected 1 page, got %d", result.PagesBuilt)
	}

	outPath := filepath.Join(cfg.RootDir, ".nexgo/out", "index.html")
	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("expected output file at %s", outPath)
	}
}

func TestBuild_StaticCopy(t *testing.T) {
	cfg := &config.NexGoConfig{
		RootDir:       t.TempDir(),
		PagesDir:      "pages",
		LayoutsDir:    "layouts",
		StaticDir:     "static",
		OutputDir:     ".nexgo/out",
		ComponentsDir: "components",
		IslandsDir:    "islands",
		ProjectName:   "test-app",
	}
	os.MkdirAll(filepath.Join(cfg.RootDir, "pages"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "layouts"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "static/css"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "components"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "islands"), 0755)

	os.WriteFile(filepath.Join(cfg.RootDir, "pages", "index.html"), []byte("<h1>Hello</h1>"), 0644)
	os.WriteFile(filepath.Join(cfg.RootDir, "static/css", "style.css"), []byte("body{}"), 0644)

	b := New(cfg)
	result, err := b.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StaticCopied != 1 {
		t.Errorf("expected 1 static file, got %d", result.StaticCopied)
	}

	cssPath := filepath.Join(cfg.RootDir, ".nexgo/out", "static", "css", "style.css")
	if _, err := os.Stat(cssPath); err != nil {
		t.Errorf("expected static file at %s", cssPath)
	}
}

func TestBuild_NoStaticDir(t *testing.T) {
	cfg := &config.NexGoConfig{
		RootDir:       t.TempDir(),
		PagesDir:      "pages",
		LayoutsDir:    "layouts",
		StaticDir:     "static",
		OutputDir:     ".nexgo/out",
		ComponentsDir: "components",
		IslandsDir:    "islands",
		ProjectName:   "test-app",
	}
	os.MkdirAll(filepath.Join(cfg.RootDir, "pages"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "layouts"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "components"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "islands"), 0755)

	os.WriteFile(filepath.Join(cfg.RootDir, "pages", "index.html"), []byte("<h1>Hello</h1>"), 0644)

	b := New(cfg)
	result, err := b.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StaticCopied != 0 {
		t.Errorf("expected 0 static files, got %d", result.StaticCopied)
	}
}

func TestRegisterDataLoader(t *testing.T) {
	cfg := &config.NexGoConfig{
		RootDir:       t.TempDir(),
		PagesDir:      "pages",
		LayoutsDir:    "layouts",
		StaticDir:     "static",
		OutputDir:     ".nexgo/out",
		ComponentsDir: "components",
		IslandsDir:    "islands",
	}
	os.MkdirAll(filepath.Join(cfg.RootDir, "pages"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "layouts"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "static"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "components"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "islands"), 0755)

	b := New(cfg)
	b.RegisterDataLoader("/test", func(req *http.Request, params map[string]string) (map[string]interface{}, error) {
		return map[string]interface{}{"key": "value"}, nil
	})
}

func TestRegisterGlobalState(t *testing.T) {
	cfg := &config.NexGoConfig{
		RootDir:       t.TempDir(),
		PagesDir:      "pages",
		LayoutsDir:    "layouts",
		StaticDir:     "static",
		OutputDir:     ".nexgo/out",
		ComponentsDir: "components",
		IslandsDir:    "islands",
	}
	os.MkdirAll(filepath.Join(cfg.RootDir, "pages"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "layouts"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "static"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "components"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "islands"), 0755)

	b := New(cfg)
	b.RegisterGlobalState("user", map[string]interface{}{"name": "John"})
}

func TestRouteToOutputPath(t *testing.T) {
	cfg := &config.NexGoConfig{
		RootDir:       t.TempDir(),
		PagesDir:      "pages",
		LayoutsDir:    "layouts",
		StaticDir:     "static",
		OutputDir:     ".nexgo/out",
		ComponentsDir: "components",
		IslandsDir:    "islands",
	}
	os.MkdirAll(filepath.Join(cfg.RootDir, "pages"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "layouts"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "static"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "components"), 0755)
	os.MkdirAll(filepath.Join(cfg.RootDir, "islands"), 0755)

	b := New(cfg)

	tests := []struct {
		pattern string
		want    string
	}{
		{"/", "index.html"},
		{"/about", "about/index.html"},
		{"/blog/my-post", "blog/my-post/index.html"},
	}

	for _, tt := range tests {
		got := b.routeToOutputPath(tt.pattern)
		if got != filepath.Join(cfg.OutputAbsDir(), tt.want) {
			t.Errorf("routeToOutputPath(%s) = %s, want %s", tt.pattern, got, filepath.Join(cfg.OutputAbsDir(), tt.want))
		}
	}
}
