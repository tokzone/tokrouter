package cli

import (
	"context"
	"strconv"

	"github.com/tokzone/tokrouter/config"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
)

var modelsCmd = &cli.Command{
	Name:  "models",
	Usage: "List all available models",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return runModels(cmd)
	},
}

func runModels(c *cli.Command) error {
	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if len(cfg.Keys) == 0 {
		pterm.Info.Println("No keys configured")
		return nil
	}

	pterm.DefaultSection.Println("Available Models")

	tableData := [][]string{{"MODEL", "PROVIDER", "FORMAT", "PRIORITY", "STATUS"}}

	for _, k := range cfg.Keys {
		status := "disabled"
		if k.Enabled {
			status = "enabled"
		}

		for _, m := range k.Models {
			tableData = append(tableData, []string{
				m.Name,
				k.Name,
				k.Format,
				formatPriority(m.Priority),
				status,
			})
		}
	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	pterm.Println()
	pterm.Info.Println("Use model name in API requests as the 'model' field")
	return nil
}

func formatPriority(p int64) string {
	if p == 0 {
		return "0 (default)"
	}
	return strconv.FormatInt(p, 10)
}