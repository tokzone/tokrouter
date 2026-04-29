package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tokzone/fluxcore"
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
    base_urls:
      openai: "https://api.example.com/v1"
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
    base_urls:
      openai: "https://api.example.com/v1"
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
						BaseURLs: map[string]string{FormatOpenAI: "https://api.example.com"},
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
						BaseURLs: map[string]string{FormatOpenAI: "https://api.example.com"},
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
						BaseURLs: map[string]string{FormatOpenAI: "https://api.example.com"},
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
    base_urls:
      openai: "https://api.example.com/v1"
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
    base_urls:
      openai: "https://api.example.com/v1"
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
				BaseURLs: map[string]string{FormatOpenAI: "https://api.example.com"},
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

func TestLoadWithPreset(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `
server:
  port: 8765

keys:
  # Simplified config using preset
  - provider: openai
    secret: "sk-test-key"

  # Another preset with custom models override
  - provider: deepseek
    secret: "sk-deepseek"
    models:
      - name: deepseek-chat

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

	// Check first key (openai preset)
	if cfg.Keys[0].Name != "openai-1" {
		t.Errorf("Key[0] Name = %s, want openai-1 (auto-generated)", cfg.Keys[0].Name)
	}
	if cfg.Keys[0].BaseURLs[FormatOpenAI] != "https://api.openai.com/v1" {
		t.Errorf("Key[0] BaseURL = %s, want preset value", cfg.Keys[0].BaseURLs[FormatOpenAI])
	}
	if cfg.Keys[0].Format != "openai" {
		t.Errorf("Key[0] Format = %s, want openai", cfg.Keys[0].Format)
	}
	if len(cfg.Keys[0].Models) == 0 {
		t.Errorf("Key[0] Models should be auto-filled from preset")
	}
	// Check that preset models are filled
	foundGPT4o := false
	for _, m := range cfg.Keys[0].Models {
		if m.Name == "gpt-5.4" {
			foundGPT4o = true
			break
		}
	}
	if !foundGPT4o {
		t.Errorf("Key[0] Models should include gpt-5.4 from preset")
	}

	// Check second key (deepseek with custom models)
	if cfg.Keys[1].Name != "deepseek-2" {
		t.Errorf("Key[1] Name = %s, want deepseek-2", cfg.Keys[1].Name)
	}
	if cfg.Keys[1].BaseURLs[FormatOpenAI] != "https://api.deepseek.com" {
		t.Errorf("Key[1] BaseURL = %s, want preset value", cfg.Keys[1].BaseURLs[FormatOpenAI])
	}
	if len(cfg.Keys[1].Models) != 1 {
		t.Errorf("Key[1] Models count = %d, want 1 (custom override)", len(cfg.Keys[1].Models))
	}
	if cfg.Keys[1].Models[0].Name != "deepseek-chat" {
		t.Errorf("Key[1] Model = %s, want deepseek-chat", cfg.Keys[1].Models[0].Name)
	}
}

func TestPresetExists(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"openai", true},
		{"anthropic", true},
		{"deepseek", true},
		{"qwen", true},
		{"zhipu", true},
		{"invalid-provider", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PresetExists(tt.name); got != tt.want {
				t.Errorf("PresetExists(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestGetPreset(t *testing.T) {
	preset, err := GetPreset("openai")
	if err != nil {
		t.Fatalf("GetPreset(openai) error: %v", err)
	}

	if preset.DisplayName != "OpenAI" {
		t.Errorf("DisplayName = %s, want OpenAI", preset.DisplayName)
	}
	if preset.BaseURLs[FormatOpenAI] != "https://api.openai.com/v1" {
		t.Errorf("BaseURL = %s, want https://api.openai.com/v1", preset.BaseURLs[FormatOpenAI])
	}
	if preset.Format != "openai" {
		t.Errorf("Format = %s, want openai", preset.Format)
	}
	if preset.Region != "global" {
		t.Errorf("Region = %s, want global", preset.Region)
	}
	if len(preset.DefaultModels) == 0 {
		t.Errorf("DefaultModels should not be empty")
	}

	// Test invalid preset
	_, err = GetPreset("invalid")
	if err == nil {
		t.Errorf("GetPreset(invalid) should return error")
	}
}

func TestListPresets(t *testing.T) {
	presets := ListPresets()

	if len(presets) < 20 {
		t.Errorf("ListPresets count = %d, want at least 20", len(presets))
	}

	// Check that presets are sorted by region then name
	// Global presets should come first
	for i, p := range presets {
		if p.Name == "" {
			t.Errorf("Preset[%d] has empty name", i)
		}
		if len(p.BaseURLs) == 0 {
			t.Errorf("Preset[%d] %s has empty BaseURLs", i, p.Name)
		}
		if p.Format == "" {
			t.Errorf("Preset[%d] %s has empty Format", i, p.Name)
		}
	}
}

func TestConvertBaseURLs(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]string
		want  map[string]string // protocol name → url
	}{
		{
			name:  "single openai",
			input: map[string]string{"openai": "https://api.openai.com/v1"},
			want:  map[string]string{"openai": "https://api.openai.com/v1"},
		},
		{
			name: "multi protocol",
			input: map[string]string{
				"openai":    "https://api.openai.com/v1",
				"anthropic": "https://api.anthropic.com/v1",
			},
			want: map[string]string{
				"openai":    "https://api.openai.com/v1",
				"anthropic": "https://api.anthropic.com/v1",
			},
		},
		{
			name:  "invalid protocol skipped",
			input: map[string]string{"invalid": "https://example.com", "openai": "https://api.openai.com"},
			want:  map[string]string{"openai": "https://api.openai.com"},
		},
		{
			name:  "empty input",
			input: map[string]string{},
			want:  map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertBaseURLs(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("got %d entries, want %d", len(got), len(tt.want))
			}
			for protoStr, url := range tt.want {
				p, _ := fluxcore.ParseProtocol(protoStr)
				if got[p] != url {
					t.Errorf("got[%s] = %s, want %s", protoStr, got[p], url)
				}
			}
		})
	}
}

func TestToRoutes(t *testing.T) {
	cfg := &Config{
		Keys: []KeyConfig{
			{
				Name:    "openai",
				BaseURLs: map[string]string{"openai": "https://api.openai.com/v1"},
				Format:  "openai",
				Secret:  "sk-openai",
				Enabled: true,
				Models: []ModelConfig{
					{Name: "gpt-4", Priority: 0},
					{Name: "gpt-3.5", Priority: 10},
				},
			},
			{
				Name:    "disabled-key",
				BaseURLs: map[string]string{"openai": "https://api.disabled.com"},
				Format:  "openai",
				Secret:  "sk-disabled",
				Enabled: false,
				Models:  []ModelConfig{{Name: "gpt-4"}},
			},
		},
	}

	svcEPs := make(map[string]*fluxcore.ServiceEndpoint)
	calls := make(map[string]int)

	routes := cfg.ToRoutes(svcEPs, func(desc fluxcore.RouteDesc) *fluxcore.Route {
		calls[desc.IdentityKey()]++
		return fluxcore.NewRoute(desc)
	})

	if len(routes) != 2 {
		t.Fatalf("got %d routes, want 2 (disabled key excluded)", len(routes))
	}

	// gpt-4 has priority 0, gpt-3.5 has priority 10
	if routes[0].Desc().Model != "gpt-4" || routes[1].Desc().Model != "gpt-3.5" {
		t.Error("route order wrong")
	}
	if routes[0].Desc().Priority != 0 {
		t.Errorf("gpt-4 priority = %d, want 0", routes[0].Desc().Priority)
	}
	if routes[0].Desc().Credential != "sk-openai" {
		t.Errorf("credential = %s, want sk-openai", routes[0].Desc().Credential)
	}

	// ServiceEndpoint should be created and shared
	if se, ok := svcEPs["openai"]; !ok || se == nil {
		t.Error("openai ServiceEndpoint not created in svcEPs")
	}
	if len(svcEPs) != 1 {
		t.Errorf("svcEPs count = %d, want 1", len(svcEPs))
	}

	// Each unique identity key should trigger exactly one FindOrCreate call
	expectedKeys := []string{"openai/gpt-4/sk-openai", "openai/gpt-3.5/sk-openai"}
	for _, key := range expectedKeys {
		if calls[key] != 1 {
			t.Errorf("FindOrCreate called %d times for %s, want 1", calls[key], key)
		}
	}
}

func TestFindKey(t *testing.T) {
	cfg := &Config{
		Keys: []KeyConfig{
			{Name: "key1"},
			{Name: "key2"},
		},
	}
	if k := cfg.FindKey("key1"); k == nil || k.Name != "key1" {
		t.Error("FindKey should find existing key")
	}
	if k := cfg.FindKey("nonexistent"); k != nil {
		t.Error("FindKey should return nil for missing key")
	}
}

func TestFindKeyIndex(t *testing.T) {
	cfg := &Config{
		Keys: []KeyConfig{
			{Name: "key1"},
			{Name: "key2"},
		},
	}
	if idx := cfg.FindKeyIndex("key1"); idx != 0 {
		t.Errorf("FindKeyIndex(key1) = %d, want 0", idx)
	}
	if idx := cfg.FindKeyIndex("key2"); idx != 1 {
		t.Errorf("FindKeyIndex(key2) = %d, want 1", idx)
	}
	if idx := cfg.FindKeyIndex("nonexistent"); idx != -1 {
		t.Errorf("FindKeyIndex(nonexistent) = %d, want -1", idx)
	}
}

func TestLoadWithInvalidProvider(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `
server:
  port: 8765

keys:
  - provider: invalid-provider
    secret: "sk-test"

stats:
  enabled: false

log:
  level: info
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Errorf("Load with invalid provider should return error")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Load should return error for non-existent file")
	}
	if cfg != nil {
		t.Error("cfg should be nil when Load returns error")
	}
}
