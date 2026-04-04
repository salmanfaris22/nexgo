package session

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()

	sess := &Session{
		ID:        "test-id",
		Data:      map[string]interface{}{"key": "value"},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	if err := store.Save(sess); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get("test-id")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("session should exist")
	}
	if got.ID != "test-id" {
		t.Errorf("ID = %q, want %q", got.ID, "test-id")
	}

	if err := store.Delete("test-id"); err != nil {
		t.Fatal(err)
	}

	got, _ = store.Get("test-id")
	if got != nil {
		t.Error("session should be deleted")
	}
}

func TestSessionData(t *testing.T) {
	sess := &Session{
		ID:   "test",
		Data: make(map[string]interface{}),
	}

	sess.Set("name", "John")
	if sess.GetString("name") != "John" {
		t.Errorf("GetString = %q, want %q", sess.GetString("name"), "John")
	}

	if !sess.Has("name") {
		t.Error("Has should be true")
	}
	if sess.Has("missing") {
		t.Error("Has should be false for missing key")
	}

	sess.Delete("name")
	if sess.Has("name") {
		t.Error("key should be deleted")
	}

	sess.Set("a", 1)
	sess.Set("b", 2)
	sess.Clear()
	if sess.Has("a") || sess.Has("b") {
		t.Error("Clear should remove all keys")
	}
}

func TestFlashMessages(t *testing.T) {
	sess := &Session{
		ID:   "test",
		Data: make(map[string]interface{}),
	}

	sess.Flash("success", "Account created!")

	msg := sess.GetFlash("success")
	if msg != "Account created!" {
		t.Errorf("GetFlash = %v, want %q", msg, "Account created!")
	}

	// Second read should return nil
	msg = sess.GetFlash("success")
	if msg != nil {
		t.Errorf("GetFlash second call = %v, want nil", msg)
	}
}

func TestManagerStartCreatesSession(t *testing.T) {
	store := NewMemoryStore()
	cfg := DefaultConfig()
	mgr := NewManager(store, cfg)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	sess, err := mgr.Start(w, req)
	if err != nil {
		t.Fatal(err)
	}
	if sess == nil {
		t.Fatal("session should not be nil")
	}
	if sess.ID == "" {
		t.Error("session ID should not be empty")
	}

	// Check cookie was set
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == cfg.CookieName {
			found = true
		}
	}
	if !found {
		t.Error("session cookie should be set")
	}
}

func TestManagerDestroy(t *testing.T) {
	store := NewMemoryStore()
	cfg := DefaultConfig()
	mgr := NewManager(store, cfg)

	// Create session
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	sess, _ := mgr.Start(w, req)
	sessID := sess.ID

	// Destroy it
	cookie := w.Result().Cookies()[0]
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.AddCookie(cookie)
	w2 := httptest.NewRecorder()

	mgr.Destroy(w2, req2)

	got, _ := store.Get(sessID)
	if got != nil {
		t.Error("session should be deleted from store")
	}
}

func TestFileStore(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	sess := &Session{
		ID:        "file-test",
		Data:      map[string]interface{}{"key": "value"},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	if err := store.Save(sess); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get("file-test")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("session should exist")
	}
	if got.ID != "file-test" {
		t.Errorf("ID = %q, want %q", got.ID, "file-test")
	}

	if err := store.Delete("file-test"); err != nil {
		t.Fatal(err)
	}

	got, _ = store.Get("file-test")
	if got != nil {
		t.Error("session should be deleted")
	}
}

func TestMemoryStoreCount(t *testing.T) {
	store := NewMemoryStore()
	if store.Count() != 0 {
		t.Errorf("Count = %d, want 0", store.Count())
	}

	store.Save(&Session{ID: "1", ExpiresAt: time.Now().Add(time.Hour)})
	store.Save(&Session{ID: "2", ExpiresAt: time.Now().Add(time.Hour)})
	if store.Count() != 2 {
		t.Errorf("Count = %d, want 2", store.Count())
	}
}

func TestSessionMiddleware(t *testing.T) {
	store := NewMemoryStore()
	mgr := NewManager(store, DefaultConfig())

	var gotSess *Session
	handler := SessionMiddleware(mgr)(func(w http.ResponseWriter, r *http.Request) {
		gotSess = FromRequest(r)
		w.WriteHeader(200)
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if gotSess == nil {
		t.Fatal("session should be injected into context")
	}
}
