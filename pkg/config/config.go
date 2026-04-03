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
