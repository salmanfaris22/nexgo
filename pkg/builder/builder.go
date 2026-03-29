package builder

import (
	"fmt"
	"log"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nexgo/nexgo/pkg/config"
	"github.com/nexgo/nexgo/pkg/renderer"
	"github.com/nexgo/nexgo/pkg/router"
)

// BuildResult contains build statistics
type BuildResult struct {
	PagesBuilt   int
	StaticCopied int
	Duration     time.Duration
	OutputDir    string
	Errors       []error
}

// Builder handles static site generation
type Builder struct {
	cfg      *config.NexGoConfig
	router   *router.Router
	renderer *renderer.Renderer
}

// New creates a new Builder
func New(cfg *config.NexGoConfig) *Builder {
	return &Builder{
		cfg:      cfg,
		router:   router.New(cfg.PagesAbsDir()),
		renderer: renderer.New(cfg),
	}
}

// Build runs the full static site generation
func (b *Builder) Build() (*BuildResult, error) {
	start := time.Now()
	result := &BuildResult{OutputDir: b.cfg.OutputAbsDir()}

	log.Println("[NexGo] 🔨 Building...")

	// Clean output directory
	if err := os.RemoveAll(b.cfg.OutputAbsDir()); err != nil {
		return nil, fmt.Errorf("cleaning output dir: %w", err)
	}
	if err := os.MkdirAll(b.cfg.OutputAbsDir(), 0755); err != nil {
		return nil, fmt.Errorf("creating output dir: %w", err)
	}

	// Scan routes
	if err := b.router.Scan(); err != nil {
		return nil, fmt.Errorf("scanning routes: %w", err)
	}

	// Load templates
	if err := b.renderer.LoadAll(); err != nil {
		return nil, fmt.Errorf("loading templates: %w", err)
	}

	// Build each page
	for _, route := range b.router.GetRoutes() {
		if route.Type != router.RouteTypePage {
			continue
		}

		if err := b.buildPage(route); err != nil {
			log.Printf("[NexGo] Warning: failed to build %s: %v", route.Pattern, err)
			result.Errors = append(result.Errors, err)
		} else {
			result.PagesBuilt++
			log.Printf("[NexGo]   ✓ %s", route.Pattern)
		}
	}

	// Copy static files
	n, err := b.copyStatic()
	if err != nil {
		log.Printf("[NexGo] Warning: static copy error: %v", err)
	}
	result.StaticCopied = n

	result.Duration = time.Since(start)

	log.Printf("\n[NexGo] ✅ Built %d pages, %d static files in %s",
		result.PagesBuilt, result.StaticCopied, result.Duration.Round(time.Millisecond))
	log.Printf("[NexGo]    Output: %s\n", result.OutputDir)

	return result, nil
}

func (b *Builder) buildPage(route *router.Route) error {
	// Create fake request for rendering
	req := httptest.NewRequest("GET", route.Pattern, nil)
	w := httptest.NewRecorder()

	if err := b.renderer.RenderPage(w, req, route.FilePath, map[string]string{}); err != nil {
		return err
	}

	// Determine output path
	outPath := b.routeToOutputPath(route.Pattern)
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(outPath, w.Body.Bytes(), 0644)
}

func (b *Builder) routeToOutputPath(pattern string) string {
	// "/" → index.html
	// "/about" → about/index.html
	// "/blog/my-post" → blog/my-post/index.html
	clean := strings.TrimPrefix(pattern, "/")
	if clean == "" {
		return filepath.Join(b.cfg.OutputAbsDir(), "index.html")
	}
	return filepath.Join(b.cfg.OutputAbsDir(), clean, "index.html")
}

func (b *Builder) copyStatic() (int, error) {
	src := b.cfg.StaticAbsDir()
	dst := filepath.Join(b.cfg.OutputAbsDir(), "static")

	if _, err := os.Stat(src); os.IsNotExist(err) {
		return 0, nil
	}

	count := 0
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		dest := filepath.Join(dst, rel)

		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dest, data, 0644); err != nil {
			return err
		}
		count++
		return nil
	})

	return count, err
}
