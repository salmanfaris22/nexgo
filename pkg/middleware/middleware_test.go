package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	called := false
	Logger(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})(w, r)
	if !called {
		t.Error("expected handler to be called")
	}
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCORS(t *testing.T) {
	t.Run("with origins", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		CORS("https://example.com")(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		})(w, r)
		if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Error("expected CORS origin header")
		}
	})

	t.Run("no origins", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		CORS()(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		})(w, r)
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("expected no CORS header when no origins given")
		}
	})

	t.Run("OPTIONS preflight", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("OPTIONS", "/", nil)
		CORS("https://example.com")(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called for OPTIONS")
		})(w, r)
		if w.Code != http.StatusNoContent {
			t.Errorf("expected 204, got %d", w.Code)
		}
	})
}

func TestGzip(t *testing.T) {
	t.Run("with gzip accepted", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Accept-Encoding", "gzip")
		Gzip(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello"))
		})(w, r)
		if w.Header().Get("Content-Encoding") != "gzip" {
			t.Error("expected gzip content encoding")
		}
	})

	t.Run("without gzip accepted", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		Gzip(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello"))
		})(w, r)
		if w.Header().Get("Content-Encoding") == "gzip" {
			t.Error("expected no gzip encoding")
		}
	})
}

func TestSecurityHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	SecurityHeaders(func(w http.ResponseWriter, r *http.Request) {})(w, r)

	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "SAMEORIGIN",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for k, v := range headers {
		if w.Header().Get(k) != v {
			t.Errorf("expected %s=%s, got %s", k, v, w.Header().Get(k))
		}
	}
}

func TestCache(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	Cache(3600)(func(w http.ResponseWriter, r *http.Request) {})(w, r)
	if w.Header().Get("Cache-Control") != "public, max-age=3600" {
		t.Errorf("expected cache header, got %s", w.Header().Get("Cache-Control"))
	}
}

func TestRecover(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	Recover(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestChain(t *testing.T) {
	var order []string
	mw1 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw1-before")
			next(w, r)
			order = append(order, "mw1-after")
		}
	}
	mw2 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw2-before")
			next(w, r)
			order = append(order, "mw2-after")
		}
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	Chain(mw1, mw2)(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})(w, r)

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(order))
	}
	for i := range expected {
		if order[i] != expected[i] {
			t.Errorf("step %d: expected %s, got %s", i, expected[i], order[i])
		}
	}
}

func TestCSP(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	CSP("default-src 'self'")(func(w http.ResponseWriter, r *http.Request) {})(w, r)
	if w.Header().Get("Content-Security-Policy") != "default-src 'self'" {
		t.Errorf("wrong CSP: %s", w.Header().Get("Content-Security-Policy"))
	}
}

func TestCSPWithNonce(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	CSPWithNonce("script-src 'nonce-{nonce}'")(func(w http.ResponseWriter, r *http.Request) {})(w, r)
	csp := w.Header().Get("Content-Security-Policy")
	nonce := w.Header().Get("X-CSP-Nonce")
	if nonce == "" {
		t.Error("expected nonce")
	}
	if !strings.Contains(csp, nonce) {
		t.Errorf("expected CSP to contain nonce")
	}
}

func TestRouteMiddleware(t *testing.T) {
	t.Run("matching prefix", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/users", nil)
		called := false
		RouteMiddleware("/api/*", func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				called = true
				next(w, r)
			}
		})(func(w http.ResponseWriter, r *http.Request) {})(w, r)
		if !called {
			t.Error("expected middleware to be called")
		}
	})

	t.Run("non-matching", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/other", nil)
		called := false
		RouteMiddleware("/api/*", func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				called = true
				next(w, r)
			}
		})(func(w http.ResponseWriter, r *http.Request) {})(w, r)
		if called {
			t.Error("expected middleware NOT to be called")
		}
	})
}

func TestRouteGroup(t *testing.T) {
	t.Run("matching prefix", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/admin/dashboard", nil)
		called := false
		RouteGroup("/admin", func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				called = true
				next(w, r)
			}
		})(func(w http.ResponseWriter, r *http.Request) {})(w, r)
		if !called {
			t.Error("expected middleware to be called")
		}
	})

	t.Run("non-matching", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/public", nil)
		called := false
		RouteGroup("/admin", func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				called = true
				next(w, r)
			}
		})(func(w http.ResponseWriter, r *http.Request) {})(w, r)
		if called {
			t.Error("expected middleware NOT to be called")
		}
	})
}

func TestTimeout(t *testing.T) {
	t.Run("fast handler", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		Timeout(1*time.Second)(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		})(w, r)
		if w.Code != 200 {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("slow handler", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		Timeout(10*time.Millisecond)(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(200)
		})(w, r)
		if w.Code != http.StatusGatewayTimeout {
			t.Errorf("expected 504, got %d", w.Code)
		}
	})
}

func TestRequestID(t *testing.T) {
	t.Run("auto-generate", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		RequestID(func(w http.ResponseWriter, r *http.Request) {})(w, r)
		id := w.Header().Get("X-Request-ID")
		if id == "" {
			t.Error("expected request ID")
		}
	})

	t.Run("use existing", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("X-Request-ID", "my-id-123")
		RequestID(func(w http.ResponseWriter, r *http.Request) {})(w, r)
		if w.Header().Get("X-Request-ID") != "my-id-123" {
			t.Error("expected existing request ID to be preserved")
		}
	})
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		path    string
		pattern string
		want    bool
	}{
		{"/api/users", "/api/*", true},
		{"/api", "/api/*", true},
		{"/other", "/api/*", false},
		{"/exact", "/exact", true},
		{"/exact", "/other", false},
	}
	for _, tt := range tests {
		if got := matchPattern(tt.path, tt.pattern); got != tt.want {
			t.Errorf("matchPattern(%s, %s) = %v, want %v", tt.path, tt.pattern, got, tt.want)
		}
	}
}
