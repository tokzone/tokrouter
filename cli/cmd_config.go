package cli

import (
	"context"
	"fmt"

	"tokflux/tokrouter/config"

	"github.com/urfave/cli/v3"
)

var configCmd = &cli.Command{
	Name:  "config",
	Usage: "Show configuration",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return runConfigShow(cmd)
	},
}

func runConfigShow(c *cli.Command) error {
	configPath := getConfigPath(c)

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	fmt.Printf("Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Keys: %d configured\n", len(cfg.Keys))
	for _, k := range cfg.Keys {
		fmt.Printf("  - %s (%s): %d models\n", k.Name, k.Format, len(k.Models))
	}
	fmt.Printf("Stats: enabled=%v, db=%s\n", cfg.Stats.Enabled, cfg.Stats.DBPath)
	fmt.Printf("Retry: max=%d, timeout=%s\n", cfg.Router.Retry.MaxRetries, cfg.Router.Retry.Timeout)
	return nil
}
