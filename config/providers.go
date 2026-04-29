package config

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed presets.yaml
var presetsData []byte

// Provider name constants for type-safe preset references.
const (
	// International providers
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
	ProviderGoogle    = "google"
	ProviderMistral   = "mistral"
	ProviderCohere    = "cohere"
	ProviderGroq      = "groq"
	ProviderDeepSeek  = "deepseek"

	// Chinese providers
	ProviderZhipu       = "zhipu"
	ProviderQwen        = "qwen"
	ProviderTencent     = "tencent"
	ProviderBaidu       = "baidu"
	ProviderQianfan     = "qianfan"
	ProviderHuawei      = "huawei"
	ProviderMoonshot    = "moonshot"
	ProviderMinimax     = "minimax"
	ProviderSiliconflow = "siliconflow"
	ProviderYi          = "yi"
	ProviderStepfun     = "stepfun"
	ProviderBaichuan    = "baichuan"
	ProviderXunfei      = "xunfei"
	ProviderDoubao      = "doubao"
	ProviderParallel    = "parallel"
	ProviderModelScope  = "modelscope"

	// Aggregation platforms
	ProviderTogether   = "together"
	ProviderReplicate  = "replicate"
	ProviderOpenRouter = "openrouter"
)

// ProviderPreset defines a preset configuration for a LLM provider.
type ProviderPreset struct {
	Name          string            `yaml:"-"` // populated from map key
	DisplayName   string            `yaml:"display_name"`
	BaseURLs      map[string]string `yaml:"base_urls"`
	Format        string            `yaml:"format"`
	Protocols     []string          `yaml:"protocols"`
	DefaultModels []ModelInfo       `yaml:"models"`
	DocURL        string            `yaml:"doc_url"`
	Region        string            `yaml:"region"`
}

// ModelInfo defines a model with optional alias and context length.
type ModelInfo struct {
	Name    string `yaml:"name"`
	Alias   string `yaml:"alias"`
	Context int    `yaml:"context"`
}

// presetsFile is the top-level YAML structure of presets.yaml.
type presetsFile struct {
	Presets map[string]ProviderPreset `yaml:"presets"`
}

// BuiltinPresets contains all built-in provider presets, loaded from embedded presets.yaml.
var BuiltinPresets map[string]ProviderPreset

func init() {
	BuiltinPresets = loadPresets()
}

func loadPresets() map[string]ProviderPreset {
	var pf presetsFile
	if err := yaml.Unmarshal(presetsData, &pf); err != nil {
		slog.Error("failed to parse embedded presets.yaml", "error", err)
		return nil
	}
	presetsData = nil // free embedded YAML bytes after parse

	for name := range pf.Presets {
		if name == "" {
			delete(pf.Presets, name)
			continue
		}
		p := pf.Presets[name]
		p.Name = name
		pf.Presets[name] = p
	}
	return pf.Presets
}

var externalPresetsOnce sync.Once

// MergeExternalPresets reads presets.yaml from the given directory and
// merges them into BuiltinPresets. Entries in the external file override
// built-in ones with the same name; new entries are added.
// If no presets.yaml exists in dir, it silently uses built-in only.
// Multiple calls are no-ops — external presets are loaded at most once.
func MergeExternalPresets(configDir string) {
	externalPresetsOnce.Do(func() {
		path := filepath.Join(configDir, "presets.yaml")
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		var pf presetsFile
		if err := yaml.Unmarshal(data, &pf); err != nil {
			slog.Warn("failed to parse external presets.yaml, using built-in", "error", err)
			return
		}
		merged := 0
		for name := range pf.Presets {
			if name == "" {
				continue
			}
			p := pf.Presets[name]
			p.Name = name
			BuiltinPresets[name] = p
			merged++
		}
		slog.Info("merged external presets", "count", merged, "path", path)
	})
}

// GetPreset returns a preset by name, or an error if not found.
func GetPreset(name string) (ProviderPreset, error) {
	preset, ok := BuiltinPresets[name]
	if !ok {
		return ProviderPreset{}, fmt.Errorf("unknown provider preset: %q", name)
	}
	return preset, nil
}

// ListPresets returns all built-in presets sorted by region then name.
func ListPresets() []ProviderPreset {
	presets := make([]ProviderPreset, 0, len(BuiltinPresets))
	for _, p := range BuiltinPresets {
		presets = append(presets, p)
	}
	// Sort by region (global first) then by name
	sort.Slice(presets, func(i, j int) bool {
		if presets[i].Region != presets[j].Region {
			return presets[i].Region < presets[j].Region
		}
		return presets[i].Name < presets[j].Name
	})
	return presets
}

// PresetExists checks if a preset name exists.
func PresetExists(name string) bool {
	_, ok := BuiltinPresets[name]
	return ok
}

