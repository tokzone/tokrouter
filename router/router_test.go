package router

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/tokzone/fluxcore/endpoint"
	"github.com/tokzone/fluxcore/flux"
	"github.com/tokzone/fluxcore/provider"
	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/usage"
)

// Helper functions for test data construction
func newTestProvider() *provider.Provider {
	return provider.NewProvider(1, "https://api.example.com")
}

func newTestAPIKey(prov *provider.Provider) *flux.APIKey {
	k, _ := flux.NewAPIKey(prov, "test-key")
	return k
}

func newTestUserEndpoint(prov *provider.Provider, model string, priority int64) *flux.UserEndpoint {
	endpoint.RegisterEndpoint(1, prov, model, []provider.Protocol{provider.ProtocolOpenAI})
	k := newTestAPIKey(prov)
	ue, _ := flux.NewUserEndpoint(model, k, priority)
	return ue
}

func setupTestRouter(model string) *router {
	endpoint.GlobalRegistry().Clear()
	prov := newTestProvider()
	ue := newTestUserEndpoint(prov, model, 0)
	return New([]*flux.UserEndpoint{ue}, nil, 2, nil, nil, nil).(*router)
}

func newTestRouter(userEndpoints []*flux.UserEndpoint) *router {
	return New(userEndpoints, nil, 2, nil, nil, nil).(*router)
}

func TestNewRouter(t *testing.T) {
	endpoint.GlobalRegistry().Clear()
	prov := newTestProvider()
	ue := newTestUserEndpoint(prov, "gpt-4", 10000)
	userEndpoints := []*flux.UserEndpoint{ue}

	svc := New(userEndpoints, nil, 2, nil, nil, nil)
	if svc == nil {
		t.Fatal("Router is nil")
	}
}

