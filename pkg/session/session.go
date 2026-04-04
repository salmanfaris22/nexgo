// Package session provides cookie-based session management with pluggable backends.
package session

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store is the session storage backend interface.
type Store interface {
	Get(id string) (*Session, error)
	Save(s *Session) error
	Delete(id string) error
	GC() error
}

// Session holds session data for a single user.
type Session struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt time.Time              `json:"created_at"`
	ExpiresAt time.Time              `json:"expires_at"`
	mu        sync.RWMutex
}

// Config holds session configuration.
type Config struct {
	CookieName string
	Secret     string
	MaxAge     time.Duration
	Path       string
	Domain     string
	Secure     bool
	HTTPOnly   bool
	SameSite   http.SameSite
}

// DefaultConfig returns sensible session defaults.
func DefaultConfig() Config {
	return Config{
		CookieName: "nexgo_session",
		Secret:     "change-me-in-production",
		MaxAge:     24 * time.Hour,
		Path:       "/",
		HTTPOnly:   true,
		SameSite:   http.SameSiteLaxMode,
	}
}

// Manager handles session lifecycle.
type Manager struct {
	store  Store
	config Config
}

// NewManager creates a session manager with the given store and config.
func NewManager(store Store, cfg Config) *Manager {
	return &Manager{store: store, config: cfg}
}

// Start retrieves or creates a session from the request.
func (m *Manager) Start(w http.ResponseWriter, r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(m.config.CookieName)
	if err == nil {
		id, valid := m.verifyID(cookie.Value)
		if valid {
			sess, err := m.store.Get(id)
			if err == nil && sess != nil && time.Now().Before(sess.ExpiresAt) {
				return sess, nil
			}
		}
	}

	sess := &Session{
		ID:        generateID(),
		Data:      make(map[string]interface{}),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(m.config.MaxAge),
	}

	if err := m.store.Save(sess); err != nil {
		return nil, err
	}

	m.setCookie(w, sess.ID)
	return sess, nil
}

// Destroy removes a session and clears the cookie.
func (m *Manager) Destroy(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(m.config.CookieName)
	if err != nil {
		return nil
	}
	id, valid := m.verifyID(cookie.Value)
	if valid {
		if err := m.store.Delete(id); err != nil {
			return err
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     m.config.CookieName,
		Value:    "",
		Path:     m.config.Path,
		MaxAge:   -1,
		HttpOnly: m.config.HTTPOnly,
	})
	return nil
}

// Save persists session changes.
func (m *Manager) Save(sess *Session) error {
	return m.store.Save(sess)
}

func (m *Manager) setCookie(w http.ResponseWriter, id string) {
	signed := m.signID(id)
	http.SetCookie(w, &http.Cookie{
		Name:     m.config.CookieName,
		Value:    signed,
		Path:     m.config.Path,
		Domain:   m.config.Domain,
		MaxAge:   int(m.config.MaxAge.Seconds()),
		Secure:   m.config.Secure,
		HttpOnly: m.config.HTTPOnly,
		SameSite: m.config.SameSite,
	})
}

func (m *Manager) signID(id string) string {
	mac := hmac.New(sha256.New, []byte(m.config.Secret))
	mac.Write([]byte(id))
	sig := hex.EncodeToString(mac.Sum(nil))
	return id + "." + sig
}

