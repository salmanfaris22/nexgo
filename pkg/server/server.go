package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/salmanfaris22/nexgo/v2/pkg/cache"
	"github.com/salmanfaris22/nexgo/v2/pkg/cluster"
	"github.com/salmanfaris22/nexgo/v2/pkg/config"
	"github.com/salmanfaris22/nexgo/v2/pkg/devtools"
	"github.com/salmanfaris22/nexgo/v2/pkg/islands"
	"github.com/salmanfaris22/nexgo/v2/pkg/middleware"
	"github.com/salmanfaris22/nexgo/v2/pkg/renderer"
	"github.com/salmanfaris22/nexgo/v2/pkg/router"
	"github.com/salmanfaris22/nexgo/v2/pkg/seo"
	"github.com/salmanfaris22/nexgo/v2/pkg/watcher"
)

// HMRClient is a connected hot-reload SSE client
type HMRClient struct {
	ch chan string
}

// routeEntry holds a custom handler registered before Start
type routeEntry struct {
	pattern string
	handler http.HandlerFunc
}

// Server is the NexGo HTTP server
type Server struct {
	cfg           *config.NexGoConfig
	router        *router.Router
	renderer      *renderer.Renderer
	watcher       *watcher.Watcher
	httpServer    *http.Server
	cluster       *cluster.Cluster
	responseCache *cache.Cache
	hmrMu         sync.RWMutex
	hmrClients    map[*HMRClient]bool
	pendingRoutes []routeEntry
}

// New creates a NexGo server
func New(cfg *config.NexGoConfig) (*Server, error) {
	s := &Server{
		cfg:        cfg,
		router:     router.New(cfg.PagesAbsDir()),
		renderer:   renderer.New(cfg),
		hmrClients: make(map[*HMRClient]bool),
	}
	if cfg.DevMode {
		s.watcher = watcher.New(500 * time.Millisecond)
	}
	return s, nil
}

// RegisterDataLoader adds a data loader for a route (getServerSideProps equivalent)
func (s *Server) RegisterDataLoader(route string, loader renderer.DataLoader) {
	s.renderer.RegisterDataLoader(route, loader)
}

// RegisterGlobalState adds state that is available to all pages
func (s *Server) RegisterGlobalState(key string, value interface{}) {
	s.renderer.RegisterGlobalState(key, value)
}

// RegisterRoute registers a custom HTTP handler for a path pattern.
// Handlers registered here take priority over the catch-all page/API router.
func (s *Server) RegisterRoute(pattern string, handler http.HandlerFunc) {
	s.pendingRoutes = append(s.pendingRoutes, routeEntry{pattern, handler})
}

