package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nexgo/nexgo/pkg/config"
	"github.com/nexgo/nexgo/pkg/middleware"
	"github.com/nexgo/nexgo/pkg/renderer"
	"github.com/nexgo/nexgo/pkg/router"
	"github.com/nexgo/nexgo/pkg/watcher"
)

// HMRClient is a connected hot-reload SSE client
type HMRClient struct {
	ch chan string
}

// Server is the NexGo HTTP server
type Server struct {
	cfg        *config.NexGoConfig
	router     *router.Router
	renderer   *renderer.Renderer
	watcher    *watcher.Watcher
	httpServer *http.Server
	hmrMu      sync.RWMutex
	hmrClients map[*HMRClient]bool
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

// Start boots the server
func (s *Server) Start(ctx context.Context) error {
	if err := s.router.Scan(); err != nil {
		return fmt.Errorf("scanning routes: %w", err)
	}
	if err := s.renderer.LoadAll(); err != nil {
		return fmt.Errorf("loading templates: %w", err)
	}

	mux := http.NewServeMux()

	// Static assets
	mux.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir(s.cfg.StaticAbsDir()))))

	// NexGo runtime JS
	mux.HandleFunc("/_nexgo/runtime.js", s.handleRuntime)

	if s.cfg.DevMode {
		mux.HandleFunc("/_nexgo/hmr", s.handleHMR)
		mux.HandleFunc("/_nexgo/routes", s.handleDevRoutes)
		mux.HandleFunc("/_nexgo/reload", s.handleManualReload)

		s.watcher.Watch(s.cfg.PagesAbsDir())
		s.watcher.Watch(s.cfg.AbsPath(s.cfg.LayoutsDir))
		s.watcher.Watch(s.cfg.AbsPath(s.cfg.ComponentsDir))
		s.watcher.OnChange(func(e watcher.Event) {
			log.Printf("[NexGo] 🔄 %s", filepath.Base(e.Path))
			s.reload()
		})
		s.watcher.Start()
	}

	mainHandler := middleware.Chain(
		middleware.Recover,
		middleware.Logger,
		middleware.SecurityHeaders,
	)(s.handleRequest)

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

	s.printBanner(addr)

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpServer.Shutdown(shutCtx)
	}()

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.httpServer.Serve(ln)
}

func (s *Server) handleRequest(w http.ResponseWriter, req *http.Request) {
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
		s.broadcastHMR(`{"type":"error","message":"` + err.Error() + `"}`)
		return
	}
	s.router.Scan()
	s.broadcastHMR(`{"type":"reload"}`)
	log.Println("[NexGo] ✅ Reloaded")
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
		list = append(list, info{rt.Pattern, rt.FilePath, t})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) handleManualReload(w http.ResponseWriter, r *http.Request) {
	s.reload()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleRuntime(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	if s.cfg.DevMode {
		w.Header().Set("Cache-Control", "no-cache")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}
	w.Write([]byte(nexgoRuntime))
}

func (s *Server) printBanner(addr string) {
	mode := "\033[32mproduction\033[0m"
	if s.cfg.DevMode {
		mode = "\033[33mdevelopment\033[0m (hot reload on)"
	}
	fmt.Println()
	fmt.Println("  \033[36m⚡ NexGo — Go-powered web framework\033[0m")
	fmt.Println()
	fmt.Printf("  \033[1mLocal:\033[0m   \033[4mhttp://%s\033[0m\n", addr)
	fmt.Printf("  \033[1mMode:\033[0m    %s\n", mode)
	fmt.Printf("  \033[1mPages:\033[0m   ./%s/\n", s.cfg.PagesDir)
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
        root.style.cssText='opacity:0;transform:translateY(6px);transition:opacity 0.15s,transform 0.15s';
        await new Promise(r=>setTimeout(r,150));
        root.innerHTML=newRoot.innerHTML;
        document.title=doc.title;
        root.style.cssText='opacity:0;transform:translateY(6px)';
        requestAnimationFrame(()=>{root.style.cssText='opacity:1;transform:translateY(0);transition:opacity 0.2s,transform 0.2s';});
        this._initLinks();
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

  _initHMR(){
    if(!document.documentElement.dataset.nexgoDev)return;
    console.log('%c[NexGo HMR] Connected','color:#00d2ff;font-weight:bold');
    const es=new EventSource('/_nexgo/hmr');
    es.onmessage=e=>{
      const m=JSON.parse(e.data);
      if(m.type==='reload'){console.log('%c[NexGo] Reloading...','color:#00d2ff');location.reload();}
      else if(m.type==='error'){console.error('[NexGo] Error:',m.message);}
    };
    es.onerror=()=>console.warn('[NexGo] HMR reconnecting...');
  },

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

  init(){
    this.router._initLinks();
    this._initHMR();
    this._initPrefetch();
    this._initLazy();
    window.addEventListener('popstate',()=>this.router._load(location.pathname));
    document.dispatchEvent(new CustomEvent('nexgo:ready',{}));
  }
};

document.addEventListener('DOMContentLoaded',()=>NexGo.init());
window.NexGo=NexGo;
})();
`
