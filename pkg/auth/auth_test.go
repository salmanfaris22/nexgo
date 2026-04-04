package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGenerateAndVerifyToken(t *testing.T) {
	a := New(DefaultConfig())

	token, err := a.GenerateToken(Claims{
		Sub:   "user-123",
		Email: "test@example.com",
		Role:  "admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}

	claims, err := a.VerifyToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Sub != "user-123" {
		t.Errorf("Sub = %q, want %q", claims.Sub, "user-123")
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "test@example.com")
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %q, want %q", claims.Role, "admin")
	}
}

func TestExpiredToken(t *testing.T) {
	a := New(DefaultConfig())

	token, _ := a.GenerateToken(Claims{
		Sub: "user-123",
		Exp: time.Now().Add(-1 * time.Hour).Unix(),
	})

	_, err := a.VerifyToken(token)
	if err == nil {
		t.Error("expired token should fail verification")
	}
}

func TestInvalidSignature(t *testing.T) {
	a1 := New(Config{Secret: "secret-1", TokenExpiry: time.Hour})
	a2 := New(Config{Secret: "secret-2", TokenExpiry: time.Hour})

	token, _ := a1.GenerateToken(Claims{Sub: "user-123"})
	_, err := a2.VerifyToken(token)
	if err == nil {
		t.Error("token signed with different secret should fail")
	}
}

func TestInvalidTokenFormat(t *testing.T) {
	a := New(DefaultConfig())

	_, err := a.VerifyToken("not-a-jwt")
	if err == nil {
		t.Error("invalid format should fail")
	}

	_, err = a.VerifyToken("")
	if err == nil {
		t.Error("empty token should fail")
	}
}

func TestMiddlewareBlocks(t *testing.T) {
	a := New(DefaultConfig())
	handler := a.Middleware()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	// No token
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("no token: got %d, want %d", w.Code, http.StatusUnauthorized)
	}

	// Valid token
	token, _ := a.GenerateToken(Claims{Sub: "user-123"})
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("valid token: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestClaimsFromContext(t *testing.T) {
	a := New(DefaultConfig())
	var gotClaims *Claims

	handler := a.Middleware()(func(w http.ResponseWriter, r *http.Request) {
		gotClaims = GetClaims(r)
		w.WriteHeader(200)
	})

	token, _ := a.GenerateToken(Claims{Sub: "user-123", Role: "admin"})
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler(w, req)

	if gotClaims == nil {
		t.Fatal("claims should be in context")
	}
	if gotClaims.Sub != "user-123" {
		t.Errorf("Sub = %q, want %q", gotClaims.Sub, "user-123")
	}
}

func TestRequireRole(t *testing.T) {
	a := New(DefaultConfig())
	handler := a.RequireRole("admin")(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	// Admin token
	token, _ := a.GenerateToken(Claims{Sub: "user-1", Role: "admin"})
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != 200 {
		t.Errorf("admin should pass, got %d", w.Code)
	}

	// User token (not admin)
	token, _ = a.GenerateToken(Claims{Sub: "user-2", Role: "user"})
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("non-admin should be forbidden, got %d", w.Code)
	}
}

func TestPasswordHashing(t *testing.T) {
	salt := "test-salt"
	hash := HashPassword("mypassword", salt)
	if !CheckPassword("mypassword", salt, hash) {
		t.Error("password should match")
	}
	if CheckPassword("wrongpassword", salt, hash) {
		t.Error("wrong password should not match")
	}
}

func TestExtractTokenFromCookie(t *testing.T) {
	a := New(DefaultConfig())
	token, _ := a.GenerateToken(Claims{Sub: "user-123"})

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: a.config.CookieName, Value: token})

	extracted := a.ExtractToken(req)
	if extracted != token {
		t.Errorf("ExtractToken from cookie = %q, want %q", extracted, token)
	}
}
