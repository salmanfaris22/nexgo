package renderer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nexgo/nexgo/pkg/config"
)

// PageData is passed to every template
type PageData struct {
	// Page metadata
	Title       string
	Description string
	Keywords    string
	OGImage     string
	Canonical   string

	// Route info
	Path   string
	Params map[string]string
	Query  map[string][]string

	// User data (from GetServerSideProps equivalent)
	Props map[string]interface{}

	// Framework internals
	NexGoVersion string
	DevMode      bool
	BuildID      string

	// Request
	Request *http.Request
}

// DataLoader is a function that loads data for a page (like getServerSideProps)
type DataLoader func(req *http.Request, params map[string]string) (map[string]interface{}, error)

// Renderer handles all template rendering
type Renderer struct {
	mu          sync.RWMutex
	cfg         *config.NexGoConfig
	templates   map[string]*template.Template
	layouts     map[string]*template.Template
	components  map[string]*template.Template
	dataLoaders map[string]DataLoader
	funcMap     template.FuncMap
	buildID     string
}

// New creates a new Renderer
func New(cfg *config.NexGoConfig) *Renderer {
	r := &Renderer{
		cfg:         cfg,
		templates:   make(map[string]*template.Template),
		layouts:     make(map[string]*template.Template),
		components:  make(map[string]*template.Template),
		dataLoaders: make(map[string]DataLoader),
		buildID:     fmt.Sprintf("%d", time.Now().Unix()),
	}
	r.funcMap = r.buildFuncMap()
	return r
}

// RegisterDataLoader registers a server-side data loader for a route
func (r *Renderer) RegisterDataLoader(route string, loader DataLoader) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dataLoaders[route] = loader
}

// LoadAll scans and compiles all templates
func (r *Renderer) LoadAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load layouts first
	if err := r.loadLayouts(); err != nil {
		return fmt.Errorf("loading layouts: %w", err)
	}

	// Load components
	if err := r.loadComponents(); err != nil {
		return fmt.Errorf("loading components: %w", err)
	}

	// Load pages
	if err := r.loadPages(); err != nil {
		return fmt.Errorf("loading pages: %w", err)
	}

	return nil
}

func (r *Renderer) loadLayouts() error {
	layoutsDir := r.cfg.AbsPath(r.cfg.LayoutsDir)
	if _, err := os.Stat(layoutsDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.WalkDir(layoutsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := filepath.Ext(path)
		if ext != ".html" && ext != ".gohtml" && ext != ".tmpl" {
			return nil
		}
		rel, _ := filepath.Rel(layoutsDir, path)
		name := strings.TrimSuffix(filepath.ToSlash(rel), ext)

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		tmpl, err := template.New(name).Funcs(r.funcMap).Parse(string(data))
		if err != nil {
			return fmt.Errorf("parsing layout %s: %w", name, err)
		}
		r.layouts[name] = tmpl
		return nil
	})
}

func (r *Renderer) loadComponents() error {
	compDir := r.cfg.AbsPath(r.cfg.ComponentsDir)
	if _, err := os.Stat(compDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.WalkDir(compDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := filepath.Ext(path)
		if ext != ".html" && ext != ".gohtml" && ext != ".tmpl" {
			return nil
		}
		rel, _ := filepath.Rel(compDir, path)
		name := strings.TrimSuffix(filepath.ToSlash(rel), ext)

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		tmpl, err := template.New(name).Funcs(r.funcMap).Parse(string(data))
		if err != nil {
			return fmt.Errorf("parsing component %s: %w", name, err)
		}
		r.components[name] = tmpl
		return nil
	})
}

func (r *Renderer) loadPages() error {
	pagesDir := r.cfg.PagesAbsDir()

	return filepath.WalkDir(pagesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := filepath.Ext(path)
		if ext != ".html" && ext != ".gohtml" && ext != ".tmpl" {
			return nil
		}

		rel, _ := filepath.Rel(pagesDir, path)
		name := strings.TrimSuffix(filepath.ToSlash(rel), ext)

		if err := r.loadPage(name, path); err != nil {
			fmt.Printf("[NexGo] Warning: failed to load page %s: %v\n", name, err)
		}
		return nil
	})
}

func (r *Renderer) loadPage(name, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Create template with all components available
	tmpl := template.New(name).Funcs(r.funcMap)

	// Add all components
	for compName, comp := range r.components {
		compClone, _ := comp.Clone()
		_ = compClone
		tmpl.AddParseTree(compName, comp.Tree)
	}

	// Parse page template
	if _, err := tmpl.Parse(string(data)); err != nil {
		return fmt.Errorf("parsing page %s: %w", name, err)
	}

	r.templates[name] = tmpl
	return nil
}

