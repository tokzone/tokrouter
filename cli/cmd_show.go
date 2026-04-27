package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/router"
	"github.com/tokzone/tokrouter/usage"

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
			Usage: "Show current configuration",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runShowConfig(cmd)
			},
		},
		{
			Name:  "health",
			Usage: "Show endpoint health status",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "watch",
					Aliases: []string{"w"},
					Usage:   "Watch mode: refresh every 2 seconds",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runShowHealth(cmd)
			},
		},
		{
			Name:  "usage",
			Usage: "Show usage statistics",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "today",
					Usage: "Show today's stats",
				},
				&cli.BoolFlag{
					Name:  "week",
					Usage: "Show this week's stats",
				},
				&cli.BoolFlag{
					Name:  "month",
					Usage: "Show this month's stats (default)",
				},
				&cli.StringFlag{
					Name:  "export",
					Usage: "Export format (csv/json)",
				},
				&cli.BoolFlag{
					Name:  "chart",
					Usage: "Show bar chart for usage distribution",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runShowUsage(cmd)
			},
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
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
	pterm.Println("  tr config " + name + " --enable")
	pterm.Println("  tr config " + name + " --disable")
	pterm.Println("  tr config " + name + " --secret sk-new")
	pterm.Println("  tr remove " + name)
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
	pterm.Printf("  tr add %s --secret YOUR_API_KEY\n", name)
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

// --- show health (from old cmd_status.go) ---

func runShowHealth(c *cli.Command) error {
	configPath := getConfigPath(c)

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	routerSvc, err := router.NewFromConfig(cfg)
	if err != nil {
		return err
	}
	defer routerSvc.Close()

	watch := c.Bool("watch")
	if watch {
		return runHealthWatch(routerSvc)
	}

	printHealth(routerSvc)
	return nil
}

func runHealthWatch(routerSvc router.Router) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	clearScreen()
	printHealth(routerSvc)

	for {
		select {
		case <-sigCh:
			pterm.Println()
			pterm.Info.Println("Watch mode stopped.")
			return nil
		case <-ticker.C:
			clearScreen()
			printHealth(routerSvc)
		}
	}
}

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func printHealth(routerSvc router.Router) {
	statuses := routerSvc.ProviderStatuses()

	pterm.DefaultSection.Println("Endpoint Health")

	tableData := [][]string{{"NAME", "PROTOCOL", "HEALTHY", "MODELS"}}
	for _, s := range statuses {
		healthy := "✓"
		healthyCount := 0
		for _, m := range s.Models {
			if m.Healthy {
				healthyCount++
			}
		}
		if healthyCount == 0 {
			healthy = "✗"
		}
		tableData = append(tableData, []string{
			s.Name,
			s.Protocol,
			healthy,
			fmt.Sprintf("%d/%d", healthyCount, len(s.Models)),
		})
	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	pterm.Println()
	pterm.Info.Println("Press Ctrl+C to exit watch mode")
}

// --- show usage (from old cmd_summary.go) ---

func runShowUsage(c *cli.Command) error {
	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if !cfg.Stats.Enabled {
		return fmt.Errorf("stats feature is disabled. Enable it in config.yaml: stats.enabled: true")
	}

	routerSvc, err := router.NewFromConfig(cfg)
	if err != nil {
		return err
	}
	defer routerSvc.Close()

	period := "month"
	if c.Bool("today") {
		period = "today"
	} else if c.Bool("week") {
		period = "week"
	}

	var start, end time.Time
	switch period {
	case "today":
		start, end = usage.DayRange(time.Now())
	case "week":
		start, end = usage.WeekRange()
	case "month":
		start, end = usage.MonthRange()
	}

	filter := usage.QueryFilter{
		Start:   start,
		End:     end,
		GroupBy: usage.GroupByProvider,
	}

	stats, err := routerSvc.Stats(filter)
	if err != nil {
		return err
	}

	exportFormat := c.String("export")
	showChart := c.Bool("chart")

	switch exportFormat {
	case "csv":
		exportUsageCSV(stats, period)
	case "json":
		exportUsageJSON(stats, period)
	default:
		if showChart {
			printUsageChart(stats, period)
		}
		printUsageTable(stats, period)
	}
	return nil
}

func printUsageTable(stats []usage.StatRow, period string) {
	pterm.DefaultSection.Printf("Usage Summary (%s)", period)

	tableData := [][]string{{"SERVICE", "INPUT", "OUTPUT", "REQUESTS", "AVG LATENCY", "SUCCESS"}}
	for _, row := range stats {
		tableData = append(tableData, []string{
			row.GroupKey,
			fmt.Sprintf("%d", row.InputTokens),
			fmt.Sprintf("%d", row.OutputTokens),
			fmt.Sprintf("%d", row.RequestCount),
			fmt.Sprintf("%dms", row.AvgLatency),
			fmt.Sprintf("%.1f%%", row.SuccessRate),
		})
	}
	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

func printUsageChart(stats []usage.StatRow, period string) {
	if len(stats) == 0 {
		return
	}

	sorted := make([]usage.StatRow, len(stats))
	copy(sorted, stats)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].InputTokens < sorted[j].InputTokens {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	bars := make(pterm.Bars, len(sorted))
	for i, row := range sorted {
		bars[i] = pterm.Bar{
			Label: row.GroupKey,
			Value: int(row.InputTokens),
		}
	}

	pterm.DefaultSection.Printf("Input Tokens Distribution (%s)", period)
	pterm.DefaultBarChart.WithBars(bars).WithShowValue().Render()
}

func exportUsageCSV(stats []usage.StatRow, period string) {
	fmt.Printf("# Usage stats: %s\n", period)
	fmt.Println("service,input_tokens,output_tokens,requests,avg_latency_ms,success_rate")
	for _, row := range stats {
		fmt.Printf("%s,%d,%d,%d,%d,%.1f\n",
			row.GroupKey, row.InputTokens, row.OutputTokens,
			row.RequestCount, row.AvgLatency, row.SuccessRate)
	}
}

func exportUsageJSON(stats []usage.StatRow, period string) {
	fmt.Printf("[period: %s]\n", period)
	for _, row := range stats {
		fmt.Printf("  %s: in=%d out=%d req=%d latency=%dms success=%.1f%%\n",
			row.GroupKey, row.InputTokens, row.OutputTokens,
			row.RequestCount, row.AvgLatency, row.SuccessRate)
	}
}
