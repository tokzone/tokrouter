package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tokzone/fluxcore"

	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/usage"
)

func testConfig(keys ...config.KeyConfig) *config.Config {
	return &config.Config{
		Server: config.ServerConfig{Port: 8765},
		Keys:   keys,
		Router: config.RouterConfig{Retry: config.RetryConfig{MaxRetries: 2}},
		Log:    config.LogConfig{Level: "info"},
	}
}

func singleKeyConfig(model string) *config.Config {
	return testConfig(config.KeyConfig{
		Name:    "test",
		BaseURLs: map[string]string{"openai": "https://api.example.com"},
		Format:  "openai",
		Secret:  "sk-test",
		Enabled: true,
		Models:  []config.ModelConfig{{Name: model, Priority: 0}},
	})
}

func TestNewRouter(t *testing.T) {
	cfg := singleKeyConfig("gpt-4")
	svc, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if svc == nil {
		t.Fatal("Router is nil")
	}
	svc.Close()
}

func TestResolveAlias(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		body      []byte
		aliases   map[string]string
		wantModel string
	}{
		{
			name:      "no alias match",
			model:     "gpt-4",
			body:      []byte(`{"model": "gpt-4"}`),
			aliases:   map[string]string{},
			wantModel: "gpt-4",
		},
		{
			name:      "alias resolved",
			model:     "4o",
			body:      []byte(`{"model": "4o"}`),
			aliases:   map[string]string{"4o": "gpt-4o"},
			wantModel: "gpt-4o",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, body := resolveAlias(tt.model, tt.body, tt.aliases)
			if model != tt.wantModel {
				t.Errorf("model = %q, want %q", model, tt.wantModel)
			}
			if len(tt.aliases) > 0 {
				var req map[string]interface{}
				if err := json.Unmarshal(body, &req); err == nil {
					if req["model"] != tt.wantModel {
						t.Errorf("body model = %v, want %v", req["model"], tt.wantModel)
					}
				}
			}
		})
	}
}

