package cli

import (
	"context"

	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/router"
	"github.com/tokzone/tokrouter/server"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
)

var startCmd = &cli.Command{
	Name:  "start",
	Usage: "Start tokrouter server",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "host",
			Usage: "server host",
			Value: "127.0.0.1",
		},
		&cli.IntFlag{
			Name:  "port",
			Usage: "server port",
			Value: 8765,
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return runStart(cmd)
	},
}

func runStart(c *cli.Command) error {
	configPath := getConfigPath(c)
	host := c.String("host")
	port := c.Int("port")

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if host != "127.0.0.1" {
		cfg.Server.Host = host
	}
	if port != 8765 {
		cfg.Server.Port = port
	}

	routerSvc, err := router.NewServiceFromConfig(cfg)
	if err != nil {
		return err
	}

	pterm.Success.Printf("Starting tokrouter on %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	pterm.Printf("Config: %s\n", configPath)
	pterm.Printf("Services: %d\n", len(cfg.Keys))

	// Calculate total models
	totalModels := 0
	for _, k := range cfg.Keys {
		if k.Enabled {
			totalModels += len(k.Models)
		}
	}
	pterm.Printf("Models: %d\n", totalModels)

	pterm.Println()
	pterm.Info.Println("Press Ctrl+C to stop")
	pterm.Println()

	// Create and run server (Run() handles signals internally)
	srv := server.NewServer(routerSvc, config.TraceConfig{
		Enabled: cfg.Trace.Enabled,
		Header:  cfg.Trace.Header,
	}, configPath)
	srv.Run()
	return nil
}

var stopCmd = &cli.Command{
	Name:  "stop",
	Usage: "Stop tokrouter server (if running)",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return runStop(cmd)
	},
}

func runStop(c *cli.Command) error {
	// TODO: Implement stop via PID file or signal
	pterm.Info.Println("Stop not fully implemented yet")
	pterm.Info.Println("If tokrouter is running in foreground, press Ctrl+C to stop")
	pterm.Info.Println("If running in background, find and kill the process:")
	pterm.Println("  ps aux | grep tokrouter")
	pterm.Println("  kill <PID>")
	return nil
}