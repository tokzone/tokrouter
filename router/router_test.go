package router

import (
	"testing"

	"github.com/tokzone/fluxcore/routing"
)

func TestNewService(t *testing.T) {
	key := &routing.Key{
		BaseURL:  "https://api.example.com",
		APIKey:   "test-key",
		Protocol: routing.ProtocolOpenAI,
	}
	endpoints := []*routing.Endpoint{
		routing.NewEndpoint(1, key, "gpt-4", 0.01, 0.03),
	}

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

	endpoints := []*routing.Endpoint{
		routing.NewEndpoint(1, key1, "gpt-4", 0.01, 0.03),
		routing.NewEndpoint(2, key1, "gpt-3.5-turbo", 0.001, 0.002),
		routing.NewEndpoint(3, key2, "claude-3", 0.01, 0.03),
	}

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
	endpoints := []*routing.Endpoint{
		routing.NewEndpoint(1, key, "gpt-4", 0.01, 0.03),
	}

	svc, _ := NewService(endpoints, nil, 2)
	ep := svc.CurrentEndpoint()

	if ep == nil {
		t.Fatal("CurrentEndpoint returned nil")
	}

	if ep.Model != "gpt-4" {
		t.Errorf("Model = %s, want gpt-4", ep.Model)
	}
}