func (m *Manager) verifyID(signed string) (string, bool) {
	idx := -1
	for i := len(signed) - 1; i >= 0; i-- {
		if signed[i] == '.' {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "", false
	}
	id := signed[:idx]
	expected := m.signID(id)
	return id, hmac.Equal([]byte(signed), []byte(expected))
}

func generateID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("session: failed to generate random ID")
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// --- Session data methods ---

// Get retrieves a value from the session.
func (s *Session) Get(key string) interface{} {
	s.mu.RLock()
	v := s.Data[key]
	s.mu.RUnlock()
	return v
}

// GetString retrieves a string value.
func (s *Session) GetString(key string) string {
	if str, ok := s.Get(key).(string); ok {
		return str
	}
	return ""
}

// Set stores a value in the session.
func (s *Session) Set(key string, value interface{}) {
	s.mu.Lock()
	s.Data[key] = value
	s.mu.Unlock()
}

// Delete removes a key from the session.
func (s *Session) Delete(key string) {
	s.mu.Lock()
	delete(s.Data, key)
	s.mu.Unlock()
}

// Clear removes all session data.
func (s *Session) Clear() {
	s.mu.Lock()
	s.Data = make(map[string]interface{})
	s.mu.Unlock()
}

// Has checks if a key exists.
func (s *Session) Has(key string) bool {
	s.mu.RLock()
	_, ok := s.Data[key]
	s.mu.RUnlock()
	return ok
}

// Flash sets a value that is removed after the next read.
func (s *Session) Flash(key string, value interface{}) {
	s.Set("_flash_"+key, value)
}

// GetFlash retrieves and removes a flash value.
func (s *Session) GetFlash(key string) interface{} {
	fk := "_flash_" + key
	v := s.Get(fk)
	if v != nil {
		s.Delete(fk)
	}
	return v
}

// --- Memory Store ---

// MemoryStore stores sessions in memory (single-instance / development).
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewMemoryStore creates an in-memory session store.
func NewMemoryStore() *MemoryStore {
	ms := &MemoryStore{sessions: make(map[string]*Session)}
	go ms.gcLoop()
	return ms
}

func (m *MemoryStore) Get(id string) (*Session, error) {
	m.mu.RLock()
	s := m.sessions[id]
	m.mu.RUnlock()
	return s, nil
}

func (m *MemoryStore) Save(s *Session) error {
	m.mu.Lock()
	m.sessions[s.ID] = s
	m.mu.Unlock()
	return nil
}

func (m *MemoryStore) Delete(id string) error {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()
	return nil
}

func (m *MemoryStore) GC() error {
	m.mu.Lock()
	now := time.Now()
	for id, s := range m.sessions {
		if now.After(s.ExpiresAt) {
			delete(m.sessions, id)
		}
	}
	m.mu.Unlock()
	return nil
}

func (m *MemoryStore) gcLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		m.GC()
	}
}

// Count returns the number of active sessions.
func (m *MemoryStore) Count() int {
	m.mu.RLock()
	n := len(m.sessions)
	m.mu.RUnlock()
	return n
}

// --- File Store ---

// FileStore persists sessions as JSON files on disk.
type FileStore struct {
	dir string
	mu  sync.RWMutex
}

// NewFileStore creates a file-based session store.
func NewFileStore(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &FileStore{dir: dir}, nil
}

func (f *FileStore) Get(id string) (*Session, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	data, err := os.ReadFile(filepath.Join(f.dir, id+".json"))
	if err != nil {
		return nil, nil
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (f *FileStore) Save(s *Session) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(f.dir, s.ID+".json"), data, 0600)
}

func (f *FileStore) Delete(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return os.Remove(filepath.Join(f.dir, id+".json"))
}

func (f *FileStore) GC() error {
	entries, err := os.ReadDir(f.dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		path := filepath.Join(f.dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var s Session
		if err := json.Unmarshal(data, &s); err != nil || time.Now().After(s.ExpiresAt) {
			os.Remove(path)
		}
	}
	return nil
}

// --- Context helpers ---

type ctxKey string

const sessionCtxKey ctxKey = "nexgo_session"

// WithSession stores a session in the context.
func WithSession(ctx context.Context, sess *Session) context.Context {
	return context.WithValue(ctx, sessionCtxKey, sess)
}

// FromRequest retrieves the session from request context.
func FromRequest(r *http.Request) *Session {
	if s, ok := r.Context().Value(sessionCtxKey).(*Session); ok {
		return s
	}
	return nil
}

// SessionMiddleware returns HTTP middleware that injects a session into request context.
func SessionMiddleware(mgr *Manager) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			sess, err := mgr.Start(w, r)
			if err != nil {
				http.Error(w, "Session error", http.StatusInternalServerError)
				return
			}
			r = r.WithContext(WithSession(r.Context(), sess))
			next(w, r)
			mgr.Save(sess)
		}
	}
}
