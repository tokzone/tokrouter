package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `
server:
  host: "127.0.0.1"
  port: 8765

keys:
  - name: test-key
    base_url: "https://api.example.com/v1"
    format: openai
    secret: "test-secret"
    enabled: true
    models:
      - name: "gpt-4"
        pricing:
          input: 0.03
          output: 0.06

stats:
  enabled: false

log:
  level: info
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Server.Port != 8765 {
		t.Errorf("Port = %d, want 8765", cfg.Server.Port)
	}

	if len(cfg.Keys) != 1 {
		t.Errorf("Keys count = %d, want 1", len(cfg.Keys))
	}

	if cfg.Keys[0].Name != "test-key" {
		t.Errorf("Key name = %s, want test-key", cfg.Keys[0].Name)
	}
}

func TestLoadWithEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	os.Setenv("TEST_API_KEY", "env-secret-value")
	defer os.Unsetenv("TEST_API_KEY")

	content := `
server:
  port: 8765

keys:
  - name: test-key
    base_url: "https://api.example.com/v1"
    format: openai
    secret: "${TEST_API_KEY}"
    enabled: true
    models:
      - name: "gpt-4"
        pricing:
          input: 0.03
          output: 0.06

stats:
  enabled: false
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Keys[0].Secret != "env-secret-value" {
		t.Errorf("Secret = %s, want env-secret-value", cfg.Keys[0].Secret)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Server: ServerConfig{Port: 8765},
				Keys: []KeyConfig{
					{
						Name:    "test",
						BaseURL: "https://api.example.com",
						Format:  "openai",
						Secret:  "key",
						Enabled: true,
						Models:  []ModelConfig{{Name: "gpt-4"}},
					},
				},
				Log: LogConfig{Level: "info"},
			},
			wantErr: false,
		},
		{
			name: "missing secret",
			config: &Config{
				Server: ServerConfig{Port: 8765},
				Keys: []KeyConfig{
					{
						Name:    "test",
						BaseURL: "https://api.example.com",
						Format:  "openai",
						Enabled: true,
						Models:  []ModelConfig{{Name: "gpt-4"}},
					},
				},
				Log: LogConfig{Level: "info"},
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: &Config{
				Server: ServerConfig{Port: 0},
				Keys: []KeyConfig{
					{
						Name:    "test",
						BaseURL: "https://api.example.com",
						Format:  "openai",
						Secret:  "key",
						Enabled: true,
						Models:  []ModelConfig{{Name: "gpt-4"}},
					},
				},
				Log: LogConfig{Level: "info"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestToEndpoints(t *testing.T) {
	cfg := &Config{
		Keys: []KeyConfig{
			{
				Name:    "openai",
				BaseURL: "https://api.openai.com/v1",
				Format:  "openai",
				Secret:  "sk-test",
				Enabled: true,
				Models: []ModelConfig{
					{Name: "gpt-4", Pricing: PricingConfig{Input: 0.03, Output: 0.06}},
					{Name: "gpt-3.5", Pricing: PricingConfig{Input: 0.001, Output: 0.002}},
				},
			},
			{
				Name:    "disabled-key",
				BaseURL: "https://api.example.com",
				Format:  "openai",
				Secret:  "sk-disabled",
				Enabled: false,
				Models:  []ModelConfig{{Name: "model"}},
			},
		},
	}

	endpoints := cfg.ToEndpoints()

	// Should have 2 endpoints (from enabled key with 2 models)
	if len(endpoints) != 2 {
		t.Errorf("Endpoints count = %d, want 2", len(endpoints))
	}

	// Check first endpoint
	if endpoints[0].Key.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("BaseURL = %s, want https://api.openai.com/v1", endpoints[0].Key.BaseURL)
	}
	if endpoints[0].Model != "gpt-4" {
		t.Errorf("Model = %s, want gpt-4", endpoints[0].Model)
	}
	if endpoints[0].InputPrice != 0.03 {
		t.Errorf("InputPrice = %f, want 0.03", endpoints[0].InputPrice)
	}
}
