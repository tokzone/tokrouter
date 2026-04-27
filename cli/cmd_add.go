package cli

import (
	"context"
	"fmt"

	"github.com/tokzone/tokrouter/config"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
)

var addCmd = &cli.Command{
	Name:  "add",
	Usage: "Add a service by preset name or custom config",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "secret",
			Usage: "API key secret",
		},
		&cli.StringFlag{
			Name:  "base-url",
			Usage: "API base URL (for custom service)",
		},
		&cli.StringFlag{
			Name:  "format",
			Usage: "API format (openai/anthropic/gemini/cohere)",
		},
		&cli.StringFlag{
			Name:  "name",
			Usage: "Service name (for custom service)",
		},
		&cli.StringSliceFlag{
			Name:  "model",
			Usage: "Models to add (can specify multiple)",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return runAdd(cmd)
	},
}

func runAdd(c *cli.Command) error {
	args := c.Args()
	secret := c.String("secret")

	// No preset and no custom name → interactive
	if args.Len() == 0 && c.String("name") == "" {
		return runAddInteractive(c)
	}

	// Validate secret
	if secret == "" {
		return fmt.Errorf("--secret is required")
	}

	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	var newKey config.KeyConfig

	if args.Len() > 0 {
		// Preset mode
		presetName := args.First()
		preset, err := config.GetPreset(presetName)
		if err != nil {
			return fmt.Errorf("preset '%s' not found. Use 'tr list presets' to see available presets", presetName)
		}

		serviceName := fmt.Sprintf("%s-%d", presetName, len(cfg.Keys)+1)
		for _, k := range cfg.Keys {
			if k.Provider == presetName || k.Name == serviceName {
				serviceName = fmt.Sprintf("%s-%d", presetName, len(cfg.Keys)+2)
			}
		}

		newKey = config.KeyConfig{
			Provider: presetName,
			Name:     serviceName,
			BaseURL:  preset.BaseURL,
			Format:   preset.Format,
			Secret:   secret,
			Enabled:  true,
		}

		models := c.StringSlice("model")
		if len(models) > 0 {
			for _, m := range models {
				newKey.Models = append(newKey.Models, config.ModelConfig{Name: m})
			}
		} else {
			for _, pm := range preset.DefaultModels {
				newKey.Models = append(newKey.Models, config.ModelConfig{
					Name:  pm.Name,
					Alias: pm.Alias,
				})
			}
		}

		pterm.Success.Printf("Adding service '%s' (preset: %s)\n", serviceName, presetName)
	} else {
		// Custom mode
		name := c.String("name")
		baseURL := c.String("base-url")
		format := c.String("format")

		if name == "" {
			return fmt.Errorf("--name is required for custom service")
		}
		if baseURL == "" {
			return fmt.Errorf("--base-url is required for custom service")
		}
		if format == "" {
			return fmt.Errorf("--format is required for custom service")
		}
		if !config.IsValidFormat(format) {
			return fmt.Errorf("invalid format: %s (must be openai/anthropic/gemini/cohere)", format)
		}
		if cfg.FindKey(name) != nil {
			return fmt.Errorf("service '%s' already exists", name)
		}

		newKey = config.KeyConfig{
			Name:    name,
			BaseURL: baseURL,
			Format:  format,
			Secret:  secret,
			Enabled: true,
		}

		for _, m := range c.StringSlice("model") {
			newKey.Models = append(newKey.Models, config.ModelConfig{Name: m})
		}

		pterm.Success.Printf("Adding custom service '%s'\n", name)
	}

	cfg.Keys = append(cfg.Keys, newKey)

	if err := config.Save(configPath, cfg); err != nil {
		return err
	}

	pterm.Success.Printf("Service added successfully. Models: %d\n", len(newKey.Models))
	pterm.Println("\nStart tokrouter to use:")
	pterm.Println("  tr start")
	return nil
}

