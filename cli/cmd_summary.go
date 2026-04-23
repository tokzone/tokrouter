package cli

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/usage"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
)

var summaryCmd = &cli.Command{
	Name:  "summary",
	Usage: "Show usage statistics",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "today",
			Usage: "show today's stats",
		},
		&cli.BoolFlag{
			Name:  "week",
			Usage: "show this week's stats",
		},
		&cli.BoolFlag{
			Name:  "month",
			Usage: "show this month's stats",
		},
		&cli.StringFlag{
			Name:  "export",
			Usage: "export format (csv/json)",
		},
		&cli.BoolFlag{
			Name:  "chart",
			Usage: "show bar chart for usage distribution",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return runSummary(cmd)
	},
}

func runSummary(c *cli.Command) error {
	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if !cfg.Stats.Enabled {
		return fmt.Errorf("stats feature is disabled")
	}

	routerSvc, err := createRouter(cfg)
	if err != nil {
		return err
	}
	defer routerSvc.Close()

	// Determine period
	period := "month"
	if c.Bool("today") {
		period = "today"
	} else if c.Bool("week") {
		period = "week"
	}

	// Determine time range
	var start, end time.Time
	switch period {
	case "today":
		start, end = usage.DayRange(time.Now())
	case "week":
		start, end = usage.WeekRange()
	case "month":
		start, end = usage.MonthRange()
	}

	// Query stats
	filter := usage.QueryFilter{
		Start:   start,
		End:     end,
		GroupBy: usage.GroupByProvider,
	}

	stats, err := queryStats(routerSvc, filter)
	if err != nil {
		return err
	}

	// Output based on format
	exportFormat := c.String("export")
	showChart := c.Bool("chart")
	switch exportFormat {
	case "csv":
		exportCSV(stats, period)
	case "json":
		exportJSON(stats, period)
	default:
		if showChart {
			printSummaryChart(stats, period)
		}
		printSummaryTable(stats, period)
	}
	return nil
}

func printSummaryTable(stats []usage.StatRow, period string) {
	pterm.DefaultSection.Printf("Usage Summary (%s)", period)

	tableData := [][]string{
		{"KEY", "INPUT", "OUTPUT", "COST", "REQUESTS"},
	}

	for _, row := range stats {
		tableData = append(tableData, []string{
			row.GroupKey,
			fmt.Sprintf("%d", row.InputTokens),
			fmt.Sprintf("%d", row.OutputTokens),
			fmt.Sprintf("$%.4f", float64(row.Cost)/10000),
			fmt.Sprintf("%d", row.RequestCount),
		})
	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

func printSummaryChart(stats []usage.StatRow, period string) {
	if len(stats) == 0 {
		return
	}

	// Sort by InputTokens descending
	sorted := make([]usage.StatRow, len(stats))
	copy(sorted, stats)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].InputTokens > sorted[j].InputTokens
	})

	// Build bar chart data
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

func exportCSV(stats []usage.StatRow, period string) {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Header
	writer.Write([]string{"key", "input_tokens", "output_tokens", "cost", "requests"})

	// Data
	for _, row := range stats {
		writer.Write([]string{
			row.GroupKey,
			fmt.Sprintf("%d", row.InputTokens),
			fmt.Sprintf("%d", row.OutputTokens),
			fmt.Sprintf("%.4f", float64(row.Cost)/10000),
			fmt.Sprintf("%d", row.RequestCount),
		})
	}
}

func exportJSON(stats []usage.StatRow, period string) {
	output := map[string]interface{}{
		"period": period,
		"stats":  stats,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(output)
}
