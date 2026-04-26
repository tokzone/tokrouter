package router

import (
	"context"

	"github.com/tokzone/fluxcore/flux"
	"github.com/tokzone/fluxcore/message"
	"github.com/tokzone/fluxcore/provider"

	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/usage"
)

// MockRouterService implements RouterService for testing.
type MockRouterService struct {
	ForwardFunc         func(ctx context.Context, rawReq []byte, model string, clientFormat provider.Protocol) ([]byte, *message.Usage, error)
	ForwardStreamFunc   func(ctx context.Context, rawReq []byte, model string, clientFormat provider.Protocol) (*flux.StreamResult, string, string, error)
	RecordStreamUsageFunc func(usage *message.Usage, model string, providerURL string)
	ProviderStatusesFunc func() []ProviderStatus
	StatsFunc           func(filter usage.QueryFilter) ([]usage.StatRow, error)
	ServerConfigFunc    func() config.ServerConfig
	ReloadFunc          func(cfg *config.Config) error
	CloseFunc           func() error
}

var _ RouterService = (*MockRouterService)(nil)

func (m *MockRouterService) Forward(ctx context.Context, rawReq []byte, model string, clientFormat provider.Protocol) ([]byte, *message.Usage, error) {
	if m.ForwardFunc != nil {
		return m.ForwardFunc(ctx, rawReq, model, clientFormat)
	}
	return []byte(`{"choices": [{"message": {"content": "mock response"}}]}`), &message.Usage{InputTokens: 10, OutputTokens: 5}, nil
}

func (m *MockRouterService) ForwardStream(ctx context.Context, rawReq []byte, model string, clientFormat provider.Protocol) (*flux.StreamResult, string, string, error) {
	if m.ForwardStreamFunc != nil {
		return m.ForwardStreamFunc(ctx, rawReq, model, clientFormat)
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

func (m *MockRouterService) RecordStreamUsage(usage *message.Usage, model string, providerURL string) {
	if m.RecordStreamUsageFunc != nil {
		m.RecordStreamUsageFunc(usage, model, providerURL)
	}
}

func (m *MockRouterService) ProviderStatuses() []ProviderStatus {
	if m.ProviderStatusesFunc != nil {
		return m.ProviderStatusesFunc()
	}
	return []ProviderStatus{
		{Name: "https://mock.example.com", Protocol: "openai", Healthy: true,
			Models: []ModelStatus{{Name: "mock-model", Healthy: true}}},
	}
}

func (m *MockRouterService) Stats(filter usage.QueryFilter) ([]usage.StatRow, error) {
	if m.StatsFunc != nil {
		return m.StatsFunc(filter)
	}
	return nil, usage.ErrDisabled
}

func (m *MockRouterService) ServerConfig() config.ServerConfig {
	if m.ServerConfigFunc != nil {
		return m.ServerConfigFunc()
	}
	return config.ServerConfig{Host: "127.0.0.1", Port: 8765}
}

func (m *MockRouterService) Reload(cfg *config.Config) error {
	if m.ReloadFunc != nil {
		return m.ReloadFunc(cfg)
	}
	return nil
}

func (m *MockRouterService) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}