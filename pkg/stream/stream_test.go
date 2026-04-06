package stream

import (
	"net/http/httptest"
	"testing"
)

func TestWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := New(rec)

	n, err := sw.WriteString("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes, got %d", n)
	}
}

func TestWritef(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := New(rec)

	n, err := sw.Writef("count: %d", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 9 {
		t.Errorf("expected 9 bytes, got %d", n)
	}
	if rec.Body.String() != "count: 42" {
		t.Errorf("expected 'count: 42', got %s", rec.Body.String())
	}
}

func TestStreamHTML(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := StreamHTML(rec)

	sw.WriteString("<html><body>")
	sw.WriteString("<h1>Hello</h1>")
	sw.WriteString("</body></html>")

	if rec.Body.String() != "<html><body><h1>Hello</h1></body></html>" {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}

func TestSSEWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	sse := NewSSE(rec)

	err := sse.Send("message", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = sse.SendData("simple data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = sse.Ping()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := rec.Body.String()
	if body == "" {
		t.Error("expected SSE output")
	}
}

func TestFlush(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := New(rec)
	sw.Flush() // Should not panic
}