func runAddInteractive(c *cli.Command) error {
	pterm.DefaultSection.Println("Add a service")

	presets := config.ListPresets()

	// Build selection: all presets + "Custom (manual)"
	options := make([]string, len(presets)+1)
	for i, p := range presets {
		options[i] = fmt.Sprintf("%s  (%s)", p.Name, p.DisplayName)
	}
	customLabel := "Custom (manual config)"
	options[len(presets)] = customLabel

	var selected string
	selectPrompt := &survey.Select{
		Message: "Select provider:",
		Options: options,
	}
	if err := survey.AskOne(selectPrompt, &selected); err != nil {
		return err
	}

	var serviceName, baseURL, format, secret string
	var models []config.ModelConfig

	if selected == customLabel {
		// Custom service
		namePrompt := &survey.Input{
			Message: "Service name:",
		}
		if err := survey.AskOne(namePrompt, &serviceName); err != nil || serviceName == "" {
			return fmt.Errorf("service name is required")
		}

		formatSelect := &survey.Select{
			Message: "API format:",
			Options: []string{config.FormatOpenAI, config.FormatAnthropic, config.FormatGemini, config.FormatCohere},
			Default: config.FormatOpenAI,
		}
		if err := survey.AskOne(formatSelect, &format); err != nil {
			return err
		}

		urlPrompt := &survey.Input{
			Message: "Base URL:",
			Default: getDefaultURL(format),
		}
		if err := survey.AskOne(urlPrompt, &baseURL); err != nil || baseURL == "" {
			return fmt.Errorf("base URL is required")
		}

		secretPrompt := &survey.Password{
			Message: "API Key:",
		}
		if err := survey.AskOne(secretPrompt, &secret); err != nil || secret == "" {
			return fmt.Errorf("API key is required")
		}

		// Add models
		pterm.Println()
		pterm.DefaultSection.WithLevel(2).Println("Add models")
		for {
			var modelName string
			modelPrompt := &survey.Input{
				Message: "Model name (empty to finish):",
			}
			if err := survey.AskOne(modelPrompt, &modelName); err != nil || modelName == "" {
				break
			}
			models = append(models, config.ModelConfig{Name: modelName})
		}
	} else {
		// Extract preset name from selection
		presetIdx := -1
		for i, opt := range options {
			if opt == selected {
				presetIdx = i
				break
			}
		}
		presetName := presets[presetIdx].Name

		preset, err := config.GetPreset(presetName)
		if err != nil {
			return err
		}

		secretPrompt := &survey.Password{
			Message: fmt.Sprintf("API Key for %s:", preset.DisplayName),
		}
		if err := survey.AskOne(secretPrompt, &secret); err != nil || secret == "" {
			return fmt.Errorf("API key is required")
		}

		configPath := getConfigPath(c)
		cfg, _ := config.Load(configPath)
		serviceName = fmt.Sprintf("%s-%d", presetName, len(cfg.Keys)+1)

		baseURL = preset.BaseURL
		format = preset.Format

		for _, pm := range preset.DefaultModels {
			models = append(models, config.ModelConfig{Name: pm.Name, Alias: pm.Alias})
		}
	}

	// Save
	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if cfg.FindKey(serviceName) != nil {
		return fmt.Errorf("service '%s' already exists", serviceName)
	}

	newKey := config.KeyConfig{
		Name:    serviceName,
		BaseURL: baseURL,
		Format:  format,
		Secret:  secret,
		Enabled: true,
		Models:  models,
	}

	cfg.Keys = append(cfg.Keys, newKey)

	if err := config.Save(configPath, cfg); err != nil {
		return err
	}

	pterm.Success.Printf("Service '%s' added successfully\n", serviceName)
	pterm.Printf("Models: %d\n", len(models))
	pterm.Println("\nStart tokrouter:")
	pterm.Println("  tr start")
	return nil
}

// getDefaultURL returns a default base URL for a given format.
func getDefaultURL(format string) string {
	switch format {
	case config.FormatOpenAI:
		return "https://api.openai.com/v1"
	case config.FormatAnthropic:
		return "https://api.anthropic.com/v1"
	case config.FormatGemini:
		return "https://generativelanguage.googleapis.com/v1"
	case config.FormatCohere:
		return "https://api.cohere.ai/v1"
	default:
		return ""
	}
}
