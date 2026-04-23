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

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
)

var statusCmd = &cli.Command{
	Name:  "status",
	Usage: "Show key status",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "watch",
			Aliases: []string{"w"},
			Usage:   "Watch mode: refresh status every 2 seconds",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return runStatus(cmd)
	},
}

func runStatus(c *cli.Command) error {
	configPath := getConfigPath(c)

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	routerSvc, err := createRouter(cfg)
	if err != nil {
		return err
	}
	defer routerSvc.Close()

	watch := c.Bool("watch")
	if watch {
		return runStatusWatch(routerSvc)
	}

	printStatus(routerSvc)
	return nil
}

func runStatusWatch(routerSvc *router.Service) error {
	// Setup signal handling for graceful exit
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Print initial status
	clearScreen()
	printStatus(routerSvc)

	for {
		select {
		case <-sigCh:
			// Clear the "watching" message and exit cleanly
			pterm.Println()
			pterm.Info.Println("Watch mode stopped.")
			return nil
		case <-ticker.C:
			clearScreen()
			printStatus(routerSvc)
		}
	}
}

func clearScreen() {
	// ANSI escape codes: clear screen and move cursor to home position
	fmt.Print("\033[2J\033[H")
}

func printStatus(routerSvc *router.Service) {
	statuses := routerSvc.GetProviderStatuses()

	pterm.DefaultSection.Println("Key Status")

	tableData := [][]string{
		{"NAME", "PROTOCOL", "HEALTHY", "MODELS"},
	}

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
