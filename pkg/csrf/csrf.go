// Package csrf provides Cross-Site Request Forgery protection.
package csrf

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"
)

// Config holds CSRF configuration.
type Config struct {
	Secret       string
	CookieName   string
	HeaderName   string
	FieldName    string
	MaxAge       time.Duration
	Secure       bool
	SameSite     http.SameSite
	ErrorHandler http.HandlerFunc
}

// DefaultConfig returns sensible defaults for CSRF protection.
func DefaultConfig() Config {
	return Config{
		Secret:     "change-me-csrf-secret",
		CookieName: "_nexgo_csrf",
		HeaderName: "X-CSRF-Token",
		FieldName:  "_csrf",
		MaxAge:     12 * time.Hour,
		SameSite:   http.SameSiteStrictMode,
	}
}

// CSRF provides CSRF token generation and validation.
type CSRF struct {
	config Config
}

// New creates a CSRF protector.
func New(cfg Config) *CSRF {
	return &CSRF{config: cfg}
}

// GenerateToken creates a new CSRF token.
func (c *CSRF) GenerateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	nonce := base64.RawURLEncoding.EncodeToString(b)
	ts := fmt.Sprintf("%d", time.Now().Unix())
	data := nonce + "|" + ts
	sig := c.sign(data)
	return data + "|" + sig
}

// ValidateToken checks if a CSRF token is valid.
func (c *CSRF) ValidateToken(token string) bool {
	parts := strings.SplitN(token, "|", 3)
	if len(parts) != 3 {
		return false
	}
	data := parts[0] + "|" + parts[1]
	expectedSig := c.sign(data)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return false
	}
	var ts int64
	fmt.Sscanf(parts[1], "%d", &ts)
	created := time.Unix(ts, 0)
	return time.Since(created) < c.config.MaxAge
}

// Middleware returns HTTP middleware that enforces CSRF protection.
func (c *CSRF) Middleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := c.GenerateToken()
			http.SetCookie(w, &http.Cookie{
				Name:     c.config.CookieName,
				Value:    token,
				Path:     "/",
				MaxAge:   int(c.config.MaxAge.Seconds()),
				HttpOnly: false,
				Secure:   c.config.Secure,
				SameSite: c.config.SameSite,
			})

			r = r.WithContext(context.WithValue(r.Context(), csrfKey, token))

			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next(w, r)
				return
			}

			submitted := r.Header.Get(c.config.HeaderName)
			if submitted == "" {
				submitted = r.FormValue(c.config.FieldName)
			}

			if !c.ValidateToken(submitted) {
				if c.config.ErrorHandler != nil {
					c.config.ErrorHandler(w, r)
					return
				}
				http.Error(w, `{"error":"invalid CSRF token","status":403}`, http.StatusForbidden)
				return
			}

			next(w, r)
		}
	}
}

// TemplateField returns an HTML hidden input field with the CSRF token.
func (c *CSRF) TemplateField(token string) template.HTML {
	return template.HTML(fmt.Sprintf(
		`<input type="hidden" name="%s" value="%s">`,
		template.HTMLEscapeString(c.config.FieldName),
		template.HTMLEscapeString(token),
	))
}

// TemplateFuncMap returns template functions for CSRF.
func TemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"csrfToken": func(r *http.Request) string {
			return GetToken(r)
		},
		"csrfField": func(token string) template.HTML {
			return template.HTML(fmt.Sprintf(
				`<input type="hidden" name="_csrf" value="%s">`,
				template.HTMLEscapeString(token),
			))
		},
	}
}

func (c *CSRF) sign(data string) string {
	mac := hmac.New(sha256.New, []byte(c.config.Secret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

type ctxKey string

const csrfKey ctxKey = "nexgo_csrf_token"

// GetToken retrieves the CSRF token from request context.
func GetToken(r *http.Request) string {
	if t, ok := r.Context().Value(csrfKey).(string); ok {
		return t
	}
	return ""
}
