package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tokzone/tokrouter/config"
)

// TestConfigBuilder helps build test configurations.
type TestConfigBuilder struct {
	cfg *config.Config
}

// NewTestConfigBuilder creates a new TestConfigBuilder with default values.
func NewTestConfigBuilder() *TestConfigBuilder {
	return &TestConfigBuilder{
		cfg: &config.Config{
			Server: config.ServerConfig{Host: "127.0.0.1", Port: 8765},
			Log:    config.LogConfig{Level: "info"},
			Stats:  config.StatsConfig{Enabled: false},
		},
	}
}

// WithKey adds a key configuration.
func (b *TestConfigBuilder) WithKey(name, provider, format, secret, baseURL string, enabled bool) *TestConfigBuilder {
	key := config.KeyConfig{
		Name:    name,
		Provider: provider,
		Format:  format,
		Secret:  secret,
		BaseURL: baseURL,
		Enabled: enabled,
	}
	if provider != "" {
		// Apply preset if provider is set
		preset, err := config.GetPreset(provider)
		if err == nil {
			if key.BaseURL == "" {
				key.BaseURL = preset.BaseURL
			}
			if key.Format == "" {
				key.Format = preset.Format
			}
			if len(key.Models) == 0 {
				key.Models = make([]config.ModelConfig, len(preset.DefaultModels))
				for i, m := range preset.DefaultModels {
					key.Models[i] = config.ModelConfig{Name: m.Name, Alias: m.Alias}
				}
			}
		}
	} else {
		// Add default model for non-preset keys
		if len(key.Models) == 0 {
			key.Models = []config.ModelConfig{{Name: "test-model"}}
		}
	}
	b.cfg.Keys = append(b.cfg.Keys, key)
	return b
}

// WithOpenAIKey adds an OpenAI key with test defaults.
func (b *TestConfigBuilder) WithOpenAIKey(name, secret string) *TestConfigBuilder {
	return b.WithKey(name, "openai", "", secret, "", true)
}

// WithAnthropicKey adds an Anthropic key with test defaults.
func (b *TestConfigBuilder) WithAnthropicKey(name, secret string) *TestConfigBuilder {
	return b.WithKey(name, "anthropic", "", secret, "", true)
}

// Build returns the built configuration.
func (b *TestConfigBuilder) Build() *config.Config {
	return b.cfg
}

// WriteTempConfig writes the configuration to a temporary file.
// Returns the file path. The file will be cleaned up by t.Cleanup.
func WriteTempConfig(t *testing.T, cfg *config.Config) string {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Build YAML content manually for test
	content := "server:\n  host: " + cfg.Server.Host + "\n  port: " + itoa(cfg.Server.Port) + "\n"
	content += "log:\n  level: " + cfg.Log.Level + "\n"
	content += "stats:\n  enabled: false\n"
	content += "keys:\n"

	for _, key := range cfg.Keys {
		content += "  - name: " + key.Name + "\n"
		if key.Provider != "" {
			content += "    provider: " + key.Provider + "\n"
		}
		if key.BaseURL != "" {
			content += "    base_url: " + key.BaseURL + "\n"
		}
		if key.Format != "" {
			content += "    format: " + key.Format + "\n"
		}
		content += "    secret: " + key.Secret + "\n"
		content += "    enabled: " + boolStr(key.Enabled) + "\n"
		content += "    models:\n"
		for _, m := range key.Models {
			content += "      - name: " + m.Name + "\n"
		}
	}

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteTempConfig: %v", err)
	}

	return configPath
}

// itoa converts int to string (simple helper).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var s string
	for n > 0 {
		s = string('0'+byte(n%10)) + s
		n /= 10
	}
	return s
}

// boolStr converts bool to string.
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}