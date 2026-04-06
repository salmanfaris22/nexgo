package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewContext(t *testing.T) {
	ctx := NewContext()
	if ctx == nil {
		t.Fatal("expected context")
	}
}

func TestOn(t *testing.T) {
	ctx := NewContext()
	called := false

	ctx.On(HookBeforeStart, "test", func() {
		called = true
	})

	err := ctx.Emit(HookBeforeStart)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected hook to be called")
	}
}

func TestOnPriority(t *testing.T) {
	ctx := NewContext()
	var order []string

	ctx.OnPriority(HookBeforeStart, "low", 10, func() {
		order = append(order, "low")
	})
	ctx.OnPriority(HookBeforeStart, "high", 1, func() {
		order = append(order, "high")
	})

	ctx.Emit(HookBeforeStart)

	if len(order) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(order))
	}
	if order[0] != "high" {
		t.Errorf("expected high first, got %v", order)
	}
}

func TestEmit_WithError(t *testing.T) {
	ctx := NewContext()

	ctx.On(HookBeforeStart, "fail", func() error {
		return nil
	})

	err := ctx.Emit(HookBeforeStart)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	ctx.On(HookBeforeBuild, "fail", func() error {
		return nil
	})

	err = ctx.Emit(HookBeforeBuild)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMiddleware(t *testing.T) {
	ctx := NewContext()

	ctx.AddMiddleware(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Plugin", "true")
			next(w, r)
		}
	})

	mw := ctx.GetMiddleware()
	if len(mw) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(mw))
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	mw[0](func(w http.ResponseWriter, r *http.Request) {})(w, r)

	if w.Header().Get("X-Plugin") != "true" {
		t.Error("expected X-Plugin header")
	}
}

func TestRoutes(t *testing.T) {
	ctx := NewContext()

	ctx.AddRoute("/api/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	routes := ctx.GetRoutes()
	if len(routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Pattern != "/api/test" {
		t.Errorf("expected /api/test, got %s", routes[0].Pattern)
	}
}

func TestTemplateFuncs(t *testing.T) {
	ctx := NewContext()

	ctx.AddTemplateFunc("upper", func(s string) string { return s })

	fns := ctx.GetTemplateFuncs()
	if _, ok := fns["upper"]; !ok {
		t.Error("expected upper template function")
	}
}

func TestConfig(t *testing.T) {
	ctx := NewContext()

	ctx.SetConfig("key", "value")
	if ctx.GetConfig("key") != "value" {
		t.Error("expected value")
	}
	if ctx.GetConfig("missing") != nil {
		t.Error("expected nil for missing key")
	}
}

func TestManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("expected manager")
	}
	if m.Context() == nil {
		t.Error("expected context")
	}
}

type testPlugin struct {
	name    string
	version string
}

func (p *testPlugin) Name() string    { return p.name }
func (p *testPlugin) Version() string { return p.version }
func (p *testPlugin) Init(ctx *Context) error {
	ctx.SetConfig("plugin_name", p.name)
	return nil
}

func TestManager_Register(t *testing.T) {
	m := NewManager()

	err := m.Register(&testPlugin{name: "test", version: "1.0"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list := m.List()
	if len(list) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(list))
	}
	if list[0].Name != "test" {
		t.Errorf("expected name test, got %s", list[0].Name)
	}
	if list[0].Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", list[0].Version)
	}

	// Verify config was set during Init
	if m.Context().GetConfig("plugin_name") != "test" {
		t.Error("expected plugin_name config")
	}
}

func TestManager_EmitHook(t *testing.T) {
	m := NewManager()

	m.Context().On(HookAfterStart, "test", func() {})

	err := m.EmitHook(HookAfterStart)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHookConstants(t *testing.T) {
	hooks := []Hook{
		HookBeforeStart,
		HookAfterStart,
		HookBeforeStop,
		HookAfterStop,
		HookBeforeRequest,
		HookAfterRequest,
		HookBeforeRender,
		HookAfterRender,
		HookBeforeBuild,
		HookAfterBuild,
		HookOnRouteMatch,
		HookOnError,
	}

	for _, h := range hooks {
		if h == "" {
			t.Errorf("hook %v is empty", h)
		}
	}
}