func TestRewriteModelInRequest(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		newModel string
		want     string
	}{
		{
			name:     "simple model change",
			input:    []byte(`{"model": "gpt-4", "messages": []}`),
			newModel: "gpt-4-turbo",
			want:     "gpt-4-turbo",
		},
		{
			name:     "with other fields preserved",
			input:    []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "hi"}], "stream": true}`),
			newModel: "gpt-3.5",
			want:     "gpt-3.5",
		},
		{
			name:     "invalid JSON returns original",
			input:    []byte(`{"model": "gpt-4"`),
			newModel: "new-model",
			want:     "gpt-4",
		},
		{
			name:     "empty input",
			input:    []byte(``),
			newModel: "new-model",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rewriteModelInRequest(tt.input, tt.newModel)

			var req map[string]interface{}
			if err := json.Unmarshal(result, &req); err != nil {
				if tt.name == "invalid JSON returns original" || tt.name == "empty input" {
					return
				}
				t.Fatalf("result is not valid JSON: %v", err)
			}

			gotModel, _ := req["model"].(string)
			if gotModel != tt.want {
				t.Errorf("model = %q, want %q", gotModel, tt.want)
			}
		})
	}
}

func TestProviderStatuses(t *testing.T) {
	cfg := testConfig(
		config.KeyConfig{
			Name:    "openai",
			BaseURLs: map[string]string{"openai": "https://api.openai.com"},
			Format:  "openai",
			Secret:  "key1",
			Enabled: true,
			Models: []config.ModelConfig{
				{Name: "gpt-4", Priority: 0},
				{Name: "gpt-3.5-turbo", Priority: 1},
			},
		},
		config.KeyConfig{
			Name:    "anthropic",
			BaseURLs: map[string]string{"anthropic": "https://api.anthropic.com"},
			Format:  "anthropic",
			Secret:  "key2",
			Enabled: true,
			Models:  []config.ModelConfig{{Name: "claude-3", Priority: 0}},
		},
	)

	svc, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer svc.Close()

	statuses := svc.ProviderStatuses()
	if len(statuses) != 2 {
		t.Fatalf("Statuses count = %d, want 2", len(statuses))
	}

	// Find openai status
	var openaiStatus *ProviderStatus
	for i := range statuses {
		if statuses[i].Name == "openai" {
			openaiStatus = &statuses[i]
			break
		}
	}
	if openaiStatus == nil {
		t.Fatal("openai status not found")
	}
	if len(openaiStatus.Models) != 2 {
		t.Errorf("OpenAI models count = %d, want 2", len(openaiStatus.Models))
	}
	if !openaiStatus.Healthy {
		t.Error("OpenAI should be healthy")
	}
}

func TestReloadFailurePreservesState(t *testing.T) {
	cfg := singleKeyConfig("gpt-4")
	svc, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer svc.Close()

	r := svc.(*router)

	invalidCfg := &config.Config{
		Keys: []config.KeyConfig{
			{Name: "test", Enabled: true},
		},
	}

	if err := r.Reload(invalidCfg); err == nil {
		t.Fatal("expected Reload to fail with invalid config")
	}

	r.mu.RLock()
	_, ok := r.ctx.oaTables["gpt-4"]
	r.mu.RUnlock()

	if !ok {
		t.Error("RouteTables should still contain gpt-4 after failed reload")
	}
}

func TestRecordStreamUsageNilInputs(t *testing.T) {
	cfg := singleKeyConfig("gpt-4")
	svc, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer svc.Close()

	svc.RecordStreamUsage(nil, "gpt-4", "https://api.example.com")
	svc.RecordStreamUsage(nil, "", "")
}

func TestReload(t *testing.T) {
	cfg := singleKeyConfig("gpt-4")
	svc, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer svc.Close()

	r := svc.(*router)

	newCfg := testConfig(config.KeyConfig{
		Name:    "test",
		BaseURLs: map[string]string{"openai": "https://api.example.com"},
		Format:  "openai",
		Secret:  "test-key",
		Enabled: true,
		Models:  []config.ModelConfig{{Name: "gpt-3.5-turbo", Priority: 10}},
	})

	if err := r.Reload(newCfg); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	r.mu.RLock()
	_, ok1 := r.ctx.oaTables["gpt-3.5-turbo"]
	_, ok2 := r.ctx.oaTables["gpt-4"]
	r.mu.RUnlock()

	if !ok1 {
		t.Error("Tables should contain gpt-3.5-turbo after reload")
	}
	if ok2 {
		t.Error("Tables should not contain gpt-4 after reload")
	}
}

func TestReloadPreservesCircuitBreaker(t *testing.T) {
	cfg := singleKeyConfig("gpt-4")
	svc, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer svc.Close()

	r := svc.(*router)

	// Trip the route's CB
	desc := fluxcore.RouteDesc{
		SvcEP:      r.svcEPs["test"],
		Model:      "gpt-4",
		Credential: "sk-test",
	}
	route := r.routeRepo.FindOrCreate(desc.IdentityKey(), func() *fluxcore.Route {
		return fluxcore.NewRoute(desc)
	})
	if route == nil {
		t.Fatal("route not found in repo")
	}
	route.MarkModelFailure()
	route.MarkModelFailure()
	route.MarkModelFailure()
	if route.IsAvailable() {
		t.Fatal("route should be tripped")
	}

	// Reload with same config
	newCfg := singleKeyConfig("gpt-4")
	if err := r.Reload(newCfg); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	// Verify route is still tripped (CB state survived reload)
	routeAfter := r.routeRepo.FindOrCreate(desc.IdentityKey(), func() *fluxcore.Route {
		return fluxcore.NewRoute(desc)
	})
	if routeAfter == nil {
		t.Fatal("route lost after reload")
	}
	if routeAfter.IsAvailable() {
		t.Error("circuit breaker state should have survived reload")
	}
}

func TestForwardOpenAI(t *testing.T) {
	cfg := singleKeyConfig("gpt-4")
	svc, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer svc.Close()

	req := []byte(`{"model": "unknown-model", "messages": []}`)
	_, _, err = svc.ForwardOpenAI(context.Background(), req, "unknown-model")
	if err == nil {
		t.Error("expected error for unknown model")
	}
}

func TestForwardAnthropic(t *testing.T) {
	cfg := testConfig(config.KeyConfig{
		Name:    "anthropic",
		BaseURLs: map[string]string{"anthropic": "https://api.anthropic.com"},
		Format:  "anthropic",
		Secret:  "sk-test",
		Enabled: true,
		Models:  []config.ModelConfig{{Name: "claude-3", Priority: 0}},
	})
	svc, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer svc.Close()

	req := []byte(`{"model": "unknown-model", "messages": []}`)
	_, _, err = svc.ForwardAnthropic(context.Background(), req, "unknown-model")
	if err == nil {
		t.Error("expected error for unknown model")
	}
}

func TestForwardStreamOpenAI(t *testing.T) {
	cfg := singleKeyConfig("gpt-4")
	svc, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer svc.Close()

	req := []byte(`{"model": "unknown-model", "messages": []}`)
	_, _, _, err = svc.ForwardStreamOpenAI(context.Background(), req, "unknown-model")
	if err == nil {
		t.Error("expected error for unknown model")
	}
}

func TestServerConfig(t *testing.T) {
	cfg := singleKeyConfig("gpt-4")
	svc, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer svc.Close()

	serverCfg := svc.ServerConfig()
	if serverCfg.Port != 8765 {
		t.Errorf("default port = %d, want 8765", serverCfg.Port)
	}

	fullCfg := testConfig(config.KeyConfig{
		Name:    "test",
		BaseURLs: map[string]string{"openai": "https://api.example.com"},
		Format:  "openai",
		Secret:  "test-key",
		Enabled: true,
		Models:  []config.ModelConfig{{Name: "gpt-4"}},
	})
	fullCfg.Server = config.ServerConfig{Port: 9000, Host: "0.0.0.0"}

	svc2, err := NewFromConfig(fullCfg)
	if err != nil {
		t.Fatalf("NewFromConfig failed: %v", err)
	}
	defer svc2.Close()

	cfg2 := svc2.ServerConfig()
	if cfg2.Port != 9000 {
		t.Errorf("port = %d, want 9000", cfg2.Port)
	}
	if cfg2.Host != "0.0.0.0" {
		t.Errorf("host = %s, want 0.0.0.0", cfg2.Host)
	}
}

func TestStats(t *testing.T) {
	cfg := singleKeyConfig("gpt-4")
	svc, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer svc.Close()

	_, err = svc.Stats(usage.QueryFilter{})
	if err != usage.ErrDisabled {
		t.Errorf("expected ErrDisabled, got %v", err)
	}
}

func TestClose(t *testing.T) {
	cfg := singleKeyConfig("gpt-4")
	svc, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer svc.Close()

	if err := svc.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestAliasMapping(t *testing.T) {
	cfg := testConfig(config.KeyConfig{
		Name:    "test",
		BaseURLs: map[string]string{"openai": "https://api.example.com"},
		Format:  "openai",
		Secret:  "test-key",
		Enabled: true,
		Models: []config.ModelConfig{
			{Name: "gpt-4-1106-preview", Alias: "gpt-4-turbo"},
		},
	})

	svc, err := NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig failed: %v", err)
	}
	defer svc.Close()

	req := []byte(`{"model": "gpt-4-turbo", "messages": []}`)
	model, body := resolveAlias("gpt-4-turbo", req, cfg.AliasMap())
	if model != "gpt-4-1106-preview" {
		t.Errorf("model = %s, want gpt-4-1106-preview", model)
	}

	var modified map[string]interface{}
	if err := json.Unmarshal(body, &modified); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if modified["model"] != "gpt-4-1106-preview" {
		t.Errorf("model = %v, want gpt-4-1106-preview", modified["model"])
	}
}

func TestNewFromConfig(t *testing.T) {
	cfg := singleKeyConfig("gpt-4")
	svc, err := NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig failed: %v", err)
	}
	defer svc.Close()

	// Verify internal state
	r := svc.(*router)
	if r.oaRouter == nil {
		t.Error("oaRouter should not be nil")
	}
	if r.anthRouter == nil {
		t.Error("anthRouter should not be nil")
	}
	if len(r.ctx.oaTables) != 1 {
		t.Errorf("oaTables count = %d, want 1", len(r.ctx.oaTables))
	}
	if _, ok := r.ctx.oaTables[fluxcore.Model("gpt-4")]; !ok {
		t.Error("oaTables should contain gpt-4")
	}
}

func TestForwardOpenAISuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"chat-1","model":"gpt-4","choices":[{"index":0,"message":{"role":"assistant"},"finish_reason":"stop"}],"usage":{"input_tokens":10,"output_tokens":5}}`))
	}))
	defer ts.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8765},
		Keys: []config.KeyConfig{
			{
				Name:    "test",
				BaseURLs: map[string]string{"openai": ts.URL},
				Format:  "openai",
				Secret:  "sk-test",
				Enabled: true,
				Models:  []config.ModelConfig{{Name: "gpt-4", Priority: 0}},
			},
		},
		Router: config.RouterConfig{Retry: config.RetryConfig{MaxRetries: 1}},
		Log:    config.LogConfig{Level: "info"},
	}

	svc, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer svc.Close()

	req := []byte(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}]}`)
	resp, usage, err := svc.ForwardOpenAI(context.Background(), req, "gpt-4")
	if err != nil {
		t.Fatalf("ForwardOpenAI failed: %v", err)
	}
	if len(resp) == 0 {
		t.Error("expected non-empty response body")
	}
	if usage == nil {
		t.Fatal("expected non-nil usage")
	}
	if usage.InputTokens != 10 {
		t.Errorf("input tokens = %d, want 10", usage.InputTokens)
	}
}

func TestForwardStreamOpenAISuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer ts.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8765},
		Keys: []config.KeyConfig{
			{
				Name:    "test",
				BaseURLs: map[string]string{"openai": ts.URL},
				Format:  "openai",
				Secret:  "sk-test",
				Enabled: true,
				Models:  []config.ModelConfig{{Name: "gpt-4", Priority: 0}},
			},
		},
		Router: config.RouterConfig{Retry: config.RetryConfig{MaxRetries: 1}},
		Log:    config.LogConfig{Level: "info"},
	}

	svc, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer svc.Close()

	req := []byte(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}],"stream":true}`)
	result, _, _, err := svc.ForwardStreamOpenAI(context.Background(), req, "gpt-4")
	if err != nil {
		t.Fatalf("ForwardStreamOpenAI failed: %v", err)
	}
	defer result.Close()

	// Drain the channel
	for range result.Ch {
	}
}

