package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

var version = "dev"

// Execute runs the CLI
func Execute() {
	app := &cli.Command{
		Name:    "tokrouter",
		Usage:   "LLM API Router (Personal)",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "config file path",
				Value:   "./config.yaml",
			},
		},
		Commands: []*cli.Command{
			initCmd,
			serveCmd,
			statusCmd,
			keysCmd,
			summaryCmd,
			configCmd,
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// getConfigPath gets config path from flag or environment
func getConfigPath(c *cli.Command) string {
	// From flag
	if path := c.String("config"); path != "" {
		return path
	}
	// From environment
	if envPath := os.Getenv("TOKROUTER_CONFIG"); envPath != "" {
		return envPath
	}
	// Default
	return "./config.yaml"
}
