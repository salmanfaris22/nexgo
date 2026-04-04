// Package auth provides JWT and cookie-based authentication middleware.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Claims represents JWT claims.
type Claims struct {
	Sub       string                 `json:"sub"`
	Email     string                 `json:"email,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Role      string                 `json:"role,omitempty"`
	Exp       int64                  `json:"exp"`
	Iat       int64                  `json:"iat"`
	ExtraData map[string]interface{} `json:"data,omitempty"`
}

// IsExpired checks if the token has expired.
func (c Claims) IsExpired() bool {
	return time.Now().Unix() > c.Exp
}

// HasRole checks if the user has a specific role.
func (c Claims) HasRole(role string) bool {
	return c.Role == role
}

// Config holds authentication configuration.
type Config struct {
	Secret        string
	TokenExpiry   time.Duration
	RefreshExpiry time.Duration
	CookieName    string
	CookiePath    string
	CookieDomain  string
	CookieSecure  bool
	HeaderName    string // "Authorization" by default
}

// DefaultConfig returns sensible auth defaults.
func DefaultConfig() Config {
	return Config{
		Secret:        "change-me-in-production",
		TokenExpiry:   1 * time.Hour,
		RefreshExpiry: 7 * 24 * time.Hour,
		CookieName:    "nexgo_token",
		CookiePath:    "/",
		CookieSecure:  false,
		HeaderName:    "Authorization",
	}
}

// Auth manages authentication.
type Auth struct {
	config Config
}

// New creates an Auth manager.
func New(cfg Config) *Auth {
	return &Auth{config: cfg}
}

// GenerateToken creates a signed JWT token.
func (a *Auth) GenerateToken(claims Claims) (string, error) {
	if claims.Iat == 0 {
		claims.Iat = time.Now().Unix()
	}
	if claims.Exp == 0 {
		claims.Exp = time.Now().Add(a.config.TokenExpiry).Unix()
	}

	header := base64URLEncode([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payloadB64 := base64URLEncode(payload)

	sigInput := header + "." + payloadB64
	sig := a.sign([]byte(sigInput))
	sigB64 := base64URLEncode(sig)

	return sigInput + "." + sigB64, nil
}

// VerifyToken validates and parses a JWT token.
func (a *Auth) VerifyToken(token string) (*Claims, error) {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return nil, errors.New("auth: invalid token format")
	}

	sigInput := parts[0] + "." + parts[1]
	expectedSig := a.sign([]byte(sigInput))
	actualSig, err := base64URLDecode(parts[2])
	if err != nil {
		return nil, errors.New("auth: invalid signature encoding")
	}
	if !hmac.Equal(expectedSig, actualSig) {
		return nil, errors.New("auth: invalid signature")
	}

	payload, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, errors.New("auth: invalid payload encoding")
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("auth: invalid claims: %w", err)
	}

	if claims.IsExpired() {
		return nil, errors.New("auth: token expired")
	}

	return &claims, nil
}

// GenerateRefreshToken creates a longer-lived refresh token.
func (a *Auth) GenerateRefreshToken(sub string) (string, error) {
	claims := Claims{
		Sub: sub,
		Iat: time.Now().Unix(),
		Exp: time.Now().Add(a.config.RefreshExpiry).Unix(),
	}
	return a.GenerateToken(claims)
}

// SetTokenCookie sets the JWT as an HTTP-only cookie.
func (a *Auth) SetTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.config.CookieName,
		Value:    token,
		Path:     a.config.CookiePath,
		Domain:   a.config.CookieDomain,
		MaxAge:   int(a.config.TokenExpiry.Seconds()),
		Secure:   a.config.CookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearTokenCookie removes the auth cookie.
func (a *Auth) ClearTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.config.CookieName,
		Value:    "",
		Path:     a.config.CookiePath,
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// ExtractToken gets the token from Authorization header or cookie.
func (a *Auth) ExtractToken(r *http.Request) string {
	// Check Authorization header first
	authHeader := r.Header.Get(a.config.HeaderName)
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Fallback to cookie
	cookie, err := r.Cookie(a.config.CookieName)
	if err == nil {
		return cookie.Value
	}

	return ""
}

// Middleware returns HTTP middleware that requires authentication.
func (a *Auth) Middleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := a.ExtractToken(r)
			if token == "" {
				http.Error(w, `{"error":"unauthorized","status":401}`, http.StatusUnauthorized)
				return
			}

			claims, err := a.VerifyToken(token)
			if err != nil {
				http.Error(w, `{"error":"invalid token","status":401}`, http.StatusUnauthorized)
				return
			}

			r = r.WithContext(WithClaims(r.Context(), claims))
			next(w, r)
		}
	}
}

// OptionalMiddleware extracts claims if present but doesn't require auth.
func (a *Auth) OptionalMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := a.ExtractToken(r)
			if token != "" {
				claims, err := a.VerifyToken(token)
				if err == nil {
					r = r.WithContext(WithClaims(r.Context(), claims))
				}
			}
			next(w, r)
		}
	}
}

// RequireRole returns middleware that requires a specific role.
func (a *Auth) RequireRole(roles ...string) func(http.HandlerFunc) http.HandlerFunc {
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return a.Middleware()(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r)
			if claims == nil || !roleSet[claims.Role] {
				http.Error(w, `{"error":"forbidden","status":403}`, http.StatusForbidden)
				return
			}
			next(w, r)
		})
	}
}

func (a *Auth) sign(data []byte) []byte {
	mac := hmac.New(sha256.New, []byte(a.config.Secret))
	mac.Write(data)
	return mac.Sum(nil)
}

func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// --- Context helpers ---

type ctxKey string

const claimsKey ctxKey = "nexgo_auth_claims"

// WithClaims stores claims in the context.
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// GetClaims retrieves claims from request context.
func GetClaims(r *http.Request) *Claims {
	if c, ok := r.Context().Value(claimsKey).(*Claims); ok {
		return c
	}
	return nil
}

// IsAuthenticated checks if the request has valid auth claims.
func IsAuthenticated(r *http.Request) bool {
	return GetClaims(r) != nil
}

// --- Password hashing (simple SHA256-based, no bcrypt dep) ---

// HashPassword creates a salted hash of a password.
func HashPassword(password, salt string) string {
	h := sha256.New()
	h.Write([]byte(salt + password + salt))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// CheckPassword verifies a password against its hash.
func CheckPassword(password, salt, hash string) bool {
	return hmac.Equal([]byte(HashPassword(password, salt)), []byte(hash))
}
