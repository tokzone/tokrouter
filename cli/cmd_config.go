package cli

import (
	"context"
	"fmt"

	"github.com/tokzone/tokrouter/config"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
)

var configCmd = &cli.Command{
	Name:  "config",
	Usage: "Configure service or assistant",
	Commands: []*cli.Command{
		{
			Name:  "service",
			Usage: "Configure service settings",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "secret",
					Usage: "Update API key secret",
				},
				&cli.BoolFlag{
					Name:  "enable",
					Usage: "Enable the service",
				},
				&cli.BoolFlag{
					Name:  "disable",
					Usage: "Disable the service",
				},
				&cli.StringFlag{
					Name:  "add-model",
					Usage: "Add a model to the service",
				},
				&cli.StringFlag{
					Name:  "remove-model",
					Usage: "Remove a model from the service",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runConfigService(cmd)
			},
		},
		{
			Name:  "assistant",
			Usage: "Configure AI assistant to use tokrouter",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "auto",
					Usage: "Auto-detect and configure all installed assistants",
				},
				&cli.StringFlag{
					Name:  "url",
					Usage: "Custom tokrouter URL (default: http://localhost:8765)",
					Value: "http://localhost:8765",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runConfigAssistant(cmd)
			},
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		// Default: show usage
		pterm.Info.Println("Specify what to configure:")
		pterm.Println("  tr config service <name> --enable/--disable/--secret/--add-model/--remove-model")
		pterm.Println("  tr config assistant <name>")
		pterm.Println("  tr config assistant --auto")
		return nil
	},
}

func runConfigService(c *cli.Command) error {
	args := c.Args()
	if args.Len() == 0 {
		return fmt.Errorf(`Service name is required.

Usage: tr config service <name> [flags]
Example: tr config service openai-1 --enable`)
	}

	name := args.First()
	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	key := cfg.FindKey(name)
	if key == nil {
		return fmt.Errorf("service '%s' not found. Use 'tr list services' to see all services.", name)
	}

	// Track changes
	changes := 0

	// Update secret
	if c.String("secret") != "" {
		key.Secret = c.String("secret")
		pterm.Success.Printf("Updated secret for '%s'\n", name)
		changes++
	}

	// Enable/disable
	if c.Bool("enable") && c.Bool("disable") {
		return fmt.Errorf("cannot use both --enable and --disable")
	}
	if c.Bool("enable") {
		key.Enabled = true
		pterm.Success.Printf("Enabled '%s'\n", name)
		changes++
	}
	if c.Bool("disable") {
		key.Enabled = false
		pterm.Success.Printf("Disabled '%s'\n", name)
		changes++
	}

	// Add model
	if c.String("add-model") != "" {
		modelName := c.String("add-model")
		if key.HasModel(modelName) {
			pterm.Warning.Printf("Model '%s' already exists in '%s'\n", modelName, name)
		} else {
			key.AddModel(modelName)
			pterm.Success.Printf("Added model '%s' to '%s'\n", modelName, name)
			changes++
		}
	}

	// Remove model
	if c.String("remove-model") != "" {
		modelName := c.String("remove-model")
		if key.RemoveModel(modelName) {
			pterm.Success.Printf("Removed model '%s' from '%s'\n", modelName, name)
			changes++
		} else {
			pterm.Warning.Printf("Model '%s' not found in '%s'\n", modelName, name)
		}
	}

	// Save if any changes
	if changes > 0 {
		// Update the key in config
		idx := cfg.FindKeyIndex(name)
		cfg.Keys[idx] = *key

		if err := config.Save(configPath, cfg); err != nil {
			return err
		}

		pterm.Println()
		pterm.Info.Println("Changes saved. Restart tokrouter to apply:")
		pterm.Println("  tr stop")
		pterm.Println("  tr start")
	} else {
		pterm.Info.Println("No changes specified")
	}

	return nil
}

func runConfigAssistant(c *cli.Command) error {
	url := c.String("url")

	if c.Bool("auto") {
		return runConfigAssistantAuto(c, url)
	}

	args := c.Args()
	if args.Len() == 0 {
		return fmt.Errorf(`Assistant name is required.

Usage: tr config assistant <name>
Example: tr config assistant cursor

Use 'tr list assistants' to see supported assistants.`)
	}

	name := args.First()
	assistant := findAssistant(name)
	if assistant == nil {
		return fmt.Errorf("assistant '%s' not supported. Use 'tr list assistants' to see supported assistants.", name)
	}

	// Check if installed
	if !checkAssistantInstalled(assistant) {
		pterm.Warning.Printf("Assistant '%s' may not be installed (config file not found)\n", name)
		pterm.Info.Println("Continue anyway? (creates config file if needed)")
		if !askConfirm("Continue?", true) {
			return nil
		}
	}

	// Show what will be written
	pterm.DefaultSection.Printf("Configuring %s", assistant.DisplayName)
	pterm.Printf("  Tokrouter URL: %s\n", url)

	// Get config file path
	configPath := getAssistantConfigPath(assistant)
	pterm.Printf("  Config file:   %s\n", configPath)

	// Show content preview
	content := generateAssistantConfig(assistant, url)
	pterm.Println()
	pterm.Info.Println("Will write:")
	pterm.Println(content)
	pterm.Println()

	// Confirm
	if !askConfirm("Confirm write?", true) {
		pterm.Info.Println("Cancelled")
		return nil
	}

	// Write config
	if err := writeAssistantConfig(assistant, url); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	pterm.Success.Printf("Config written to %s\n", configPath)
	pterm.Println()
	pterm.Info.Println("Restart the assistant to use tokrouter:")
	pterm.Println(getAssistantRestartHint(assistant))
	return nil
}

func runConfigAssistantAuto(c *cli.Command, url string) error {
	assistants := getAssistantConfigs()

	pterm.DefaultSection.Println("Auto-detecting installed assistants")

	installed := []AssistantConfig{}
	for _, a := range assistants {
		if checkAssistantInstalled(&a) {
			pterm.Success.Printf("  ✓ %s (installed)\n", a.DisplayName)
			installed = append(installed, a)
		} else {
			pterm.Printf("  ○ %s (not installed)\n", a.DisplayName)
		}
	}

	if len(installed) == 0 {
		pterm.Info.Println("No installed assistants detected")
		return nil
	}

	pterm.Println()
	pterm.Info.Printf("Will configure %d assistants with URL: %s\n", len(installed), url)

	if !askConfirm("Continue?", true) {
		pterm.Info.Println("Cancelled")
		return nil
	}

	pterm.Println()
	for _, a := range installed {
		if err := writeAssistantConfig(&a, url); err != nil {
			pterm.Error.Printf("Failed to configure %s: %v\n", a.DisplayName, err)
			continue
		}
		pterm.Success.Printf("✓ %s: configured\n", a.DisplayName)
	}

	pterm.Println()
	pterm.Info.Println("Restart the assistants to use tokrouter:")
	for _, a := range installed {
		pterm.Printf("  %s: %s\n", a.DisplayName, getAssistantRestartHint(&a))
	}
	return nil
}