package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// NexGoConfig holds all framework configuration
type NexGoConfig struct {
	// Project settings
	ProjectName string `json:"projectName"`
	Port        int    `json:"port"`
	Host        string `json:"host"`

	// Directories
	PagesDir      string `json:"pagesDir"`
	StaticDir     string `json:"staticDir"`
	OutputDir     string `json:"outputDir"`
	ComponentsDir string `json:"componentsDir"`
	LayoutsDir    string `json:"layoutsDir"`

	// Build settings
	Minify        bool   `json:"minify"`
	SourceMaps    bool   `json:"sourceMaps"`
	BaseURL       string `json:"baseURL"`
	TrailingSlash bool   `json:"trailingSlash"`

	// SSR / SSG settings
	DefaultRenderMode string `json:"defaultRenderMode"` // "ssr", "ssg", "spa"

	// Dev settings
	HotReload bool `json:"hotReload"`
	DevTools  bool `json:"devTools"`

	// Performance
	CacheControl string `json:"cacheControl"`
	Compression  bool   `json:"compression"`

	// Auth & Session
	SessionSecret string `json:"sessionSecret"`
	SessionMaxAge int    `json:"sessionMaxAge"` // seconds
	AuthSecret    string `json:"authSecret"`
	CSRFSecret    string `json:"csrfSecret"`

	// Rate Limiting
	RateLimitPerMinute int `json:"rateLimitPerMinute"`

	// i18n
	DefaultLocale  string   `json:"defaultLocale"`
	Locales        []string `json:"locales"`
	TranslationDir string   `json:"translationDir"`

	// Image Optimization
	ImageOptimization bool `json:"imageOptimization"`

	// Database
	DatabaseDriver string `json:"databaseDriver"` // "json", "sqlite3", "postgres"
	DatabaseDSN    string `json:"databaseDSN"`
	DatabaseDir    string `json:"databaseDir"`

	// Logging
	LogLevel  string `json:"logLevel"`  // "debug", "info", "warn", "error"
	LogFormat string `json:"logFormat"` // "text", "json"
	LogDir    string `json:"logDir"`

	// Metrics
	MetricsEnabled bool `json:"metricsEnabled"`

	// Health
	HealthEnabled bool `json:"healthEnabled"`

	// WebSocket
	WebSocketEnabled bool `json:"webSocketEnabled"`

	// Plugins
	Plugins []string `json:"plugins"`

	// Internal (not in config file)
	RootDir string `json:"-"`
	DevMode bool   `json:"-"`
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *NexGoConfig {
	return &NexGoConfig{
		ProjectName:       "nexgo-app",
		Port:              3000,
		Host:              "localhost",
		PagesDir:          "pages",
		StaticDir:         "static",
		OutputDir:         ".nexgo/out",
		ComponentsDir:     "components",
		LayoutsDir:        "layouts",
		Minify:            true,
		SourceMaps:        false,
		BaseURL:           "",
		TrailingSlash:     false,
		DefaultRenderMode: "ssr",
		HotReload:         true,
		DevTools:          true,
		CacheControl:      "public, max-age=31536000",
		Compression:       true,
		// Auth & Session
		SessionSecret:  "change-me-in-production",
		SessionMaxAge:  86400,
		AuthSecret:     "change-me-in-production",
		CSRFSecret:     "change-me-csrf-secret",
		// Rate Limiting
		RateLimitPerMinute: 60,
		// i18n
		DefaultLocale:  "en",
		TranslationDir: "locales",
		// Image Optimization
		ImageOptimization: true,
		// Database
		DatabaseDriver: "json",
		DatabaseDir:    ".nexgo/data",
		// Logging
		LogLevel:  "info",
		LogFormat: "text",
		LogDir:    ".nexgo/logs",
		// Metrics & Health
		MetricsEnabled: true,
		HealthEnabled:  true,
		// WebSocket
		WebSocketEnabled: true,
	}
}

// Load reads nexgo.config.json from the project root
func Load(rootDir string) (*NexGoConfig, error) {
	cfg := DefaultConfig()
	cfg.RootDir = rootDir

	configPath := filepath.Join(rootDir, "nexgo.config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Use defaults if no config file
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.RootDir = rootDir
	return cfg, nil
}

// AbsPath returns absolute path for a config-relative path
func (c *NexGoConfig) AbsPath(rel string) string {
	return filepath.Join(c.RootDir, rel)
}

func (c *NexGoConfig) PagesAbsDir() string  { return c.AbsPath(c.PagesDir) }
func (c *NexGoConfig) StaticAbsDir() string { return c.AbsPath(c.StaticDir) }
func (c *NexGoConfig) OutputAbsDir() string { return c.AbsPath(c.OutputDir) }

// SetDevMode safely changes DevMode with a mutex-free atomic operation.
// DevMode is only read by the builder during single-threaded builds,
// so direct assignment is safe in that context.
func (c *NexGoConfig) SetDevMode(v bool) {
	c.DevMode = v
}
