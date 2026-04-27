package router

import (
	"context"

	"github.com/tokzone/fluxcore/flux"
	"github.com/tokzone/fluxcore/message"

	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/usage"
)

// MockRouter implements Router for testing.
type MockRouter struct {
	ForwardOpenAIFunc         func(ctx context.Context, body []byte, model string) ([]byte, *message.Usage, error)
	ForwardAnthropicFunc      func(ctx context.Context, body []byte, model string) ([]byte, *message.Usage, error)
	ForwardStreamOpenAIFunc   func(ctx context.Context, body []byte, model string) (*flux.StreamResult, string, string, error)
	ForwardStreamAnthropicFunc func(ctx context.Context, body []byte, model string) (*flux.StreamResult, string, string, error)
	RecordStreamUsageFunc     func(usage *message.Usage, model string, providerURL string)
	ProviderStatusesFunc      func() []ProviderStatus
	StatsFunc                 func(filter usage.QueryFilter) ([]usage.StatRow, error)
	ServerConfigFunc          func() config.ServerConfig
	ReloadFunc                func(cfg *config.Config) error
	CloseFunc                 func() error
}

var _ Router = (*MockRouter)(nil)

func (m *MockRouter) ForwardOpenAI(ctx context.Context, body []byte, model string) ([]byte, *message.Usage, error) {
	if m.ForwardOpenAIFunc != nil {
		return m.ForwardOpenAIFunc(ctx, body, model)
	}
	return []byte(`{"choices": [{"message": {"content": "mock response"}}]}`), &message.Usage{InputTokens: 10, OutputTokens: 5}, nil
}

func (m *MockRouter) ForwardAnthropic(ctx context.Context, body []byte, model string) ([]byte, *message.Usage, error) {
	if m.ForwardAnthropicFunc != nil {
		return m.ForwardAnthropicFunc(ctx, body, model)
	}
	return []byte(`{"content": [{"text": "mock response"}]}`), &message.Usage{InputTokens: 10, OutputTokens: 5}, nil
}

func (m *MockRouter) ForwardStreamOpenAI(ctx context.Context, body []byte, model string) (*flux.StreamResult, string, string, error) {
	if m.ForwardStreamOpenAIFunc != nil {
		return m.ForwardStreamOpenAIFunc(ctx, body, model)
	}
	ch := make(chan []byte, 1)
	ch <- []byte(`data: {"choices": [{"delta": {"content": "mock"}}]}\n\n`)
	close(ch)
	return &flux.StreamResult{
		Ch:    ch,
		Usage: func() *message.Usage { return &message.Usage{InputTokens: 10, OutputTokens: 5} },
		Error: func() error { return nil },
	}, model, "https://mock.example.com", nil
}

func (m *MockRouter) ForwardStreamAnthropic(ctx context.Context, body []byte, model string) (*flux.StreamResult, string, string, error) {
	if m.ForwardStreamAnthropicFunc != nil {
		return m.ForwardStreamAnthropicFunc(ctx, body, model)
	}
	ch := make(chan []byte, 1)
	ch <- []byte(`data: {"type": "content_block_delta"}\n\n`)
	close(ch)
	return &flux.StreamResult{
		Ch:    ch,
		Usage: func() *message.Usage { return &message.Usage{InputTokens: 10, OutputTokens: 5} },
		Error: func() error { return nil },
	}, model, "https://mock.example.com", nil
}

func (m *MockRouter) RecordStreamUsage(usage *message.Usage, model string, providerURL string) {
	if m.RecordStreamUsageFunc != nil {
		m.RecordStreamUsageFunc(usage, model, providerURL)
	}
}

func (m *MockRouter) ProviderStatuses() []ProviderStatus {
	if m.ProviderStatusesFunc != nil {
		return m.ProviderStatusesFunc()
	}
	return []ProviderStatus{
		{Name: "https://mock.example.com", Protocol: "openai", Healthy: true,
			Models: []ModelStatus{{Name: "mock-model", Healthy: true}}},
	}
}

func (m *MockRouter) Stats(filter usage.QueryFilter) ([]usage.StatRow, error) {
	if m.StatsFunc != nil {
		return m.StatsFunc(filter)
	}
	return nil, usage.ErrDisabled
}

func (m *MockRouter) ServerConfig() config.ServerConfig {
	if m.ServerConfigFunc != nil {
		return m.ServerConfigFunc()
	}
	return config.ServerConfig{Host: "127.0.0.1", Port: 8765}
}

func (m *MockRouter) Reload(cfg *config.Config) error {
	if m.ReloadFunc != nil {
		return m.ReloadFunc(cfg)
	}
	return nil
}

func (m *MockRouter) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