// Start boots the server
func (s *Server) Start(ctx context.Context) error {
	if err := s.router.Scan(); err != nil {
		return fmt.Errorf("scanning routes: %w", err)
	}
	s.router.BindAPIHandlers()

	if err := s.renderer.LoadAll(); err != nil {
		return fmt.Errorf("loading templates: %w", err)
	}

	mux := http.NewServeMux()

	// Apply middleware to custom routes
	mainMW := middleware.Chain(
		middleware.Recover,
		middleware.Logger,
		middleware.SecurityHeaders,
	)
	for _, re := range s.pendingRoutes {
		mux.HandleFunc(re.pattern, mainMW(re.handler))
	}

	mux.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir(s.cfg.StaticAbsDir()))))

	mux.HandleFunc("/_nexgo/runtime.js", s.handleRuntime)
	mux.HandleFunc("/_nexgo/islands/", s.handleIslandJS)
	mux.HandleFunc("/_nexgo/island-runtime.js", s.handleIslandRuntime)
	mux.HandleFunc("/_nexgo/live", s.handleLive)

	// SEO: auto sitemap.xml
	if s.cfg.SEO.AutoSitemap {
		mux.HandleFunc("/sitemap.xml", s.handleSitemap)
	}
	// SEO: auto robots.txt
	if s.cfg.SEO.AutoRobotsTxt {
		mux.HandleFunc("/robots.txt", s.handleRobotsTxt)
	}

	if s.cfg.DevMode {
		mux.HandleFunc("/_nexgo/hmr", s.handleHMR)
		mux.HandleFunc("/_nexgo/routes", s.handleDevRoutes)
		mux.HandleFunc("/_nexgo/reload", s.handleManualReload)
		mux.HandleFunc("/_nexgo/devtools", s.handleDevTools)

		s.watcher.Watch(s.cfg.PagesAbsDir())
		s.watcher.Watch(s.cfg.AbsPath(s.cfg.LayoutsDir))
		s.watcher.Watch(s.cfg.AbsPath(s.cfg.ComponentsDir))
		s.watcher.Watch(s.cfg.AbsPath(s.cfg.IslandsDir))
		s.watcher.OnChange(func(e watcher.Event) {
			log.Printf("[NexGo] %s", filepath.Base(e.Path))
			s.reload()
		})
		s.watcher.Start()
	}

	// Choose logger middleware: async for production, sync for dev
	loggerMW := middleware.Logger
	if s.cfg.AsyncLogging && !s.cfg.DevMode {
		loggerMW = middleware.AsyncLogger
	}

	seoLang := s.cfg.SEO.Language
	mainHandler := middleware.Chain(
		middleware.Recover,
		loggerMW,
		middleware.SecurityHeaders,
	)(seo.SEOHeaders(seoLang, s.handleRequest))

	// Response caching for production GET requests
	if s.cfg.ResponseCache && !s.cfg.DevMode {
		ttl := time.Duration(s.cfg.ResponseCacheTTL) * time.Second
		if ttl == 0 {
			ttl = 5 * time.Minute
		}
		s.responseCache = cache.New(ttl)
		mainHandler = cache.Middleware(s.responseCache, ttl)(mainHandler)
	}

	if s.cfg.Compression {
		mainHandler = middleware.Gzip(mainHandler)
	}
	mux.HandleFunc("/", mainHandler)

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if s.cfg.ReadBufferSize > 0 {
		s.httpServer.ReadHeaderTimeout = 5 * time.Second
		s.httpServer.MaxHeaderBytes = s.cfg.ReadBufferSize
	}

	s.printBanner(addr)

	// Cluster mode: multi-worker for production
	if s.cfg.ClusterMode && !s.cfg.DevMode {
		clusterCfg := cluster.Config{
			Workers:         s.cfg.ClusterWorkers,
			GracefulTimeout: 30 * time.Second,
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     60 * time.Second,
		}
		s.cluster = cluster.New(clusterCfg, mux)

		go func() {
			<-ctx.Done()
			if err := s.cluster.Shutdown(); err != nil {
				log.Printf("[NexGo] Cluster shutdown error: %v", err)
			}
			if s.responseCache != nil {
				s.responseCache.Stop()
			}
		}()

		return s.cluster.ListenAndServe(addr)
	}

	// Single-server mode (dev or non-cluster)
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutCtx); err != nil {
			log.Printf("[NexGo] Shutdown error: %v", err)
		}
		if s.watcher != nil {
			s.watcher.Stop()
		}
		if s.responseCache != nil {
			s.responseCache.Stop()
		}
	}()

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.httpServer.Serve(ln)
}
func (s *Server) handleRequest(w http.ResponseWriter, req *http.Request) {
	if !s.cfg.DevMode && strings.HasPrefix(req.URL.Path, "/_nexgo/") {
		http.NotFound(w, req)
		return
	}

	route, params := s.router.Match(req.URL.Path)
	if route == nil {
		s.renderer.RenderError(w, http.StatusNotFound, "Page not found: "+req.URL.Path)
		return
	}
	switch route.Type {
	case router.RouteTypeAPI:
		if route.Handler != nil {
			ctx := router.WithParams(req.Context(), params)
			route.Handler(w, req.WithContext(ctx))
		} else {
			http.Error(w, "API route not implemented", http.StatusNotImplemented)
		}
	case router.RouteTypePage:
		if err := s.renderer.RenderPage(w, req, route.FilePath, params); err != nil {
			log.Printf("[NexGo] Render error %s: %v", req.URL.Path, err)
			s.renderer.RenderError(w, http.StatusInternalServerError, err.Error())
		}
	}
}

