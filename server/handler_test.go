package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tokzone/fluxcore/endpoint"
	"github.com/tokzone/fluxcore/flux"
	"github.com/tokzone/fluxcore/provider"
	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/router"
)

func setupTestRouterService() *router.Service {
	endpoint.GlobalRegistry().Clear()
	prov := provider.NewProvider(1, "https://api.example.com", provider.ProtocolOpenAI)
	endpoint.RegisterEndpoint(1, prov, "gpt-4")
	k, _ := flux.NewAPIKey(prov, "test-key")
	ue, _ := flux.NewUserEndpoint("gpt-4", k, 0)
	return router.NewService([]*flux.UserEndpoint{ue}, nil, 2, nil)
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
	svc := setupTestRouterService()
	defer svc.Close()

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
	endpoint.GlobalRegistry().Clear()
	prov := provider.NewProvider(1, "https://api.example.com", provider.ProtocolOpenAI)
	ep := endpoint.RegisterEndpoint(1, prov, "gpt-4")
	ep.MarkEndpointFail()
	ep.MarkEndpointFail()
	ep.MarkEndpointFail()

	k, _ := flux.NewAPIKey(prov, "test-key")
	ue, _ := flux.NewUserEndpoint("gpt-4", k, 0)
	svc := router.NewService([]*flux.UserEndpoint{ue}, nil, 2, nil)
	defer svc.Close()

	handler := HandleHealth(svc)
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	// Should return degraded status
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

func TestHandleRequestInvalidJSON(t *testing.T) {
	svc := setupTestRouterService()
	defer svc.Close()

	handler := HandleRequest(svc, provider.ProtocolOpenAI)
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader("invalid json"))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want 400", w.Code)
	}
}

func TestHandleRequestMissingModel(t *testing.T) {
	svc := setupTestRouterService()
	defer svc.Close()

	handler := HandleRequest(svc, provider.ProtocolOpenAI)
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"messages": []}`))
	w := httptest.NewRecorder()

	handler(w, req)

	// Should attempt to route (will fail with unknown model error)
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
	errResp := NewErrorResponse(errors.New("test error"))

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
	svc := setupTestRouterService()
	defer svc.Close()

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
	errResp := NewErrorResponseWithCode("INVALID_REQUEST", "test message")

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

	if resp["code"] != "INVALID_REQUEST" {
		t.Errorf("code = %v, want INVALID_REQUEST", resp["code"])
	}
}

func TestNewServer(t *testing.T) {
	svc := setupTestRouterService()
	defer svc.Close()

	traceCfg := config.TraceConfig{Enabled: true, Header: "x-request-id"}
	srv := NewServer(svc, traceCfg, "config.yaml")

	if srv == nil {
		t.Fatal("NewServer returned nil")
	}
	if srv.server == nil {
		t.Error("server is nil")
	}
}

func TestGenerateTraceID(t *testing.T) {
	id := generateTraceID()
	if id == "" {
		t.Error("generateTraceID returned empty string")
	}
	// Should contain pid and timestamp separated by dash
	if !strings.Contains(id, "-") {
		t.Errorf("traceID format = %s, expected format pid-timestamp", id)
	}
}
