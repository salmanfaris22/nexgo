package islands

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create island HTML template
	os.WriteFile(filepath.Join(dir, "counter.html"), []byte(
		`<div class="counter"><span>{{ .count }}</span></div>`,
	), 0644)

	// Create island JS
	os.WriteFile(filepath.Join(dir, "counter.js"), []byte(
		`export default function(el, props) { console.log(props); }`,
	), 0644)

	// Create an island with no JS (SSR only)
	os.WriteFile(filepath.Join(dir, "banner.html"), []byte(
		`<div class="banner">{{ .text }}</div>`,
	), 0644)

	return dir
}

func TestRegistryScan(t *testing.T) {
	dir := setupTestDir(t)
	r := NewRegistry(dir, template.FuncMap{})

	if err := r.Scan(); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should find both islands
	names := r.Names()
	if len(names) != 2 {
		t.Fatalf("expected 2 islands, got %d: %v", len(names), names)
	}

	// Counter should have JS
	counter, ok := r.Get("counter")
	if !ok {
		t.Fatal("counter island not found")
	}
	if !counter.HasJS {
		t.Error("counter should have JS")
	}
	if counter.Template == nil {
		t.Error("counter should have a template")
	}

	// Banner should NOT have JS
	banner, ok := r.Get("banner")
	if !ok {
		t.Fatal("banner island not found")
	}
	if banner.HasJS {
		t.Error("banner should not have JS")
	}
}

func TestRegistryScanEmptyDir(t *testing.T) {
	r := NewRegistry("/nonexistent/path", template.FuncMap{})
	if err := r.Scan(); err != nil {
		t.Fatalf("Scan on nonexistent dir should not error: %v", err)
	}
	if len(r.Names()) != 0 {
		t.Error("expected 0 islands for nonexistent dir")
	}
}

func TestGetJS(t *testing.T) {
	dir := setupTestDir(t)
	r := NewRegistry(dir, template.FuncMap{})
	r.Scan()

	data, ok := r.GetJS("counter")
	if !ok {
		t.Fatal("expected JS for counter")
	}
	if !strings.Contains(string(data), "export default") {
		t.Error("JS content missing export")
	}

	_, ok = r.GetJS("banner")
	if ok {
		t.Error("banner should not have JS")
	}

	_, ok = r.GetJS("nonexistent")
	if ok {
		t.Error("nonexistent island should not have JS")
	}
}

func TestRender(t *testing.T) {
	dir := setupTestDir(t)
	r := NewRegistry(dir, template.FuncMap{})
	r.Scan()

	t.Run("basic render with props", func(t *testing.T) {
		html := r.Render("counter", map[string]interface{}{"count": 42}, "client:load")
		s := string(html)

		if !strings.Contains(s, "<nexgo-island") {
			t.Error("missing <nexgo-island> wrapper")
		}
		if !strings.Contains(s, `data-name="counter"`) {
			t.Error("missing data-name attribute")
		}
		if !strings.Contains(s, `data-strategy="client:load"`) {
			t.Error("missing data-strategy attribute")
		}
		if !strings.Contains(s, `data-has-js="true"`) {
			t.Error("missing data-has-js attribute")
		}
		if !strings.Contains(s, "<span>42</span>") {
			t.Error("template not rendered with props")
		}
	})

	t.Run("client:visible strategy", func(t *testing.T) {
		html := r.Render("counter", map[string]interface{}{"count": 0}, "client:visible")
		if !strings.Contains(string(html), `data-strategy="client:visible"`) {
			t.Error("wrong strategy")
		}
	})

	t.Run("client:none returns raw HTML", func(t *testing.T) {
		html := r.Render("banner", map[string]interface{}{"text": "Hello"}, StrategyNone)
		s := string(html)
		if strings.Contains(s, "<nexgo-island") {
			t.Error("client:none should NOT wrap in <nexgo-island>")
		}
		if !strings.Contains(s, "Hello") {
			t.Error("template content missing")
		}
	})

	t.Run("island without JS", func(t *testing.T) {
		html := r.Render("banner", map[string]interface{}{"text": "Hi"}, "client:load")
		if !strings.Contains(string(html), `data-has-js="false"`) {
			t.Error("banner should have data-has-js=false")
		}
	})

	t.Run("nonexistent island", func(t *testing.T) {
		html := r.Render("missing", nil, "")
		if !strings.Contains(string(html), "not found") {
			t.Error("expected not found comment")
		}
	})

	t.Run("default strategy", func(t *testing.T) {
		html := r.Render("counter", map[string]interface{}{"count": 1}, "")
		if !strings.Contains(string(html), `data-strategy="client:load"`) {
			t.Error("empty strategy should default to client:load")
		}
	})

	t.Run("nil props", func(t *testing.T) {
		html := r.Render("counter", nil, "client:load")
		s := string(html)
		if !strings.Contains(s, "<nexgo-island") {
			t.Error("should still render with nil props")
		}
	})
}

func TestRuntimeJS(t *testing.T) {
	js := RuntimeJS()
	if js == "" {
		t.Fatal("runtime JS is empty")
	}
	if !strings.Contains(js, "nexgo-island") {
		t.Error("runtime should reference nexgo-island elements")
	}
	if !strings.Contains(js, "client:visible") {
		t.Error("runtime should handle client:visible strategy")
	}
	if !strings.Contains(js, "client:idle") {
		t.Error("runtime should handle client:idle strategy")
	}
	if !strings.Contains(js, "IntersectionObserver") {
		t.Error("runtime should use IntersectionObserver for visible strategy")
	}
}