func (s *Server) reload() {
	if err := s.renderer.Reload(); err != nil {
		log.Printf("[NexGo] Reload error: %v", err)
		msg, _ := json.Marshal(map[string]string{"type": "error", "message": err.Error()})
		s.broadcastHMR(string(msg))
		return
	}
	if err := s.router.Scan(); err != nil {
		log.Printf("[NexGo] Rescan error: %v", err)
		return
	}
	s.broadcastHMR(`{"type":"reload"}`)
	log.Println("[NexGo] Reloaded")
}

func (s *Server) handleHMR(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	c := &HMRClient{ch: make(chan string, 10)}
	s.hmrMu.Lock()
	s.hmrClients[c] = true
	s.hmrMu.Unlock()
	defer func() {
		s.hmrMu.Lock()
		delete(s.hmrClients, c)
		s.hmrMu.Unlock()
	}()

	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	tick := time.NewTicker(20 * time.Second)
	defer tick.Stop()

	for {
		select {
		case msg := <-c.ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-tick.C:
			fmt.Fprintf(w, ": ping\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) broadcastHMR(msg string) {
	s.hmrMu.RLock()
	defer s.hmrMu.RUnlock()
	for c := range s.hmrClients {
		select {
		case c.ch <- msg:
		default:
		}
	}
}

func (s *Server) handleDevRoutes(w http.ResponseWriter, r *http.Request) {
	type info struct {
		Pattern string `json:"pattern"`
		File    string `json:"file"`
		Type    string `json:"type"`
	}
	var list []info
	for _, rt := range s.router.GetRoutes() {
		t := "page"
		if rt.Type == router.RouteTypeAPI {
			t = "api"
		}
		rel := strings.TrimPrefix(rt.FilePath, s.cfg.RootDir)
		list = append(list, info{rt.Pattern, rel, t})
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(list); err != nil {
		log.Printf("[NexGo] JSON encode error: %v", err)
	}
}

func (s *Server) handleDevTools(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if _, err := w.Write([]byte(devtools.DevToolsHTML)); err != nil {
		log.Printf("[NexGo] Write error: %v", err)
	}
}

func (s *Server) handleManualReload(w http.ResponseWriter, r *http.Request) {
	s.reload()
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
		log.Printf("[NexGo] Write error: %v", err)
	}
}

func (s *Server) handleLive(w http.ResponseWriter, r *http.Request) {
	text := r.URL.Query().Get("text")
	tag := r.URL.Query().Get("tag")
	if tag == "" {
		tag = "h1"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if text != "" {
		escaped := template.HTMLEscapeString(text)
		fmt.Fprintf(w, "<%s class=\"gradient\" style=\"font-size:3rem;font-weight:900;\">%s</%s>", tag, escaped, tag)
	} else {
		fmt.Fprint(w, `<p style="color:var(--muted);">Start typing above...</p>`)
	}
}

// handleIslandJS serves individual island JS files.
// GET /_nexgo/islands/counter.js → serves islands/counter.js
func (s *Server) handleIslandJS(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/_nexgo/islands/")
	name = strings.TrimSuffix(name, ".js")

	data, ok := s.renderer.Islands().GetJS(name)
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	if s.cfg.DevMode {
		w.Header().Set("Cache-Control", "no-cache")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	w.Write(data)
}

// handleIslandRuntime serves the island hydration runtime.
func (s *Server) handleIslandRuntime(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	if s.cfg.DevMode {
		w.Header().Set("Cache-Control", "no-cache")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}
	w.Write([]byte(islands.RuntimeJS()))
}

// handleSitemap auto-generates sitemap.xml from the route table.
func (s *Server) handleSitemap(w http.ResponseWriter, r *http.Request) {
	routes := s.router.GetRoutes()
	patterns := make([]string, 0, len(routes))
	for _, rt := range routes {
		patterns = append(patterns, rt.Pattern)
	}

	baseURL := s.cfg.SEO.SiteURL
	if baseURL == "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		baseURL = fmt.Sprintf("%s://%s", scheme, r.Host)
	}

	entries := seo.AutoSitemap(baseURL, patterns, 0.5)
	data, err := seo.RenderSitemap(entries)
	if err != nil {
		http.Error(w, "sitemap error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(data)
}

// handleRobotsTxt auto-generates robots.txt from SEO config.
func (s *Server) handleRobotsTxt(w http.ResponseWriter, r *http.Request) {
	baseURL := s.cfg.SEO.SiteURL
	if baseURL == "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		baseURL = fmt.Sprintf("%s://%s", scheme, r.Host)
	}
	sitemapURL := ""
	if s.cfg.SEO.AutoSitemap {
		sitemapURL = baseURL + "/sitemap.xml"
	}
	content := seo.RobotsTxt(s.cfg.SEO.RobotsAllow, s.cfg.SEO.RobotsDisallow, sitemapURL)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write([]byte(content))
}

func (s *Server) handleRuntime(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	if s.cfg.DevMode {
		w.Header().Set("Cache-Control", "no-cache")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}
	if _, err := w.Write([]byte(nexgoRuntime)); err != nil {
		log.Printf("[NexGo] Write error: %v", err)
	}
}

func (s *Server) printBanner(addr string) {
	mode := "\033[32mproduction\033[0m"
	if s.cfg.ClusterMode && !s.cfg.DevMode {
		mode = "\033[32mproduction\033[0m (cluster mode)"
	}
	if s.cfg.DevMode {
		mode = "\033[33mdevelopment\033[0m (hot reload on)"
	}
	fmt.Println()
	fmt.Println("  \033[36m⚡ NexGo — Go-powered web framework\033[0m")
	fmt.Println()
	fmt.Printf("  \033[1mLocal:\033[0m   \033[4mhttp://%s\033[0m\n", addr)
	fmt.Printf("  \033[1mMode:\033[0m    %s\n", mode)
	fmt.Printf("  \033[1mPages:\033[0m   ./%s/\n", s.cfg.PagesDir)
	islandNames := s.renderer.Islands().Names()
	if len(islandNames) > 0 {
		fmt.Printf("  \033[1mIslands:\033[0m %d (%s)\n", len(islandNames), strings.Join(islandNames, ", "))
	}
	if s.cfg.ClusterMode && !s.cfg.DevMode {
		workers := s.cfg.ClusterWorkers
		if workers <= 0 {
			workers = runtime.NumCPU()
		}
		fmt.Printf("  \033[1mWorkers:\033[0m %d CPU cores\n", workers)
	}
	if s.cfg.ResponseCache && !s.cfg.DevMode {
		fmt.Printf("  \033[1mCache:\033[0m   response caching (%ds TTL)\n", s.cfg.ResponseCacheTTL)
	}
	if s.cfg.AsyncLogging && !s.cfg.DevMode {
		fmt.Printf("  \033[1mLogger:\033[0m  async (non-blocking)\n")
	}
	if s.cfg.DevMode {
		fmt.Printf("  \033[1mDevtools:\033[0m \033[4mhttp://%s/_nexgo/devtools\033[0m\n", addr)
	}
	fmt.Println()

	routes := s.router.GetRoutes()
	if len(routes) > 0 {
		fmt.Println("  \033[90mRoutes discovered:\033[0m")
		for _, rt := range routes {
			icon := "\033[32m●\033[0m"
			if rt.Type == router.RouteTypeAPI {
				icon = "\033[35m◆\033[0m"
			}
			rel := strings.TrimPrefix(rt.FilePath, s.cfg.RootDir)
			fmt.Printf("  %s %-28s \033[90m%s\033[0m\n", icon, rt.Pattern, rel)
		}
		fmt.Println()
	}
}

const nexgoRuntime = `
(function(){
'use strict';

const NexGo={
  version:'1.0.0',
  state: {},
  _listeners: [],

  router:{
    navigate(href){
      if(!href||href.startsWith('http')||href.startsWith('//')||href.startsWith('#')||href.startsWith('mailto:')){
        if(href)location.href=href;return;
      }
      history.pushState(null,'',href);
      this._load(href);
    },
    async _load(url){
      const root=document.querySelector('#nexgo-root')||document.querySelector('main');
      if(!root){location.reload();return;}
      try{
        const res=await fetch(url,{headers:{'X-NexGo-SPA':'1','Accept':'text/html'}});
        if(!res.ok){location.href=url;return;}
        const html=await res.text();
        const doc=new DOMParser().parseFromString(html,'text/html');
        const newRoot=doc.querySelector('#nexgo-root')||doc.querySelector('main')||doc.body;
        
        // Handle state update from new page if present
        const newStateScript = doc.querySelector('#__nexgo_state');
        if (newStateScript) {
           try {
             const newState = JSON.parse(newStateScript.textContent);
             NexGo.updateState(newState);
           } catch(e) { console.error('Failed to update state during navigation', e); }
        }

        root.style.cssText='opacity:0;transform:translateY(6px);transition:opacity 0.15s,transform 0.15s';
        await new Promise(r=>setTimeout(r,150));
        root.innerHTML=newRoot.innerHTML;
        root.querySelectorAll('script').forEach(function(old){var s=document.createElement('script');s.textContent=old.textContent;old.replaceWith(s);});
        document.title=doc.title;
        root.style.cssText='opacity:0;transform:translateY(6px)';
        requestAnimationFrame(()=>{root.style.cssText='opacity:1;transform:translateY(0);transition:opacity 0.2s,transform 0.2s';});
        this._initLinks();
        NexGo._initLive();
        document.dispatchEvent(new CustomEvent('nexgo:navigate',{detail:{url}}));
      }catch(e){location.href=url;}
    },
    _initLinks(){
      document.querySelectorAll('a[href]:not([data-ng])').forEach(a=>{
        const h=a.getAttribute('href');
        if(!h||h.startsWith('http')||h.startsWith('//')||h.startsWith('#')||h.startsWith('mailto:'))return;
        a.dataset.ng='1';
        a.addEventListener('click',e=>{e.preventDefault();this.navigate(h);});
      });
    }
  },

  // State Management
  getState(key, defaultValue) {
    return key in this.state ? this.state[key] : defaultValue;
  },

  setState(key, value) {
    this.state[key] = value;
    this._notify();
  },

  updateState(newState) {
    this.state = { ...this.state, ...newState };
    this._notify();
  },

  subscribe(callback) {
    this._listeners.push(callback);
    return () => {
      this._listeners = this._listeners.filter(l => l !== callback);
    };
  },

  _notify() {
    this._listeners.forEach(l => l(this.state));
    document.dispatchEvent(new CustomEvent('nexgo:statechange', { detail: this.state }));
  },
_initHMR() {
  if (!document.documentElement.dataset.nexgoDev) return;
  console.log('%c[NexGo HMR] Connected','color:#00d2ff;font-weight:bold');
  const es = new EventSource('/_nexgo/hmr');
  es.onmessage = async (e) => {
    const m = JSON.parse(e.data);
    if (m.type === 'reload') {
      console.log('%c[NexGo HMR] Updating content...','color:#00d2ff');
      await this._fetchAndReplaceContent();
    } else if (m.type === 'css') {
      this._reloadCSS(m.path);
    } else if (m.type === 'error') {
      console.error('[NexGo HMR] Error:', m.message);
    }
  };
  es.onerror = () => console.warn('[NexGo HMR] Reconnecting...');
},

async _fetchAndReplaceContent() {
  const url = window.location.pathname + window.location.search;
  try {
    const res = await fetch(url, { headers: { 'X-NexGo-SPA': '1', 'Accept': 'text/html' } });
    if (!res.ok) { location.reload(); return; }
    const html = await res.text();
    const doc = new DOMParser().parseFromString(html, 'text/html');
    const newRoot = doc.querySelector('#nexgo-root') || doc.querySelector('main');
    const currentRoot = document.querySelector('#nexgo-root') || document.querySelector('main');
    if (newRoot && currentRoot) {
      currentRoot.style.opacity = '0';
      await new Promise(r => setTimeout(r, 100));
      currentRoot.innerHTML = newRoot.innerHTML;
      currentRoot.querySelectorAll('script').forEach(oldScript => {
        const newScript = document.createElement('script');
        newScript.textContent = oldScript.textContent;
        oldScript.replaceWith(newScript);
      });
      currentRoot.style.opacity = '1';
      this.router._initLinks();
      this._initLive();
      if (doc.title) document.title = doc.title;
    } else {
      location.reload();
    }
  } catch (e) {
    console.error('HMR fetch failed', e);
    location.reload();
  }
},

_reloadCSS(path) {
  const links = document.querySelectorAll('link[rel="stylesheet"]');
  links.forEach(link => {
    if (link.href && link.href.includes(path)) {
      const newLink = document.createElement('link');
      newLink.rel = 'stylesheet';
      newLink.href = link.href.split('?')[0] + '?v=' + Date.now();
      link.parentNode.replaceChild(newLink, link);
    }
  });
}
  _initPrefetch(){
    const done=new Set();
    document.addEventListener('mouseover',e=>{
      const a=e.target.closest('a[href]');
      if(!a||done.has(a.href))return;
      const h=a.getAttribute('href');
      if(!h||h.startsWith('http')||h.startsWith('#'))return;
      done.add(a.href);
      document.head.appendChild(Object.assign(document.createElement('link'),{rel:'prefetch',href:h}));
    },{passive:true});
  },

  _initLazy(){
    const imgs=document.querySelectorAll('img[data-src]');
    if(!imgs.length)return;
    if('IntersectionObserver' in window){
      const obs=new IntersectionObserver(es=>{
        es.forEach(e=>{if(e.isIntersecting){e.target.src=e.target.dataset.src;obs.unobserve(e.target);}});
      },{rootMargin:'200px'});
      imgs.forEach(i=>obs.observe(i));
    }else{imgs.forEach(i=>{i.src=i.dataset.src;});}
  },

  _initLive(){
    const input=document.getElementById('nexgo-input');
    const output=document.getElementById('live-output');
    if(!input||!output)return;
    let t=null;
    input.addEventListener('input',function(){
      clearTimeout(t);
      t=setTimeout(function(){
        var x=new XMLHttpRequest();
        x.open('GET','/_nexgo/live?text='+encodeURIComponent(input.value));
        x.onload=function(){if(x.status===200){output.innerHTML=x.responseText;}};
        x.send();
      },50);
    });
    input.addEventListener('focus',function(){input.style.borderColor='var(--accent)';});
    input.addEventListener('blur',function(){input.style.borderColor='var(--border)';});
  },

  _hydrateState() {
    const script = document.getElementById('__nexgo_state');
    if (script) {
      try {
        this.state = JSON.parse(script.textContent);
      } catch (e) {
        console.error('Failed to hydrate state:', e);
      }
    }
  },

  init(){
    this._hydrateState();
    this.router._initLinks();
    this._initForms();
    this._initHMR();
    this._initPrefetch();
    this._initLazy();
    this._initLive();
    window.addEventListener('popstate',()=>this.router._load(location.pathname));
    document.addEventListener('nexgo:navigate',()=>this._initForms());
    document.dispatchEvent(new CustomEvent('nexgo:ready',{}));
  },

  _initForms(){
    var self=this;
    document.querySelectorAll('form[data-fragment]:not([data-ng-form])').forEach(function(f){
      f.setAttribute('data-ng-form','1');
      f.addEventListener('submit',function(e){e.preventDefault();});
      var btn=f.querySelector('button[type=submit],input[type=submit]');
      if(!btn)return;
      btn.addEventListener('click',function(e){
        e.preventDefault();
        var actionUrl=f.getAttribute('action')||window.location.pathname;
        var frag=f.getAttribute('data-fragment');
        var tid=f.getAttribute('data-target');
        var fd=new FormData(f);
        f.reset();
        fetch(actionUrl,{method:'POST',body:fd,headers:{'X-Requested-With':'XMLHttpRequest'}})
          .then(function(){return fetch(frag);})
          .then(function(r){return r.text();})
          .then(function(html){
            var el=document.getElementById(tid);
            if(el){
              el.innerHTML=html;
              self._initForms();
            }
          });
      });
    });
  }
};

document.addEventListener('DOMContentLoaded',()=>NexGo.init());
window.NexGo=NexGo;
})();
`
