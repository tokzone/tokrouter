package cli

import (
	"fmt"
	"sort"

	"github.com/tokzone/tokrouter/config"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
)

type modelOption struct {
	Model       string // model name
	DisplayName string // provider display name
	Preset      string // preset name
	Configured  bool   // already in config.yaml
	Context     int    // context window size in tokens
}

// selectAssistantModel lets the user pick a model for the assistant.
// Shows a flat list of all models from all presets with provider attribution.
// Models already in config.yaml are marked with ✓.
// If the chosen preset is not yet configured, prompts for API key and adds it.
// Returns the model name, or empty string if user skips.
func selectAssistantModel(c *cli.Command) (string, error) {
	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		cfg = &config.Config{}
	}

	// Determine which providers are already configured
	configuredProviders := map[string]bool{}
	for _, k := range cfg.Keys {
		if k.Enabled && k.Provider != "" {
			configuredProviders[k.Provider] = true
		}
	}

	// Collect all models from all presets
	seen := map[string]bool{}
	var all []modelOption
	for _, preset := range config.ListPresets() {
		p, err := config.GetPreset(preset.Name)
		if err != nil {
			continue
		}
		for _, m := range p.DefaultModels {
			key := m.Name
			if seen[key] {
				continue
			}
			seen[key] = true
			all = append(all, modelOption{
				Model:       m.Name,
				DisplayName: p.DisplayName,
				Preset:      preset.Name,
				Configured:  configuredProviders[preset.Name],
				Context:     m.Context,
			})
		}
	}

	if len(all) == 0 {
		return "", fmt.Errorf("no model presets available")
	}

	// Sort by configured first, then display name, then model name
	sort.Slice(all, func(i, j int) bool {
		if all[i].Configured != all[j].Configured {
			return all[i].Configured
		}
		if all[i].DisplayName != all[j].DisplayName {
			return all[i].DisplayName < all[j].DisplayName
		}
		return all[i].Model < all[j].Model
	})

	// Build survey options
	skipLabel := "Skip (don't set a default model)"
	surveyOptions := []string{skipLabel}
	options := []modelOption{{}}

	for _, m := range all {
		ctxStr := formatContext(m.Context)
		label := fmt.Sprintf("%-28s %-12s %s", m.Model, ctxStr, m.DisplayName)
		if m.Configured {
			label += " ✓"
		}
		surveyOptions = append(surveyOptions, label)
		options = append(options, m)
	}

	pterm.Println()
	prompt := &survey.Select{
		Message:  "Select default model:",
		Options:  surveyOptions,
		PageSize: 15,
	}
	var selected string
	if err := survey.AskOne(prompt, &selected); err != nil {
		return "", err
	}

	// Find selection
	for i, opt := range surveyOptions {
		if opt == selected {
			if i == 0 || options[i].Model == "" {
				return "", nil // skip
			}
			opt := options[i]
			if opt.Configured {
				model, err := updatePresetInConfig(cfg, configPath, opt)
				if err != nil {
					pterm.Warning.Printf("Failed to update config: %v\n", err)
				}
				return model, nil
			}
			return addPresetToConfig(configPath, cfg, opt)
		}
	}
	return "", nil
}

// updatePresetInConfig updates an existing key's format and models from the preset.
// Secret and service name are preserved.
func updatePresetInConfig(cfg *config.Config, configPath string, opt modelOption) (string, error) {
	preset, err := config.GetPreset(opt.Preset)
	if err != nil {
		return "", err
	}

	for i := range cfg.Keys {
		if cfg.Keys[i].Provider == opt.Preset && cfg.Keys[i].Enabled {
			cfg.Keys[i].Format = preset.Format
			cfg.Keys[i].Models = nil
			for _, m := range preset.DefaultModels {
				cfg.Keys[i].Models = append(cfg.Keys[i].Models, config.ModelConfig{Name: m.Name, Alias: m.Alias})
			}
			if err := config.Save(configPath, cfg); err != nil {
				return "", err
			}
			pterm.Success.Printf("Updated %s with latest preset (format: %s, %d models)\n", preset.DisplayName, preset.Format, len(preset.DefaultModels))
			return opt.Model, nil
		}
	}

	// If no existing key found for this provider, skip update (shouldn't happen)
	return opt.Model, nil
}

// addPresetToConfig prompts for an API key and adds the preset provider to config.
func addPresetToConfig(configPath string, cfg *config.Config, opt modelOption) (string, error) {
	preset, err := config.GetPreset(opt.Preset)
	if err != nil {
		return "", err
	}

	pterm.Println()
	var secret string
	secretPrompt := &survey.Password{
		Message: fmt.Sprintf("API Key for %s:", preset.DisplayName),
	}
	if err := survey.AskOne(secretPrompt, &secret); err != nil || secret == "" {
		return "", fmt.Errorf("API key is required")
	}

	serviceName := fmt.Sprintf("%s-%d", opt.Preset, len(cfg.Keys)+1)
	newKey := config.KeyConfig{
		Provider: opt.Preset,
		Name:     serviceName,
		Format:   preset.Format,
		Secret:   secret,
		Enabled:  true,
	}
	for _, m := range preset.DefaultModels {
		newKey.Models = append(newKey.Models, config.ModelConfig{Name: m.Name, Alias: m.Alias})
	}

	cfg.Keys = append(cfg.Keys, newKey)
	if err := config.Save(configPath, cfg); err != nil {
		return "", err
	}

	pterm.Success.Printf("Added %s with %d models\n", preset.DisplayName, len(newKey.Models))

	return opt.Model, nil
}

func formatContext(tokens int) string {
	if tokens == 0 {
		return ""
	}
	if tokens >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1000000)
	}
	if tokens >= 1000 {
		return fmt.Sprintf("%dK", tokens/1000)
	}
	return fmt.Sprintf("%d", tokens)
}
