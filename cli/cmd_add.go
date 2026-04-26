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
	Usage: "Add a service (provider)",
	Commands: []*cli.Command{
		{
			Name:  "service",
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
				return runAddService(cmd)
			},
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		// Default: add service interactively
		return runAddServiceInteractive(cmd)
	},
}

func runAddService(c *cli.Command) error {
	args := c.Args()
	secret := c.String("secret")

	// Need either preset name (from args) or custom config flags
	if args.Len() == 0 && c.String("name") == "" {
		return runAddServiceInteractive(c)
	}

	// Validate secret is provided
	if secret == "" {
		return fmt.Errorf("--secret is required")
	}

	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	// Determine if using preset or custom
	var newKey config.KeyConfig

	if args.Len() > 0 {
		// Using preset
		presetName := args.First()
		preset, err := config.GetPreset(presetName)
		if err != nil {
			return fmt.Errorf("preset '%s' not found. Use 'tr list presets' to see available presets", presetName)
		}

		// Check if service with this preset already exists
		serviceName := fmt.Sprintf("%s-%d", presetName, len(cfg.Keys)+1)
		for _, k := range cfg.Keys {
			if k.Provider == presetName || k.Name == serviceName {
				pterm.Warning.Printf("Service with preset '%s' already exists as '%s'\n", presetName, k.Name)
				// Generate unique name
				serviceName = fmt.Sprintf("%s-%d", presetName, len(cfg.Keys)+1)
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

		// Add models from preset or from flags
		models := c.StringSlice("model")
		if len(models) > 0 {
			for _, m := range models {
				newKey.Models = append(newKey.Models, config.ModelConfig{Name: m})
			}
		} else {
			// Use default models from preset
			for _, pm := range preset.DefaultModels {
				newKey.Models = append(newKey.Models, config.ModelConfig{
					Name:  pm.Name,
					Alias: pm.Alias,
				})
			}
		}

		pterm.Success.Printf("Adding service '%s' (preset: %s)\n", serviceName, presetName)
	} else {
		// Custom service
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

		// Check if key already exists
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

		models := c.StringSlice("model")
		for _, m := range models {
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

func runAddServiceInteractive(c *cli.Command) error {
	pterm.DefaultSection.Println("Add a service")

	// Show available presets
	presets := config.ListPresets()
	pterm.Info.Println("Available presets:")
	for _, p := range presets[:min(10, len(presets))] {
		pterm.Printf("  - %s (%s)\n", p.Name, p.DisplayName)
	}
	if len(presets) > 10 {
		pterm.Printf("  ... and %d more (use 'tr list presets' to see all)\n", len(presets)-10)
	}
	pterm.Println()

	// Ask if using preset or custom
	var usePreset bool
	presetPrompt := &survey.Confirm{
		Message: "Use a preset provider?",
		Default: true,
	}
	if err := survey.AskOne(presetPrompt, &usePreset); err != nil {
		return err
	}

	var serviceName, baseURL, format, secret string
	var models []config.ModelConfig

	if usePreset {
		// Select preset
		var presetName string
		presetNames := make([]string, len(presets))
		for i, p := range presets {
			presetNames[i] = p.Name
		}
		presetSelect := &survey.Select{
			Message: "Select preset:",
			Options: presetNames,
		}
		if err := survey.AskOne(presetSelect, &presetName); err != nil {
			return err
		}

		preset, err := config.GetPreset(presetName)
		if err != nil {
			return err
		}

		// Secret
		secretPrompt := &survey.Password{
			Message: fmt.Sprintf("API Key for %s:", preset.DisplayName),
		}
		if err := survey.AskOne(secretPrompt, &secret); err != nil || secret == "" {
			return fmt.Errorf("API key is required")
		}

		// Auto-generate name
		configPath := getConfigPath(c)
		cfg, _ := config.Load(configPath)
		serviceName = fmt.Sprintf("%s-%d", presetName, len(cfg.Keys)+1)

		baseURL = preset.BaseURL
		format = preset.Format

		// Use preset models
		pterm.Info.Printf("Default models: %d\n", len(preset.DefaultModels))
		for _, pm := range preset.DefaultModels {
			models = append(models, config.ModelConfig{Name: pm.Name, Alias: pm.Alias})
		}

	} else {
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

		// Add models interactively
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
	}

	// Save
	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	// Check if name exists
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
	if usePreset {
		newKey.Provider = serviceName[:len(serviceName)-2] // Extract preset name
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}