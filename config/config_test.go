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
        priority: 100

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
        priority: 100

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
					{Name: "gpt-4", Priority: 100},
					{Name: "gpt-3.5", Priority: 10},
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
	// Priority is set from config
	if endpoints[0].Priority != 100 {
		t.Errorf("Priority = %d, want 100", endpoints[0].Priority)
	}
}

func TestResolvePaths(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `
server:
  port: 8765
  tls_cert: "./cert.pem"
  tls_key: "./key.pem"

keys:
  - name: test-key
    base_url: "https://api.example.com/v1"
    format: openai
    secret: "test-secret"
    enabled: true
    models:
      - name: "gpt-4"
        priority: 100

stats:
  enabled: true
  db_path: "./data/usage.db"

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

	// Check that relative paths are resolved to config file directory
	expectedDBPath := filepath.Join(tmpDir, "data", "usage.db")
	if cfg.Stats.DBPath != expectedDBPath {
		t.Errorf("DBPath = %s, want %s", cfg.Stats.DBPath, expectedDBPath)
	}

	expectedTLSCert := filepath.Join(tmpDir, "cert.pem")
	if cfg.Server.TLSCert != expectedTLSCert {
		t.Errorf("TLSCert = %s, want %s", cfg.Server.TLSCert, expectedTLSCert)
	}

	expectedTLSKey := filepath.Join(tmpDir, "key.pem")
	if cfg.Server.TLSKey != expectedTLSKey {
		t.Errorf("TLSKey = %s, want %s", cfg.Server.TLSKey, expectedTLSKey)
	}
}

func TestResolvePathsAbsolute(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Use absolute paths
	absPath := "/var/data/usage.db"

	content := `
server:
  port: 8765

keys:
  - name: test-key
    base_url: "https://api.example.com/v1"
    format: openai
    secret: "test-secret"
    enabled: true
    models:
      - name: "gpt-4"
        priority: 100

stats:
  enabled: true
  db_path: "` + absPath + `"

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

	// Absolute path should remain unchanged
	if cfg.Stats.DBPath != absPath {
		t.Errorf("DBPath = %s, want %s", cfg.Stats.DBPath, absPath)
	}
}

func TestParseProtocol(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"openai", "openai", false},
		{"anthropic", "anthropic", false},
		{"gemini", "gemini", false},
		{"cohere", "cohere", false},
		{"invalid", "invalid", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseProtocol(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseProtocol(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
			}
		})
	}
}

func TestAliasMap(t *testing.T) {
	cfg := &Config{
		Keys: []KeyConfig{
			{
				Name: "test",
				Models: []ModelConfig{
					// Name is actual model, Alias is what user requests
					{Name: "gpt-4-turbo", Alias: "gpt-4"},
					{Name: "gpt-3.5", Alias: ""}, // no alias
				},
			},
		},
	}

	aliasMap := cfg.AliasMap()
	if len(aliasMap) != 1 {
		t.Errorf("aliasMap length = %d, want 1", len(aliasMap))
	}

	// User requests "gpt-4" -> rewritten to actual model "gpt-4-turbo"
	if aliasMap["gpt-4"] != "gpt-4-turbo" {
		t.Errorf("alias = %s, want gpt-4-turbo", aliasMap["gpt-4"])
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		Server: ServerConfig{Port: 8765},
		Keys: []KeyConfig{
			{
				Name:    "test",
				BaseURL: "https://api.example.com",
				Format:  "openai",
				Secret:  "test-secret",
				Enabled: true,
				Models: []ModelConfig{
					{Name: "gpt-4", Priority: 100},
					{Name: "gpt-4-turbo", Alias: "gpt-4-1106-preview", Priority: 50},
				},
			},
		},
		Log: LogConfig{Level: "info"},
	}

	if err := Save(configPath, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load and verify
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Keys) != 1 {
		t.Errorf("Keys count = %d, want 1", len(loaded.Keys))
	}

	if len(loaded.Keys[0].Models) != 2 {
		t.Errorf("Models count = %d, want 2", len(loaded.Keys[0].Models))
	}
}

func TestIsValidFormat(t *testing.T) {
	tests := []struct {
		format string
		want   bool
	}{
		{FormatOpenAI, true},
		{FormatAnthropic, true},
		{FormatGemini, true},
		{FormatCohere, true},
		{"invalid", false},
		{"", false},
		{"OPENAI", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			if got := IsValidFormat(tt.format); got != tt.want {
				t.Errorf("IsValidFormat(%s) = %v, want %v", tt.format, got, tt.want)
			}
		})
	}
}
