package cli

import (
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

func TestFindKey(t *testing.T) {
	cfg := &config.Config{
		Keys: []config.KeyConfig{
			{Name: "key1", Format: config.FormatOpenAI},
			{Name: "key2", Format: config.FormatAnthropic},
		},
	}

	// Existing key
	k := cfg.FindKey("key1")
	if k == nil {
		t.Error("FindKey should find existing key")
	}
	if k.Name != "key1" {
		t.Errorf("FindKey returned wrong key: %s", k.Name)
	}

	// Missing key
	k = cfg.FindKey("nonexistent")
	if k != nil {
		t.Error("FindKey should return nil for nonexistent key")
	}
}

func TestFindKeyIndex(t *testing.T) {
	cfg := &config.Config{
		Keys: []config.KeyConfig{
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

func TestConfigKeyOperations(t *testing.T) {
	key := config.KeyConfig{
		Name:    "test-key",
		Format:  config.FormatOpenAI,
		Models:  []config.ModelConfig{{Name: "gpt-4"}},
		Enabled: true,
	}

	// HasModel
	if !key.HasModel("gpt-4") {
		t.Error("HasModel should return true for existing model")
	}
	if key.HasModel("nonexistent") {
		t.Error("HasModel should return false for missing model")
	}

	// AddModel
	if key.AddModel("gpt-4") {
		t.Error("AddModel should return false when model already exists")
	}
	if !key.AddModel("gpt-3.5") {
		t.Error("AddModel should return true when adding new model")
	}
	if len(key.Models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(key.Models))
	}

	// RemoveModel
	if key.RemoveModel("nonexistent") {
		t.Error("RemoveModel should return false for missing model")
	}
	if !key.RemoveModel("gpt-3.5") {
		t.Error("RemoveModel should return true when removing existing model")
	}
	if len(key.Models) != 1 {
		t.Errorf("Expected 1 model after removal, got %d", len(key.Models))
	}
}
