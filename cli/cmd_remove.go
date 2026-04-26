package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/tokzone/tokrouter/config"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
)

var removeCmd = &cli.Command{
	Name:  "remove",
	Usage: "Remove a service",
	Commands: []*cli.Command{
		{
			Name:  "service",
			Usage: "Remove a service by name",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runRemoveService(cmd)
			},
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		// Default: remove service
		return runRemoveService(cmd)
	},
}

func runRemoveService(c *cli.Command) error {
	args := c.Args()
	if args.Len() == 0 {
		return fmt.Errorf(`Service name is required.

Usage: tr remove service <name>
Example: tr remove service openai-1

Use 'tr list services' to see all services.`)
	}

	name := args.First()
	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	idx := cfg.FindKeyIndex(name)
	if idx < 0 {
		var serviceNames []string
		for _, k := range cfg.Keys {
			serviceNames = append(serviceNames, k.Name)
		}
		return fmt.Errorf(`Service '%s' not found.

Available services: %s

Use 'tr list services' to see all services.`, name, strings.Join(serviceNames, ", "))
	}

	cfg.Keys = append(cfg.Keys[:idx], cfg.Keys[idx+1:]...)

	if err := config.Save(configPath, cfg); err != nil {
		return err
	}

	pterm.Success.Printf("Service '%s' removed successfully\n", name)
	pterm.Println("\nNote: If tokrouter is running, restart it to apply the change:")
	pterm.Println("  tr stop")
	pterm.Println("  tr start")
	return nil
}