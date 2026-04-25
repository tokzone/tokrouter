package router

import (
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
	return provider.NewProvider(1, "https://api.example.com", provider.ProtocolOpenAI)
}

func newTestAPIKey(prov *provider.Provider) *flux.APIKey {
	k, _ := flux.NewAPIKey(prov, "test-key")
	return k
}

func newTestUserEndpoint(prov *provider.Provider, model string, priority int64) *flux.UserEndpoint {
	endpoint.RegisterEndpoint(1, prov, model)
	k := newTestAPIKey(prov)
	ue, _ := flux.NewUserEndpoint(model, k, priority)
	return ue
}

func setupTestService(model string) *Service {
	endpoint.GlobalRegistry().Clear()
	prov := newTestProvider()
	ue := newTestUserEndpoint(prov, model, 0)
	return NewService([]*flux.UserEndpoint{ue}, nil, 2)
}

func newTestService(userEndpoints []*flux.UserEndpoint) *Service {
	return NewService(userEndpoints, nil, 2)
}

func TestNewService(t *testing.T) {
	endpoint.GlobalRegistry().Clear()
	prov := newTestProvider()
	ue := newTestUserEndpoint(prov, "gpt-4", 10000)
	userEndpoints := []*flux.UserEndpoint{ue}

	svc := NewService(userEndpoints, nil, 2)
	if svc == nil {
		t.Fatal("Service is nil")
	}
}

