package router

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestWithContext(t *testing.T) {
	ctx := context.Background()
	params := map[string]string{"id": "123"}
	ctx = WithParams(ctx, params)

	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(ctx)

	val := Param(r, "id")
	if val != "123" {
		t.Errorf("expected 123, got %s", val)
	}
}

func TestParam_Missing(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	val := Param(r, "id")
	if val != "" {
		t.Errorf("expected empty string, got %s", val)
	}
}

func TestParams(t *testing.T) {
	ctx := context.Background()
	params := map[string]string{"id": "123", "name": "test"}
	ctx = WithParams(ctx, params)

	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(ctx)

	all := Params(r)
	if all["id"] != "123" {
		t.Errorf("expected id=123, got %s", all["id"])
	}
	if all["name"] != "test" {
		t.Errorf("expected name=test, got %s", all["name"])
	}
}

func TestParams_NoContext(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	all := Params(r)
	if len(all) != 0 {
		t.Errorf("expected empty map, got %v", all)
	}
}
