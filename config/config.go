package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/tokzone/fluxcore/endpoint"
	"github.com/tokzone/fluxcore/flux"
	"github.com/tokzone/fluxcore/provider"
)

// Config is the main configuration structure
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Keys   []KeyConfig  `mapstructure:"keys"`
	Router RouterConfig `mapstructure:"router"`
	Stats  StatsConfig  `mapstructure:"stats"`
	Log    LogConfig    `mapstructure:"log"`
	Trace  TraceConfig  `mapstructure:"trace"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	TLSCert  string `mapstructure:"tls_cert"`
	TLSKey   string `mapstructure:"tls_key"`
	LogLevel string `mapstructure:"log_level"` // Optional, defaults to log.level
}

// KeyConfig holds API key configuration (immutable config, no runtime state)
type KeyConfig struct {
	Name    string        `mapstructure:"name"`
	BaseURL string        `mapstructure:"base_url"`
	Format  string        `mapstructure:"format"` // "openai", "anthropic", "gemini", "cohere"
	Secret  string        `mapstructure:"secret"`
	Enabled bool          `mapstructure:"enabled"` // kept for config file compatibility
	Models  []ModelConfig `mapstructure:"models"`
}

// Protocol returns the provider.Protocol for this key config
// Returns ProtocolOpenAI as fallback if format is invalid (should not happen after Validate)
func (k *KeyConfig) Protocol() provider.Protocol {
	p, err := ParseProtocol(k.Format)
	if err != nil {
		return provider.ProtocolOpenAI // explicit fallback
	}
	return p
}

// ModelConfig holds model configuration
type ModelConfig struct {
	Name     string `mapstructure:"name"`
	Alias    string `mapstructure:"alias"`    // Optional: map request model name to actual model
	Priority int64  `mapstructure:"priority"` // Optional: endpoint priority (lower = preferred). Default 0.
}

// validLogLevels is the set of valid log levels
var validLogLevels = map[string]bool{
	"debug": true,
	"info":  true,
	"warn":  true,
	"error": true,
}

// RouterConfig holds router configuration
type RouterConfig struct {
	Retry RetryConfig `mapstructure:"retry"`
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries int    `mapstructure:"max_retries"`
	Timeout    string `mapstructure:"timeout"`
	Backoff    string `mapstructure:"backoff"`
}

// StatsConfig holds statistics configuration
type StatsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	DBPath  string `mapstructure:"db_path"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// TraceConfig holds trace configuration
type TraceConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Header  string `mapstructure:"header"`
}

// Load loads configuration from file
func Load(configPath string) (*Config, error) {
	// Get absolute path and directory of config file
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}
	configDir := filepath.Dir(absConfigPath)

	v := viper.New()

	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// Support environment variables
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Expand environment variables in secrets
	for i := range cfg.Keys {
		cfg.Keys[i].Secret = expandEnv(cfg.Keys[i].Secret, cfg.Keys[i].Name)
	}

	// Resolve relative paths based on config file directory
	cfg.resolvePaths(configDir)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	v.SetDefault("server.host", "127.0.0.1") // localhost only for personal use
	v.SetDefault("server.port", 8765)        // avoid Clash default port 7890

	v.SetDefault("router.retry.max_retries", 2)
	v.SetDefault("router.retry.timeout", "30s")
	v.SetDefault("router.retry.backoff", "exponential")

	v.SetDefault("stats.enabled", true)
	v.SetDefault("stats.db_path", "./data/usage.db")

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "stdout")

	v.SetDefault("trace.enabled", true)
	v.SetDefault("trace.header", "x-request-id")
}

