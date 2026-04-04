// Package env provides .env file loading and environment variable management.
// Supports .env, .env.local, .env.development, .env.production with priority ordering.
package env

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Store holds loaded environment variables with thread-safe access.
type Store struct {
	mu   sync.RWMutex
	vars map[string]string
}

// New creates an empty environment store.
func New() *Store {
	return &Store{vars: make(map[string]string)}
}

// Load reads .env files from the given root directory in priority order.
// Later files override earlier ones. Priority (lowest to highest):
//
//	.env  <  .env.local  <  .env.{mode}  <  .env.{mode}.local
//
// Real OS environment variables always win over file-based values.
func Load(rootDir string, mode string) (*Store, error) {
	s := New()

	// Load files in priority order (lowest first)
	files := []string{
		".env",
		".env.local",
	}
	if mode != "" {
		files = append(files, ".env."+mode)
		files = append(files, ".env."+mode+".local")
	}

	for _, name := range files {
		path := filepath.Join(rootDir, name)
		if err := s.loadFile(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("loading %s: %w", name, err)
		}
	}

	return s, nil
}

// loadFile parses a single .env file.
func (s *Store) loadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle export prefix
		line = strings.TrimPrefix(line, "export ")

		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Remove surrounding quotes
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		// Expand ${VAR} references
		value = os.Expand(value, func(key string) string {
			if v, ok := s.vars[key]; ok {
				return v
			}
			return os.Getenv(key)
		})

		s.mu.Lock()
		s.vars[key] = value
		s.mu.Unlock()
	}

	return scanner.Err()
}

// Get returns an environment variable value.
// OS environment takes precedence over .env files.
func (s *Store) Get(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	s.mu.RLock()
	v := s.vars[key]
	s.mu.RUnlock()
	return v
}

// GetDefault returns the value or a default if not set.
func (s *Store) GetDefault(key, defaultVal string) string {
	v := s.Get(key)
	if v == "" {
		return defaultVal
	}
	return v
}

// MustGet returns the value or panics if not set.
func (s *Store) MustGet(key string) string {
	v := s.Get(key)
	if v == "" {
		panic(fmt.Sprintf("env: required variable %q is not set", key))
	}
	return v
}

// Set stores a value (does not modify OS environment).
func (s *Store) Set(key, value string) {
	s.mu.Lock()
	s.vars[key] = value
	s.mu.Unlock()
}

// All returns a copy of all loaded variables.
func (s *Store) All() map[string]string {
	s.mu.RLock()
	out := make(map[string]string, len(s.vars))
	for k, v := range s.vars {
		out[k] = v
	}
	s.mu.RUnlock()
	return out
}

// Has checks if a variable is set (in file or OS).
func (s *Store) Has(key string) bool {
	return s.Get(key) != ""
}

// SetToOS exports all loaded vars into the process environment.
func (s *Store) SetToOS() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, v := range s.vars {
		if os.Getenv(k) == "" {
			if err := os.Setenv(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// IsProduction returns true if mode is "production".
func (s *Store) IsProduction() bool {
	mode := s.Get("NEXGO_MODE")
	return mode == "production" || mode == "prod"
}

// IsDevelopment returns true if mode is "development" or empty.
func (s *Store) IsDevelopment() bool {
	mode := s.Get("NEXGO_MODE")
	return mode == "" || mode == "development" || mode == "dev"
}

// --- Global convenience functions ---

var globalStore = New()

// LoadGlobal loads .env files into the global store.
func LoadGlobal(rootDir, mode string) error {
	s, err := Load(rootDir, mode)
	if err != nil {
		return err
	}
	globalStore = s
	return nil
}

// GetEnv returns a variable from the global store.
func GetEnv(key string) string { return globalStore.Get(key) }

// GetEnvDefault returns a variable with a fallback from the global store.
func GetEnvDefault(key, def string) string { return globalStore.GetDefault(key, def) }

// MustGetEnv returns a variable or panics from the global store.
func MustGetEnv(key string) string { return globalStore.MustGet(key) }

// SetEnv sets a variable in the global store.
func SetEnv(key, value string) { globalStore.Set(key, value) }

// Global returns the global store.
func Global() *Store { return globalStore }
