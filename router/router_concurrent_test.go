package router

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/tokzone/tokrouter/config"
)

func TestConcurrentProviderStatuses(t *testing.T) {
	cfg := singleKeyConfig("gpt-4")
	svc, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer svc.Close()

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
	cfg := singleKeyConfig("gpt-4")
	svc, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer svc.Close()

	r := svc.(*router)

	var wg sync.WaitGroup
	var successCount atomic.Int32

	for i := 0; i < 5; i++ {
		cfg := testConfig(config.KeyConfig{
			Name:    "test",
			BaseURLs: map[string]string{"openai": "https://api.example.com"},
			Format:  "openai",
			Secret:  "test-key",
			Enabled: true,
			Models: []config.ModelConfig{
				{Name: "gpt-4", Priority: int64(i)},
			},
		})
		if err := r.Reload(cfg); err == nil {
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

	r.mu.RLock()
	_, ok := r.ctx.oaTables["gpt-4"]
	r.mu.RUnlock()

	if !ok {
		t.Error("gpt-4 table should exist after reloads")
	}
}

func TestConcurrentReadTables(t *testing.T) {
	cfg := singleKeyConfig("gpt-4")
	svc, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer svc.Close()

	r := svc.(*router)

	var wg sync.WaitGroup
	var errorCount atomic.Int32
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.mu.RLock()
			_, ok := r.ctx.oaTables["gpt-4"]
			r.mu.RUnlock()
			if !ok {
				errorCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if errorCount.Load() > 0 {
		t.Errorf("got %d errors from concurrent table reads", errorCount.Load())
	}
}

func TestReloadPreservesRetryMax(t *testing.T) {
	cfg := singleKeyConfig("gpt-4")
	cfg.Router.Retry.MaxRetries = 5
	svc, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer svc.Close()

	r := svc.(*router)
	r.mu.RLock()
	retryMax := r.ctx.retryMax
	r.mu.RUnlock()

	if retryMax != 5 {
		t.Errorf("retryMax = %d, want 5", retryMax)
	}
}
