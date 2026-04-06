// Package islands implements the Islands Architecture for NexGo.
//
// Pages are static HTML by default — zero JavaScript shipped.
// Only interactive "islands" send JS to the browser, with configurable
// hydration strategies: client:load, client:visible, client:idle, client:media, client:none.
//
// Usage in templates:
//
//	{{ island "counter" }}
//	{{ island "counter" (props "count" 5) }}
//	{{ island "counter" (props "count" 5) "client:visible" }}
//	{{ island "chart" (props "data" .Props.chart) "client:idle" }}
//	{{ island "mobile-menu" nil "client:media=(max-width:768px)" }}
package islands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Strategy controls when an island's JS is loaded and executed on the client.
const (
	StrategyLoad    = "client:load"    // Hydrate immediately on page load
	StrategyVisible = "client:visible" // Hydrate when scrolled into viewport
	StrategyIdle    = "client:idle"    // Hydrate when browser is idle
	StrategyMedia   = "client:media"   // Hydrate when media query matches
	StrategyNone    = "client:none"    // SSR only — never hydrate (no JS shipped)
)

// Island represents a registered island component.
type Island struct {
	Name     string
	Template *template.Template
	HasJS    bool   // Whether a .js file exists for client hydration
	JSPath   string // Absolute path to the .js file
}

// Registry manages all discovered island components.
type Registry struct {
	mu      sync.RWMutex
	islands map[string]*Island
	dir     string
	funcMap template.FuncMap
}

// NewRegistry creates an island registry that scans the given directory.
func NewRegistry(islandsDir string, funcMap template.FuncMap) *Registry {
	return &Registry{
		islands: make(map[string]*Island),
		dir:     islandsDir,
		funcMap: funcMap,
	}
}

// Scan discovers island templates and JS files from the islands directory.
func (r *Registry) Scan() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.islands = make(map[string]*Island)

	if _, err := os.Stat(r.dir); os.IsNotExist(err) {
		return nil // No islands directory — that's fine
	}

	return filepath.WalkDir(r.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		ext := filepath.Ext(path)
		rel, _ := filepath.Rel(r.dir, path)
		name := strings.TrimSuffix(filepath.ToSlash(rel), ext)

		switch ext {
		case ".html", ".gohtml", ".tmpl":
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading island %s: %w", name, err)
			}
			tmpl, err := template.New(name).Funcs(r.funcMap).Parse(string(data))
			if err != nil {
				return fmt.Errorf("parsing island %s: %w", name, err)
			}

			island := r.getOrCreate(name)
			island.Template = tmpl

		case ".js":
			island := r.getOrCreate(name)
			island.HasJS = true
			island.JSPath = path
		}

		return nil
	})
}

func (r *Registry) getOrCreate(name string) *Island {
	if isl, ok := r.islands[name]; ok {
		return isl
	}
	isl := &Island{Name: name}
	r.islands[name] = isl
	return isl
}

// Get returns an island by name.
func (r *Registry) Get(name string) (*Island, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	isl, ok := r.islands[name]
	return isl, ok
}

// GetJS returns the JS source for an island, or empty if none exists.
func (r *Registry) GetJS(name string) ([]byte, bool) {
	r.mu.RLock()
	isl, ok := r.islands[name]
	r.mu.RUnlock()
	if !ok || !isl.HasJS {
		return nil, false
	}
	data, err := os.ReadFile(isl.JSPath)
	if err != nil {
		return nil, false
	}
	return data, true
}

// Names returns all registered island names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.islands))
	for name := range r.islands {
		names = append(names, name)
	}
	return names
}

