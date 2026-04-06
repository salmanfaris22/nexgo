package middleware

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Logger middleware logs all requests
func Logger(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, status: 200}
		next(wrapped, r)
		dur := time.Since(start)

		status := wrapped.status
		color := "\033[32m" // green
		if status >= 400 {
			color = "\033[33m" // yellow
		}
		if status >= 500 {
			color = "\033[31m" // red
		}

		log.Printf("%s%d\033[0m %s %s %s",
			color, status, r.Method, r.URL.Path, dur)
	}
}

// asyncLogEntry holds data for background log writing.
type asyncLogEntry struct {
	status int
	method string
	path   string
	dur    time.Duration
}

var (
	asyncLogCh   chan asyncLogEntry
	asyncLogOnce sync.Once
)

func startAsyncLogger() {
	asyncLogCh = make(chan asyncLogEntry, 8192)
	go func() {
		for e := range asyncLogCh {
			color := "\033[32m"
			if e.status >= 400 {
				color = "\033[33m"
			}
			if e.status >= 500 {
				color = "\033[31m"
			}
			log.Printf("%s%d\033[0m %s %s %s",
				color, e.status, e.method, e.path, e.dur)
		}
	}()
}

// AsyncLogger is a non-blocking logger that writes to a buffered channel.
// Under high throughput this avoids blocking request goroutines on I/O.
func AsyncLogger(next http.HandlerFunc) http.HandlerFunc {
	asyncLogOnce.Do(startAsyncLogger)
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, status: 200}
		next(wrapped, r)
		select {
		case asyncLogCh <- asyncLogEntry{
			status: wrapped.status,
			method: r.Method,
			path:   r.URL.Path,
			dur:    time.Since(start),
		}:
		default:
			// Drop log entry if buffer is full — never block the request
		}
	}
}

// CORS middleware adds cross-origin headers
func CORS(origins ...string) func(http.HandlerFunc) http.HandlerFunc {
	if len(origins) == 0 {
		return func(next http.HandlerFunc) http.HandlerFunc {
			return next
		}
	}
	allowed := strings.Join(origins, ", ")
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowed)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next(w, r)
		}
	}
}

// Gzip middleware compresses responses
func Gzip(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next(w, r)
			return
		}
		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			next(w, r)
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")
		next(&gzipResponseWriter{ResponseWriter: w, writer: gz}, r)
	}
}

// SecurityHeaders adds security-related HTTP headers
func SecurityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next(w, r)
	}
}

// Cache sets cache-control headers for static assets
func Cache(maxAge int) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(maxAge))
			next(w, r)
		}
	}
}

// Recover from panics gracefully
func Recover(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[NexGo] PANIC: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next(w, r)
	}
}

// Chain combines multiple middleware into one
func Chain(middlewares ...func(http.HandlerFunc) http.HandlerFunc) func(http.HandlerFunc) http.HandlerFunc {
	return func(final http.HandlerFunc) http.HandlerFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// CSP adds Content-Security-Policy headers.
func CSP(policy string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Security-Policy", policy)
			next(w, r)
		}
	}
}

// CSPWithNonce generates a nonce-based CSP header per request.
func CSPWithNonce(basePolicy string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			nonce := generateNonce()
			policy := strings.ReplaceAll(basePolicy, "{nonce}", nonce)
			w.Header().Set("Content-Security-Policy", policy)
			w.Header().Set("X-CSP-Nonce", nonce)
			next(w, r)
		}
	}
}

func generateNonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// RouteMiddleware applies middleware only to specific route patterns.
func RouteMiddleware(pattern string, mw func(http.HandlerFunc) http.HandlerFunc) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if matchPattern(r.URL.Path, pattern) {
				mw(next)(w, r)
				return
			}
			next(w, r)
		}
	}
}

// RouteGroup applies middleware to a group of routes matching a prefix.
func RouteGroup(prefix string, middlewares ...func(http.HandlerFunc) http.HandlerFunc) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		chained := Chain(middlewares...)(next)
		return func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, prefix) {
				chained(w, r)
				return
			}
			next(w, r)
		}
	}
}

// Timeout wraps a handler with a timeout.
func Timeout(d time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			done := make(chan struct{})
			go func() {
				next(w, r.WithContext(ctx))
				close(done)
			}()
			select {
			case <-done:
			case <-ctx.Done():
				http.Error(w, "Request Timeout", http.StatusGatewayTimeout)
			}
		}
	}
}

// RequestID adds a unique request ID header.
func RequestID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			b := make([]byte, 16)
			rand.Read(b)
			id = fmt.Sprintf("%x", b)
		}
		w.Header().Set("X-Request-ID", id)
		next(w, r)
	}
}

func matchPattern(path, pattern string) bool {
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(path, strings.TrimSuffix(pattern, "*"))
	}
	return path == pattern
}

// --- internal helpers ---

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying writer does not support Hijack")
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer io.Writer
}

func (g *gzipResponseWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data)
}

func (g *gzipResponseWriter) Flush() {
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (g *gzipResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := g.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying writer does not support Hijack")
}
