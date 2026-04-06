package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ProjectName != "nexgo-app" {
		t.Errorf("expected nexgo-app, got %s", cfg.ProjectName)
	}
	if cfg.Port != 3000 {
		t.Errorf("expected port 3000, got %d", cfg.Port)
	}
	if cfg.Host != "localhost" {
		t.Errorf("expected localhost, got %s", cfg.Host)
	}
	if !cfg.HotReload {
		t.Error("expected HotReload to be true")
	}
	if !cfg.Compression {
		t.Error("expected Compression to be true")
	}
}

func TestLoad_NoConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ProjectName != "nexgo-app" {
		t.Error("expected defaults when no config file exists")
	}
	if cfg.RootDir != dir {
		t.Errorf("expected RootDir=%s, got %s", dir, cfg.RootDir)
	}
}

func TestLoad_WithConfigFile(t *testing.T) {
	dir := t.TempDir()
	configJSON := `{
		"projectName": "my-app",
		"port": 8080,
		"host": "0.0.0.0",
		"hotReload": false,
		"compression": false
	}`
	err := os.WriteFile(filepath.Join(dir, "nexgo.config.json"), []byte(configJSON), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ProjectName != "my-app" {
		t.Errorf("expected my-app, got %s", cfg.ProjectName)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected 8080, got %d", cfg.Port)
	}
	if cfg.Host != "0.0.0.0" {
		t.Errorf("expected 0.0.0.0, got %s", cfg.Host)
	}
	if cfg.HotReload {
		t.Error("expected HotReload to be false")
	}
	if cfg.Compression {
		t.Error("expected Compression to be false")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "nexgo.config.json"), []byte(`{invalid`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Load(dir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestAbsPath(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RootDir = "/project"
	if cfg.AbsPath("pages") != "/project/pages" {
		t.Error("AbsPath failed")
	}
}

func TestPathHelpers(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RootDir = "/project"
	cfg.PagesDir = "pages"
	cfg.StaticDir = "static"
	cfg.OutputDir = ".nexgo/out"

	if cfg.PagesAbsDir() != "/project/pages" {
		t.Errorf("PagesAbsDir: %s", cfg.PagesAbsDir())
	}
	if cfg.StaticAbsDir() != "/project/static" {
		t.Errorf("StaticAbsDir: %s", cfg.StaticAbsDir())
	}
	if cfg.OutputAbsDir() != "/project/.nexgo/out" {
		t.Errorf("OutputAbsDir: %s", cfg.OutputAbsDir())
	}
}

func TestSetDevMode(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DevMode {
		t.Error("expected DevMode false by default")
	}
	cfg.SetDevMode(true)
	if !cfg.DevMode {
		t.Error("expected DevMode true after SetDevMode(true)")
	}
	cfg.SetDevMode(false)
	if cfg.DevMode {
		t.Error("expected DevMode false after SetDevMode(false)")
	}
}
