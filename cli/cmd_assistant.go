package cli

import (
	"context"
	"fmt"

	"github.com/tokzone/tokrouter/config"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
)

var assistantCmd = &cli.Command{
	Name:  "assistant",
	Usage: "Configure AI assistants to use tokrouter",
	Commands: []*cli.Command{
		{
			Name:  "list",
			Usage: "List supported AI assistants",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runAssistantList(cmd)
			},
		},
		{
			Name:  "auto",
			Usage: "Auto-detect and configure all installed assistants",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "url",
					Usage: "Custom tokrouter URL",
					Value: "http://localhost:8765",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runAssistantAuto(cmd)
			},
		},
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "url",
			Usage: "Custom tokrouter URL",
			Value: "http://localhost:8765",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		args := cmd.Args()
		if args.Len() == 0 {
			pterm.Info.Println("Specify an assistant name or use a subcommand:")
			pterm.Println("  tr assistant list                  List supported assistants")
			pterm.Println("  tr assistant auto [--url URL]      Auto-detect and configure all")
			pterm.Println("  tr assistant <name> [--url URL]    Configure a specific assistant")
			pterm.Println()
			pterm.Println("Supported names: claude-code, cursor, aider, windsurf, cline, continue, codex")
			return nil
		}
		return runAssistant(cmd)
	},
}

func runAssistantList(c *cli.Command) error {
	assistants := getAssistantConfigs()

	pterm.DefaultSection.Println("Supported AI Assistants")

	tableData := [][]string{{"NAME", "DISPLAY", "CONFIG TYPE", "INSTALL CHECK"}}
	for i := range assistants {
		installed := checkAssistantInstalled(&assistants[i])
		status := "not installed"
		if installed {
			status = "installed"
		}
		tableData = append(tableData, []string{
			assistants[i].Name,
			assistants[i].DisplayName,
			assistants[i].ConfigType,
			status,
		})
	}
	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()

	pterm.Println()
	pterm.Info.Println("Configure an assistant:")
	pterm.Println("  tr assistant cursor")
	pterm.Println("  tr assistant auto  # auto-detect and configure all")
	return nil
}

func runAssistant(c *cli.Command) error {
	name := c.Args().First()
	url := c.String("url")

	a := findAssistant(name)
	if a == nil {
		return fmt.Errorf("assistant '%s' not supported. Use 'tr assistant list' to see supported assistants.", name)
	}

	if !checkAssistantInstalled(a) {
		pterm.Warning.Printf("Assistant '%s' may not be installed (config file not found)\n", name)
		if !askConfirm("Continue anyway?", true) {
			return nil
		}
	}

	model, err := selectAssistantModel(c)
	if err != nil {
		pterm.Warning.Printf("Model selection skipped: %v\n", err)
		model = ""
	}

	pterm.DefaultSection.Printf("Configuring %s", a.DisplayName)
	pterm.Printf("  Tokrouter URL: %s\n", url)
	if model != "" {
		pterm.Printf("  Default model: %s\n", model)
	}

	configPath := getAssistantConfigPath(a)
	pterm.Printf("  Config file:   %s\n", configPath)

	content := generateAssistantConfig(a, url, model)
	pterm.Println()
	pterm.Info.Println("Will write:")
	pterm.Println(content)
	pterm.Println()

	if !askConfirm("Confirm write?", true) {
		pterm.Info.Println("Cancelled")
		return nil
	}

	if err := writeAssistantConfig(a, url, model); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	pterm.Success.Printf("Config written to %s\n", configPath)
	pterm.Println()
	pterm.Info.Printf("Restart %s to use tokrouter\n", a.DisplayName)
	pterm.Println(getAssistantRestartHint(a))

	if a.ModelNote != "" {
		note := resolveTemplate(a.ModelNote, url, model)
		pterm.Println()
		pterm.Info.Println(note)
	}
	return nil
}

func runAssistantAuto(c *cli.Command) error {
	url := c.String("url")
	assistants := getAssistantConfigs()

	pterm.DefaultSection.Println("Auto-detecting installed assistants")

	installed := []AssistantConfig{}
	for _, a := range assistants {
		if checkAssistantInstalled(&a) {
			pterm.Success.Printf("  ✓ %s (installed)\n", a.DisplayName)
			installed = append(installed, a)
		} else {
			pterm.Printf("  ○ %s (not detected)\n", a.DisplayName)
		}
	}

	if len(installed) == 0 {
		pterm.Info.Println("No installed assistants detected")
		return nil
	}

	model := findDefaultModel(c)

	pterm.Println()
	if model != "" {
		pterm.Info.Printf("Will configure %d assistants with URL: %s (model: %s)\n", len(installed), url, model)
	} else {
		pterm.Info.Printf("Will configure %d assistants with URL: %s\n", len(installed), url)
	}

	if !askConfirm("Continue?", true) {
		pterm.Info.Println("Cancelled")
		return nil
	}

	pterm.Println()
	for _, a := range installed {
		if err := writeAssistantConfig(&a, url, model); err != nil {
			pterm.Error.Printf("Failed to configure %s: %v\n", a.DisplayName, err)
			continue
		}
		pterm.Success.Printf("✓ %s: configured\n", a.DisplayName)
		if a.ModelNote != "" {
			note := resolveTemplate(a.ModelNote, url, model)
			pterm.Printf("  Hint: %s\n", note)
		}
	}

	pterm.Println()
	pterm.Info.Println("Restart the assistants to use tokrouter:")
	for _, a := range installed {
		pterm.Printf("  %s: %s\n", a.DisplayName, getAssistantRestartHint(&a))
	}
	return nil
}

// findDefaultModel reads the config and returns the first model from the first enabled key.
// Returns empty string if no config or no models found.
func findDefaultModel(c *cli.Command) string {
	cfg, err := config.Load(getConfigPath(c))
	if err != nil || cfg == nil {
		return ""
	}
	for _, k := range cfg.Keys {
		if !k.Enabled {
			continue
		}
		if len(k.Models) > 0 {
			return k.Models[0].Name
		}
	}
	return ""
}
