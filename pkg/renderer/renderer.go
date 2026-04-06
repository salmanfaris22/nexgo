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
	"sync/atomic"
	"time"

	"github.com/salmanfaris22/nexgo/v2/pkg/config"
	"github.com/salmanfaris22/nexgo/v2/pkg/islands"
	"github.com/salmanfaris22/nexgo/v2/pkg/seo"
	"github.com/salmanfaris22/nexgo/v2/pkg/worker"
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

	// Global state (shared across all pages)
	State map[string]interface{}

	// Framework internals
	NexGoVersion string
	DevMode      bool
	BuildID      string

	// Request
	Request *http.Request
}

// DataLoader is a function that loads data for a page (like getServerSideProps)
type DataLoader func(req *http.Request, params map[string]string) (map[string]interface{}, error)

// ParallelLoader loads multiple data sources concurrently and merges results.
type ParallelLoader struct {
	loaders map[string]DataLoader
}

// NewParallelLoader creates a new parallel data loader.
func NewParallelLoader() *ParallelLoader {
	return &ParallelLoader{loaders: make(map[string]DataLoader)}
}

// Add registers a named data loader.
func (p *ParallelLoader) Add(name string, loader DataLoader) *ParallelLoader {
	p.loaders[name] = loader
	return p
}

// Execute runs all loaders concurrently and merges results.
func (p *ParallelLoader) Execute(req *http.Request, params map[string]string) (map[string]interface{}, error) {
	type result struct {
		name string
		data map[string]interface{}
		err  error
	}

	results := worker.Map(4, keys(p.loaders), func(name string) result {
		data, err := p.loaders[name](req, params)
		return result{name, data, err}
	})

	merged := make(map[string]interface{})
	for _, r := range results {
		if r.err != nil {
			return nil, fmt.Errorf("loader %q failed: %w", r.name, r.err)
		}
		for k, v := range r.data {
			merged[k] = v
		}
	}
	return merged, nil
}

func keys(m map[string]DataLoader) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TemplateCache stores compiled templates for fast access.
type TemplateCache struct {
	mu         sync.RWMutex
	templates  map[string]*template.Template
	layouts    map[string]*template.Template
	components map[string]*template.Template
	loadedAt   time.Time
}

// NewTemplateCache creates an empty template cache.
func NewTemplateCache() *TemplateCache {
	return &TemplateCache{
		templates:  make(map[string]*template.Template),
		layouts:    make(map[string]*template.Template),
		components: make(map[string]*template.Template),
	}
}

// Get retrieves a template by name.
func (c *TemplateCache) Get(name string) (*template.Template, bool) {
	c.mu.RLock()
	t, ok := c.templates[name]
	c.mu.RUnlock()
	return t, ok
}

// Set stores a compiled template.
func (c *TemplateCache) Set(name string, tmpl *template.Template) {
	c.mu.Lock()
	c.templates[name] = tmpl
	c.mu.Unlock()
}

// GetLayout retrieves a layout by name.
func (c *TemplateCache) GetLayout(name string) (*template.Template, bool) {
	c.mu.RLock()
	t, ok := c.layouts[name]
	c.mu.RUnlock()
	return t, ok
}

// SetLayout stores a compiled layout.
func (c *TemplateCache) SetLayout(name string, tmpl *template.Template) {
	c.mu.Lock()
	c.layouts[name] = tmpl
	c.mu.Unlock()
}

// Clear resets the entire cache.
func (c *TemplateCache) Clear() {
	c.mu.Lock()
	c.templates = make(map[string]*template.Template)
	c.layouts = make(map[string]*template.Template)
	c.components = make(map[string]*template.Template)
	c.loadedAt = time.Now()
	c.mu.Unlock()
}

// templateSnapshot is an immutable snapshot used for lock-free reads.
type templateSnapshot struct {
	templates  map[string]*template.Template
	layouts    map[string]*template.Template
	components map[string]*template.Template
}

// Renderer handles all template rendering
type Renderer struct {
	mu          sync.RWMutex
	cfg         *config.NexGoConfig
	templates   map[string]*template.Template
	layouts     map[string]*template.Template
	components  map[string]*template.Template
	dataLoaders map[string]DataLoader
	globalState map[string]interface{}
	funcMap     template.FuncMap
	buildID     string
	cache       *TemplateCache
	islands     *islands.Registry
	snap        atomic.Value // stores *templateSnapshot for lock-free reads
}

