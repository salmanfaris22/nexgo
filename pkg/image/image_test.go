package image

import (
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Quality != 80 {
		t.Errorf("expected quality 80, got %d", cfg.Quality)
	}
	if !cfg.LazyLoad {
		t.Error("expected LazyLoad true")
	}
	if len(cfg.Sizes) != 6 {
		t.Errorf("expected 6 sizes, got %d", len(cfg.Sizes))
	}
}

func TestImageTag(t *testing.T) {
	cfg := DefaultConfig()
	o := New(cfg, "/project")

	tag := string(o.ImageTag("/static/photo.jpg", "A photo", 800, 600))

	checks := []string{
		`src="/static/photo.jpg"`,
		`alt="A photo"`,
		`width="800"`,
		`height="600"`,
		`loading="lazy"`,
		`decoding="async"`,
	}

	for _, check := range checks {
		if !strings.Contains(tag, check) {
			t.Errorf("expected %q in tag", check)
		}
	}
}

func TestImageTag_NoDimensions(t *testing.T) {
	cfg := DefaultConfig()
	o := New(cfg, "/project")

	tag := string(o.ImageTag("/static/photo.jpg", "Photo", 0, 0))

	if !strings.Contains(tag, `src="/static/photo.jpg"`) {
		t.Error("expected src")
	}
	if strings.Contains(tag, "width=") {
		t.Error("expected no width when 0")
	}
}

func TestImageTag_Escaping(t *testing.T) {
	cfg := DefaultConfig()
	o := New(cfg, "/project")

	tag := string(o.ImageTag("/static/photo.jpg", "<script>alert('xss')</script>", 0, 0))

	if strings.Contains(tag, "<script>") {
		t.Error("expected alt escaping")
	}
}

func TestPictureTag(t *testing.T) {
	cfg := DefaultConfig()
	o := New(cfg, "/project")

	tag := string(o.PictureTag("/static/photo.jpg", "Photo", 800, 600))

	checks := []string{
		"<picture>",
		`type="image/webp"`,
		`src="/static/photo.jpg"`,
		`alt="Photo"`,
		"</picture>",
	}

	for _, check := range checks {
		if !strings.Contains(tag, check) {
			t.Errorf("expected %q in picture tag", check)
		}
	}
}

func TestPictureTag_LazyLoad(t *testing.T) {
	cfg := DefaultConfig()
	o := New(cfg, "/project")

	tag := string(o.PictureTag("/static/photo.jpg", "Photo", 800, 600))
	if !strings.Contains(tag, `loading="lazy"`) {
		t.Error("expected lazy loading in picture tag")
	}
}

func TestBlurPlaceholder(t *testing.T) {
	placeholder := string(BlurPlaceholder(100, 100, "#1a1a1a"))

	if !strings.Contains(placeholder, "data:image/svg+xml") {
		t.Error("expected data URI")
	}
	if !strings.Contains(placeholder, "width=\"100\"") {
		t.Error("expected width in SVG")
	}
	if !strings.Contains(placeholder, "height=\"100\"") {
		t.Error("expected height in SVG")
	}
}

func TestBlurPlaceholder_DefaultColor(t *testing.T) {
	placeholder := string(BlurPlaceholder(50, 50, ""))
	if !strings.Contains(placeholder, "#1a1a1a") {
		t.Error("expected default color #1a1a1a")
	}
}

func TestTemplateFuncMap(t *testing.T) {
	cfg := DefaultConfig()
	o := New(cfg, "/project")

	fns := o.TemplateFuncMap()

	if _, ok := fns["img"]; !ok {
		t.Error("expected img function")
	}
	if _, ok := fns["imgSize"]; !ok {
		t.Error("expected imgSize function")
	}
	if _, ok := fns["picture"]; !ok {
		t.Error("expected picture function")
	}
	if _, ok := fns["placeholder"]; !ok {
		t.Error("expected placeholder function")
	}
}

func TestGenerateSrcSet(t *testing.T) {
	cfg := DefaultConfig()
	o := New(cfg, "/project")

	srcset := o.generateSrcSet("/static/photo.jpg", 1024)
	if srcset == "" {
		t.Error("expected non-empty srcset")
	}
	if !strings.Contains(srcset, "320w") {
		t.Error("expected 320w breakpoint")
	}
	if !strings.Contains(srcset, "1024w") {
		t.Error("expected 1024w breakpoint")
	}
}

func TestCacheKey(t *testing.T) {
	cfg := DefaultConfig()
	o := New(cfg, "/project")

	k1 := o.cacheKey("photo.jpg", 640, 80)
	k2 := o.cacheKey("photo.jpg", 640, 80)
	k3 := o.cacheKey("photo.jpg", 320, 80)

	if k1 != k2 {
		t.Error("same params should produce same key")
	}
	if k1 == k3 {
		t.Error("different params should produce different keys")
	}
}
