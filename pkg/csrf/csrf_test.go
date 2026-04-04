package csrf

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateAndValidateToken(t *testing.T) {
	c := New(DefaultConfig())

	token := c.GenerateToken()
	if token == "" {
		t.Fatal("token should not be empty")
	}

	if !c.ValidateToken(token) {
		t.Error("valid token should pass validation")
	}
}

func TestInvalidToken(t *testing.T) {
	c := New(DefaultConfig())

	if c.ValidateToken("") {
		t.Error("empty token should fail")
	}
	if c.ValidateToken("not-valid") {
		t.Error("garbage token should fail")
	}
	if c.ValidateToken("a|b") {
		t.Error("two-part token should fail")
	}
}

func TestDifferentSecrets(t *testing.T) {
	c1 := New(Config{Secret: "secret-1", MaxAge: DefaultConfig().MaxAge})
	c2 := New(Config{Secret: "secret-2", MaxAge: DefaultConfig().MaxAge})

	token := c1.GenerateToken()
	if c2.ValidateToken(token) {
		t.Error("token from different secret should fail")
	}
}

func TestMiddlewareAllowsGET(t *testing.T) {
	c := New(DefaultConfig())
	handler := c.Middleware()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != 200 {
		t.Errorf("GET should pass, got %d", w.Code)
	}
}

func TestMiddlewareBlocksPOSTWithoutToken(t *testing.T) {
	c := New(DefaultConfig())
	handler := c.Middleware()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	req := httptest.NewRequest("POST", "/", strings.NewReader("data=test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("POST without token: got %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestMiddlewareAllowsPOSTWithHeader(t *testing.T) {
	c := New(DefaultConfig())

	token := c.GenerateToken()

	handler := c.Middleware()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("X-CSRF-Token", token)
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != 200 {
		t.Errorf("POST with valid header token: got %d, want 200", w.Code)
	}
}

func TestTemplateField(t *testing.T) {
	c := New(DefaultConfig())
	field := c.TemplateField("test-token")
	html := string(field)
	if !strings.Contains(html, `name="_csrf"`) {
		t.Error("field should contain name attribute")
	}
	if !strings.Contains(html, `value="test-token"`) {
		t.Error("field should contain token value")
	}
}