// Render executes an island's server-side template and wraps it in a
// <nexgo-island> custom element for client-side hydration.
func (r *Registry) Render(name string, props map[string]interface{}, strategy string) template.HTML {
	r.mu.RLock()
	isl, ok := r.islands[name]
	r.mu.RUnlock()

	if !ok || isl.Template == nil {
		return template.HTML(fmt.Sprintf("<!-- island %q not found -->", name))
	}

	if strategy == "" {
		strategy = StrategyLoad
	}

	// Execute the island template server-side
	var buf bytes.Buffer
	if err := isl.Template.Execute(&buf, props); err != nil {
		return template.HTML(fmt.Sprintf("<!-- island %q render error: %v -->", name, err))
	}

	// If strategy is client:none, just return raw HTML — no wrapper, no JS
	if strategy == StrategyNone {
		return template.HTML(buf.String())
	}

	// Marshal props for the client
	propsJSON := "{}"
	if props != nil {
		if b, err := json.Marshal(props); err == nil {
			propsJSON = string(b)
		}
	}

	// Determine if this island has client JS
	hasJS := "false"
	if isl.HasJS {
		hasJS = "true"
	}

	// Wrap in custom element
	return template.HTML(fmt.Sprintf(
		`<nexgo-island data-name="%s" data-strategy="%s" data-has-js="%s" data-props='%s'>%s</nexgo-island>`,
		template.HTMLEscapeString(name),
		template.HTMLEscapeString(strategy),
		hasJS,
		template.HTMLEscapeString(propsJSON),
		buf.String(),
	))
}

// RuntimeJS returns the client-side island hydration runtime.
// This is a small (~1KB) script that scans for <nexgo-island> elements
// and hydrates them according to their strategy.
func RuntimeJS() string {
	return islandRuntimeJS
}

const islandRuntimeJS = `(function(){
'use strict';

if(window.__nexgo_islands)return;
window.__nexgo_islands=true;

function hydrateIsland(el){
  var name=el.getAttribute('data-name');
  var hasJS=el.getAttribute('data-has-js')==='true';
  if(!name||!hasJS)return;

  var props={};
  try{props=JSON.parse(el.getAttribute('data-props')||'{}');}catch(e){}

  var script=document.createElement('script');
  script.type='module';
  script.textContent='import i from "/_nexgo/islands/'+name+'.js";if(typeof i==="function")i(document.querySelector(\'nexgo-island[data-name="'+name+'"]\'),'+JSON.stringify(props)+');';
  document.head.appendChild(script);
  el.setAttribute('data-hydrated','true');
}

var strategies={
  'client:load':function(el,fn){fn();},
  'client:visible':function(el,fn){
    if(!('IntersectionObserver' in window)){fn();return;}
    var obs=new IntersectionObserver(function(entries){
      if(entries[0].isIntersecting){obs.disconnect();fn();}
    },{rootMargin:'200px'});
    obs.observe(el);
  },
  'client:idle':function(el,fn){
    if('requestIdleCallback' in window)requestIdleCallback(fn);
    else setTimeout(fn,200);
  },
  'client:media':function(el,fn,query){
    if(!query){fn();return;}
    var mq=matchMedia(query);
    if(mq.matches){fn();return;}
    mq.addEventListener('change',function handler(e){if(e.matches){mq.removeEventListener('change',handler);fn();}});
  }
};

function initIslands(){
  var islands=document.querySelectorAll('nexgo-island[data-name]:not([data-hydrated])');
  islands.forEach(function(el){
    var raw=el.getAttribute('data-strategy')||'client:load';
    var eqIdx=raw.indexOf('=');
    var strat=eqIdx>-1?raw.substring(0,eqIdx):raw;
    var param=eqIdx>-1?raw.substring(eqIdx+1):'';
    var handler=strategies[strat]||strategies['client:load'];
    handler(el,function(){hydrateIsland(el);},param);
  });
}

if(document.readyState==='loading'){
  document.addEventListener('DOMContentLoaded',initIslands);
}else{
  initIslands();
}

// Re-init after SPA navigations
document.addEventListener('nexgo:navigate',function(){
  setTimeout(initIslands,50);
});
})();
`
