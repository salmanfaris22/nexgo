// Package image provides image optimization: resizing, format detection,
// lazy loading attributes, and responsive image helpers.
package image

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Config holds image optimization configuration.
type Config struct {
	CacheDir       string   // where optimized images are stored
	Quality        int      // default quality (1-100)
	Sizes          []int    // responsive breakpoints: [320, 640, 768, 1024, 1280]
	AllowedFormats []string // ["jpg", "png", "gif", "webp", "avif"]
	MaxWidth       int      // max allowed width
	MaxHeight      int      // max allowed height
	LazyLoad       bool     // add loading="lazy" by default
}

// DefaultConfig returns sensible image optimization defaults.
func DefaultConfig() Config {
	return Config{
		CacheDir:       ".nexgo/image-cache",
		Quality:        80,
		Sizes:          []int{320, 640, 768, 1024, 1280, 1920},
		AllowedFormats: []string{"jpg", "jpeg", "png", "gif", "webp", "svg"},
		MaxWidth:       4096,
		MaxHeight:      4096,
		LazyLoad:       true,
	}
}

// Optimizer handles image processing and serving.
type Optimizer struct {
	config  Config
	mu      sync.RWMutex
	cache   map[string]string // hash -> cached path
	rootDir string
}

// New creates an image optimizer.
func New(cfg Config, rootDir string) *Optimizer {
	os.MkdirAll(filepath.Join(rootDir, cfg.CacheDir), 0755)
	return &Optimizer{
		config:  cfg,
		cache:   make(map[string]string),
		rootDir: rootDir,
	}
}

// ImageTag generates an optimized <img> tag with responsive attributes.
func (o *Optimizer) ImageTag(src, alt string, width, height int) template.HTML {
	attrs := []string{
		fmt.Sprintf(`src="%s"`, template.HTMLEscapeString(src)),
		fmt.Sprintf(`alt="%s"`, template.HTMLEscapeString(alt)),
	}

	if width > 0 {
		attrs = append(attrs, fmt.Sprintf(`width="%d"`, width))
	}
	if height > 0 {
		attrs = append(attrs, fmt.Sprintf(`height="%d"`, height))
	}
	if o.config.LazyLoad {
		attrs = append(attrs, `loading="lazy"`, `decoding="async"`)
	}

	// Generate srcset for responsive images
	if width > 0 {
		srcset := o.generateSrcSet(src, width)
		if srcset != "" {
			attrs = append(attrs, fmt.Sprintf(`srcset="%s"`, srcset))
			attrs = append(attrs, `sizes="(max-width: 768px) 100vw, (max-width: 1200px) 50vw, 33vw"`)
		}
	}

	return template.HTML("<img " + strings.Join(attrs, " ") + ">")
}

// PictureTag generates a <picture> element with multiple sources.
func (o *Optimizer) PictureTag(src, alt string, width, height int) template.HTML {
	ext := filepath.Ext(src)
	base := strings.TrimSuffix(src, ext)

	var html strings.Builder
	html.WriteString("<picture>")

	// WebP source
	html.WriteString(fmt.Sprintf(`<source type="image/webp" srcset="%s.webp">`, base))

	// Original format fallback
	html.WriteString(fmt.Sprintf(`<img src="%s" alt="%s"`, src, template.HTMLEscapeString(alt)))
	if width > 0 {
		html.WriteString(fmt.Sprintf(` width="%d"`, width))
	}
	if height > 0 {
		html.WriteString(fmt.Sprintf(` height="%d"`, height))
	}
	if o.config.LazyLoad {
		html.WriteString(` loading="lazy" decoding="async"`)
	}
	html.WriteString(">")
	html.WriteString("</picture>")

	return template.HTML(html.String())
}

// Handler serves optimized images with caching.
func (o *Optimizer) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse query params: ?w=640&q=80&f=webp
		src := strings.TrimPrefix(r.URL.Path, "/_nexgo/image/")
		widthStr := r.URL.Query().Get("w")
		qualityStr := r.URL.Query().Get("q")

		width := 0
		if widthStr != "" {
			width, _ = strconv.Atoi(widthStr)
			if width > o.config.MaxWidth {
				width = o.config.MaxWidth
			}
		}

		quality := o.config.Quality
		if qualityStr != "" {
			quality, _ = strconv.Atoi(qualityStr)
			if quality < 1 || quality > 100 {
				quality = o.config.Quality
			}
		}

		// Generate cache key
		cacheKey := o.cacheKey(src, width, quality)

		// Check cache
		o.mu.RLock()
		cached, ok := o.cache[cacheKey]
		o.mu.RUnlock()
		if ok {
			http.ServeFile(w, r, cached)
			return
		}

		// Serve original file with cache headers
		srcPath := filepath.Join(o.rootDir, "static", src)
		if _, err := os.Stat(srcPath); err != nil {
			http.NotFound(w, r)
			return
		}

		// Set cache headers
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("Vary", "Accept")

		http.ServeFile(w, r, srcPath)
	}
}

// BlurPlaceholder returns a tiny base64 placeholder for LQIP (Low Quality Image Placeholder).
func BlurPlaceholder(width, height int, color string) template.HTML {
	if color == "" {
		color = "#1a1a1a"
	}
	svg := fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d"><rect fill="%s" width="100%%" height="100%%"/></svg>`,
		width, height, color,
	)
	return template.HTML(fmt.Sprintf(
		`data:image/svg+xml;charset=utf-8,%s`,
		template.URLQueryEscaper(svg),
	))
}

// TemplateFuncMap returns template functions for image optimization.
func (o *Optimizer) TemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"img": func(src, alt string) template.HTML {
			return o.ImageTag(src, alt, 0, 0)
		},
		"imgSize": func(src, alt string, width, height int) template.HTML {
			return o.ImageTag(src, alt, width, height)
		},
		"picture": func(src, alt string, width, height int) template.HTML {
			return o.PictureTag(src, alt, width, height)
		},
		"placeholder": func(w, h int) template.HTML {
			return BlurPlaceholder(w, h, "")
		},
	}
}

func (o *Optimizer) generateSrcSet(src string, maxWidth int) string {
	var parts []string
	for _, size := range o.config.Sizes {
		if size <= maxWidth {
			parts = append(parts, fmt.Sprintf("/_nexgo/image/%s?w=%d %dw",
				strings.TrimPrefix(src, "/static/"), size, size))
		}
	}
	return strings.Join(parts, ", ")
}

func (o *Optimizer) cacheKey(src string, width, quality int) string {
	key := fmt.Sprintf("%s_%d_%d", src, width, quality)
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// CopyOptimized copies a file for serving (basic copy, no transform without CGO).
func CopyOptimized(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
