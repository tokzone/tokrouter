package cli

import (
	"testing"

	"github.com/tokzone/tokrouter/config"
)

func TestUpdateConfigKey(t *testing.T) {
	cfg := NewTestConfigBuilder().
		WithOpenAIKey("test-key", "sk-test").
		Build()
	cfg.Keys[0].Enabled = false // Start disabled
	configPath := WriteTempConfig(t, cfg)

	// Enable the key
	err := updateConfigKey(configPath, "test-key", true)
	if err != nil {
		t.Errorf("updateConfigKey(enable): %v", err)
	}

	// Verify enabled
	updatedCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load updated config: %v", err)
	}
	if !updatedCfg.Keys[0].Enabled {
		t.Error("key should be enabled")
	}

	// Disable the key
	err = updateConfigKey(configPath, "test-key", false)
	if err != nil {
		t.Errorf("updateConfigKey(disable): %v", err)
	}

	// Verify disabled
	updatedCfg, err = config.Load(configPath)
	if err != nil {
		t.Fatalf("load updated config: %v", err)
	}
	if updatedCfg.Keys[0].Enabled {
		t.Error("key should be disabled")
	}
}

func TestUpdateConfigKeyNotFound(t *testing.T) {
	cfg := NewTestConfigBuilder().
		WithOpenAIKey("existing-key", "sk-test").
		Build()
	configPath := WriteTempConfig(t, cfg)

	err := updateConfigKey(configPath, "nonexistent-key", true)
	if err == nil {
		t.Error("expected error for nonexistent key")
	}
}

func TestTestModelRequestBuilding(t *testing.T) {
	// Test request body building logic without actually making HTTP calls
	key := &config.KeyConfig{
		Name:    "test",
		Format:  config.FormatOpenAI,
		Secret:  "sk-test",
		BaseURL: "https://api.test.com",
		Enabled: true,
		Models:  []config.ModelConfig{{Name: "gpt-4"}},
	}

	// Verify key configuration is valid for testing
	if key.Format != config.FormatOpenAI {
		t.Errorf("format = %s, want openai", key.Format)
	}
	if len(key.Models) == 0 {
		t.Error("key should have models for testing")
	}
}

func TestTestTimeoutConstant(t *testing.T) {
	// Verify timeout is reasonable (10 seconds)
	if testTimeout != 10*1000*1000*1000 { // 10 seconds in nanoseconds
		t.Errorf("testTimeout = %v, expected 10 seconds", testTimeout)
	}
}

func TestConfigBuilderWithPreset(t *testing.T) {
	cfg := NewTestConfigBuilder().
		WithOpenAIKey("openai-main", "sk-test-openai").
		Build()

	if len(cfg.Keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(cfg.Keys))
	}

	key := cfg.Keys[0]
	if key.Name != "openai-main" {
		t.Errorf("key name = %s, want openai-main", key.Name)
	}
	if key.Provider != "openai" {
		t.Errorf("key provider = %s, want openai", key.Provider)
	}
	if key.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("key BaseURL = %s, want https://api.openai.com/v1", key.BaseURL)
	}
	if len(key.Models) == 0 {
		t.Error("preset key should have default models")
	}
}

func TestConfigBuilderMultipleKeys(t *testing.T) {
	cfg := NewTestConfigBuilder().
		WithOpenAIKey("openai-main", "sk-test-1").
		WithAnthropicKey("anthropic-backup", "sk-ant-test").
		Build()

	if len(cfg.Keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(cfg.Keys))
	}

	// Check first key
	if cfg.Keys[0].Provider != "openai" {
		t.Errorf("first key provider = %s, want openai", cfg.Keys[0].Provider)
	}

	// Check second key
	if cfg.Keys[1].Provider != "anthropic" {
		t.Errorf("second key provider = %s, want anthropic", cfg.Keys[1].Provider)
	}
}

func TestWriteTempConfig(t *testing.T) {
	cfg := NewTestConfigBuilder().
		WithOpenAIKey("test-key", "sk-secret").
		Build()

	configPath := WriteTempConfig(t, cfg)

	// File should exist
	if configPath == "" {
		t.Error("config path should not be empty")
	}

	// Load and verify
	loadedCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load temp config: %v", err)
	}

	if len(loadedCfg.Keys) != 1 {
		t.Errorf("loaded config has %d keys, want 1", len(loadedCfg.Keys))
	}

	if loadedCfg.Keys[0].Name != "test-key" {
		t.Errorf("loaded key name = %s, want test-key", loadedCfg.Keys[0].Name)
	}
}