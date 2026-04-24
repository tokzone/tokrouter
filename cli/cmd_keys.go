package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/tokzone/tokrouter/config"

	"github.com/AlecAivazis/survey/v2"
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
			Usage: "Add a new key (interactive if no flags provided)",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "name",
					Usage: "key name",
				},
				&cli.StringFlag{
					Name:  "format",
					Usage: "format (openai/anthropic/gemini/cohere)",
				},
				&cli.StringFlag{
					Name:  "secret",
					Usage: "API key secret",
				},
				&cli.StringFlag{
					Name:  "base-url",
					Usage: "API base URL",
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

		tableData := [][]string{{"MODEL", "PRIORITY"}}
		for _, m := range k.Models {
			tableData = append(tableData, []string{
				m.Name,
				fmt.Sprintf("%d", m.Priority),
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

	// Interactive mode if no flags provided
	if name == "" && format == "" && secret == "" && baseURL == "" {
		return runKeysAddInteractive(c)
	}

	// Validate required flags in non-interactive mode
	if name == "" {
		return fmt.Errorf("flag --name is required")
	}
	if format == "" {
		return fmt.Errorf("flag --format is required")
	}
	if !config.IsValidFormat(format) {
		return fmt.Errorf("invalid format: %s (must be openai/anthropic/gemini/cohere)", format)
	}
	if secret == "" {
		return fmt.Errorf("flag --secret is required")
	}
	if baseURL == "" {
		return fmt.Errorf("flag --base-url is required")
	}

	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	// Check if key already exists
	if err := checkKeyExists(cfg, name); err != nil {
		return err
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

func runKeysAddInteractive(c *cli.Command) error {
	pterm.DefaultSection.Println("Add a new key")

	// Key name
	var name string
	namePrompt := &survey.Input{
		Message: "Key name:",
	}
	if err := survey.AskOne(namePrompt, &name); err != nil || name == "" {
		return fmt.Errorf("key name is required")
	}

	// Format
	var format string
	formatPrompt := &survey.Select{
		Message: "Format:",
		Options: []string{config.FormatOpenAI, config.FormatAnthropic, config.FormatGemini, config.FormatCohere},
		Default: config.FormatOpenAI,
	}
	if err := survey.AskOne(formatPrompt, &format); err != nil {
		return err
	}

	// Base URL
	defaultURL := getDefaultURL(format)
	var baseURL string
	urlPrompt := &survey.Input{
		Message: "Base URL:",
		Default: defaultURL,
	}
	if err := survey.AskOne(urlPrompt, &baseURL); err != nil || baseURL == "" {
		return fmt.Errorf("base URL is required")
	}

	// Secret
	var secret string
	secretPrompt := &survey.Password{
		Message: "API Key:",
	}
	if err := survey.AskOne(secretPrompt, &secret); err != nil || secret == "" {
		return fmt.Errorf("API key is required")
	}

	// Models
	var models []config.ModelConfig
	pterm.Println()
	pterm.DefaultSection.WithLevel(2).Println("Add models")

	for {
		var modelName string
		modelPrompt := &survey.Input{
			Message: "Model name (empty to finish):",
		}
		if err := survey.AskOne(modelPrompt, &modelName); err != nil || modelName == "" {
			break
		}

		var priority int64
		priorityPrompt := &survey.Input{
			Message: "Priority (lower = preferred, default 0):",
			Default: "0",
		}
		survey.AskOne(priorityPrompt, &priority)

		models = append(models, config.ModelConfig{
			Name:     modelName,
			Priority: priority,
		})

		var addAnother bool
		addPrompt := &survey.Confirm{
			Message: "Add another model?",
			Default: false,
		}
		survey.AskOne(addPrompt, &addAnother)
		if !addAnother {
			break
		}
	}

	if len(models) == 0 {
		pterm.Warning.Println("No models added, key will be created without models")
	}

	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	// Check if key already exists
	if err := checkKeyExists(cfg, name); err != nil {
		return err
	}

	// Add new key
	newKey := config.KeyConfig{
		Name:    name,
		Format:  format,
		Secret:  secret,
		BaseURL: baseURL,
		Enabled: true,
		Models:  models,
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
		return fmt.Errorf(`Key name is required.

Usage: tokrouter keys remove <name>
Example: tokrouter keys remove openai-backup`)
	}

	name := args.First()
	configPath := getConfigPath(c)
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	idx := cfg.FindKeyIndex(name)
	if idx < 0 {
		return listAvailableKeysError(cfg, name)
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
		action := "enable"
		if !enable {
			action = "disable"
		}
		return fmt.Errorf(`Key name is required.

Usage: tokrouter keys %s <name>
Example: tokrouter keys %s openai-main`, action, action)
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
		return fmt.Errorf(`Key name is required.

Usage: tokrouter keys ping <name>
Example: tokrouter keys ping openai-main

Use 'tokrouter keys list' to see all keys.`)
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
		return listAvailableKeysError(cfg, keyName)
	}

	if !targetKey.Enabled {
		pterm.Warning.Printf("Key '%s' is disabled\n", keyName)
	}

	pterm.DefaultSection.Printf("Testing key '%s'", keyName)
	pterm.Printf("  Format:  %s\n", targetKey.Format)
	pterm.Printf("  BaseURL: %s\n", targetKey.BaseURL)
	pterm.Println()

	// Test each model
	passed := 0
	failed := 0
	for _, model := range targetKey.Models {
		pterm.Printf("  Testing model '%s'... ", model.Name)

		start := time.Now()
		err := testModel(targetKey, model.Name)
		latency := time.Since(start)

		if err != nil {
			pterm.Error.Printf("FAILED: %v\n", err)
			failed++
		} else {
			pterm.Success.Printf("OK (%dms)\n", latency.Milliseconds())
			passed++
		}
	}

	// Print summary
	pterm.Println()
	if passed > 0 && failed == 0 {
		pterm.Success.Printf("Summary: %d passed, %d failed\n", passed, failed)
	} else if failed > 0 {
		pterm.Warning.Printf("Summary: %d passed, %d failed\n", passed, failed)
	} else {
		pterm.Info.Println("No models to test")
	}
	return nil
}

const testTimeout = 10 * time.Second

func testModel(key *config.KeyConfig, modelName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Build test request
	var reqBody []byte
	var url string

	switch key.Format {
	case config.FormatAnthropic:
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
	case config.FormatAnthropic:
		req.Header.Set("x-api-key", key.Secret)
		req.Header.Set("anthropic-version", "2023-06-01")
	default:
		req.Header.Set("Authorization", "Bearer "+key.Secret)
	}

	client := &http.Client{Timeout: testTimeout}
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

// checkKeyExists returns error if key already exists
func checkKeyExists(cfg *config.Config, name string) error {
	if cfg.FindKey(name) != nil {
		return fmt.Errorf(`Key '%s' already exists.

Suggestions:
  - Use 'tokrouter keys enable %s' to enable it
  - Choose a different name for the new key
  - Use 'tokrouter keys list' to see existing keys`, name, name)
	}
	return nil
}

// listAvailableKeysError returns error with available key names
func listAvailableKeysError(cfg *config.Config, keyName string) error {
	var keyNames []string
	for _, k := range cfg.Keys {
		keyNames = append(keyNames, k.Name)
	}
	return fmt.Errorf(`Key '%s' not found.

Available keys: %s

Use 'tokrouter keys list' to see all keys.`, keyName, strings.Join(keyNames, ", "))
}