func TestParseModelFromRequest(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "valid model",
			input: []byte(`{"model": "gpt-4", "messages": []}`),
			want:  "gpt-4",
		},
		{
			name:  "empty model",
			input: []byte(`{"model": "", "messages": []}`),
			want:  "",
		},
		{
			name:  "no model field",
			input: []byte(`{"messages": []}`),
			want:  "",
		},
		{
			name:  "invalid JSON",
			input: []byte(`{"model": "gpt-4"`),
			want:  "",
		},
		{
			name:  "empty JSON",
			input: []byte(``),
			want:  "",
		},
		{
			name:  "model with special chars",
			input: []byte(`{"model": "gpt-4-1106-preview", "messages": []}`),
			want:  "gpt-4-1106-preview",
		},
		{
			name:  "model with numbers",
			input: []byte(`{"model": "gpt-3.5-turbo", "messages": []}`),
			want:  "gpt-3.5-turbo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseModelFromRequest(tt.input)
			if got != tt.want {
				t.Errorf("parseModelFromRequest() = %q, want %q", got, tt.want)
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

	prov1 := provider.NewProvider(1, "https://api.openai.com", provider.ProtocolOpenAI)
	prov2 := provider.NewProvider(2, "https://api.anthropic.com", provider.ProtocolAnthropic)

	ep1, _ := endpoint.NewEndpoint(1, prov1, "gpt-4")
	ep2, _ := endpoint.NewEndpoint(2, prov1, "gpt-3.5-turbo")
	ep3, _ := endpoint.NewEndpoint(3, prov2, "claude-3")

	endpoint.GlobalRegistry().Register(ep1)
	endpoint.GlobalRegistry().Register(ep2)
	endpoint.GlobalRegistry().Register(ep3)

	k1, _ := flux.NewAPIKey(prov1, "key1")
	k2, _ := flux.NewAPIKey(prov2, "key2")

	ue1, _ := flux.NewUserEndpoint("gpt-4", k1, 10000)
	ue2, _ := flux.NewUserEndpoint("gpt-3.5-turbo", k1, 2000)
	ue3, _ := flux.NewUserEndpoint("claude-3", k2, 10000)
	userEndpoints := []*flux.UserEndpoint{ue1, ue2, ue3}

	svc := NewService(userEndpoints, nil, 2)
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

	svc := newTestService(userEndpoints)

	invalidCfg := &config.Config{
		Keys: []config.KeyConfig{
			{Name: "test", Enabled: true},
		},
	}

	if err := svc.Reload(invalidCfg); err == nil {
		t.Fatal("expected Reload to fail with invalid config")
	}

	svc.mu.RLock()
	_, ok := svc.state.clients["gpt-4"]
	svc.mu.RUnlock()

	if !ok {
		t.Error("Clients should still contain gpt-4 after failed reload")
	}
}

func TestRecordStreamUsageNilInputs(t *testing.T) {
	svc := newTestService(nil)

	svc.RecordStreamUsage(nil, "gpt-4", "https://api.example.com")
	svc.RecordStreamUsage(nil, "", "")
}

func TestReload(t *testing.T) {
	endpoint.GlobalRegistry().Clear()
	prov := newTestProvider()
	ue := newTestUserEndpoint(prov, "gpt-4", 10000)
	userEndpoints := []*flux.UserEndpoint{ue}

	svc := newTestService(userEndpoints)

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
	_, ok1 := svc.state.clients["gpt-3.5-turbo"]
	_, ok2 := svc.state.clients["gpt-4"]
	svc.mu.RUnlock()

	if !ok1 {
		t.Error("Clients should contain gpt-3.5-turbo after reload")
	}
	if ok2 {
		t.Error("Clients should not contain gpt-4 after reload")
	}
}

func TestForward(t *testing.T) {
	endpoint.GlobalRegistry().Clear()
	prov := newTestProvider()
	ue := newTestUserEndpoint(prov, "gpt-4", 0)
	svc := newTestService([]*flux.UserEndpoint{ue})

	req := []byte(`{"model": "unknown-model"}`)
	_, _, err := svc.Forward(nil, req, provider.ProtocolOpenAI)
	if err == nil {
		t.Error("expected error for unknown model")
	}
}

func TestForwardStream(t *testing.T) {
	endpoint.GlobalRegistry().Clear()
	prov := newTestProvider()
	ue := newTestUserEndpoint(prov, "gpt-4", 0)
	svc := newTestService([]*flux.UserEndpoint{ue})

	req := []byte(`{"model": "unknown-model"}`)
	_, _, _, err := svc.ForwardStream(nil, req, provider.ProtocolOpenAI)
	if err == nil {
		t.Error("expected error for unknown model")
	}
}

func TestServerConfig(t *testing.T) {
	endpoint.GlobalRegistry().Clear()
	prov := newTestProvider()
	ue := newTestUserEndpoint(prov, "gpt-4", 0)
	svc := newTestService([]*flux.UserEndpoint{ue})

	cfg := svc.ServerConfig()
	if cfg.Port != 8765 {
		t.Errorf("default port = %d, want 8765", cfg.Port)
	}

	// Service with config
	fullCfg := &config.Config{
		Server: config.ServerConfig{Port: 9000, Host: "0.0.0.0"},
		Keys: []config.KeyConfig{
			{Name: "test", BaseURL: "https://api.example.com", Format: "openai", Secret: "test-key", Enabled: true,
				Models: []config.ModelConfig{{Name: "gpt-4"}}},
		},
		Log: config.LogConfig{Level: "info"},
	}
	svc2, err := NewServiceFromConfig(fullCfg)
	if err != nil {
		t.Fatalf("NewServiceFromConfig failed: %v", err)
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
	svc := newTestService([]*flux.UserEndpoint{ue})

	_, err := svc.Stats(usage.QueryFilter{})
	if err != usage.ErrDisabled {
		t.Errorf("expected ErrDisabled, got %v", err)
	}
}

func TestClose(t *testing.T) {
	endpoint.GlobalRegistry().Clear()
	prov := newTestProvider()
	ue := newTestUserEndpoint(prov, "gpt-4", 0)
	svc := newTestService([]*flux.UserEndpoint{ue})

	if err := svc.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestAliasMapping(t *testing.T) {
	endpoint.GlobalRegistry().Clear()

	cfg := &config.Config{
		Keys: []config.KeyConfig{
			{
				Name:    "test",
				BaseURL: "https://api.example.com",
				Format:  "openai",
				Secret:  "test-key",
				Enabled: true,
				Models: []config.ModelConfig{
					// Endpoint created with actual model name "gpt-4-1106-preview"
					// User can request alias "gpt-4-turbo" -> rewritten to actual name
					{Name: "gpt-4-1106-preview", Alias: "gpt-4-turbo"},
				},
			},
		},
		Server: config.ServerConfig{Port: 8765},
		Log:    config.LogConfig{Level: "info"},
	}

	svc, err := NewServiceFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewServiceFromConfig failed: %v", err)
	}
	defer svc.Close()

	// Request for alias should be rewritten to actual model
	req := []byte(`{"model": "gpt-4-turbo"}`)
	client, model, providerURL, modifiedReq, err := svc.prepareRequestWithDetails(req)
	if err != nil {
		t.Fatalf("prepareRequestWithDetails failed: %v", err)
	}
	if client == nil {
		t.Error("client is nil")
	}
	if model != "gpt-4-1106-preview" {
		t.Errorf("model = %s, want gpt-4-1106-preview", model)
	}
	if providerURL != "https://api.example.com" {
		t.Errorf("providerURL = %s, want https://api.example.com", providerURL)
	}

	// Check model was rewritten from alias to actual name
	var modified map[string]interface{}
	if err := json.Unmarshal(modifiedReq, &modified); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if modified["model"] != "gpt-4-1106-preview" {
		t.Errorf("model = %v, want gpt-4-1106-preview", modified["model"])
	}
}