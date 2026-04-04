package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFile(t *testing.T) {
	dir := t.TempDir()

	// Write .env file
	os.WriteFile(filepath.Join(dir, ".env"), []byte(`
# Comment
APP_NAME=NexGo
PORT=3000
SECRET="my-secret"
SINGLE='single-quoted'
EXPANDED=${APP_NAME}-app
export EXPORTED=yes
`), 0644)

	store, err := Load(dir, "")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"APP_NAME", "NexGo"},
		{"PORT", "3000"},
		{"SECRET", "my-secret"},
		{"SINGLE", "single-quoted"},
		{"EXPANDED", "NexGo-app"},
		{"EXPORTED", "yes"},
	}

	for _, tt := range tests {
		got := store.Get(tt.key)
		if got != tt.expected {
			t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}

func TestPriorityOrder(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, ".env"), []byte("KEY=base\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".env.local"), []byte("KEY=local\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".env.production"), []byte("KEY=prod\n"), 0644)

	store, err := Load(dir, "production")
	if err != nil {
		t.Fatal(err)
	}

	if got := store.Get("KEY"); got != "prod" {
		t.Errorf("Get(KEY) = %q, want %q", got, "prod")
	}
}

func TestOSEnvWins(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".env"), []byte("TEST_ENV_VAR=from-file\n"), 0644)

	os.Setenv("TEST_ENV_VAR", "from-os")
	defer os.Unsetenv("TEST_ENV_VAR")

	store, err := Load(dir, "")
	if err != nil {
		t.Fatal(err)
	}

	if got := store.Get("TEST_ENV_VAR"); got != "from-os" {
		t.Errorf("Get(TEST_ENV_VAR) = %q, want %q (OS should win)", got, "from-os")
	}
}

func TestGetDefault(t *testing.T) {
	store := New()
	if got := store.GetDefault("MISSING", "fallback"); got != "fallback" {
		t.Errorf("GetDefault = %q, want %q", got, "fallback")
	}
	store.Set("EXISTS", "value")
	if got := store.GetDefault("EXISTS", "fallback"); got != "value" {
		t.Errorf("GetDefault = %q, want %q", got, "value")
	}
}

func TestMustGetPanics(t *testing.T) {
	store := New()
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGet should have panicked")
		}
	}()
	store.MustGet("MISSING")
}

func TestHas(t *testing.T) {
	store := New()
	store.Set("KEY", "val")
	if !store.Has("KEY") {
		t.Error("Has(KEY) should be true")
	}
	if store.Has("MISSING") {
		t.Error("Has(MISSING) should be false")
	}
}

func TestModeHelpers(t *testing.T) {
	store := New()
	if !store.IsDevelopment() {
		t.Error("empty mode should be development")
	}
	store.Set("NEXGO_MODE", "production")
	if !store.IsProduction() {
		t.Error("should be production")
	}
	if store.IsDevelopment() {
		t.Error("should not be development")
	}
}
