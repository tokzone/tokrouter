package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"tokflux/tokrouter/config"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
)

var keysCmd = &cli.Command{
	Name:  "keys",
	Usage: "Manage keys - default lists all keys",
	Commands: []*cli.Command{
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "List all keys",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runKeysList(cmd)
			},
		},
		{
			Name:  "add",
			Usage: "Add a new key",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "name",
					Usage:    "key name",
					Required: true,
				},
				&cli.StringFlag{
					Name:     "format",
					Usage:    "format (openai/anthropic/gemini/cohere)",
					Required: true,
				},
				&cli.StringFlag{
					Name:     "secret",
					Usage:    "API key secret",
					Required: true,
				},
				&cli.StringFlag{
					Name:     "base-url",
					Usage:    "API base URL",
					Required: true,
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runKeysAdd(cmd)
			},
		},
		{
			Name:    "remove",
			Aliases: []string{"rm", "delete"},
			Usage:   "Remove a key",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runKeysRemove(cmd)
			},
		},
		{
			Name:  "enable",
			Usage: "Enable a key",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runKeysEnable(cmd, true)
			},
		},
		{
			Name:  "disable",
			Usage: "Disable a key",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runKeysEnable(cmd, false)
			},
		},
		{
			Name:  "ping",
			Usage: "Test key connectivity",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runKeysPing(cmd)
			},
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return runKeysList(cmd)
	}, // default action when no subcommand
}

func runKeysList(c *cli.Command) error {
	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if len(cfg.Keys) == 0 {
		pterm.Info.Println("No keys configured")
		return nil
	}

	pterm.DefaultSection.Println("Keys")

	for _, k := range cfg.Keys {
		status := pterm.FgRed.Sprint("disabled")
		if k.Enabled {
			status = pterm.FgGreen.Sprint("enabled")
		}

		pterm.DefaultSection.WithLevel(2).Printf("%s (%s)", k.Name, status)
		pterm.Printf("  Format:  %s\n", k.Format)
		pterm.Printf("  BaseURL: %s\n", k.BaseURL)
		pterm.Printf("  Models:  %d\n", len(k.Models))

		tableData := [][]string{{"MODEL", "INPUT PRICE", "OUTPUT PRICE"}}
		for _, m := range k.Models {
			tableData = append(tableData, []string{
				m.Name,
				fmt.Sprintf("$%.4f/1K", m.Pricing.Input),
				fmt.Sprintf("$%.4f/1K", m.Pricing.Output),
			})
		}
		pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
		pterm.Println()
	}
	return nil
}

func runKeysAdd(c *cli.Command) error {
	name := c.String("name")
	format := c.String("format")
	secret := c.String("secret")
	baseURL := c.String("base-url")

	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	// Check if key already exists
	for _, k := range cfg.Keys {
		if k.Name == name {
			return fmt.Errorf("key '%s' already exists", name)
		}
	}

	// Add new key
	newKey := config.KeyConfig{
		Name:    name,
		Format:  format,
		Secret:  secret,
		BaseURL: baseURL,
		Enabled: true,
	}
	cfg.Keys = append(cfg.Keys, newKey)

	if err := config.Save(configPath, cfg); err != nil {
		return err
	}

	pterm.Success.Printf("Key '%s' added successfully\n", name)
	return nil
}

func runKeysRemove(c *cli.Command) error {
	args := c.Args()
	if args.Len() == 0 {
		return fmt.Errorf("key name required")
	}

	name := args.First()
	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	idx := cfg.FindKeyIndex(name)
	if idx < 0 {
		return fmt.Errorf("key '%s' not found", name)
	}

	cfg.Keys = append(cfg.Keys[:idx], cfg.Keys[idx+1:]...)

	if err := config.Save(configPath, cfg); err != nil {
		return err
	}

	pterm.Success.Printf("Key '%s' removed successfully\n", name)
	return nil
}

func runKeysEnable(c *cli.Command, enable bool) error {
	args := c.Args()
	if args.Len() == 0 {
		return fmt.Errorf("key name required")
	}

	keyName := args.First()
	configPath := getConfigPath(c)

	if err := updateConfigKey(configPath, keyName, enable); err != nil {
		return err
	}

	action := "disabled"
	if enable {
		action = "enabled"
	}
	pterm.Success.Printf("Key '%s' %s\n", keyName, action)
	fmt.Println("\nNote: If tokrouter is running, restart it to apply the change:")
	fmt.Println("  tokrouter serve")
	return nil
}

// updateConfigKey updates a key's enabled status in the config file
func updateConfigKey(configPath, keyName string, enable bool) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	idx := cfg.FindKeyIndex(keyName)
	if idx < 0 {
		return fmt.Errorf("key '%s' not found", keyName)
	}

	cfg.Keys[idx].Enabled = enable
	return config.Save(configPath, cfg)
}

func runKeysPing(c *cli.Command) error {
	args := c.Args()
	if args.Len() == 0 {
		return fmt.Errorf("key name required")
	}

	keyName := args.First()
	configPath := getConfigPath(c)

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	// Find the key
	targetKey := cfg.FindKey(keyName)
	if targetKey == nil {
		return fmt.Errorf("key '%s' not found", keyName)
	}

	if !targetKey.Enabled {
		pterm.Warning.Printf("Key '%s' is disabled\n", keyName)
	}

	pterm.DefaultSection.Printf("Testing key '%s'", keyName)
	pterm.Printf("  Format:  %s\n", targetKey.Format)
	pterm.Printf("  BaseURL: %s\n", targetKey.BaseURL)
	pterm.Println()

	// Test each model
	for _, model := range targetKey.Models {
		pterm.Printf("  Testing model '%s'... ", model.Name)

		err := testModel(targetKey, model.Name)
		if err != nil {
			pterm.Error.Printf("FAILED: %v\n", err)
		} else {
			pterm.Success.Println("OK")
		}
	}
	return nil
}

func testModel(key *config.KeyConfig, modelName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Build test request
	var reqBody []byte
	var url string

	switch key.Format {
	case "anthropic":
		url = key.BaseURL + "/v1/messages"
		req := map[string]interface{}{
			"model":      modelName,
			"max_tokens": 10,
			"messages": []map[string]string{
				{"role": "user", "content": "Hi"},
			},
		}
		reqBody, _ = json.Marshal(req)
	default: // openai and compatible
		url = key.BaseURL + "/v1/chat/completions"
		req := map[string]interface{}{
			"model": modelName,
			"messages": []map[string]string{
				{"role": "user", "content": "Hi"},
			},
			"max_tokens": 10,
		}
		reqBody, _ = json.Marshal(req)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	switch key.Format {
	case "anthropic":
		req.Header.Set("x-api-key", key.Secret)
		req.Header.Set("anthropic-version", "2023-06-01")
	default:
		req.Header.Set("Authorization", "Bearer "+key.Secret)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("server error: %d", resp.StatusCode)
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("authentication failed: %d", resp.StatusCode)
	}
	if resp.StatusCode == 404 {
		return fmt.Errorf("model not found: %d", resp.StatusCode)
	}

	// 400 might be due to invalid request format, but key is valid
	// 200-299 is success
	if resp.StatusCode >= 400 {
		return fmt.Errorf("request error: %d", resp.StatusCode)
	}

	return nil
}
