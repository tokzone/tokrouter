package cli

import (
	"context"
	"fmt"

	"github.com/tokzone/tokrouter/config"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
)

var listCmd = &cli.Command{
	Name:  "list",
	Usage: "List resources",
	Commands: []*cli.Command{
		{
			Name:  "services",
			Usage: "List all configured services",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runListServices(cmd)
			},
		},
		{
			Name:  "models",
			Usage: "List all available models",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runListModels(cmd)
			},
		},
		{
			Name:  "presets",
			Usage: "List all provider presets",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runListPresets(cmd)
			},
		},
		{
			Name:  "assistants",
			Usage: "List supported AI assistants",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runListAssistants(cmd)
			},
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		// Default: list services
		return runListServices(cmd)
	},
}

func runListServices(c *cli.Command) error {
	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if len(cfg.Keys) == 0 {
		pterm.Info.Println("No services configured")
		pterm.Println("\nAdd a service:")
		pterm.Println("  tr add openai --secret sk-xxx")
		return nil
	}

	pterm.DefaultSection.Println("Services")

	tableData := [][]string{{"NAME", "PROVIDER", "FORMAT", "MODELS", "STATUS"}}
	for _, k := range cfg.Keys {
		status := "disabled"
		if k.Enabled {
			status = "enabled"
		}
		provider := k.Provider
		if provider == "" {
			provider = "custom"
		}
		tableData = append(tableData, []string{
			k.Name,
			provider,
			k.Format,
			fmt.Sprintf("%d", len(k.Models)),
			status,
		})
	}
	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	return nil
}

func runListModels(c *cli.Command) error {
	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if len(cfg.Keys) == 0 {
		pterm.Info.Println("No services configured")
		return nil
	}

	pterm.DefaultSection.Println("Available Models")

	tableData := [][]string{{"MODEL", "SERVICE", "FORMAT", "PRIORITY", "STATUS"}}
	for _, k := range cfg.Keys {
		status := "disabled"
		if k.Enabled {
			status = "enabled"
		}
		for _, m := range k.Models {
			priority := "-"
			if m.Priority > 0 {
				priority = fmt.Sprintf("%d", m.Priority)
			}
			tableData = append(tableData, []string{
				m.Name,
				k.Name,
				k.Format,
				priority,
				status,
			})
		}
	}
	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	return nil
}

func runListPresets(c *cli.Command) error {
	presets := config.ListPresets()

	pterm.DefaultSection.Println("Provider Presets")

	tableData := [][]string{{"NAME", "DISPLAY", "FORMAT", "REGION", "MODELS"}}
	for _, p := range presets {
		region := p.Region
		if region == "" {
			region = "global"
		}
		tableData = append(tableData, []string{
			p.Name,
			p.DisplayName,
			p.Format,
			region,
			fmt.Sprintf("%d", len(p.DefaultModels)),
		})
	}
	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()

	pterm.Println()
	pterm.Info.Println("Add a service using preset:")
	pterm.Println("  tr add openai --secret sk-xxx")
	pterm.Println("  tr add deepseek --secret sk-xxx")
	return nil
}

func runListAssistants(c *cli.Command) error {
	assistants := getAssistantConfigs()

	pterm.DefaultSection.Println("Supported AI Assistants")

	tableData := [][]string{{"NAME", "DISPLAY", "CONFIG TYPE", "INSTALL CHECK"}}
	for i := range assistants {
		installed := checkAssistantInstalled(&assistants[i])
		installStatus := "not installed"
		if installed {
			installStatus = "installed"
		}
		tableData = append(tableData, []string{
			assistants[i].Name,
			assistants[i].DisplayName,
			assistants[i].ConfigType,
			installStatus,
		})
	}
	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()

	pterm.Println()
	pterm.Info.Println("Configure an assistant:")
	pterm.Println("  tr assistant cursor")
	pterm.Println("  tr assistant auto  # auto-detect and configure all")
	return nil
}