package cli

import (
	"context"

	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/server"

	"github.com/urfave/cli/v3"
)

var serveCmd = &cli.Command{
	Name:  "serve",
	Usage: "Start HTTP server",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "port",
			Usage: "server port",
			Value: 8765,
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return runServe(cmd)
	},
}

func runServe(c *cli.Command) error {
	configPath := getConfigPath(c)
	port := c.Int("port")

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if port != 8765 {
		cfg.Server.Port = port
	}

	routerSvc, err := createRouter(cfg)
	if err != nil {
		return err
	}

	// Create and run server (blocks)
	srv := server.NewServer(routerSvc, config.TraceConfig{
		Enabled: cfg.Trace.Enabled,
		Header:  cfg.Trace.Header,
	}, configPath)
	srv.Run()
	return nil
}
