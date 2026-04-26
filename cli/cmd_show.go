package cli

import (
	"context"
	"fmt"

	"github.com/tokzone/tokrouter/config"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
)

var showCmd = &cli.Command{
	Name:  "show",
	Usage: "Show details",
	Commands: []*cli.Command{
		{
			Name:  "service",
			Usage: "Show service details",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runShowService(cmd)
			},
		},
		{
			Name:  "preset",
			Usage: "Show preset details",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runShowPreset(cmd)
			},
		},
		{
			Name:  "config",
			Usage: "Show full configuration",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runShowConfig(cmd)
			},
		},
		{
			Name:  "status",
			Usage: "Show server status",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runShowStatus(cmd)
			},
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		// Default: show config
		return runShowConfig(cmd)
	},
}

func runShowService(c *cli.Command) error {
	args := c.Args()
	if args.Len() == 0 {
		return fmt.Errorf(`Service name is required.

Usage: tr show service <name>
Example: tr show service openai-1`)
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

	pterm.DefaultSection.Printf("Service: %s", name)
	pterm.Printf("  Provider:  %s\n", key.Provider)
	pterm.Printf("  Format:    %s\n", key.Format)
	pterm.Printf("  BaseURL:   %s\n", key.BaseURL)
	pterm.Printf("  Status:    %s\n", func() string {
		if key.Enabled {
			return pterm.FgGreen.Sprint("enabled")
		}
		return pterm.FgRed.Sprint("disabled")
	}())
	pterm.Printf("  Models:    %d\n", len(key.Models))

	if len(key.Models) > 0 {
		pterm.Println()
		pterm.DefaultSection.WithLevel(2).Println("Models")
		tableData := [][]string{{"NAME", "ALIAS", "PRIORITY"}}
		for _, m := range key.Models {
			alias := "-"
			if m.Alias != "" {
				alias = m.Alias
			}
			priority := "-"
			if m.Priority > 0 {
				priority = fmt.Sprintf("%d", m.Priority)
			}
			tableData = append(tableData, []string{m.Name, alias, priority})
		}
		pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	}

	pterm.Println()
	pterm.Info.Println("Actions:")
	pterm.Println("  tr config service " + name + " --enable")
	pterm.Println("  tr config service " + name + " --disable")
	pterm.Println("  tr config service " + name + " --secret sk-new")
	pterm.Println("  tr remove service " + name)
	return nil
}

func runShowPreset(c *cli.Command) error {
	args := c.Args()
	if args.Len() == 0 {
		return fmt.Errorf(`Preset name is required.

Usage: tr show preset <name>
Example: tr show preset openai

Use 'tr list presets' to see all presets.`)
	}

	name := args.First()
	preset, err := config.GetPreset(name)
	if err != nil {
		return fmt.Errorf("preset '%s' not found. Use 'tr list presets' to see all presets.", name)
	}

	pterm.DefaultSection.Printf("Preset: %s", name)
	pterm.Printf("  Display:   %s\n", preset.DisplayName)
	pterm.Printf("  BaseURL:   %s\n", preset.BaseURL)
	pterm.Printf("  Format:    %s\n", preset.Format)
	pterm.Printf("  Region:    %s\n", preset.Region)
	pterm.Printf("  DocURL:    %s\n", preset.DocURL)

	if len(preset.DefaultModels) > 0 {
		pterm.Println()
		pterm.DefaultSection.WithLevel(2).Println("Default Models")
		tableData := [][]string{{"NAME", "ALIAS", "CONTEXT"}}
		for _, m := range preset.DefaultModels {
			alias := "-"
			if m.Alias != "" {
				alias = m.Alias
			}
			ctx := "-"
			if m.Context > 0 {
				ctx = fmt.Sprintf("%d", m.Context)
			}
			tableData = append(tableData, []string{m.Name, alias, ctx})
		}
		pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	}

	pterm.Println()
	pterm.Info.Println("Add service using this preset:")
	pterm.Printf("  tr add service %s --secret YOUR_API_KEY\n", name)
	return nil
}

func runShowConfig(c *cli.Command) error {
	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	pterm.DefaultSection.Println("Configuration")
	pterm.Printf("  Server:    %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	pterm.Printf("  Services:  %d configured\n", len(cfg.Keys))
	pterm.Printf("  Stats:     enabled=%v, db=%s\n", cfg.Stats.Enabled, cfg.Stats.DBPath)
	pterm.Printf("  Trace:     enabled=%v, header=%s\n", cfg.Trace.Enabled, cfg.Trace.Header)
	pterm.Printf("  Retry:     max=%d, timeout=%s\n", cfg.Router.Retry.MaxRetries, cfg.Router.Retry.Timeout)

	pterm.Println()
	pterm.Info.Printf("Config file: %s\n", configPath)
	return nil
}

func runShowStatus(c *cli.Command) error {
	// Check if server is running
	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	host := cfg.Server.Host
	port := cfg.Server.Port
	url := fmt.Sprintf("http://%s:%d/v1/models", host, port)

	pterm.DefaultSection.Println("Server Status")
	pterm.Printf("  Address:   %s:%d\n", host, port)
	pterm.Printf("  URL:       %s\n", url)

	// Try to connect
	status := checkServerStatus(url)
	if status == "running" {
		pterm.Printf("  Status:    %s\n", pterm.FgGreen.Sprint("running"))
		pterm.Println()
		pterm.Info.Println("Server is running. Test connection:")
		pterm.Println("  curl " + url)
	} else {
		pterm.Printf("  Status:    %s\n", pterm.FgRed.Sprint("not running"))
		pterm.Println()
		pterm.Info.Println("Start the server:")
		pterm.Println("  tr start")
	}
	return nil
}