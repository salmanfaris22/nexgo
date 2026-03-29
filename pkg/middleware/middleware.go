package middleware

import (
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strings"
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

// CORS middleware adds cross-origin headers
func CORS(origins ...string) func(http.HandlerFunc) http.HandlerFunc {
	allowed := "*"
	if len(origins) > 0 {
		allowed = strings.Join(origins, ", ")
	}
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
			w.Header().Set("Cache-Control", "public, max-age="+string(rune(maxAge)))
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

// --- internal helpers ---

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer io.Writer
}

func (g *gzipResponseWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data)
}
