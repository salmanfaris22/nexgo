// Package logger provides structured logging with JSON output, log levels, and rotation.
package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Level represents a log severity level.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel parses a level string.
func ParseLevel(s string) Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN", "WARNING":
		return LevelWarn
	case "ERROR":
		return LevelError
	case "FATAL":
		return LevelFatal
	default:
		return LevelInfo
	}
}

// Entry is a single log entry.
type Entry struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"msg"`
	Timestamp string                 `json:"time"`
	Caller    string                 `json:"caller,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// Logger is a structured logger.
type Logger struct {
	mu        sync.Mutex
	output    io.Writer
	level     Level
	fields    map[string]interface{}
	json      bool
	caller    bool
	colorize  bool
}

// Config holds logger configuration.
type Config struct {
	Level    string // "debug", "info", "warn", "error"
	JSON     bool   // output as JSON
	Caller   bool   // include caller info
	Colorize bool   // colorize console output
	Output   io.Writer
}

// DefaultConfig returns sensible logger defaults.
func DefaultConfig() Config {
	return Config{
		Level:    "info",
		JSON:     false,
		Caller:   false,
		Colorize: true,
		Output:   os.Stdout,
	}
}

// New creates a structured logger.
func New(cfg Config) *Logger {
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}
	return &Logger{
		output:   output,
		level:    ParseLevel(cfg.Level),
		fields:   make(map[string]interface{}),
		json:     cfg.JSON,
		caller:   cfg.Caller,
		colorize: cfg.Colorize,
	}
}

// With returns a new logger with added fields.
func (l *Logger) With(key string, value interface{}) *Logger {
	newFields := make(map[string]interface{}, len(l.fields)+1)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value
	return &Logger{
		output:   l.output,
		level:    l.level,
		fields:   newFields,
		json:     l.json,
		caller:   l.caller,
		colorize: l.colorize,
	}
}

// WithFields returns a new logger with multiple added fields.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newFields := make(map[string]interface{}, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	return &Logger{
		output:   l.output,
		level:    l.level,
		fields:   newFields,
		json:     l.json,
		caller:   l.caller,
		colorize: l.colorize,
	}
}

// Debug logs at debug level.
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(LevelDebug, msg, args...)
}

// Info logs at info level.
func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(LevelInfo, msg, args...)
}

// Warn logs at warn level.
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(LevelWarn, msg, args...)
}

// Error logs at error level.
func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(LevelError, msg, args...)
}

// Fatal logs at fatal level and exits.
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.log(LevelFatal, msg, args...)
	os.Exit(1)
}

func (l *Logger) log(level Level, msg string, args ...interface{}) {
	if level < l.level {
		return
	}

	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	entry := Entry{
		Level:     level.String(),
		Message:   msg,
		Timestamp: time.Now().Format(time.RFC3339),
		Fields:    l.fields,
	}

	if l.caller {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			entry.Caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.json {
		data, _ := json.Marshal(entry)
		fmt.Fprintln(l.output, string(data))
	} else {
		l.writeText(entry, level)
	}
}

func (l *Logger) writeText(entry Entry, level Level) {
	ts := time.Now().Format("15:04:05")
	levelStr := entry.Level

	if l.colorize {
		switch level {
		case LevelDebug:
			levelStr = "\033[90mDEBUG\033[0m"
		case LevelInfo:
			levelStr = "\033[36mINFO\033[0m "
		case LevelWarn:
			levelStr = "\033[33mWARN\033[0m "
		case LevelError:
			levelStr = "\033[31mERROR\033[0m"
		case LevelFatal:
			levelStr = "\033[35mFATAL\033[0m"
		}
	}

	line := fmt.Sprintf("%s %s %s", ts, levelStr, entry.Message)
	if entry.Caller != "" {
		line += fmt.Sprintf(" \033[90m(%s)\033[0m", entry.Caller)
	}
	if len(entry.Fields) > 0 {
		for k, v := range entry.Fields {
			line += fmt.Sprintf(" \033[90m%s=\033[0m%v", k, v)
		}
	}
	fmt.Fprintln(l.output, line)
}

// RequestLogger returns HTTP middleware that logs requests in structured format.
func (l *Logger) RequestLogger() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &statusWriter{ResponseWriter: w, status: 200}
			next(wrapped, r)
			duration := time.Since(start)

			l.With("method", r.Method).
				With("path", r.URL.Path).
				With("status", wrapped.status).
				With("duration", duration.String()).
				With("ip", r.RemoteAddr).
				Info("request")
		}
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

// --- File rotation ---

// RotatingWriter writes to a file with automatic rotation.
type RotatingWriter struct {
	mu       sync.Mutex
	file     *os.File
	dir      string
	prefix   string
	maxBytes int64
	written  int64
}

// NewRotatingWriter creates a writer that rotates log files.
func NewRotatingWriter(dir, prefix string, maxBytes int64) (*RotatingWriter, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	rw := &RotatingWriter{dir: dir, prefix: prefix, maxBytes: maxBytes}
	if err := rw.rotate(); err != nil {
		return nil, err
	}
	return rw, nil
}

func (rw *RotatingWriter) Write(p []byte) (int, error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.written+int64(len(p)) > rw.maxBytes {
		if err := rw.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := rw.file.Write(p)
	rw.written += int64(n)
	return n, err
}

func (rw *RotatingWriter) rotate() error {
	if rw.file != nil {
		rw.file.Close()
	}
	name := fmt.Sprintf("%s_%s.log", rw.prefix, time.Now().Format("2006-01-02_15-04-05"))
	path := filepath.Join(rw.dir, name)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	rw.file = f
	rw.written = 0
	return nil
}

// Close closes the current log file.
func (rw *RotatingWriter) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	if rw.file != nil {
		return rw.file.Close()
	}
	return nil
}

// --- Global logger ---

var global = New(DefaultConfig())

// SetGlobal sets the global logger.
func SetGlobal(l *Logger) { global = l }

// G returns the global logger.
func G() *Logger { return global }

// Convenience functions on global logger
func Debug(msg string, args ...interface{}) { global.Debug(msg, args...) }
func Info(msg string, args ...interface{})  { global.Info(msg, args...) }
func Warn(msg string, args ...interface{})  { global.Warn(msg, args...) }
func Error(msg string, args ...interface{}) { global.Error(msg, args...) }
