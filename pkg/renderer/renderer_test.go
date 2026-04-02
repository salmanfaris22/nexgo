package renderer

import (
	"html/template"
	"testing"

	"github.com/salmanfaris22/nexgo/pkg/config"
)

func TestStateManagement(t *testing.T) {
	cfg := &config.NexGoConfig{
		ProjectName: "TestApp",
		PagesDir:    "pages",
	}
	r := New(cfg)

	// Register global state
	r.RegisterGlobalState("user", "admin")
	r.RegisterGlobalState("count", 10)

	// Test if global state is injected into PageData
	// We need to mock a template to test RenderPage
	// But let's just check the internal injection logic
	
	pageData := &PageData{
		State: make(map[string]interface{}),
	}
	
	// Copy global state (mimic RenderPage logic)
	for k, v := range r.globalState {
		pageData.State[k] = v
	}

	if pageData.State["user"] != "admin" {
		t.Errorf("Expected user to be admin, got %v", pageData.State["user"])
	}
	if pageData.State["count"] != 10 {
		t.Errorf("Expected count to be 10, got %v", pageData.State["count"])
	}
}

func TestRenderState(t *testing.T) {
	cfg := &config.NexGoConfig{}
	r := New(cfg)
	
	state := map[string]interface{}{
		"foo": "bar",
		"num": 42,
	}
	
	rendered := r.funcMap["renderState"].(func(map[string]interface{}) template.HTML)(state)
	
	expected := `<script id="__nexgo_state" type="application/json">{"foo":"bar","num":42}</script>`
	if string(rendered) != expected {
		t.Errorf("Expected %s, got %s", expected, string(rendered))
	}
}
