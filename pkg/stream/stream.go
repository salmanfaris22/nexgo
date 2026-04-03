// Package stream provides streaming SSR helpers for progressive page rendering.
package stream

import (
	"fmt"
	"net/http"
	"sync"
)

// Writer wraps http.ResponseWriter with streaming helpers.
type Writer struct {
	http.ResponseWriter
	once sync.Once
}

// New creates a streaming Writer.
func New(w http.ResponseWriter) *Writer {
	return &Writer{ResponseWriter: w}
}

// Flush sends buffered content to the client immediately.
func (s *Writer) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// WriteHeader sets streaming headers and status code.
func (s *Writer) WriteHeader(status int) {
	s.once.Do(func() {
		s.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
		s.ResponseWriter.Header().Set("Cache-Control", "no-cache")
		s.ResponseWriter.Header().Set("X-Accel-Buffering", "no")
		s.ResponseWriter.Header().Set("Transfer-Encoding", "chunked")
		s.ResponseWriter.WriteHeader(status)
	})
}

// Write sends bytes and auto-flushes.
func (s *Writer) Write(b []byte) (int, error) {
	s.WriteHeader(http.StatusOK)
	n, err := s.ResponseWriter.Write(b)
	s.Flush()
	return n, err
}

// WriteString sends a string and auto-flushes.
func (s *Writer) WriteString(str string) (int, error) {
	return s.Write([]byte(str))
}

// Writef formats and writes a string, then flushes.
func (s *Writer) Writef(format string, args ...interface{}) (int, error) {
	return s.WriteString(fmt.Sprintf(format, args...))
}

// StreamHTML streams an HTML page progressively.
// Usage:
//
//	sw := stream.StreamHTML(w)
//	sw.Write("<html><head>...</head><body>")
//	sw.Flush()
//	// ... do slow work ...
//	sw.Write("<main>content</main>")
//	sw.Write("</body></html>")
func StreamHTML(w http.ResponseWriter) *Writer {
	sw := New(w)
	sw.WriteHeader(http.StatusOK)
	sw.Flush()
	return sw
}

// SSEWriter is a Server-Sent Events stream writer.
type SSEWriter struct {
	*Writer
}

// NewSSE creates an SSE stream.
func NewSSE(w http.ResponseWriter) *SSEWriter {
	sw := New(w)
	sw.ResponseWriter.Header().Set("Content-Type", "text/event-stream")
	sw.ResponseWriter.Header().Set("Cache-Control", "no-cache")
	sw.ResponseWriter.Header().Set("Connection", "keep-alive")
	sw.ResponseWriter.Header().Set("X-Accel-Buffering", "no")
	sw.WriteHeader(http.StatusOK)
	sw.Flush()
	return &SSEWriter{Writer: sw}
}

// Send writes an SSE event with the given name and data.
func (s *SSEWriter) Send(name, data string) error {
	_, err := s.WriteString(fmt.Sprintf("event: %s\ndata: %s\n\n", name, data))
	return err
}

// SendData writes a simple SSE data-only event.
func (s *SSEWriter) SendData(data string) error {
	_, err := s.WriteString(fmt.Sprintf("data: %s\n\n", data))
	return err
}

// Ping sends an SSE ping to keep the connection alive.
func (s *SSEWriter) Ping() error {
	_, err := s.WriteString(": ping\n\n")
	return err
}