// RenderPage renders a page template to the response
func (r *Renderer) RenderPage(w http.ResponseWriter, req *http.Request, filePath string, params map[string]string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get template name from file path
	pagesDir := r.cfg.PagesAbsDir()
	rel, err := filepath.Rel(pagesDir, filePath)
	if err != nil {
		return err
	}
	ext := filepath.Ext(rel)
	name := strings.TrimSuffix(filepath.ToSlash(rel), ext)

	tmpl, ok := r.templates[name]
	if !ok {
		return fmt.Errorf("template not found: %s", name)
	}

	// Build page data
	pageData := &PageData{
		Title:        r.cfg.ProjectName,
		Path:         req.URL.Path,
		Params:       params,
		Query:        map[string][]string(req.URL.Query()),
		Props:        make(map[string]interface{}),
		NexGoVersion: "1.0.0",
		DevMode:      r.cfg.DevMode,
		BuildID:      r.buildID,
		Request:      req,
	}

	// Call data loader if registered
	if loader, ok := r.dataLoaders["/"+name]; ok {
		props, err := loader(req, params)
		if err != nil {
			return fmt.Errorf("data loader error for %s: %w", name, err)
		}
		pageData.Props = props
	}

	// Check for layout
	layout := r.detectLayout(name)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Powered-By", "NexGo")

	if layout != nil {
		return r.renderWithLayout(w, layout, tmpl, pageData)
	}

	return tmpl.Execute(w, pageData)
}

func (r *Renderer) detectLayout(pageName string) *template.Template {
	// Walk up directory tree looking for _layout
	parts := strings.Split(pageName, "/")
	for i := len(parts); i > 0; i-- {
		dir := strings.Join(parts[:i-1], "/")
		layoutName := "default"
		if dir != "" {
			layoutName = dir + "/layout"
		}
		if tmpl, ok := r.layouts[layoutName]; ok {
			return tmpl
		}
	}
	if tmpl, ok := r.layouts["default"]; ok {
		return tmpl
	}
	return nil
}

func (r *Renderer) renderWithLayout(w http.ResponseWriter, layout, page *template.Template, data *PageData) error {
	// Render page content first
	var contentBuf bytes.Buffer
	if err := page.Execute(&contentBuf, data); err != nil {
		return err
	}

	// Add rendered content to data
	type layoutData struct {
		*PageData
		Content template.HTML
	}
	ld := layoutData{
		PageData: data,
		Content:  template.HTML(contentBuf.String()),
	}

	return layout.Execute(w, ld)
}

// RenderError renders an error page
func (r *Renderer) RenderError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, errorPageHTML, status, http.StatusText(status), message)
}

// Reload recompiles all templates (used in dev mode)
func (r *Renderer) Reload() error {
	r.mu.Lock()
	r.templates = make(map[string]*template.Template)
	r.layouts = make(map[string]*template.Template)
	r.components = make(map[string]*template.Template)
	r.mu.Unlock()
	return r.LoadAll()
}

// buildFuncMap returns template helper functions
func (r *Renderer) buildFuncMap() template.FuncMap {
	return template.FuncMap{
		// JSON encode a value
		"json": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
		// Safe HTML
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		// Asset URL with cache busting
		"asset": func(path string) string {
			return fmt.Sprintf("/_nexgo/static/%s?v=%s", strings.TrimPrefix(path, "/"), r.buildID)
		},
		// Link to a page
		"link": func(path string) string {
			return path
		},
		// Iterate n times
		"times": func(n int) []int {
			result := make([]int, n)
			for i := range result {
				result[i] = i
			}
			return result
		},
		// String formatting
		"upper":   strings.ToUpper,
		"lower":   strings.ToLower,
		"title":   strings.Title,
		"replace": strings.ReplaceAll,
		"trim":    strings.TrimSpace,
		"split":   strings.Split,
		"join":    strings.Join,
		// Math
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		// Default value
		"default": func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},
	}
}

const errorPageHTML = `<!DOCTYPE html>
<html>
<head>
  <title>%d %s | NexGo</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body { 
      font-family: 'SF Mono', 'Fira Code', monospace;
      background: #0d0d0d; color: #e0e0e0;
      display: flex; align-items: center; justify-content: center;
      min-height: 100vh; padding: 2rem;
    }
    .container { max-width: 600px; width: 100%%; }
    .code { 
      font-size: 7rem; font-weight: 900; line-height: 1;
      background: linear-gradient(135deg, #ff4757, #ff6b81);
      -webkit-background-clip: text; -webkit-text-fill-color: transparent;
    }
    .status { font-size: 1.5rem; color: #888; margin: 0.5rem 0 1.5rem; }
    .message { 
      background: #1a1a1a; border: 1px solid #2a2a2a; border-radius: 8px;
      padding: 1rem 1.5rem; font-size: 0.9rem; color: #ccc;
    }
    .footer { margin-top: 2rem; font-size: 0.8rem; color: #444; }
    a { color: #00d2ff; text-decoration: none; }
  </style>
</head>
<body>
  <div class="container">
    <div class="code">%d</div>
    <div class="status">%s</div>
    <div class="message">%s</div>
    <div class="footer">
      <a href="/">← Go back home</a> &nbsp;·&nbsp; Powered by NexGo
    </div>
  </div>
</body>
</html>`
