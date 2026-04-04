package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	log := New(Config{Level: "warn", Output: &buf, JSON: true})

	log.Debug("debug msg")
	log.Info("info msg")
	log.Warn("warn msg")
	log.Error("error msg")

	output := buf.String()
	if strings.Contains(output, "debug msg") {
		t.Error("debug should be filtered at warn level")
	}
	if strings.Contains(output, "info msg") {
		t.Error("info should be filtered at warn level")
	}
	if !strings.Contains(output, "warn msg") {
		t.Error("warn should be logged")
	}
	if !strings.Contains(output, "error msg") {
		t.Error("error should be logged")
	}
}

func TestJSONFormat(t *testing.T) {
	var buf bytes.Buffer
	log := New(Config{Level: "info", Output: &buf, JSON: true})

	log.Info("test message")

	var entry Entry
	if err := json.NewDecoder(&buf).Decode(&entry); err != nil {
		t.Fatal(err)
	}
	if entry.Level != "INFO" {
		t.Errorf("Level = %q, want INFO", entry.Level)
	}
	if entry.Message != "test message" {
		t.Errorf("Message = %q, want %q", entry.Message, "test message")
	}
	if entry.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	log := New(Config{Level: "info", Output: &buf, JSON: true})

	log.With("user_id", "123").With("action", "login").Info("user action")

	var entry Entry
	json.NewDecoder(&buf).Decode(&entry)
	if entry.Fields["user_id"] != "123" {
		t.Errorf("user_id = %v, want 123", entry.Fields["user_id"])
	}
	if entry.Fields["action"] != "login" {
		t.Errorf("action = %v, want login", entry.Fields["action"])
	}
}

func TestWithFieldsImmutable(t *testing.T) {
	var buf bytes.Buffer
	log := New(Config{Level: "info", Output: &buf, JSON: true})

	log1 := log.With("key", "val1")
	log2 := log.With("key", "val2")

	buf.Reset()
	log1.Info("msg1")
	var e1 Entry
	json.NewDecoder(&buf).Decode(&e1)

	buf.Reset()
	log2.Info("msg2")
	var e2 Entry
	json.NewDecoder(&buf).Decode(&e2)

	if e1.Fields["key"] != "val1" {
		t.Errorf("log1 key = %v, want val1", e1.Fields["key"])
	}
	if e2.Fields["key"] != "val2" {
		t.Errorf("log2 key = %v, want val2", e2.Fields["key"])
	}
}

func TestParseLevel(t *testing.T) {
	tests := map[string]Level{
		"debug":   LevelDebug,
		"DEBUG":   LevelDebug,
		"info":    LevelInfo,
		"warn":    LevelWarn,
		"WARNING": LevelWarn,
		"error":   LevelError,
		"fatal":   LevelFatal,
		"unknown": LevelInfo,
	}
	for input, want := range tests {
		if got := ParseLevel(input); got != want {
			t.Errorf("ParseLevel(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestTextFormat(t *testing.T) {
	var buf bytes.Buffer
	log := New(Config{Level: "info", Output: &buf, Colorize: false})

	log.Info("hello world")
	if !strings.Contains(buf.String(), "hello world") {
		t.Errorf("output should contain message: %q", buf.String())
	}
}

func TestFormatArgs(t *testing.T) {
	var buf bytes.Buffer
	log := New(Config{Level: "info", Output: &buf, JSON: true})

	log.Info("user %s logged in from %s", "john", "127.0.0.1")

	var entry Entry
	json.NewDecoder(&buf).Decode(&entry)
	if entry.Message != "user john logged in from 127.0.0.1" {
		t.Errorf("Message = %q", entry.Message)
	}
}