// resolvePaths resolves relative paths to absolute paths based on config file directory
func (c *Config) resolvePaths(configDir string) {
	if !filepath.IsAbs(c.Stats.DBPath) {
		c.Stats.DBPath = filepath.Join(configDir, c.Stats.DBPath)
	}
	if c.Server.TLSCert != "" && !filepath.IsAbs(c.Server.TLSCert) {
		c.Server.TLSCert = filepath.Join(configDir, c.Server.TLSCert)
	}
	if c.Server.TLSKey != "" && !filepath.IsAbs(c.Server.TLSKey) {
		c.Server.TLSKey = filepath.Join(configDir, c.Server.TLSKey)
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf(`server.port: invalid value %d (must be 1-65535)

Suggestion: Use a valid port number in config.yaml:
  server:
    port: 8765`, c.Server.Port)
	}

	// Validate keys
	if len(c.Keys) == 0 {
		return fmt.Errorf(`no keys configured

Suggestion: Add at least one key in config.yaml:
  keys:
    - name: openai-main
      format: openai
      secret: "sk-..."  # or use "${OPENAI_API_KEY}"
      base_url: "https://api.openai.com/v1"
      enabled: true
      models:
        - name: gpt-4
          priority: 0`)
	}

	for i, k := range c.Keys {
		if k.Enabled {
			keyPath := fmt.Sprintf("keys[%d] (%q)", i, k.Name)
			if k.Name == "" {
				return fmt.Errorf(`%s: missing required field 'name'

Suggestion: Add a name for this key in config.yaml:
  keys:
    - name: my-key-name  # required
      ...`, keyPath)
			}
			if k.Secret == "" {
				return fmt.Errorf(`%s: missing required field 'secret'

Suggestion: Add your API key in config.yaml:
  keys:
    - name: %s
      secret: "sk-..."  # or use "${OPENAI_API_KEY}"
      ...`, keyPath, k.Name)
			}
			if k.BaseURL == "" {
				return fmt.Errorf(`%s: missing required field 'base_url'

Suggestion: Add the API base URL in config.yaml:
  keys:
    - name: %s
      base_url: "https://api.openai.com/v1"
      ...`, keyPath, k.Name)
			}
			if k.Format == "" {
				return fmt.Errorf(`%s: missing required field 'format'

Suggestion: Add the format in config.yaml:
  keys:
    - name: %s
      format: openai  # one of: openai, anthropic, gemini, cohere
      ...`, keyPath, k.Name)
			}
			if !isValidProtocol(k.Format) {
				return fmt.Errorf(`%s: invalid format %q

Suggestion: Use one of the supported formats:
  - openai
  - anthropic
  - gemini
  - cohere`, keyPath, k.Format)
			}
			if len(k.Models) == 0 {
				return fmt.Errorf(`%s: no models configured

Suggestion: Add at least one model in config.yaml:
  keys:
    - name: %s
      models:
        - name: gpt-4
          priority: 0`, keyPath, k.Name)
			}
			for j, m := range k.Models {
				if m.Name == "" {
					return fmt.Errorf(`%s.models[%d]: missing required field 'name'

Suggestion: Add a model name:
  models:
    - name: gpt-4  # required
      priority: 0`, keyPath, j)
				}
			}
		}
	}

	if c.Stats.Enabled && c.Stats.DBPath == "" {
		return fmt.Errorf(`stats.db_path: required when stats.enabled=true

Suggestion: Add db_path in config.yaml:
  stats:
    enabled: true
    db_path: "./data/usage.db"`)
	}

	// Validate log config
	if !validLogLevels[c.Log.Level] {
		return fmt.Errorf(`log.level: invalid value %q

Suggestion: Use one of the supported log levels:
  - debug
  - info
  - warn
  - error`, c.Log.Level)
	}

	return nil
}

// expandEnv expands environment variables in a string.
// Logs a warning if the environment variable is not set.
func expandEnv(s string, keyName string) string {
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		envVar := s[2 : len(s)-1]
		val := os.Getenv(envVar)
		if val == "" {
			slog.Warn("environment variable not set",
				"variable", envVar,
				"key", keyName,
				"suggestion", fmt.Sprintf("Set the environment variable: export %s=your-api-key", envVar))
		}
		return val
	}
	return s
}

