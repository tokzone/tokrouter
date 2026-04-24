package router

import (
	"encoding/json"
	"testing"

	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/usage"

	"github.com/tokzone/fluxcore/message"
	"github.com/tokzone/fluxcore/routing"
)

// Helper functions for test data construction
func newTestKey() *routing.Key {
	return &routing.Key{
		BaseURL:  "https://api.example.com",
		APIKey:   "test-key",
		Protocol: routing.ProtocolOpenAI,
	}
}

func newTestEndpoint(key *routing.Key, model string, priority int64) *routing.Endpoint {
	ep, _ := routing.NewEndpoint(1, key, model, priority)
	return ep
}

func newTestService(endpoints []*routing.Endpoint) *Service {
	return NewService(endpoints, nil, 2)
}

func TestNewService(t *testing.T) {
	key := newTestKey()
	ep := newTestEndpoint(key, "gpt-4", 10000)
	endpoints := []*routing.Endpoint{ep}

	svc := NewService(endpoints, nil, 2)
	if svc == nil {
		t.Fatal("Service is nil")
	}
}

func TestParseModelFromRequest(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    string
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
	key1 := &routing.Key{
		BaseURL:  "https://api.openai.com",
		APIKey:   "key1",
		Protocol: routing.ProtocolOpenAI,
	}
	key2 := &routing.Key{
		BaseURL:  "https://api.anthropic.com",
		APIKey:   "key2",
		Protocol: routing.ProtocolAnthropic,
	}

	ep1, _ := routing.NewEndpoint(1, key1, "gpt-4", 10000)
	ep2, _ := routing.NewEndpoint(2, key1, "gpt-3.5-turbo", 2000)
	ep3, _ := routing.NewEndpoint(3, key2, "claude-3", 10000)
	endpoints := []*routing.Endpoint{ep1, ep2, ep3}

	svc := NewService(endpoints, nil, 2)
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

func TestModelPoolSelectsLowestPriority(t *testing.T) {
	key := newTestKey()
	ep1, _ := routing.NewEndpoint(1, key, "gpt-4", 100) // higher priority
	ep2, _ := routing.NewEndpoint(2, key, "gpt-4", 10)  // lower priority (preferred)
	endpoints := []*routing.Endpoint{ep1, ep2}

	svc := newTestService(endpoints)

	svc.mu.RLock()
	pool, ok := svc.state.modelPools["gpt-4"]
	svc.mu.RUnlock()

	if !ok {
		t.Fatal("model pool not found for gpt-4")
	}

	selected := pool.SelectBest()
	if selected.ID != 2 {
		t.Errorf("Selected endpoint ID = %d, want 2 (lowest priority)", selected.ID)
	}
}

func TestModelPoolSelectsLowerLatencyWhenSamePriority(t *testing.T) {
	key := newTestKey()
	ep1, _ := routing.NewEndpoint(1, key, "gpt-4", 100)
	ep2, _ := routing.NewEndpoint(2, key, "gpt-4", 100)
	endpoints := []*routing.Endpoint{ep1, ep2}

	ep1.UpdateLatency(100) // 100ms
	ep2.UpdateLatency(50)  // 50ms

	svc := newTestService(endpoints)

	svc.mu.RLock()
	pool, ok := svc.state.modelPools["gpt-4"]
	svc.mu.RUnlock()

	if !ok {
		t.Fatal("model pool not found for gpt-4")
	}

	selected := pool.SelectBest()
	if selected.ID != 2 {
		t.Errorf("Selected endpoint ID = %d, want 2 (lowest latency)", selected.ID)
	}
}

func TestReloadFailurePreservesState(t *testing.T) {
	key := newTestKey()
	ep1, _ := routing.NewEndpoint(1, key, "gpt-4", 10000)
	endpoints := []*routing.Endpoint{ep1}

	svc := newTestService(endpoints)

	invalidCfg := &config.Config{
		Keys: []config.KeyConfig{
			{Name: "test", Enabled: true},
		},
	}

	if err := svc.Reload(invalidCfg); err == nil {
		t.Fatal("expected Reload to fail with invalid config")
	}

	svc.mu.RLock()
	_, ok := svc.state.modelPools["gpt-4"]
	svc.mu.RUnlock()

	if !ok {
		t.Error("Model pools should still contain gpt-4 after failed reload")
	}
}

func TestRecordStreamUsageNilInputs(t *testing.T) {
	key := newTestKey()
	ep := newTestEndpoint(key, "gpt-4", 0)
	endpoints := []*routing.Endpoint{ep}

	svc := newTestService(endpoints)

	svc.RecordStreamUsage(nil, nil)
	svc.RecordStreamUsage(&message.Usage{}, nil)

	svc.mu.RLock()
	pool := svc.state.modelPools["gpt-4"]
	svc.mu.RUnlock()

	svc.RecordStreamUsage(nil, pool)
}

func TestReload(t *testing.T) {
	key := newTestKey()
	ep1, _ := routing.NewEndpoint(1, key, "gpt-4", 10000)
	endpoints := []*routing.Endpoint{ep1}

	svc := newTestService(endpoints)

	cfg := &config.Config{
		Keys: []config.KeyConfig{
			{
				Name:    "test",
				BaseURL: "https://api.example.com",
				Format:  "openai",
				Secret:  "test-key",
				Enabled: true,
				Models: []config.ModelConfig{{Name: "gpt-3.5-turbo", Priority: 10}},
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
	_, ok1 := svc.state.modelPools["gpt-3.5-turbo"]
	_, ok2 := svc.state.modelPools["gpt-4"]
	svc.mu.RUnlock()

	if !ok1 {
		t.Error("Model pools should contain gpt-3.5-turbo after reload")
	}
	if ok2 {
		t.Error("Model pools should not contain gpt-4 after reload")
	}
}

func TestForward(t *testing.T) {
	key := newTestKey()
	ep := newTestEndpoint(key, "gpt-4", 0)
	svc := newTestService([]*routing.Endpoint{ep})

	req := []byte(`{"model": "unknown-model"}`)
	_, _, err := svc.Forward(nil, req, routing.ProtocolOpenAI)
	if err == nil {
		t.Error("expected error for unknown model")
	}
}

func TestForwardStream(t *testing.T) {
	key := newTestKey()
	ep := newTestEndpoint(key, "gpt-4", 0)
	svc := newTestService([]*routing.Endpoint{ep})

	req := []byte(`{"model": "unknown-model"}`)
	_, _, err := svc.ForwardStream(nil, req, routing.ProtocolOpenAI)
	if err == nil {
		t.Error("expected error for unknown model")
	}
}

func TestServerConfig(t *testing.T) {
	key := newTestKey()
	ep := newTestEndpoint(key, "gpt-4", 0)
	svc := newTestService([]*routing.Endpoint{ep})

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
	key := newTestKey()
	ep := newTestEndpoint(key, "gpt-4", 0)
	svc := newTestService([]*routing.Endpoint{ep})

	_, err := svc.Stats(usage.QueryFilter{})
	if err != usage.ErrDisabled {
		t.Errorf("expected ErrDisabled, got %v", err)
	}
}

func TestClose(t *testing.T) {
	key := newTestKey()
	ep := newTestEndpoint(key, "gpt-4", 0)
	svc := newTestService([]*routing.Endpoint{ep})

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
	pool, _, modifiedReq, err := svc.prepareRequest(req)
	if err != nil {
		t.Fatalf("prepareRequest failed: %v", err)
	}
	if pool == nil {
		t.Error("pool is nil")
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