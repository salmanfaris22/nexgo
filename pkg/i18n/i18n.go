// Package i18n provides internationalization with locale-based routing,
// translation loading, locale detection, and RTL support.
package i18n

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Locale represents a supported language.
type Locale struct {
	Code      string // e.g., "en", "ar", "fr"
	Name      string // e.g., "English", "Arabic", "French"
	Direction string // "ltr" or "rtl"
}

// Config holds i18n configuration.
type Config struct {
	DefaultLocale  string
	Locales        []Locale
	TranslationDir string // directory containing {locale}.json files
	CookieName     string
	URLPrefix      bool // /en/about, /ar/about
}

// DefaultI18nConfig returns sensible i18n defaults.
func DefaultI18nConfig() Config {
	return Config{
		DefaultLocale: "en",
		Locales: []Locale{
			{Code: "en", Name: "English", Direction: "ltr"},
		},
		TranslationDir: "locales",
		CookieName:     "nexgo_locale",
		URLPrefix:      true,
	}
}

// I18n manages translations and locale detection.
type I18n struct {
	config       Config
	translations map[string]map[string]string // locale -> key -> value
	mu           sync.RWMutex
	localeMap    map[string]Locale
}

// New creates an I18n manager.
func New(cfg Config) *I18n {
	lm := make(map[string]Locale, len(cfg.Locales))
	for _, l := range cfg.Locales {
		lm[l.Code] = l
	}
	return &I18n{
		config:       cfg,
		translations: make(map[string]map[string]string),
		localeMap:    lm,
	}
}

// LoadTranslations reads all JSON translation files from the translations directory.
func (i *I18n) LoadTranslations(rootDir string) error {
	dir := filepath.Join(rootDir, i.config.TranslationDir)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		locale := strings.TrimSuffix(entry.Name(), ".json")
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return err
		}

		var translations map[string]string
		if err := json.Unmarshal(data, &translations); err != nil {
			// Try nested format
			var nested map[string]interface{}
			if err := json.Unmarshal(data, &nested); err != nil {
				continue
			}
			translations = flattenMap(nested, "")
		}

		i.translations[locale] = translations
	}

	return nil
}

// T translates a key for the given locale. Returns the key itself if no translation found.
func (i *I18n) T(locale, key string) string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if trans, ok := i.translations[locale]; ok {
		if val, ok := trans[key]; ok {
			return val
		}
	}
	// Fallback to default locale
	if trans, ok := i.translations[i.config.DefaultLocale]; ok {
		if val, ok := trans[key]; ok {
			return val
		}
	}
	return key
}

// TWithArgs translates with string replacement: {0}, {1}, etc.
func (i *I18n) TWithArgs(locale, key string, args ...string) string {
	text := i.T(locale, key)
	for idx, arg := range args {
		placeholder := "{" + string(rune('0'+idx)) + "}"
		text = strings.ReplaceAll(text, placeholder, arg)
	}
	return text
}

// DetectLocale determines the user's preferred locale from the request.
// Priority: URL prefix > Cookie > Accept-Language header > default.
func (i *I18n) DetectLocale(r *http.Request) string {
	// 1. URL prefix
	if i.config.URLPrefix {
		parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 2)
		if len(parts) > 0 && i.IsSupported(parts[0]) {
			return parts[0]
		}
	}

	// 2. Cookie
	if cookie, err := r.Cookie(i.config.CookieName); err == nil {
		if i.IsSupported(cookie.Value) {
			return cookie.Value
		}
	}

	// 3. Accept-Language header
	if locale := i.parseAcceptLanguage(r.Header.Get("Accept-Language")); locale != "" {
		return locale
	}

	return i.config.DefaultLocale
}

// IsSupported checks if a locale is in the supported list.
func (i *I18n) IsSupported(code string) bool {
	_, ok := i.localeMap[code]
	return ok
}

// IsRTL returns true if the locale uses right-to-left text direction.
func (i *I18n) IsRTL(locale string) bool {
	if l, ok := i.localeMap[locale]; ok {
		return l.Direction == "rtl"
	}
	return false
}

// Direction returns "ltr" or "rtl" for a locale.
func (i *I18n) Direction(locale string) string {
	if l, ok := i.localeMap[locale]; ok {
		return l.Direction
	}
	return "ltr"
}

// StripLocalePrefix removes the locale prefix from a URL path.
func (i *I18n) StripLocalePrefix(path string) string {
	parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
	if len(parts) > 0 && i.IsSupported(parts[0]) {
		if len(parts) > 1 {
			return "/" + parts[1]
		}
		return "/"
	}
	return path
}