// New creates a new Renderer
func New(cfg *config.NexGoConfig) *Renderer {
	r := &Renderer{
		cfg:         cfg,
		templates:   make(map[string]*template.Template),
		layouts:     make(map[string]*template.Template),
		components:  make(map[string]*template.Template),
		dataLoaders: make(map[string]DataLoader),
		globalState: make(map[string]interface{}),
		buildID:     fmt.Sprintf("%d", time.Now().Unix()),
		cache:       NewTemplateCache(),
	}
	r.funcMap = r.buildFuncMap()

	// Initialize islands registry
	islandsDir := cfg.AbsPath(cfg.IslandsDir)
	r.islands = islands.NewRegistry(islandsDir, r.funcMap)

	return r
}

// Islands returns the island registry for use by the server.
func (r *Renderer) Islands() *islands.Registry {
	return r.islands
}

// RegisterDataLoader registers a server-side data loader for a route
func (r *Renderer) RegisterDataLoader(route string, loader DataLoader) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dataLoaders[route] = loader
}

// RegisterGlobalState registers state that is available to all pages
func (r *Renderer) RegisterGlobalState(key string, value interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.globalState[key] = value
}

// LoadAll scans and compiles all templates
func (r *Renderer) LoadAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load islands
	if err := r.islands.Scan(); err != nil {
		return fmt.Errorf("loading islands: %w", err)
	}

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

	// Publish an immutable snapshot for lock-free reads
	r.publishSnapshot()

	return nil
}

// publishSnapshot stores an immutable copy of the current template maps.
// Must be called while holding r.mu write lock.
func (r *Renderer) publishSnapshot() {
	snap := &templateSnapshot{
		templates:  make(map[string]*template.Template, len(r.templates)),
		layouts:    make(map[string]*template.Template, len(r.layouts)),
		components: make(map[string]*template.Template, len(r.components)),
	}
	for k, v := range r.templates {
		snap.templates[k] = v
	}
	for k, v := range r.layouts {
		snap.layouts[k] = v
	}
	for k, v := range r.components {
		snap.components[k] = v
	}
	r.snap.Store(snap)
}

func (r *Renderer) loadTemplatesFromDir(dir string, targetMap map[string]*template.Template, typeName string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := filepath.Ext(path)
		if ext != ".html" && ext != ".gohtml" && ext != ".tmpl" {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("computing relative path for %s: %w", path, err)
		}
		name := strings.TrimSuffix(filepath.ToSlash(rel), ext)

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		tmpl, err := template.New(name).Funcs(r.funcMap).Parse(string(data))
		if err != nil {
			return fmt.Errorf("parsing %s %s: %w", typeName, name, err)
		}
		targetMap[name] = tmpl
		return nil
	})
}

func (r *Renderer) loadLayouts() error {
	return r.loadTemplatesFromDir(r.cfg.AbsPath(r.cfg.LayoutsDir), r.layouts, "layout")
}

func (r *Renderer) loadComponents() error {
	return r.loadTemplatesFromDir(r.cfg.AbsPath(r.cfg.ComponentsDir), r.components, "component")
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

		rel, err := filepath.Rel(pagesDir, path)
		if err != nil {
			return fmt.Errorf("computing relative path for %s: %w", path, err)
		}
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

	// Clone each component to avoid shared parse tree conflicts
	for compName, comp := range r.components {
		cloned, err := comp.Clone()
		if err != nil {
			return fmt.Errorf("cloning component %s: %w", compName, err)
		}
		tmpl.AddParseTree(compName, cloned.Tree)
	}

	// Parse page template
	if _, err := tmpl.Parse(string(data)); err != nil {
		return fmt.Errorf("parsing page %s: %w", name, err)
	}

	r.templates[name] = tmpl
	return nil
}

