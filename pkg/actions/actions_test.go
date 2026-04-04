package actions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestFormData(t *testing.T) {
	fd := FormData{Values: map[string][]string{
		"name":  {"John"},
		"tags":  {"go", "web"},
		"empty": {},
	}}

	if fd.Get("name") != "John" {
		t.Errorf("Get(name) = %q", fd.Get("name"))
	}
	if fd.Get("missing") != "" {
		t.Errorf("Get(missing) = %q", fd.Get("missing"))
	}
	if len(fd.GetAll("tags")) != 2 {
		t.Errorf("GetAll(tags) = %d", len(fd.GetAll("tags")))
	}
	if !fd.Has("name") {
		t.Error("Has(name) should be true")
	}
	if fd.Has("nope") {
		t.Error("Has(nope) should be false")
	}
}

func TestResultTypes(t *testing.T) {
	ok := OK(map[string]interface{}{"id": "1"})
	if !ok.Success {
		t.Error("OK should be successful")
	}

	fail := Fail(map[string]string{"name": "required"})
	if fail.Success {
		t.Error("Fail should not be successful")
	}

	redir := RedirectTo("/home")
	if redir.Redirect != "/home" {
		t.Errorf("Redirect = %q", redir.Redirect)
	}
}

func TestActionHandler(t *testing.T) {
	r := NewRegistry()
	r.Register("greet", func(ctx context.Context, form FormData) (*Result, error) {
		name := form.Get("name")
		return OK(map[string]interface{}{"greeting": "Hello, " + name}), nil
	})

	body := url.Values{"name": {"World"}}.Encode()
	req := httptest.NewRequest("POST", "/_nexgo/action/greet", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.Handler()(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var result Result
	json.NewDecoder(w.Body).Decode(&result)
	if !result.Success {
		t.Error("should be successful")
	}
	if result.Data["greeting"] != "Hello, World" {
		t.Errorf("greeting = %v", result.Data["greeting"])
	}
}

func TestActionHandlerMethodNotAllowed(t *testing.T) {
	r := NewRegistry()
	req := httptest.NewRequest("GET", "/_nexgo/action/test", nil)
	w := httptest.NewRecorder()
	r.Handler()(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestActionNotFound(t *testing.T) {
	r := NewRegistry()
	req := httptest.NewRequest("POST", "/_nexgo/action/unknown", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.Handler()(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestValidation(t *testing.T) {
	form := FormData{Values: map[string][]string{
		"name":  {"Jo"},
		"email": {"not-an-email"},
	}}

	v := NewValidator()
	v.Required(form, "name", "Name required")
	v.MinLength(form, "name", 3, "Name too short")
	v.Email(form, "email", "Invalid email")
	v.Required(form, "missing", "Missing required")

	if v.IsValid() {
		t.Error("should have errors")
	}

	errs := v.Errors()
	if errs["name"] != "Name too short" {
		t.Errorf("name error = %q", errs["name"])
	}
	if errs["missing"] != "Missing required" {
		t.Errorf("missing error = %q", errs["missing"])
	}
}

func TestValidationValid(t *testing.T) {
	form := FormData{Values: map[string][]string{
		"name":  {"John Doe"},
		"email": {"john@example.com"},
	}}

	v := NewValidator()
	v.Required(form, "name", "required")
	v.MinLength(form, "name", 3, "too short")
	v.Email(form, "email", "invalid email")

	if !v.IsValid() {
		t.Errorf("should be valid, errors: %v", v.Errors())
	}
}

func TestRedirectAction(t *testing.T) {
	r := NewRegistry()
	r.Register("redirect_test", func(ctx context.Context, form FormData) (*Result, error) {
		return RedirectTo("/success"), nil
	})

	body := url.Values{}.Encode()
	req := httptest.NewRequest("POST", "/_nexgo/action/redirect_test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.Handler()(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if w.Header().Get("Location") != "/success" {
		t.Errorf("Location = %q", w.Header().Get("Location"))
	}
}
