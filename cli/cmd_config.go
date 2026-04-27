package cli

import (
	"context"
	"fmt"

	"github.com/tokzone/tokrouter/config"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
)

var configCmd = &cli.Command{
	Name:  "config",
	Usage: "Modify service configuration",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "secret",
			Usage: "Update API key secret",
		},
		&cli.BoolFlag{
			Name:  "enable",
			Usage: "Enable the service",
		},
		&cli.BoolFlag{
			Name:  "disable",
			Usage: "Disable the service",
		},
		&cli.StringFlag{
			Name:  "add-model",
			Usage: "Add a model to the service",
		},
		&cli.StringFlag{
			Name:  "remove-model",
			Usage: "Remove a model from the service",
		},
		&cli.StringFlag{
			Name:  "alias",
			Usage: "Set a model alias (use with --add-model)",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		args := cmd.Args()
		if args.Len() == 0 {
			pterm.Info.Println("Specify a service name to configure:")
			pterm.Println("  tr config <name> [flags]")
			pterm.Println()
			pterm.Println("Flags:")
			pterm.Println("  --secret VALUE       Update API key")
			pterm.Println("  --enable             Enable the service")
			pterm.Println("  --disable            Disable the service")
			pterm.Println("  --add-model NAME     Add a model to the service")
			pterm.Println("  --remove-model NAME  Remove a model from the service")
			pterm.Println("  --alias NAME         Set model alias (use with --add-model)")
			pterm.Println()
			pterm.Info.Println("To view configuration:")
			pterm.Println("  tr show config")
			return nil
		}
		return runConfig(cmd)
	},
}

func runConfig(c *cli.Command) error {
	name := c.Args().First()
	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	key := cfg.FindKey(name)
	if key == nil {
		return fmt.Errorf("service '%s' not found. Use 'tr list services' to see all services.", name)
	}

	changes := 0

	if s := c.String("secret"); s != "" {
		key.Secret = s
		pterm.Success.Printf("Updated secret for '%s'\n", name)
		changes++
	}

	if c.Bool("enable") && c.Bool("disable") {
		return fmt.Errorf("cannot use both --enable and --disable")
	}
	if c.Bool("enable") {
		key.Enabled = true
		pterm.Success.Printf("Enabled '%s'\n", name)
		changes++
	}
	if c.Bool("disable") {
		key.Enabled = false
		pterm.Success.Printf("Disabled '%s'\n", name)
		changes++
	}

	modelName := c.String("add-model")
	if modelName != "" {
		alias := c.String("alias")
		if key.HasModel(modelName) {
			pterm.Warning.Printf("Model '%s' already exists in '%s'\n", modelName, name)
		} else {
			key.Models = append(key.Models, config.ModelConfig{Name: modelName, Alias: alias})
			pterm.Success.Printf("Added model '%s' to '%s'\n", modelName, name)
			changes++
		}
	}

	if rmName := c.String("remove-model"); rmName != "" {
		if key.RemoveModel(rmName) {
			pterm.Success.Printf("Removed model '%s' from '%s'\n", rmName, name)
			changes++
		} else {
			pterm.Warning.Printf("Model '%s' not found in '%s'\n", rmName, name)
		}
	}

	if changes == 0 {
		pterm.Info.Println("No changes specified. Use --help to see available flags.")
		return nil
	}

	idx := cfg.FindKeyIndex(name)
	cfg.Keys[idx] = *key

	if err := config.Save(configPath, cfg); err != nil {
		return err
	}

	pterm.Println()
	pterm.Info.Println("Changes saved. Restart tokrouter to apply:")
	pterm.Println("  tr stop && tr start")
	return nil
}