// LocalePath returns a URL path with locale prefix.
func (i *I18n) LocalePath(locale, path string) string {
	if !i.config.URLPrefix || locale == i.config.DefaultLocale {
		return path
	}
	return "/" + locale + path
}

// Middleware returns HTTP middleware for locale detection and routing.
func (i *I18n) Middleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			locale := i.DetectLocale(r)

			// Set locale cookie
			http.SetCookie(w, &http.Cookie{
				Name:     i.config.CookieName,
				Value:    locale,
				Path:     "/",
				MaxAge:   365 * 24 * 60 * 60,
				HttpOnly: false,
				SameSite: http.SameSiteLaxMode,
			})

			// Strip locale prefix from URL for downstream routing
			if i.config.URLPrefix {
				r.URL.Path = i.StripLocalePrefix(r.URL.Path)
			}

			// Set HTML direction header
			w.Header().Set("Content-Language", locale)

			// Inject locale into context
			ctx := context.WithValue(r.Context(), localeKey, locale)
			ctx = context.WithValue(ctx, i18nKey, i)
			next(w, r.WithContext(ctx))
		}
	}
}

// Locales returns all supported locales.
func (i *I18n) Locales() []Locale {
	return i.config.Locales
}

// TemplateFuncMap returns template helper functions for i18n.
func (i *I18n) TemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"t": func(locale, key string) string {
			return i.T(locale, key)
		},
		"dir": func(locale string) string {
			return i.Direction(locale)
		},
		"isRTL": func(locale string) bool {
			return i.IsRTL(locale)
		},
		"localePath": func(locale, path string) string {
			return i.LocalePath(locale, path)
		},
		"locales": func() []Locale {
			return i.config.Locales
		},
	}
}

func (i *I18n) parseAcceptLanguage(header string) string {
	if header == "" {
		return ""
	}
	// Parse "en-US,en;q=0.9,ar;q=0.8"
	for _, part := range strings.Split(header, ",") {
		lang := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		// Try exact match
		if i.IsSupported(lang) {
			return lang
		}
		// Try language code without region
		if idx := strings.Index(lang, "-"); idx > 0 {
			code := lang[:idx]
			if i.IsSupported(code) {
				return code
			}
		}
	}
	return ""
}

func flattenMap(m map[string]interface{}, prefix string) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch val := v.(type) {
		case string:
			result[key] = val
		case map[string]interface{}:
			for fk, fv := range flattenMap(val, key) {
				result[fk] = fv
			}
		}
	}
	return result
}

// --- Context helpers ---

type ctxKeyType string

const (
	localeKey ctxKeyType = "nexgo_locale"
	i18nKey   ctxKeyType = "nexgo_i18n"
)

// GetLocale returns the current locale from request context.
func GetLocale(r *http.Request) string {
	if l, ok := r.Context().Value(localeKey).(string); ok {
		return l
	}
	return "en"
}

// GetI18n returns the I18n instance from request context.
func GetI18n(r *http.Request) *I18n {
	if i, ok := r.Context().Value(i18nKey).(*I18n); ok {
		return i
	}
	return nil
}

// RTLLocales are common right-to-left language codes.
var RTLLocales = map[string]bool{
	"ar": true, "he": true, "fa": true, "ur": true,
	"ps": true, "sd": true, "yi": true, "ku": true,
}

// CommonLocales provides preset locale definitions.
var CommonLocales = map[string]Locale{
	"en": {Code: "en", Name: "English", Direction: "ltr"},
	"es": {Code: "es", Name: "Espanol", Direction: "ltr"},
	"fr": {Code: "fr", Name: "Francais", Direction: "ltr"},
	"de": {Code: "de", Name: "Deutsch", Direction: "ltr"},
	"ar": {Code: "ar", Name: "Arabic", Direction: "rtl"},
	"he": {Code: "he", Name: "Hebrew", Direction: "rtl"},
	"fa": {Code: "fa", Name: "Persian", Direction: "rtl"},
	"zh": {Code: "zh", Name: "Chinese", Direction: "ltr"},
	"ja": {Code: "ja", Name: "Japanese", Direction: "ltr"},
	"ko": {Code: "ko", Name: "Korean", Direction: "ltr"},
	"pt": {Code: "pt", Name: "Portuguese", Direction: "ltr"},
	"ru": {Code: "ru", Name: "Russian", Direction: "ltr"},
	"hi": {Code: "hi", Name: "Hindi", Direction: "ltr"},
	"tr": {Code: "tr", Name: "Turkish", Direction: "ltr"},
}
