package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSON(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, map[string]string{"msg": "hello"})
	if w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Errorf("wrong content type: %s", w.Header().Get("Content-Type"))
	}
	if !strings.Contains(w.Body.String(), "hello") {
		t.Errorf("expected 'hello' in body, got: %s", w.Body.String())
	}
}

func TestJSONStatus(t *testing.T) {
	w := httptest.NewRecorder()
	JSONStatus(w, http.StatusCreated, map[string]string{"id": "1"})
	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	Error(w, http.StatusBadRequest, "bad input")
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "bad input") {
		t.Errorf("expected error message in body")
	}
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	BadRequest(w, "invalid")
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	NotFound(w, "missing")
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	Unauthorized(w)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	Forbidden(w)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	InternalError(w, errors.New("db error"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestDecode(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"test"}`))
		var data struct {
			Name string `json:"name"`
		}
		ok := Decode(w, r, &data)
		if !ok {
			t.Error("expected Decode to return true")
		}
		if data.Name != "test" {
			t.Errorf("expected 'test', got '%s'", data.Name)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", strings.NewReader(`{invalid}`))
		var data struct{}
		ok := Decode(w, r, &data)
		if ok {
			t.Error("expected Decode to return false")
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unknown fields", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"test","extra":1}`))
		var data struct {
			Name string `json:"name"`
		}
		ok := Decode(w, r, &data)
		if ok {
			t.Error("expected Decode to return false for unknown fields")
		}
	})
}

func TestMethodGuard(t *testing.T) {
	t.Run("allowed method", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		ok := MethodGuard(w, r, "GET", "POST")
		if !ok {
			t.Error("expected true for allowed method")
		}
	})

	t.Run("disallowed method", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("DELETE", "/", nil)
		ok := MethodGuard(w, r, "GET", "POST")
		if ok {
			t.Error("expected false for disallowed method")
		}
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Code)
		}
		if w.Header().Get("Allow") != "GET, POST" {
			t.Errorf("wrong Allow header: %s", w.Header().Get("Allow"))
		}
	})
}

func TestRoute(t *testing.T) {
	t.Run("matched method", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)
		called := false
		Route(w, r, Methods{
			"POST": func(w http.ResponseWriter, r *http.Request) {
				called = true
			},
		})
		if !called {
			t.Error("expected handler to be called")
		}
	})

	t.Run("unmatched method", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("DELETE", "/", nil)
		Route(w, r, Methods{
			"GET":  func(w http.ResponseWriter, r *http.Request) {},
			"POST": func(w http.ResponseWriter, r *http.Request) {},
		})
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Code)
		}
	})
}

func TestPaginate(t *testing.T) {
	tests := []struct {
		url       string
		wantPage  int
		wantLimit int
	}{
		{"/", 1, 20},
		{"/?page=2&limit=10", 2, 10},
		{"/?page=-1&limit=0", 1, 20},
		{"/?page=1&limit=200", 1, 20},
		{"/?page=abc&limit=xyz", 1, 20},
	}

	for _, tt := range tests {
		r := httptest.NewRequest("GET", tt.url, nil)
		page, limit := Paginate(r)
		if page != tt.wantPage || limit != tt.wantLimit {
			t.Errorf("Paginate(%s) = %d, %d; want %d, %d", tt.url, page, limit, tt.wantPage, tt.wantLimit)
		}
	}
}

func TestHTMXHelpers(t *testing.T) {
	t.Run("IsHTMX true", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("HX-Request", "true")
		if !IsHTMX(r) {
			t.Error("expected IsHTMX to return true")
		}
	})

	t.Run("IsHTMX false", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		if IsHTMX(r) {
			t.Error("expected IsHTMX to return false")
		}
	})

	t.Run("HTMXHeader", func(t *testing.T) {
		w := httptest.NewRecorder()
		HTMXHeader(w, "HX-Redirect", "/home")
		if w.Header().Get("HX-Redirect") != "/home" {
			t.Error("expected HX-Redirect header")
		}
	})

	t.Run("HTMXHTML", func(t *testing.T) {
		w := httptest.NewRecorder()
		HTMXHTML(w, "<div>content</div>")
		if w.Body.String() != "<div>content</div>" {
			t.Errorf("wrong body: %s", w.Body.String())
		}
	})

	t.Run("HTMXTrigger", func(t *testing.T) {
		w := httptest.NewRecorder()
		HTMXTrigger(w, "saved")
		if w.Header().Get("HX-Trigger") != "saved" {
			t.Error("expected HX-Trigger header")
		}
	})
}

func TestState(t *testing.T) {
	s := NewState()
	s.Set("key", "value")
	if s.Get("key") != "value" {
		t.Error("expected 'value'")
	}
	if s.Get("missing") != nil {
		t.Error("expected nil for missing key")
	}
	s.Delete("key")
	if s.Get("key") != nil {
		t.Error("expected nil after delete")
	}
	all := s.All()
	if len(all) != 0 {
		t.Errorf("expected empty map, got %d items", len(all))
	}
}

func TestGlobalState(t *testing.T) {
	SetState("gkey", "gval")
	if GetState("gkey") != "gval" {
		t.Error("expected 'gval'")
	}
	DeleteState("gkey")
	if GetState("gkey") != nil {
		t.Error("expected nil after delete")
	}
}

func TestEscape(t *testing.T) {
	result := Escape("<script>alert('xss')</script>")
	expected := "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestHTML(t *testing.T) {
	w := httptest.NewRecorder()
	HTML(w, "<h1>Hello</h1>")
	if w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Error("expected HTML content type")
	}
	if w.Body.String() != "<h1>Hello</h1>" {
		t.Errorf("wrong body: %s", w.Body.String())
	}
}

func TestCacheHelpers(t *testing.T) {
	CacheClear()
	CacheSet("test", 200, http.Header{}, []byte("ok"))
	code, _, body, ok := CacheGet("test")
	if !ok || code != 200 || string(body) != "ok" {
		t.Error("expected cached response")
	}
	CacheDelete("test")
	_, _, _, ok = CacheGet("test")
	if ok {
		t.Error("expected cache miss after delete")
	}
}