// RenderPage renders a page template to the response.
// Uses an atomic snapshot for lock-free template lookups in production.
func (r *Renderer) RenderPage(w http.ResponseWriter, req *http.Request, filePath string, params map[string]string) error {
	pagesDir := r.cfg.PagesAbsDir()
	rel, err := filepath.Rel(pagesDir, filePath)
	if err != nil {
		return fmt.Errorf("computing relative path: %w", err)
	}
	ext := filepath.Ext(rel)
	name := strings.TrimSuffix(filepath.ToSlash(rel), ext)

	// Try lock-free read from atomic snapshot first
	var tmpl *template.Template
	var layout *template.Template
	if snapVal := r.snap.Load(); snapVal != nil {
		snap := snapVal.(*templateSnapshot)
		tmpl = snap.templates[name]
		if tmpl != nil {
			layout = r.detectLayoutFromSnap(name, snap)
		}
	}

	// Fallback to mutex-protected read (dev mode or first request before snapshot)
	if tmpl == nil {
		r.mu.RLock()
		tmpl = r.templates[name]
		r.mu.RUnlock()
		if tmpl == nil {
			return fmt.Errorf("template not found: %s", name)
		}
	}

	// Build page data
	pageData := &PageData{
		Title:        r.cfg.ProjectName,
		Path:         req.URL.Path,
		Params:       params,
		Query:        map[string][]string(req.URL.Query()),
		Props:        make(map[string]interface{}),
		State:        make(map[string]interface{}),
		NexGoVersion: "2.2.1",
		DevMode:      r.cfg.DevMode,
		BuildID:      r.buildID,
		Request:      req,
	}

	// Auto-canonical from SEO config
	if r.cfg.SEO.AutoCanonical && r.cfg.SEO.SiteURL != "" {
		pageData.Canonical = r.cfg.SEO.SiteURL + req.URL.Path
	}

	// Copy global state
	for k, v := range r.globalState {
		pageData.State[k] = v
	}

	// Call data loader if registered (use route pattern as key)
	routePattern := "/" + name
	if name == "index" {
		routePattern = "/"
	} else if strings.HasSuffix(name, "/index") {
		routePattern = "/" + strings.TrimSuffix(name, "/index")
	}
	if loader, ok := r.dataLoaders[routePattern]; ok {
		props, err := loader(req, params)
		if err != nil {
			return fmt.Errorf("data loader error for %s: %w", name, err)
		}
		pageData.Props = props
	}

	// Resolve layout if not already found via snapshot
	if layout == nil {
		r.mu.RLock()
		layout = r.detectLayout(name)
		r.mu.RUnlock()
	}

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

// detectLayoutFromSnap resolves a layout from the atomic snapshot (lock-free).
func (r *Renderer) detectLayoutFromSnap(pageName string, snap *templateSnapshot) *template.Template {
	parts := strings.Split(pageName, "/")
	for i := len(parts); i > 0; i-- {
		dir := strings.Join(parts[:i-1], "/")
		layoutName := "default"
		if dir != "" {
			layoutName = dir + "/layout"
		}
		if tmpl, ok := snap.layouts[layoutName]; ok {
			return tmpl
		}
	}
	if tmpl, ok := snap.layouts["default"]; ok {
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
	fmt.Fprintf(w, errorPageHTML, status, http.StatusText(status), status, http.StatusText(status), message)
}

// Reload recompiles all templates (used in dev mode)
func (r *Renderer) Reload() error {
	r.mu.Lock()
	r.templates = make(map[string]*template.Template)
	r.layouts = make(map[string]*template.Template)
	r.components = make(map[string]*template.Template)
	r.mu.Unlock()
	// LoadAll will call publishSnapshot at the end
	return r.LoadAll()
}

// buildFuncMap returns template helper functions
func (r *Renderer) buildFuncMap() template.FuncMap {
	return template.FuncMap{
		// JSON encode a value

		"json": func(v interface{}) template.JS {
			b, err := json.Marshal(v)
			if err != nil {
				return template.JS("{}")
			}
			return template.JS(b)
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"dict": func(values ...interface{}) map[string]interface{} {
			dict := make(map[string]interface{})
			for i := 0; i+1 < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					continue
				}
				dict[key] = values[i+1]
			}
			return dict
		},
		"slice": func(values ...interface{}) []interface{} {
			return values
		},
		// Asset URL with cache busting
		"asset": func(path string) string {
			return fmt.Sprintf("/_nexgo/static/%s?v=%s", strings.TrimPrefix(path, "/"), r.buildID)
		},
		// State hydration script
		"renderState": func(state map[string]interface{}) template.HTML {
			b, err := json.Marshal(state)
			if err != nil {
				return template.HTML("")
			}
			return template.HTML(fmt.Sprintf("<script id=\"__nexgo_state\" type=\"application/json\">%s</script>", string(b)))
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
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": func(s string) string {
			words := strings.Fields(s)
			for i, w := range words {
				if len(w) > 0 {
					words[i] = strings.ToUpper(w[:1]) + w[1:]
				}
			}
			return strings.Join(words, " ")
		},
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

		// Island Architecture — render an interactive island component.
		// Usage:
		//   {{ island "counter" }}
		//   {{ island "counter" (props "count" 5) }}
		//   {{ island "counter" (props "count" 5) "client:visible" }}
		//   {{ island "chart" (props "data" .Props.sales) "client:idle" }}
		//   {{ island "menu" nil "client:media=(max-width:768px)" }}
		//   {{ island "sidebar" (props "user" .State.user) "client:none" }}
		"island": func(name string, args ...interface{}) template.HTML {
			var props map[string]interface{}
			strategy := "client:load"

			for _, arg := range args {
				switch v := arg.(type) {
				case map[string]interface{}:
					props = v
				case string:
					strategy = v
				}
			}

			return r.islands.Render(name, props, strategy)
		},

		// Props builds a key-value map for passing to islands.
		// Usage: {{ island "counter" (props "count" 5 "label" "Clicks") }}
		"props": func(pairs ...interface{}) map[string]interface{} {
			m := make(map[string]interface{})
			for i := 0; i+1 < len(pairs); i += 2 {
				if key, ok := pairs[i].(string); ok {
					m[key] = pairs[i+1]
				}
			}
			return m
		},

		// islandRuntime injects the island hydration runtime script.
		// Place this in your layout: {{ islandRuntime }}
		"islandRuntime": func() template.HTML {
			return template.HTML(fmt.Sprintf("<script>%s</script>", islands.RuntimeJS()))
		},

		// --- SEO Template Functions ---

		// seoTags renders all meta tags from a seo.Meta struct.
		// Usage: {{ seoTags .SEOMeta }}
		"seoTags": func(m seo.Meta) template.HTML {
			sc := seo.SiteConfig{
				SiteName:       r.cfg.SEO.SiteName,
				SiteURL:        r.cfg.SEO.SiteURL,
				TitleTemplate:  r.cfg.SEO.TitleTemplate,
				DefaultOGImage: r.cfg.SEO.DefaultOGImage,
				TwitterSite:    r.cfg.SEO.TwitterSite,
				DefaultLocale:  "en_US",
				Language:       r.cfg.SEO.Language,
				FaviconURL:     r.cfg.SEO.FaviconURL,
				ThemeColor:     r.cfg.SEO.ThemeColor,
				Author:         r.cfg.SEO.Author,
			}
			return seo.RenderMetaTags(m, sc)
		},

		// seoMeta creates a basic Meta struct for use in templates.
		// Usage: {{ seoTags (seoMeta "Page Title" "Description" "/page-url") }}
		"seoMeta": func(title, description, canonical string) seo.Meta {
			return seo.DefaultMeta(title, description, canonical)
		},

		// breadcrumbs generates JSON-LD BreadcrumbList from current path.
		// Usage: {{ breadcrumbs .Path }}
		"breadcrumbs": func(path string) template.HTML {
			baseURL := r.cfg.SEO.SiteURL
			items := seo.AutoBreadcrumbs(baseURL, path)
			return seo.BreadcrumbSchema(items)
		},

		// jsonld generates a JSON-LD script tag from a map.
		// Usage: {{ jsonld (dict "@context" "https://schema.org" "@type" "WebSite" "name" "My Site") }}
		"jsonld": func(data map[string]interface{}) template.HTML {
			return seo.JSONLD(data)
		},

		// websiteSchema generates WebSite JSON-LD with optional search.
		// Usage: {{ websiteSchema "My Site" "https://example.com" "https://example.com/search?q=" }}
		"websiteSchema": func(name, url string, searchURL ...string) template.HTML {
			search := ""
			if len(searchURL) > 0 {
				search = searchURL[0]
			}
			return seo.WebSiteSchema(name, url, search)
		},

		// orgSchema generates Organization JSON-LD.
		// Usage: {{ orgSchema "My Org" "https://example.com" "https://example.com/logo.png" }}
		"orgSchema": func(name, url, logo string) template.HTML {
			return seo.OrganizationSchema(name, url, logo, nil)
		},

		// faqSchema generates FAQ JSON-LD from pairs of question/answer.
		// Usage: {{ faqSchema "Q1?" "A1." "Q2?" "A2." }}
		"faqSchema": func(pairs ...string) template.HTML {
			var items []seo.FAQItem
			for i := 0; i+1 < len(pairs); i += 2 {
				items = append(items, seo.FAQItem{Question: pairs[i], Answer: pairs[i+1]})
			}
			return seo.FAQSchema(items)
		},

		// preload generates a preload link tag.
		// Usage: {{ preload "/static/css/global.css" "style" }}
		"preload": func(href, as string) template.HTML {
			return template.HTML(fmt.Sprintf(`<link rel="preload" href="%s" as="%s">`, template.HTMLEscapeString(href), template.HTMLEscapeString(as)))
		},

		// vitals injects Core Web Vitals tracking script.
		// Usage: {{ vitals }}
		"vitals": func() template.HTML {
			if !r.cfg.SEO.CoreWebVitals {
				return template.HTML("")
			}
			return seo.CoreWebVitalsScript(r.cfg.SEO.VitalsEndpoint)
		},

		// slugify converts text to SEO-friendly slug.
		// Usage: {{ slugify "Hello World!" }}
		"slugify": seo.Slugify,
	}
}

// LoadParallel compiles all templates using worker pool for parallel processing.
func (r *Renderer) LoadParallel() error {
	r.mu.Lock()
	r.templates = make(map[string]*template.Template)
	r.layouts = make(map[string]*template.Template)
	r.components = make(map[string]*template.Template)
	r.mu.Unlock()

	var errs []error

	// Load layouts in parallel
	layoutFiles := r.collectFiles(r.cfg.AbsPath(r.cfg.LayoutsDir), "layout")
	if len(layoutFiles) > 0 {
		errs = append(errs, worker.Run(4, layoutFiles)...)
	}

	// Load components in parallel
	compFiles := r.collectFiles(r.cfg.AbsPath(r.cfg.ComponentsDir), "component")
	if len(compFiles) > 0 {
		errs = append(errs, worker.Run(4, compFiles)...)
	}

	// Load pages in parallel
	pageFiles := r.collectPageFiles()
	if len(pageFiles) > 0 {
		errs = append(errs, worker.Run(4, pageFiles)...)
	}

	if len(errs) > 0 {
		return fmt.Errorf("loading templates: %v", errs[0])
	}
	return nil
}

func (r *Renderer) collectFiles(dir string, typeName string) []worker.Task {
	var tasks []worker.Task
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return tasks
	}
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".html" && ext != ".gohtml" && ext != ".tmpl" {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		name := strings.TrimSuffix(filepath.ToSlash(rel), ext)
		tasks = append(tasks, func() error {
			return r.loadAndCacheTemplate(path, name, typeName)
		})
		return nil
	})
	return tasks
}

