package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	fluxerrors "github.com/tokzone/fluxcore/errors"
	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/router"
)

func setupTestRouter(t *testing.T) router.Router {
	t.Helper()
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 8765},
		Keys: []config.KeyConfig{
			{
				Name: "test", Format: "openai", Secret: "sk-test",
				BaseURLs: map[string]string{"openai": "https://api.example.com"},
				Enabled: true,
				Models:  []config.ModelConfig{{Name: "gpt-4", Priority: 0}},
			},
		},
		Router: config.RouterConfig{Retry: config.RetryConfig{MaxRetries: 2}},
		Log:    config.LogConfig{Level: "info", Format: "json", Output: "stdout"},
	}
	svc, err := router.New(cfg, nil)
	if err != nil {
		t.Fatalf("setupTestRouter: %v", err)
	}
	t.Cleanup(func() { svc.Close() })
	return svc
}

func TestHandleHealth(t *testing.T) {
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

func TestHandleHealthWithRouter(t *testing.T) {
	svc := setupTestRouter(t)

	handler := HandleHealth(svc)
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

func TestHandleHealthDegraded(t *testing.T) {
	// Use MockRouter to simulate degraded state
	mock := &router.MockRouter{
		ProviderStatusesFunc: func() []router.ProviderStatus {
			return []router.ProviderStatus{
				{Name: "test-provider", Protocol: "openai", Healthy: false,
					Models: []router.ModelStatus{{Name: "gpt-4", Healthy: false}}},
			}
		},
	}

	handler := HandleHealth(mock)
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	details := resp["details"].(map[string]interface{})
	endpoints := details["endpoints"].(map[string]interface{})
	if endpoints["healthy"].(float64) != 0 {
		t.Errorf("healthy count = %v, want 0", endpoints["healthy"])
	}
}

func TestHandleOpenAIInvalidJSON(t *testing.T) {
	svc := setupTestRouter(t)

	handler := HandleOpenAI(svc)
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader("invalid json"))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want 400", w.Code)
	}
}

func TestHandleOpenAIMissingModel(t *testing.T) {
	svc := setupTestRouter(t)

	handler := HandleOpenAI(svc)
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"messages": []}`))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Status code = %d, want 503", w.Code)
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

	respTraceID := w.Header().Get("x-request-id")
	if respTraceID != "existing-trace-123" {
		t.Errorf("traceID = %s, want existing-trace-123", respTraceID)
	}
}

func TestWriteErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()
	errResp := NewErrorResponseWithCode(fluxerrors.CodeServerError, "test error")

	WriteErrorResponse(w, http.StatusInternalServerError, errResp)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status code = %d, want 500", w.Code)
	}
}

func TestHandleStatus(t *testing.T) {
	handler := HandleStatus(nil)
	req := httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want 200", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", w.Header().Get("Content-Type"))
	}
}

func TestHandleStatusWithRouter(t *testing.T) {
	svc := setupTestRouter(t)

	handler := HandleStatus(svc)
	req := httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want 200", w.Code)
	}

	var statuses []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &statuses); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(statuses) != 1 {
		t.Errorf("statuses count = %d, want 1", len(statuses))
	}
}

func TestWriteErrorResponseWithCode(t *testing.T) {
	w := httptest.NewRecorder()
	errResp := NewErrorResponseWithCode(fluxerrors.CodeInvalidRequest, "test message")

	WriteErrorResponse(w, http.StatusBadRequest, errResp)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want 400", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if resp["type"] != "error" {
		t.Errorf("type = %v, want error", resp["type"])
	}

	if resp["code"] != "invalid_request" {
		t.Errorf("code = %v, want invalid_request", resp["code"])
	}
}

func TestNewServer(t *testing.T) {
	svc := setupTestRouter(t)

	traceCfg := config.TraceConfig{Enabled: true, Header: "x-request-id"}
	srv := NewServer(svc, traceCfg, "config.yaml")

	if srv == nil {
		t.Fatal("NewServer returned nil")
	}
	if srv.server == nil {
		t.Error("server is nil")
	}
}

func TestHandleOpenAIStream(t *testing.T) {
	mock := &router.MockRouter{}

	handler := HandleOpenAI(mock)
	req := httptest.NewRequest("POST", "/v1/chat/completions",
		strings.NewReader(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}],"stream":true}`))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want 200", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %s, want text/event-stream", w.Header().Get("Content-Type"))
	}
}

func TestHandleAnthropicStream(t *testing.T) {
	mock := &router.MockRouter{}

	handler := HandleAnthropic(mock)
	req := httptest.NewRequest("POST", "/v1/messages",
		strings.NewReader(`{"model":"claude-3","messages":[{"role":"user","content":"hi"}],"stream":true}`))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want 200", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %s, want text/event-stream", w.Header().Get("Content-Type"))
	}
}


func TestGenerateTraceID(t *testing.T) {
	id := generateTraceID()
	if id == "" {
		t.Error("generateTraceID returned empty string")
	}
	if !strings.Contains(id, "-") {
		t.Errorf("traceID format = %s, expected format pid-timestamp", id)
	}
}
