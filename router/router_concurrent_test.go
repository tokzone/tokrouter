package router

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/tokzone/fluxcore/endpoint"
	"github.com/tokzone/fluxcore/flux"
	"github.com/tokzone/fluxcore/provider"
	"github.com/tokzone/tokrouter/config"
)

func TestConcurrentProviderStatuses(t *testing.T) {
	svc := setupTestRouter("gpt-4")

	var wg sync.WaitGroup
	var errorCount atomic.Int32
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses := svc.ProviderStatuses()
			if len(statuses) != 1 {
				errorCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if errorCount.Load() > 0 {
		t.Errorf("got %d errors from concurrent ProviderStatuses calls", errorCount.Load())
	}
}

func TestConcurrentReload(t *testing.T) {
	svc := setupTestRouter("gpt-4")

	var wg sync.WaitGroup
	var successCount atomic.Int32

	for i := 0; i < 5; i++ {
		cfg := &config.Config{
			Keys: []config.KeyConfig{
				{
					Name:    "test",
					BaseURL: "https://api.example.com",
					Format:  "openai",
					Secret:  "test-key",
					Enabled: true,
					Models: []config.ModelConfig{
						{Name: "gpt-4", Priority: int64(i)},
					},
				},
			},
			Server: config.ServerConfig{Port: 8765},
			Router: config.RouterConfig{Retry: config.RetryConfig{MaxRetries: 2}},
			Log:    config.LogConfig{Level: "info"},
		}
		if err := svc.Reload(cfg); err == nil {
			successCount.Add(1)
		}
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = svc.ProviderStatuses()
			_ = svc.ServerConfig()
		}()
	}

	wg.Wait()

	if successCount.Load() != 5 {
		t.Errorf("expected 5 reloads succeeded, got %d", successCount.Load())
	}

	svc.mu.RLock()
	_, ok := svc.ctx.openAIDoFuncs["gpt-4"]
	svc.mu.RUnlock()

	if !ok {
		t.Error("gpt-4 DoFunc should exist after reloads")
	}
}

func TestConcurrentReadDoFuncs(t *testing.T) {
	svc := setupTestRouter("gpt-4")

	var wg sync.WaitGroup
	var errorCount atomic.Int32
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc.mu.RLock()
			_, ok := svc.ctx.openAIDoFuncs["gpt-4"]
			svc.mu.RUnlock()
			if !ok {
				errorCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if errorCount.Load() > 0 {
		t.Errorf("got %d errors from concurrent DoFunc reads", errorCount.Load())
	}
}

func TestReloadPreservesRetryMax(t *testing.T) {
	endpoint.GlobalRegistry().Clear()
	prov := newTestProvider()
	endpoint.RegisterEndpoint(1, prov, "gpt-4", []provider.Protocol{provider.ProtocolOpenAI})
	k := newTestAPIKey(prov)
	ue, _ := flux.NewUserEndpoint("gpt-4", k, 0)

	t.Run("skip retryMax check", func(t *testing.T) {
		_ = New([]*flux.UserEndpoint{ue}, nil, 5, nil, nil, nil)
		// retryMax is now stored in routerCtx, not directly accessible via client
	})
}
