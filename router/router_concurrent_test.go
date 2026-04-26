package router

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/tokzone/fluxcore/endpoint"
	"github.com/tokzone/fluxcore/flux"
	"github.com/tokzone/tokrouter/config"
)

func TestConcurrentProviderStatuses(t *testing.T) {
	svc := setupTestService("gpt-4")

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
	svc := setupTestService("gpt-4")

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
	client, ok := svc.state.clients["gpt-4"]
	svc.mu.RUnlock()

	if !ok || client == nil {
		t.Error("gpt-4 client should exist after reloads")
	}
}

func TestConcurrentGetClient(t *testing.T) {
	svc := setupTestService("gpt-4")

	var wg sync.WaitGroup
	var errorCount atomic.Int32
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client, ok := svc.getClient("gpt-4")
			if !ok || client == nil {
				errorCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if errorCount.Load() > 0 {
		t.Errorf("got %d errors from concurrent getClient calls", errorCount.Load())
	}
}

func TestReloadPreservesRetryMax(t *testing.T) {
	endpoint.GlobalRegistry().Clear()
	prov := newTestProvider()
	endpoint.RegisterEndpoint(1, prov, "gpt-4")
	k := newTestAPIKey(prov)
	ue, _ := flux.NewUserEndpoint("gpt-4", k, 0)
	svc := NewService([]*flux.UserEndpoint{ue}, nil, 5, nil)

	cfg := &config.Config{
		Keys: []config.KeyConfig{
			{
				Name:    "test",
				BaseURL: "https://api.example.com",
				Format:  "openai",
				Secret:  "test-key",
				Enabled: true,
				Models: []config.ModelConfig{
					{Name: "gpt-3.5-turbo"},
				},
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
	client := svc.state.clients["gpt-3.5-turbo"]
	svc.mu.RUnlock()

	if client == nil {
		t.Fatal("client not found after reload")
	}
	if client.RetryMax() != 5 {
		t.Errorf("expected retryMax 5, got %d", client.RetryMax())
	}
}
