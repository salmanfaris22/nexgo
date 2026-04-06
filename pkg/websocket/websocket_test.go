package websocket

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsWebSocketUpgrade(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Upgrade", "websocket")
	r.Header.Set("Connection", "Upgrade")

	if !isWebSocketUpgrade(r) {
		t.Error("expected websocket upgrade request")
	}
}

func TestIsWebSocketUpgrade_False(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)

	if isWebSocketUpgrade(r) {
		t.Error("expected non-websocket request")
	}
}

func TestComputeAcceptKey(t *testing.T) {
	key := "dGhlIHNhbXBsZSBub25jZQ=="
	accept := computeAcceptKey(key)

	// Standard WebSocket accept key for the sample nonce
	expected := "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="
	if accept != expected {
		t.Errorf("expected %s, got %s", expected, accept)
	}
}

func TestUpgrade_NotWebsocket(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	err := Upgrade(w, r, func(conn *Conn) {})
	if err == nil {
		t.Error("expected error for non-websocket request")
	}
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpgrade_MissingKey(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Upgrade", "websocket")
	r.Header.Set("Connection", "Upgrade")

	err := Upgrade(w, r, func(conn *Conn) {})
	if err == nil {
		t.Error("expected error for missing Sec-WebSocket-Key")
	}
}

func TestHub(t *testing.T) {
	h := NewHub()
	if h == nil {
		t.Fatal("expected hub")
	}
}

func TestHub_AddRemove(t *testing.T) {
	h := NewHub()

	// We can't easily create a real Conn without hijacking,
	// but we can test the hub structure
	if h.Count() != 0 {
		t.Errorf("expected 0 clients, got %d", h.Count())
	}
}

func TestHub_JoinLeave(t *testing.T) {
	h := NewHub()

	h.Join(nil, "room1")
	if h.RoomCount("room1") != 1 {
		t.Errorf("expected 1 client in room1, got %d", h.RoomCount("room1"))
	}

	h.Leave(nil, "room1")
	if h.RoomCount("room1") != 0 {
		t.Errorf("expected 0 clients in room1, got %d", h.RoomCount("room1"))
	}
}

func TestHub_Broadcast(t *testing.T) {
	h := NewHub()
	// Broadcast with no clients should not panic
	h.Broadcast("hello")
}

func TestHub_BroadcastTo(t *testing.T) {
	h := NewHub()
	// BroadcastTo with no clients should not panic
	h.BroadcastTo("room1", "hello")
}

func TestHub_Count(t *testing.T) {
	h := NewHub()
	if h.Count() != 0 {
		t.Errorf("expected 0, got %d", h.Count())
	}
}

func TestHub_RoomCount(t *testing.T) {
	h := NewHub()
	if h.RoomCount("nonexistent") != 0 {
		t.Errorf("expected 0, got %d", h.RoomCount("nonexistent"))
	}
}

func TestHub_Remove(t *testing.T) {
	h := NewHub()
	// Remove with no clients should not panic
	h.Remove(nil)
}

func TestOpCodes(t *testing.T) {
	if OpText != 1 {
		t.Errorf("expected OpText=1, got %d", OpText)
	}
	if OpBinary != 2 {
		t.Errorf("expected OpBinary=2, got %d", OpBinary)
	}
	if OpClose != 8 {
		t.Errorf("expected OpClose=8, got %d", OpClose)
	}
	if OpPing != 9 {
		t.Errorf("expected OpPing=9, got %d", OpPing)
	}
	if OpPong != 10 {
		t.Errorf("expected OpPong=10, got %d", OpPong)
	}
}

func TestUpgrade_ErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	Upgrade(w, r, func(conn *Conn) {})

	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Not a WebSocket") {
		t.Error("expected error message in body")
	}
}
