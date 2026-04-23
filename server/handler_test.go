package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tokzone/tokrouter/config"
)

func TestHandleHealth(t *testing.T) {
	// Create a minimal router service for testing
	handler := HandleHealth(nil)
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want 200", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("status = %v, want ok", resp["status"])
	}
}

func TestWithTraceID(t *testing.T) {
	traceCfg := config.TraceConfig{
		Enabled: true,
		Header:  "x-request-id",
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := WithTraceID(handler, traceCfg)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrapped(w, req)

	// Response should have trace ID header
	respTraceID := w.Header().Get("x-request-id")
	if respTraceID == "" {
		t.Error("response missing trace ID header")
	}
}

func TestWithTraceIDFromRequest(t *testing.T) {
	traceCfg := config.TraceConfig{
		Enabled: true,
		Header:  "x-request-id",
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := WithTraceID(handler, traceCfg)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("x-request-id", "existing-trace-123")
	w := httptest.NewRecorder()

	wrapped(w, req)

	// Should use existing trace ID
	respTraceID := w.Header().Get("x-request-id")
	if respTraceID != "existing-trace-123" {
		t.Errorf("traceID = %s, want existing-trace-123", respTraceID)
	}
}

func TestWriteErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()
	errResp := NewErrorResponse(errors.New("test error"))

	WriteErrorResponse(w, http.StatusInternalServerError, errResp)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status code = %d, want 500", w.Code)
	}
}
