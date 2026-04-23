package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/tokzone/fluxcore/routing"
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

// Protocol returns the routing.Protocol for this key config
func (k *KeyConfig) Protocol() routing.Protocol {
	return ParseProtocol(k.Format)
}

// ModelConfig holds model configuration
type ModelConfig struct {
	Name    string        `mapstructure:"name"`
	Alias   string        `mapstructure:"alias"`   // Optional: map request model name to actual model
	Pricing PricingConfig `mapstructure:"pricing"`
}

// PricingConfig holds pricing configuration
type PricingConfig struct {
	Input  float64 `mapstructure:"input"`
	Output float64 `mapstructure:"output"`
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
		cfg.Keys[i].Secret = expandEnv(cfg.Keys[i].Secret)
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
		return fmt.Errorf("config invalid: invalid port %d", c.Server.Port)
	}

	// Validate keys
	if len(c.Keys) == 0 {
		return fmt.Errorf("config invalid: no keys configured")
	}

	for _, k := range c.Keys {
		if k.Enabled {
			if k.Secret == "" {
				return fmt.Errorf("config invalid: key %s missing secret", k.Name)
			}
			if k.BaseURL == "" {
				return fmt.Errorf("config invalid: key %s missing base_url", k.Name)
			}
			if k.Format == "" {
				return fmt.Errorf("config invalid: key %s missing format", k.Name)
			}
			if !isValidProtocol(k.Format) {
				return fmt.Errorf("config invalid: key %s has invalid format %s", k.Name, k.Format)
			}
			if len(k.Models) == 0 {
				return fmt.Errorf("config invalid: key %s has no models", k.Name)
			}
			for j, m := range k.Models {
				if m.Name == "" {
					return fmt.Errorf("config invalid: key %s model %d missing name", k.Name, j)
				}
			}
		}
	}

	if c.Stats.Enabled && c.Stats.DBPath == "" {
		return fmt.Errorf("config invalid: stats.db_path required when stats.enabled=true")
	}

	// Validate log config
	if !validLogLevels[c.Log.Level] {
		return fmt.Errorf("config invalid: invalid log level %s", c.Log.Level)
	}

	return nil
}

// expandEnv expands environment variables in a string
func expandEnv(s string) string {
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		envVar := s[2 : len(s)-1]
		return os.Getenv(envVar)
	}
	return s
}

// ParseProtocol parses string to routing.Protocol
func ParseProtocol(s string) routing.Protocol {
	switch s {
	case "anthropic":
		return routing.ProtocolAnthropic
	case "gemini":
		return routing.ProtocolGemini
	case "cohere":
		return routing.ProtocolCohere
	default:
		return routing.ProtocolOpenAI
	}
}

// isValidProtocol checks if protocol string is valid
func isValidProtocol(s string) bool {
	switch s {
	case "openai", "anthropic", "gemini", "cohere":
		return true
	}
	return false
}

// ToEndpoints converts KeyConfig to routing.Endpoint
// Priority is calculated as (inputPrice + outputPrice) * 1_000_000 for price-first routing.
func (c *Config) ToEndpoints() []*routing.Endpoint {
	endpoints := make([]*routing.Endpoint, 0, len(c.Keys))

	const priceScale = 1_000_000
	id := uint(1)
	for _, kc := range c.Keys {
		if !kc.Enabled {
			continue
		}

		// Create Key (shared across models for this provider)
		key := &routing.Key{
			BaseURL:  kc.BaseURL,
			APIKey:   kc.Secret,
			Protocol: kc.Protocol(), // direct conversion
		}

		for _, mc := range kc.Models {
			// Calculate priority: price-first strategy (lower price = lower priority = preferred)
			priority := int64((mc.Pricing.Input + mc.Pricing.Output) * priceScale)
			ep, err := routing.NewEndpoint(id, key, mc.Name, priority)
			if err != nil {
				continue // skip invalid endpoint
			}
			endpoints = append(endpoints, ep)
			id++
		}
	}

	return endpoints
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

// GetAliasMap returns a map of model aliases (request model -> actual model)
func (c *Config) GetAliasMap() map[string]string {
	aliasMap := make(map[string]string)
	for _, kc := range c.Keys {
		for _, mc := range kc.Models {
			if mc.Alias != "" {
				aliasMap[mc.Name] = mc.Alias
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
			models = append(models, map[string]interface{}{
				"name": mc.Name,
				"pricing": map[string]interface{}{
					"input":  mc.Pricing.Input,
					"output": mc.Pricing.Output,
				},
			})
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
