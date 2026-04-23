package router

import (
	"testing"

	"github.com/tokzone/tokrouter/config"

	"github.com/tokzone/fluxcore/routing"
)

func TestNewService(t *testing.T) {
	key := &routing.Key{
		BaseURL:  "https://api.example.com",
		APIKey:   "test-key",
		Protocol: routing.ProtocolOpenAI,
	}
	ep, _ := routing.NewEndpoint(1, key, "gpt-4", 10000)
	endpoints := []*routing.Endpoint{ep}

	svc, err := NewService(endpoints, nil, 2)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	if svc == nil {
		t.Fatal("Service is nil")
	}
}

func TestGetProviderStatuses(t *testing.T) {
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

	svc, _ := NewService(endpoints, nil, 2)
	statuses := svc.GetProviderStatuses()

	// Should group by BaseURL (provider)
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

func TestCurrentEndpoint(t *testing.T) {
	key := &routing.Key{
		BaseURL:  "https://api.example.com",
		APIKey:   "test-key",
		Protocol: routing.ProtocolOpenAI,
	}
	ep, _ := routing.NewEndpoint(1, key, "gpt-4", 10000)
	endpoints := []*routing.Endpoint{ep}

	svc, _ := NewService(endpoints, nil, 2)
	ep = svc.CurrentEndpoint()

	if ep == nil {
		t.Fatal("CurrentEndpoint returned nil")
	}

	if ep.Model != "gpt-4" {
		t.Errorf("Model = %s, want gpt-4", ep.Model)
	}
}

func TestSelectEndpointForModel(t *testing.T) {
	key := &routing.Key{
		BaseURL:  "https://api.example.com",
		APIKey:   "test-key",
		Protocol: routing.ProtocolOpenAI,
	}
	ep1, _ := routing.NewEndpoint(1, key, "gpt-4", 10000)
	ep2, _ := routing.NewEndpoint(2, key, "gpt-3.5-turbo", 2000)
	endpoints := []*routing.Endpoint{ep1, ep2}

	svc, _ := NewService(endpoints, nil, 2)

	// Get model map
	svc.mu.RLock()
	modelMap := svc.modelMap
	svc.mu.RUnlock()

	// Test selecting for gpt-4
	selected := svc.selectEndpointForModel("gpt-4", modelMap)
	if selected == nil {
		t.Fatal("selectEndpointForModel returned nil for gpt-4")
	}
	if selected.Model != "gpt-4" {
		t.Errorf("Selected model = %s, want gpt-4", selected.Model)
	}

	// Test selecting for unknown model
	selected = svc.selectEndpointForModel("unknown", modelMap)
	if selected != nil {
		t.Error("selectEndpointForModel should return nil for unknown model")
	}
}

func TestSelectEndpointForModelPicksLowestPriority(t *testing.T) {
	key := &routing.Key{
		BaseURL:  "https://api.example.com",
		APIKey:   "test-key",
		Protocol: routing.ProtocolOpenAI,
	}
	// Two endpoints for same model, different priorities
	ep1, _ := routing.NewEndpoint(1, key, "gpt-4", 10000) // higher priority (higher price)
	ep2, _ := routing.NewEndpoint(2, key, "gpt-4", 5000)  // lower priority (lower price)
	endpoints := []*routing.Endpoint{ep1, ep2}

	svc, _ := NewService(endpoints, nil, 2)

	svc.mu.RLock()
	modelMap := svc.modelMap
	svc.mu.RUnlock()

	selected := svc.selectEndpointForModel("gpt-4", modelMap)
	if selected == nil {
		t.Fatal("selectEndpointForModel returned nil")
	}
	// Should select ep2 (lower priority = lower price)
	if selected.ID != 2 {
		t.Errorf("Selected endpoint ID = %d, want 2 (lowest priority)", selected.ID)
	}
}

func TestReload(t *testing.T) {
	key := &routing.Key{
		BaseURL:  "https://api.example.com",
		APIKey:   "test-key",
		Protocol: routing.ProtocolOpenAI,
	}
	ep1, _ := routing.NewEndpoint(1, key, "gpt-4", 10000)
	endpoints := []*routing.Endpoint{ep1}

	svc, _ := NewService(endpoints, nil, 2)

	// Create new config with different model
	cfg := &config.Config{
		Keys: []config.KeyConfig{
			{
				Name:    "test",
				BaseURL: "https://api.example.com",
				Format:  "openai",
				Secret:  "test-key",
				Enabled: true,
				Models: []config.ModelConfig{
					{Name: "gpt-3.5-turbo", Pricing: config.PricingConfig{Input: 0.001, Output: 0.002}},
				},
			},
		},
	}

	// Reload
	svc.Reload(cfg)

	// Check model map updated
	svc.mu.RLock()
	modelMap := svc.modelMap
	svc.mu.RUnlock()

	if _, ok := modelMap["gpt-3.5-turbo"]; !ok {
		t.Error("Model map should contain gpt-3.5-turbo after reload")
	}
	if _, ok := modelMap["gpt-4"]; ok {
		t.Error("Model map should not contain gpt-4 after reload")
	}
}