// Protocol format constants for config files
const (
	FormatOpenAI    = "openai"
	FormatAnthropic = "anthropic"
	FormatGemini    = "gemini"
	FormatCohere    = "cohere"
)

// ParseProtocol parses string to provider.Protocol
func ParseProtocol(s string) (provider.Protocol, error) {
	switch s {
	case FormatOpenAI:
		return provider.ProtocolOpenAI, nil
	case FormatAnthropic:
		return provider.ProtocolAnthropic, nil
	case FormatGemini:
		return provider.ProtocolGemini, nil
	case FormatCohere:
		return provider.ProtocolCohere, nil
	default:
		return provider.ProtocolOpenAI, fmt.Errorf("invalid protocol: %q", s)
	}
}

// IsValidFormat checks if format string is valid (exported for CLI use)
func IsValidFormat(s string) bool {
	return s == FormatOpenAI || s == FormatAnthropic || s == FormatGemini || s == FormatCohere
}

// isValidProtocol checks if protocol string is valid
func isValidProtocol(s string) bool {
	return IsValidFormat(s)
}

func (c *Config) ToUserEndpoints() []*flux.UserEndpoint {
	totalModels := 0
	for _, kc := range c.Keys {
		if kc.Enabled {
			totalModels += len(kc.Models)
		}
	}
	userEndpoints := make([]*flux.UserEndpoint, 0, totalModels)

	providerID := uint(1)
	endpointID := uint(1)

	for _, kc := range c.Keys {
		if !kc.Enabled {
			continue
		}

		prov := provider.NewProvider(providerID, kc.BaseURL, kc.Protocol())
		providerID++

		apiKey, err := flux.NewAPIKey(prov, kc.Secret)
		if err != nil {
			continue
		}

		for _, mc := range kc.Models {
			ep := endpoint.RegisterEndpoint(endpointID, prov, mc.Name)
			if ep == nil {
				continue
			}
			endpointID++

			ue, err := flux.NewUserEndpoint(mc.Name, apiKey, mc.Priority)
			if err != nil {
				continue
			}
			userEndpoints = append(userEndpoints, ue)
		}
	}

	return userEndpoints
}

// FindKey finds a key by name, returns nil if not found
func (c *Config) FindKey(name string) *KeyConfig {
	for i := range c.Keys {
		if c.Keys[i].Name == name {
			return &c.Keys[i]
		}
	}
	return nil
}

// FindKeyIndex finds a key index by name, returns -1 if not found
func (c *Config) FindKeyIndex(name string) int {
	for i := range c.Keys {
		if c.Keys[i].Name == name {
			return i
		}
	}
	return -1
}

// AliasMap returns a map of model aliases (alias -> actual model name)
// When a user requests the alias, it gets rewritten to the actual model name.
func (c *Config) AliasMap() map[string]string {
	aliasMap := make(map[string]string)
	for _, kc := range c.Keys {
		for _, mc := range kc.Models {
			if mc.Alias != "" {
				// User requests alias -> mapped to actual model name
				aliasMap[mc.Alias] = mc.Name
			}
		}
	}
	return aliasMap
}

// Save saves configuration to file
func Save(configPath string, cfg *Config) error {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Read existing to preserve structure
	_ = v.ReadInConfig()

	// Update keys
	var keys []interface{}
	for _, kc := range cfg.Keys {
		var models []interface{}
		for _, mc := range kc.Models {
			m := map[string]interface{}{
				"name":     mc.Name,
				"priority": mc.Priority,
			}
			if mc.Alias != "" {
				m["alias"] = mc.Alias
			}
			models = append(models, m)
		}
		keys = append(keys, map[string]interface{}{
			"name":     kc.Name,
			"base_url": kc.BaseURL,
			"format":   kc.Format,
			"secret":   kc.Secret,
			"enabled":  kc.Enabled,
			"models":   models,
		})
	}
	v.Set("keys", keys)

	return v.WriteConfig()
}