func (r *Renderer) collectPageFiles() []worker.Task {
	var tasks []worker.Task
	pagesDir := r.cfg.PagesAbsDir()
	filepath.WalkDir(pagesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".html" && ext != ".gohtml" && ext != ".tmpl" {
			return nil
		}
		rel, _ := filepath.Rel(pagesDir, path)
		name := strings.TrimSuffix(filepath.ToSlash(rel), ext)
		tasks = append(tasks, func() error {
			return r.loadAndCachePage(path, name)
		})
		return nil
	})
	return tasks
}

func (r *Renderer) loadAndCacheTemplate(path, name, typeName string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	tmpl, err := template.New(name).Funcs(r.funcMap).Parse(string(data))
	if err != nil {
		return fmt.Errorf("parsing %s %s: %w", typeName, name, err)
	}
	r.mu.Lock()
	if typeName == "component" {
		r.components[name] = tmpl
	} else {
		r.layouts[name] = tmpl
	}
	r.mu.Unlock()
	return nil
}

func (r *Renderer) loadAndCachePage(path, name string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	tmpl := template.New(name).Funcs(r.funcMap)
	r.mu.RLock()
	for compName, comp := range r.components {
		cloned, err := comp.Clone()
		if err != nil {
			r.mu.RUnlock()
			return fmt.Errorf("cloning component %s: %w", compName, err)
		}
		tmpl.AddParseTree(compName, cloned.Tree)
	}
	r.mu.RUnlock()
	if _, err := tmpl.Parse(string(data)); err != nil {
		return fmt.Errorf("parsing page %s: %w", name, err)
	}
	r.cache.Set(name, tmpl)
	r.mu.Lock()
	r.templates[name] = tmpl
	r.mu.Unlock()
	return nil
}

// CacheInfo returns template cache statistics.
func (r *Renderer) CacheInfo() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return map[string]interface{}{
		"templates":  len(r.templates),
		"layouts":    len(r.layouts),
		"components": len(r.components),
		"loaded_at":  r.cache.loadedAt,
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
