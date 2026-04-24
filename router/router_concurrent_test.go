package router

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/tokzone/tokrouter/config"

	"github.com/tokzone/fluxcore/routing"
)

func TestConcurrentProviderStatuses(t *testing.T) {
	key := &routing.Key{
		BaseURL:  "https://api.example.com",
		APIKey:   "test-key",
		Protocol: routing.ProtocolOpenAI,
	}
	ep, _ := routing.NewEndpoint(1, key, "gpt-4", 0)
	endpoints := []*routing.Endpoint{ep}

	svc := NewService(endpoints, nil, 2)

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
	key := &routing.Key{
		BaseURL:  "https://api.example.com",
		APIKey:   "test-key",
		Protocol: routing.ProtocolOpenAI,
	}
	ep, _ := routing.NewEndpoint(1, key, "gpt-4", 0)
	endpoints := []*routing.Endpoint{ep}

	svc := NewService(endpoints, nil, 2)

	var wg sync.WaitGroup
	var successCount atomic.Int32
	var failCount atomic.Int32

	// Concurrent reloads
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cfg := &config.Config{
				Keys: []config.KeyConfig{
					{
						Name:    "test",
						BaseURL: "https://api.example.com",
						Format:  "openai",
						Secret:  "test-key",
						Enabled: true,
						Models: []config.ModelConfig{
							{Name: "gpt-4", Priority: int64(idx)},
						},
					},
				},
				Server: config.ServerConfig{Port: 8765},
				Router: config.RouterConfig{Retry: config.RetryConfig{MaxRetries: 2}},
				Log:    config.LogConfig{Level: "info"},
			}
			if err := svc.Reload(cfg); err != nil {
				failCount.Add(1)
			} else {
				successCount.Add(1)
			}
		}(i)
	}

	// Concurrent reads during reload
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = svc.ProviderStatuses()
			_ = svc.ServerConfig()
		}()
	}

	wg.Wait()

	// Verify at least one reload succeeded
	if successCount.Load() == 0 {
		t.Error("no reload succeeded")
	}

	// Verify final state is valid
	svc.mu.RLock()
	pool, ok := svc.state.modelPools["gpt-4"]
	svc.mu.RUnlock()

	if !ok {
		t.Error("gpt-4 pool should exist after reloads")
	}
	if pool == nil {
		t.Error("pool should not be nil")
	}
}

func TestConcurrentGetPool(t *testing.T) {
	key := &routing.Key{
		BaseURL:  "https://api.example.com",
		APIKey:   "test-key",
		Protocol: routing.ProtocolOpenAI,
	}
	ep, _ := routing.NewEndpoint(1, key, "gpt-4", 0)
	endpoints := []*routing.Endpoint{ep}

	svc := NewService(endpoints, nil, 2)

	var wg sync.WaitGroup
	var errorCount atomic.Int32
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pool, ok := svc.getPool("gpt-4")
			if !ok {
				errorCount.Add(1)
				return
			}
			if pool == nil {
				errorCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if errorCount.Load() > 0 {
		t.Errorf("got %d errors from concurrent getPool calls", errorCount.Load())
	}
}

func TestReloadPreservesRetryMax(t *testing.T) {
	key := &routing.Key{
		BaseURL:  "https://api.example.com",
		APIKey:   "test-key",
		Protocol: routing.ProtocolOpenAI,
	}
	ep, _ := routing.NewEndpoint(1, key, "gpt-4", 0)
	endpoints := []*routing.Endpoint{ep}

	svc := NewService(endpoints, nil, 5) // retryMax = 5

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

	// Check retryMax preserved
	svc.mu.RLock()
	pool := svc.state.modelPools["gpt-3.5-turbo"]
	svc.mu.RUnlock()

	if pool == nil {
		t.Fatal("pool not found after reload")
	}
	if pool.RetryMax() != 5 {
		t.Errorf("expected retryMax 5, got %d", pool.RetryMax())
	}
}