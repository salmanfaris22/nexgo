package asset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.SourceDir != "static" {
		t.Errorf("expected static, got %s", cfg.SourceDir)
	}
	if !cfg.Minify {
		t.Error("expected Minify true")
	}
	if !cfg.Fingerprint {
		t.Error("expected Fingerprint true")
	}
}

func TestPipelineBuild(t *testing.T) {
	rootDir := t.TempDir()
	srcDir := filepath.Join(rootDir, "static")
	os.MkdirAll(filepath.Join(srcDir, "css"), 0755)
	os.MkdirAll(filepath.Join(srcDir, "js"), 0755)

	os.WriteFile(filepath.Join(srcDir, "css", "style.css"), []byte("body { color: red; }"), 0644)
	os.WriteFile(filepath.Join(srcDir, "js", "app.js"), []byte("console.log('hello');"), 0644)

	cfg := DefaultConfig()
	cfg.SourceDir = "static"
	cfg.OutputDir = ".nexgo/assets"

	p := New(cfg, rootDir)
	result, err := p.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CSSBundled != 1 {
		t.Errorf("expected 1 CSS file, got %d", result.CSSBundled)
	}
	if result.JSBundled != 1 {
		t.Errorf("expected 1 JS file, got %d", result.JSBundled)
	}
	if len(result.Manifest) == 0 {
		t.Error("expected non-empty manifest")
	}
}

func TestPipelineResolve(t *testing.T) {
	rootDir := t.TempDir()
	srcDir := filepath.Join(rootDir, "static")
	os.MkdirAll(filepath.Join(srcDir, "css"), 0755)
	os.WriteFile(filepath.Join(srcDir, "css", "style.css"), []byte("body{}"), 0644)

	cfg := DefaultConfig()
	p := New(cfg, rootDir)
	p.Build()

	resolved := p.Resolve("/static/css/style.css")
	if !strings.HasPrefix(resolved, "/assets/") {
		t.Errorf("expected /assets/ prefix, got %s", resolved)
	}
}

func TestPipelineResolve_NotFound(t *testing.T) {
	cfg := DefaultConfig()
	p := New(cfg, t.TempDir())
	resolved := p.Resolve("/static/missing.css")
	if resolved != "/static/missing.css" {
		t.Errorf("expected original path, got %s", resolved)
	}
}

func TestMinifyCSS(t *testing.T) {
	input := `
		/* Comment */
		body {
			color: red;
			background: white;
		}
		
		.empty { }
	`
	result := MinifyCSS(input)

	if strings.Contains(result, "/* Comment */") {
		t.Error("expected comments removed")
	}
	if strings.Contains(result, "  ") {
		t.Error("expected whitespace collapsed")
	}
	if strings.Contains(result, ".empty") {
		t.Error("expected empty rules removed")
	}
}

func TestMinifyJS(t *testing.T) {
	input := `
		// Single line comment
		function hello() {
			/* Multi-line
			   comment */
			console.log("hello");
		}
	`
	result := MinifyJS(input)

	if strings.Contains(result, "// Single line comment") {
		t.Error("expected single-line comments removed")
	}
	if strings.Contains(result, "/* Multi-line") {
		t.Error("expected multi-line comments removed")
	}
	if !strings.Contains(result, "console.log") {
		t.Error("expected code preserved")
	}
}

func TestMinifyHTML(t *testing.T) {
	input := `
		<!-- Comment -->
		<html>
			<body>
				<h1>Hello</h1>
			</body>
		</html>
	`
	result := MinifyHTML(input)

	if strings.Contains(result, "<!-- Comment -->") {
		t.Error("expected comments removed")
	}
	if strings.Contains(result, "  ") {
		t.Error("expected whitespace collapsed")
	}
	if !strings.Contains(result, "<h1>Hello</h1>") {
		t.Error("expected content preserved")
	}
}

func TestContentHash(t *testing.T) {
	h1 := contentHash([]byte("hello"))
	h2 := contentHash([]byte("hello"))
	h3 := contentHash([]byte("world"))

	if h1 != h2 {
		t.Error("same content should produce same hash")
	}
	if h1 == h3 {
		t.Error("different content should produce different hash")
	}
}

func TestInlineCSS(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "style.css")
	os.WriteFile(file, []byte("body { color: red; }"), 0644)

	result := InlineCSS(file)
	if !strings.HasPrefix(result, "<style>") {
		t.Error("expected style tag")
	}
	if !strings.Contains(result, "color:red") {
		t.Error("expected minified CSS")
	}
}

func TestInlineCSS_NotFound(t *testing.T) {
	result := InlineCSS("/nonexistent.css")
	if result != "" {
		t.Errorf("expected empty string, got %s", result)
	}
}

func TestInlineJS(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "app.js")
	os.WriteFile(file, []byte("console.log('hello');"), 0644)

	result := InlineJS(file)
	if !strings.HasPrefix(result, "<script>") {
		t.Error("expected script tag")
	}
	if !strings.Contains(result, "console.log") {
		t.Error("expected JS code preserved")
	}
}

func TestInlineJS_NotFound(t *testing.T) {
	result := InlineJS("/nonexistent.js")
	if result != "" {
		t.Errorf("expected empty string, got %s", result)
	}
}

func TestCriticalCSS(t *testing.T) {
	css := `
		.hero { color: red; font-size: 2rem; }
		.footer { color: gray; }
		.hero-title { font-weight: bold; }
	`
	result := CriticalCSS(css, []string{"hero"})

	if !strings.Contains(result, ".hero") {
		t.Error("expected .hero rule")
	}
	if !strings.Contains(result, ".hero-title") {
		t.Error("expected .hero-title rule")
	}
	if strings.Contains(result, ".footer") {
		t.Error("expected .footer rule excluded")
	}
}

func TestPipeline_NoBundle(t *testing.T) {
	rootDir := t.TempDir()
	srcDir := filepath.Join(rootDir, "static")
	os.MkdirAll(filepath.Join(srcDir, "css"), 0755)
	os.WriteFile(filepath.Join(srcDir, "css", "a.css"), []byte("body{}"), 0644)
	os.WriteFile(filepath.Join(srcDir, "css", "b.css"), []byte("p{}"), 0644)

	cfg := DefaultConfig()
	cfg.BundleCSS = false
	cfg.BundleJS = false

	p := New(cfg, rootDir)
	result, err := p.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CSSBundled != 0 {
		t.Errorf("expected 0 bundled, got %d", result.CSSBundled)
	}
}

func TestPipeline_OtherFiles(t *testing.T) {
	rootDir := t.TempDir()
	srcDir := filepath.Join(rootDir, "static")
	os.MkdirAll(filepath.Join(srcDir, "images"), 0755)
	os.WriteFile(filepath.Join(srcDir, "images", "logo.png"), []byte("PNG"), 0644)

	cfg := DefaultConfig()
	cfg.BundleCSS = false
	cfg.BundleJS = false

	p := New(cfg, rootDir)
	result, err := p.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OtherCopied != 1 {
		t.Errorf("expected 1 other copied, got %d", result.OtherCopied)
	}
}

func TestPipelineManifest(t *testing.T) {
	rootDir := t.TempDir()
	srcDir := filepath.Join(rootDir, "static")
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "style.css"), []byte("body{}"), 0644)

	cfg := DefaultConfig()
	cfg.BundleCSS = false
	cfg.BundleJS = false

	p := New(cfg, rootDir)
	p.Build()

	m := p.Manifest()
	if len(m) == 0 {
		t.Error("expected non-empty manifest")
	}
}
