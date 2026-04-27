package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/router"
	"github.com/tokzone/tokrouter/server"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
)

const portFile = "/tmp/tokrouter.port"

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

	// Write port file for stop command
	os.WriteFile(portFile, []byte(strconv.Itoa(cfg.Server.Port)), 0644)

	routerSvc, err := router.NewFromConfig(cfg)
	if err != nil {
		return err
	}

	pterm.Success.Printf("Starting tokrouter on %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	pterm.Printf("Config: %s\n", configPath)
	pterm.Printf("Services: %d\n", len(cfg.Keys))

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

	srv := server.NewServer(routerSvc, config.TraceConfig{
		Enabled: cfg.Trace.Enabled,
		Header:  cfg.Trace.Header,
	}, configPath)
	srv.Run()

	// Clean up port file on exit
	os.Remove(portFile)
	return nil
}

var stopCmd = &cli.Command{
	Name:  "stop",
	Usage: "Stop tokrouter server",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return runStop(cmd)
	},
}

func runStop(c *cli.Command) error {
	// Read port from file
	data, err := os.ReadFile(portFile)
	if err != nil {
		pterm.Info.Println("Port file not found. Is tokrouter running?")
		pterm.Println()
		pterm.Info.Println("If running in background, find and kill the process:")
		pterm.Println("  pkill tokrouter")
		return nil
	}

	port := strings.TrimSpace(string(data))
	url := fmt.Sprintf("http://127.0.0.1:%s/shutdown", port)

	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to reach server on port %s: %w\n\nIf the server is running in foreground, press Ctrl+C to stop.", port, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		pterm.Success.Println("Server stopped.")
	} else {
		pterm.Warning.Printf("Server responded with status %d\n", resp.StatusCode)
	}

	os.Remove(portFile)
	return nil
}
