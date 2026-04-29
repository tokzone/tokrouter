package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

var version = "v0.7.3"

// Execute runs the CLI
func Execute() {
	app := &cli.Command{
		Name:                  "tokrouter",
		Aliases:               []string{"tr"},
		Usage:                 "LLM API Router - route API requests to multiple providers",
		Version:               version,
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "config file path",
				Value:   "./config.yaml",
			},
		},
		Commands: []*cli.Command{
			addCmd,
			removeCmd,
			listCmd,
			showCmd,
			configCmd,
			assistantCmd,
			startCmd,
			stopCmd,
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// getConfigPath gets config path from flag or environment
func getConfigPath(c *cli.Command) string {
	if path := c.String("config"); path != "" {
		return path
	}
	if envPath := os.Getenv("TOKROUTER_CONFIG"); envPath != "" {
		return envPath
	}
	return "./config.yaml"
}