func TestResolveAlias(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		body     []byte
		aliases  map[string]string
		wantModel string
	}{
		{
			name:     "no alias match",
			model:    "gpt-4",
			body:     []byte(`{"model": "gpt-4"}`),
			aliases:  map[string]string{},
			wantModel: "gpt-4",
		},
		{
			name:     "alias resolved",
			model:    "4o",
			body:     []byte(`{"model": "4o"}`),
			aliases:  map[string]string{"4o": "gpt-4o"},
			wantModel: "gpt-4o",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, body := resolveAlias(tt.model, tt.body, tt.aliases)
			if model != tt.wantModel {
				t.Errorf("model = %q, want %q", model, tt.wantModel)
			}
			if tt.aliases != nil && len(tt.aliases) > 0 {
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
			want:     "gpt-4", // original preserved
		},
		{
			name:     "empty input",
			input:    []byte(``),
			newModel: "new-model",
			want:     "", // original preserved
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rewriteModelInRequest(tt.input, tt.newModel)

			// Parse result to check model field
			var req map[string]interface{}
			if err := json.Unmarshal(result, &req); err != nil {
				// For invalid JSON cases, original should be preserved
				if tt.name == "invalid JSON returns original" || tt.name == "empty input" {
					return // Expected behavior
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
	// Clear registry
	endpoint.GlobalRegistry().Clear()

	prov1 := provider.NewProvider(1, "https://api.openai.com")
	prov2 := provider.NewProvider(2, "https://api.anthropic.com")

	ep1, _ := endpoint.NewEndpoint(1, prov1, "gpt-4", []provider.Protocol{provider.ProtocolOpenAI})
	ep2, _ := endpoint.NewEndpoint(2, prov1, "gpt-3.5-turbo", []provider.Protocol{provider.ProtocolOpenAI})
	ep3, _ := endpoint.NewEndpoint(3, prov2, "claude-3", []provider.Protocol{provider.ProtocolAnthropic})

	endpoint.GlobalRegistry().Register(ep1)
	endpoint.GlobalRegistry().Register(ep2)
	endpoint.GlobalRegistry().Register(ep3)

	k1, _ := flux.NewAPIKey(prov1, "key1")
	k2, _ := flux.NewAPIKey(prov2, "key2")

	ue1, _ := flux.NewUserEndpoint("gpt-4", k1, 10000)
	ue2, _ := flux.NewUserEndpoint("gpt-3.5-turbo", k1, 2000)
	ue3, _ := flux.NewUserEndpoint("claude-3", k2, 10000)
	userEndpoints := []*flux.UserEndpoint{ue1, ue2, ue3}

	svc := New(userEndpoints, nil, 2, nil, nil, nil)
	statuses := svc.ProviderStatuses()

	if len(statuses) != 2 {
		t.Errorf("Statuses count = %d, want 2", len(statuses))
	}

	// Find openai status
	var openaiStatus *ProviderStatus
	for i := range statuses {
		if statuses[i].Name == "https://api.openai.com" {
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
	endpoint.GlobalRegistry().Clear()
	prov := newTestProvider()
	ue := newTestUserEndpoint(prov, "gpt-4", 10000)
	userEndpoints := []*flux.UserEndpoint{ue}

	svc := newTestRouter(userEndpoints)

	invalidCfg := &config.Config{
		Keys: []config.KeyConfig{
			{Name: "test", Enabled: true},
		},
	}

	if err := svc.Reload(invalidCfg); err == nil {
		t.Fatal("expected Reload to fail with invalid config")
	}

	svc.mu.RLock()
	_, ok := svc.ctx.openAIDoFuncs["gpt-4"]
	svc.mu.RUnlock()

	if !ok {
		t.Error("DoFuncs should still contain gpt-4 after failed reload")
	}
}

func TestRecordStreamUsageNilInputs(t *testing.T) {
	svc := newTestRouter(nil)

	svc.RecordStreamUsage(nil, "gpt-4", "https://api.example.com")
	svc.RecordStreamUsage(nil, "", "")
}

func TestReload(t *testing.T) {
	endpoint.GlobalRegistry().Clear()
	prov := newTestProvider()
	ue := newTestUserEndpoint(prov, "gpt-4", 10000)
	userEndpoints := []*flux.UserEndpoint{ue}

	svc := newTestRouter(userEndpoints)

	cfg := &config.Config{
		Keys: []config.KeyConfig{
			{
				Name:    "test",
				BaseURL: "https://api.example.com",
				Format:  "openai",
				Secret:  "test-key",
				Enabled: true,
				Models:  []config.ModelConfig{{Name: "gpt-3.5-turbo", Priority: 10}},
			},
		},
		Server: config.ServerConfig{Port: 8765},
		Router: config.RouterConfig{Retry: config.RetryConfig{MaxRetries: 2}},
		Log:    config.LogConfig{Level: "info"},
	}

	if err := svc.Reload(cfg); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	svc.mu.RLock()
	_, ok1 := svc.ctx.openAIDoFuncs["gpt-3.5-turbo"]
	_, ok2 := svc.ctx.openAIDoFuncs["gpt-4"]
	svc.mu.RUnlock()

	if !ok1 {
		t.Error("DoFuncs should contain gpt-3.5-turbo after reload")
	}
	if ok2 {
		t.Error("DoFuncs should not contain gpt-4 after reload")
	}
}

func TestForwardOpenAI(t *testing.T) {
	endpoint.GlobalRegistry().Clear()
	prov := newTestProvider()
	ue := newTestUserEndpoint(prov, "gpt-4", 0)
	svc := newTestRouter([]*flux.UserEndpoint{ue})

	req := []byte(`{"model": "unknown-model", "messages": []}`)
	_, _, err := svc.ForwardOpenAI(context.Background(), req, "unknown-model")
	if err == nil {
		t.Error("expected error for unknown model")
	}
}

func TestForwardAnthropic(t *testing.T) {
	endpoint.GlobalRegistry().Clear()
	prov := provider.NewProvider(1, "https://api.example.com")
	endpoint.RegisterEndpoint(1, prov, "claude-3", []provider.Protocol{provider.ProtocolAnthropic})
	k, _ := flux.NewAPIKey(prov, "test-key")
	ue, _ := flux.NewUserEndpoint("claude-3", k, 0)
	svc := newTestRouter([]*flux.UserEndpoint{ue})

	req := []byte(`{"model": "unknown-model", "messages": []}`)
	_, _, err := svc.ForwardAnthropic(context.Background(), req, "unknown-model")
	if err == nil {
		t.Error("expected error for unknown model")
	}
}

func TestForwardStreamOpenAI(t *testing.T) {
	endpoint.GlobalRegistry().Clear()
	prov := newTestProvider()
	ue := newTestUserEndpoint(prov, "gpt-4", 0)
	svc := newTestRouter([]*flux.UserEndpoint{ue})

	req := []byte(`{"model": "unknown-model", "messages": []}`)
	_, _, _, err := svc.ForwardStreamOpenAI(context.Background(), req, "unknown-model")
	if err == nil {
		t.Error("expected error for unknown model")
	}
}

func TestServerConfig(t *testing.T) {
	endpoint.GlobalRegistry().Clear()
	prov := newTestProvider()
	ue := newTestUserEndpoint(prov, "gpt-4", 0)
	svc := newTestRouter([]*flux.UserEndpoint{ue})

	cfg := svc.ServerConfig()
	if cfg.Port != 8765 {
		t.Errorf("default port = %d, want 8765", cfg.Port)
	}

	// Router with config
	fullCfg := &config.Config{
		Server: config.ServerConfig{Port: 9000, Host: "0.0.0.0"},
		Keys: []config.KeyConfig{
			{Name: "test", BaseURL: "https://api.example.com", Format: "openai", Secret: "test-key", Enabled: true,
				Models: []config.ModelConfig{{Name: "gpt-4"}}},
		},
		Log: config.LogConfig{Level: "info"},
	}
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
	endpoint.GlobalRegistry().Clear()
	prov := newTestProvider()
	ue := newTestUserEndpoint(prov, "gpt-4", 0)
	svc := newTestRouter([]*flux.UserEndpoint{ue})

	_, err := svc.Stats(usage.QueryFilter{})
	if err != usage.ErrDisabled {
		t.Errorf("expected ErrDisabled, got %v", err)
	}
}

func TestClose(t *testing.T) {
	endpoint.GlobalRegistry().Clear()
	prov := newTestProvider()
	ue := newTestUserEndpoint(prov, "gpt-4", 0)
	svc := newTestRouter([]*flux.UserEndpoint{ue})

	if err := svc.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestAliasMapping(t *testing.T) {
	cfg := &config.Config{
		Keys: []config.KeyConfig{
			{
				Name:    "test",
				BaseURL: "https://api.example.com",
				Format:  "openai",
				Secret:  "test-key",
				Enabled: true,
				Models: []config.ModelConfig{
					{Name: "gpt-4-1106-preview", Alias: "gpt-4-turbo"},
				},
			},
		},
		Server: config.ServerConfig{Port: 8765},
		Log:    config.LogConfig{Level: "info"},
	}

	svc, err := NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig failed: %v", err)
	}
	defer svc.Close()

	// Test alias resolution
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
