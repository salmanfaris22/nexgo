package i18n

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestTranslation(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "locales"), 0755)
	os.WriteFile(filepath.Join(dir, "locales", "en.json"), []byte(`{
		"hello": "Hello",
		"bye": "Goodbye"
	}`), 0644)
	os.WriteFile(filepath.Join(dir, "locales", "fr.json"), []byte(`{
		"hello": "Bonjour",
		"bye": "Au revoir"
	}`), 0644)

	cfg := DefaultI18nConfig()
	cfg.Locales = []Locale{
		{Code: "en", Name: "English", Direction: "ltr"},
		{Code: "fr", Name: "French", Direction: "ltr"},
	}
	i := New(cfg)
	i.LoadTranslations(dir)

	if got := i.T("en", "hello"); got != "Hello" {
		t.Errorf("T(en, hello) = %q, want %q", got, "Hello")
	}
	if got := i.T("fr", "hello"); got != "Bonjour" {
		t.Errorf("T(fr, hello) = %q, want %q", got, "Bonjour")
	}
}

func TestFallbackToDefault(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "locales"), 0755)
	os.WriteFile(filepath.Join(dir, "locales", "en.json"), []byte(`{"hello": "Hello"}`), 0644)
	os.WriteFile(filepath.Join(dir, "locales", "fr.json"), []byte(`{}`), 0644)

	cfg := DefaultI18nConfig()
	cfg.Locales = []Locale{
		{Code: "en", Name: "English", Direction: "ltr"},
		{Code: "fr", Name: "French", Direction: "ltr"},
	}
	i := New(cfg)
	i.LoadTranslations(dir)

	// fr doesn't have "hello", should fallback to en
	if got := i.T("fr", "hello"); got != "Hello" {
		t.Errorf("T(fr, hello) = %q, want %q (fallback)", got, "Hello")
	}

	// Missing key returns the key itself
	if got := i.T("en", "missing_key"); got != "missing_key" {
		t.Errorf("T(en, missing_key) = %q, want %q", got, "missing_key")
	}
}

func TestTWithArgs(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "locales"), 0755)
	os.WriteFile(filepath.Join(dir, "locales", "en.json"), []byte(`{
		"greeting": "Hello, {0}! Welcome to {1}."
	}`), 0644)

	i := New(DefaultI18nConfig())
	i.LoadTranslations(dir)

	got := i.TWithArgs("en", "greeting", "John", "NexGo")
	if got != "Hello, John! Welcome to NexGo." {
		t.Errorf("TWithArgs = %q", got)
	}
}

func TestIsSupported(t *testing.T) {
	cfg := DefaultI18nConfig()
	cfg.Locales = []Locale{
		{Code: "en", Name: "English", Direction: "ltr"},
		{Code: "ar", Name: "Arabic", Direction: "rtl"},
	}
	i := New(cfg)

	if !i.IsSupported("en") {
		t.Error("en should be supported")
	}
	if !i.IsSupported("ar") {
		t.Error("ar should be supported")
	}
	if i.IsSupported("fr") {
		t.Error("fr should not be supported")
	}
}

func TestRTL(t *testing.T) {
	cfg := DefaultI18nConfig()
	cfg.Locales = []Locale{
		{Code: "en", Name: "English", Direction: "ltr"},
		{Code: "ar", Name: "Arabic", Direction: "rtl"},
	}
	i := New(cfg)

	if i.IsRTL("en") {
		t.Error("en should not be RTL")
	}
	if !i.IsRTL("ar") {
		t.Error("ar should be RTL")
	}
	if i.Direction("ar") != "rtl" {
		t.Errorf("Direction(ar) = %q, want rtl", i.Direction("ar"))
	}
}

func TestDetectLocaleFromURL(t *testing.T) {
	cfg := DefaultI18nConfig()
	cfg.URLPrefix = true
	cfg.Locales = []Locale{
		{Code: "en", Name: "English", Direction: "ltr"},
		{Code: "fr", Name: "French", Direction: "ltr"},
	}
	i := New(cfg)

	req := httptest.NewRequest("GET", "/fr/about", nil)
	if got := i.DetectLocale(req); got != "fr" {
		t.Errorf("DetectLocale = %q, want %q", got, "fr")
	}
}

func TestDetectLocaleFromAcceptLanguage(t *testing.T) {
	cfg := DefaultI18nConfig()
	cfg.URLPrefix = false
	cfg.Locales = []Locale{
		{Code: "en", Name: "English", Direction: "ltr"},
		{Code: "fr", Name: "French", Direction: "ltr"},
	}
	i := New(cfg)

	req := httptest.NewRequest("GET", "/about", nil)
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en;q=0.8")
	if got := i.DetectLocale(req); got != "fr" {
		t.Errorf("DetectLocale = %q, want %q", got, "fr")
	}
}

func TestStripLocalePrefix(t *testing.T) {
	cfg := DefaultI18nConfig()
	cfg.Locales = []Locale{
		{Code: "en", Name: "English", Direction: "ltr"},
		{Code: "fr", Name: "French", Direction: "ltr"},
	}
	i := New(cfg)

	tests := []struct {
		input, expected string
	}{
		{"/fr/about", "/about"},
		{"/en/blog/post", "/blog/post"},
		{"/about", "/about"},
		{"/fr", "/"},
	}

	for _, tt := range tests {
		got := i.StripLocalePrefix(tt.input)
		if got != tt.expected {
			t.Errorf("StripLocalePrefix(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMiddlewareSetsLocale(t *testing.T) {
	cfg := DefaultI18nConfig()
	cfg.URLPrefix = true
	cfg.Locales = []Locale{
		{Code: "en", Name: "English", Direction: "ltr"},
		{Code: "fr", Name: "French", Direction: "ltr"},
	}
	i := New(cfg)

	var gotLocale string
	handler := i.Middleware()(func(w http.ResponseWriter, r *http.Request) {
		gotLocale = GetLocale(r)
		w.WriteHeader(200)
	})

	req := httptest.NewRequest("GET", "/fr/about", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if gotLocale != "fr" {
		t.Errorf("locale = %q, want %q", gotLocale, "fr")
	}
}
