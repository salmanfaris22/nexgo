// Package plugin provides a hook-based plugin system for extending NexGo.
package plugin

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
)

// Hook represents a point in the request lifecycle where plugins can inject behavior.
type Hook string

const (
	// Server lifecycle hooks
	HookBeforeStart  Hook = "before_start"
	HookAfterStart   Hook = "after_start"
	HookBeforeStop   Hook = "before_stop"
	HookAfterStop    Hook = "after_stop"

	// Request lifecycle hooks
	HookBeforeRequest  Hook = "before_request"
	HookAfterRequest   Hook = "after_request"
	HookBeforeRender   Hook = "before_render"
	HookAfterRender    Hook = "after_render"

	// Build lifecycle hooks
	HookBeforeBuild Hook = "before_build"
	HookAfterBuild  Hook = "after_build"

	// Route hooks
	HookOnRouteMatch Hook = "on_route_match"
	HookOnError      Hook = "on_error"
)

// Plugin defines the interface for NexGo plugins.
type Plugin interface {
	Name() string
	Version() string
	Init(ctx *Context) error
}

// Context provides the plugin API for interacting with the framework.
type Context struct {
	mu          sync.RWMutex
	hooks       map[Hook][]hookEntry
	middleware  []func(http.HandlerFunc) http.HandlerFunc
	routes      []RouteRegistration
	templateFns map[string]interface{}
	config      map[string]interface{}
}

type hookEntry struct {
	name     string
	priority int
	handler  interface{}
}

// RouteRegistration is a route added by a plugin.
type RouteRegistration struct {
	Pattern string
	Handler http.HandlerFunc
}

// NewContext creates a plugin context.
func NewContext() *Context {
	return &Context{
		hooks:       make(map[Hook][]hookEntry),
		templateFns: make(map[string]interface{}),
		config:      make(map[string]interface{}),
	}
}

// On registers a hook handler with default priority.
func (c *Context) On(hook Hook, name string, handler interface{}) {
	c.OnPriority(hook, name, 0, handler)
}

// OnPriority registers a hook handler with custom priority (lower runs first).
func (c *Context) OnPriority(hook Hook, name string, priority int, handler interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hooks[hook] = append(c.hooks[hook], hookEntry{
		name:     name,
		priority: priority,
		handler:  handler,
	})
	sort.Slice(c.hooks[hook], func(i, j int) bool {
		return c.hooks[hook][i].priority < c.hooks[hook][j].priority
	})
}

// Emit triggers all handlers for a hook.
func (c *Context) Emit(hook Hook, args ...interface{}) error {
	c.mu.RLock()
	entries := c.hooks[hook]
	c.mu.RUnlock()

	for _, entry := range entries {
		switch h := entry.handler.(type) {
		case func():
			h()
		case func() error:
			if err := h(); err != nil {
				return fmt.Errorf("plugin %s hook %s: %w", entry.name, hook, err)
			}
		case func(http.ResponseWriter, *http.Request):
			// Handled in middleware context
		}
	}
	return nil
}

// AddMiddleware registers middleware from a plugin.
func (c *Context) AddMiddleware(mw func(http.HandlerFunc) http.HandlerFunc) {
	c.mu.Lock()
	c.middleware = append(c.middleware, mw)
	c.mu.Unlock()
}

// GetMiddleware returns all registered plugin middleware.
func (c *Context) GetMiddleware() []func(http.HandlerFunc) http.HandlerFunc {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]func(http.HandlerFunc) http.HandlerFunc, len(c.middleware))
	copy(result, c.middleware)
	return result
}

// AddRoute registers a route from a plugin.
func (c *Context) AddRoute(pattern string, handler http.HandlerFunc) {
	c.mu.Lock()
	c.routes = append(c.routes, RouteRegistration{Pattern: pattern, Handler: handler})
	c.mu.Unlock()
}

// GetRoutes returns all registered plugin routes.
func (c *Context) GetRoutes() []RouteRegistration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]RouteRegistration, len(c.routes))
	copy(result, c.routes)
	return result
}

// AddTemplateFunc adds a template function from a plugin.
func (c *Context) AddTemplateFunc(name string, fn interface{}) {
	c.mu.Lock()
	c.templateFns[name] = fn
	c.mu.Unlock()
}

// GetTemplateFuncs returns all registered template functions.
func (c *Context) GetTemplateFuncs() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string]interface{}, len(c.templateFns))
	for k, v := range c.templateFns {
		result[k] = v
	}
	return result
}

// SetConfig stores plugin configuration.
func (c *Context) SetConfig(key string, value interface{}) {
	c.mu.Lock()
	c.config[key] = value
	c.mu.Unlock()
}

// GetConfig retrieves plugin configuration.
func (c *Context) GetConfig(key string) interface{} {
	c.mu.RLock()
	v := c.config[key]
	c.mu.RUnlock()
	return v
}

// --- Plugin Manager ---

// Manager manages plugin registration and lifecycle.
type Manager struct {
	mu      sync.RWMutex
	plugins []Plugin
	ctx     *Context
}

// NewManager creates a plugin manager.
func NewManager() *Manager {
	return &Manager{
		ctx: NewContext(),
	}
}

// Register adds a plugin.
func (m *Manager) Register(p Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := p.Init(m.ctx); err != nil {
		return fmt.Errorf("plugin %s init failed: %w", p.Name(), err)
	}
	m.plugins = append(m.plugins, p)
	return nil
}

// Context returns the shared plugin context.
func (m *Manager) Context() *Context {
	return m.ctx
}

// List returns all registered plugins.
func (m *Manager) List() []PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var info []PluginInfo
	for _, p := range m.plugins {
		info = append(info, PluginInfo{
			Name:    p.Name(),
			Version: p.Version(),
		})
	}
	return info
}

// PluginInfo describes a registered plugin.
type PluginInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// EmitHook triggers a hook across all plugins.
func (m *Manager) EmitHook(hook Hook, args ...interface{}) error {
	return m.ctx.Emit(hook, args...)
}
