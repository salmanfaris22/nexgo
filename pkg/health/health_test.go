package health

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthyEndpoint(t *testing.T) {
	h := New("1.0.0")
	h.AddCheck("db", func() error { return nil })

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	h.Handler()(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != StatusUp {
		t.Errorf("status = %q, want %q", resp.Status, StatusUp)
	}
	if resp.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", resp.Version, "1.0.0")
	}
	if resp.Checks["db"] != "up" {
		t.Errorf("db check = %q, want %q", resp.Checks["db"], "up")
	}
}

func TestUnhealthyEndpoint(t *testing.T) {
	h := New("1.0.0")
	h.AddCheck("db", func() error { return errors.New("connection refused") })

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	h.Handler()(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != StatusDown {
		t.Errorf("status = %q, want %q", resp.Status, StatusDown)
	}
}

func TestReadyEndpointIncludesSystem(t *testing.T) {
	h := New("1.0.0")

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	h.ReadyHandler()(w, req)

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.System == nil {
		t.Fatal("system info should be present in ready endpoint")
	}
	if resp.System.NumCPU == 0 {
		t.Error("NumCPU should be > 0")
	}
	if resp.System.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}
}

func TestLiveEndpoint(t *testing.T) {
	h := New("1.0.0")

	req := httptest.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()
	h.LiveHandler()(w, req)

	if w.Code != 200 {
		t.Errorf("live status = %d, want 200", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("live status = %q, want %q", resp["status"], "ok")
	}
}
