package cli

import (
	"strings"
	"testing"

	"github.com/tokzone/tokrouter/config"
)

func TestGetDefaultURL(t *testing.T) {
	tests := []struct {
		format string
		want   string
	}{
		{config.FormatOpenAI, "https://api.openai.com/v1"},
		{config.FormatAnthropic, "https://api.anthropic.com/v1"},
		{config.FormatGemini, "https://generativelanguage.googleapis.com/v1"},
		{config.FormatCohere, "https://api.cohere.ai/v1"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			got := getDefaultURL(tt.format)
			if got != tt.want {
				t.Errorf("getDefaultURL(%s) = %s, want %s", tt.format, got, tt.want)
			}
		})
	}
}

func TestFormatPriority(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 (default)"},
		{100, "100"},
		{-1, "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatPriority(tt.input)
			if got != tt.want {
				t.Errorf("formatPriority(%d) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

func TestCheckKeyExists(t *testing.T) {
	cfg := &config.Config{
		Keys: []config.KeyConfig{
			{Name: "existing-key", Format: config.FormatOpenAI},
		},
	}

	// Key exists - should return error
	err := checkKeyExists(cfg, "existing-key")
	if err == nil {
		t.Error("checkKeyExists should return error for existing key")
	}

	// Key does not exist - should return nil
	err = checkKeyExists(cfg, "new-key")
	if err != nil {
		t.Errorf("checkKeyExists should return nil for new key, got: %v", err)
	}
}

func TestListAvailableKeysError(t *testing.T) {
	cfg := &config.Config{
		Keys: []config.KeyConfig{
			{Name: "key1"},
			{Name: "key2"},
		},
	}

	err := listAvailableKeysError(cfg, "missing-key")
	if err == nil {
		t.Error("listAvailableKeysError should return error")
	}

	// Error message should contain available keys
	errMsg := err.Error()
	if !strings.Contains(errMsg, "key1") || !strings.Contains(errMsg, "key2") {
		t.Error("error message should contain available key names")
	}
}