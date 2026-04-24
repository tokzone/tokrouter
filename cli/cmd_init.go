package cli

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/tokzone/tokrouter/config"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
)

var initCmd = &cli.Command{
	Name:  "init",
	Usage: "Interactive configuration",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return runInit(cmd)
	},
}

func runInit(c *cli.Command) error {
	pterm.DefaultSection.Println("tokrouter init - Interactive configuration")
	pterm.Info.Println("Press Ctrl+C at any time to cancel")
	pterm.Println()

	// Ask for server port with validation
	var port int
	for {
		portPrompt := &survey.Input{
			Message: "Server port:",
			Default: "8765",
		}
		var portStr string
		if err := survey.AskOne(portPrompt, &portStr); err != nil {
			return err
		}
		var err error
		port, err = strconv.Atoi(portStr)
		if err != nil || port < 1 || port > 65535 {
			pterm.Error.Println("Invalid port. Must be a number between 1 and 65535")
			continue
		}
		break
	}

	// Ask for keys
	var keys []config.KeyConfig

	for {
		pterm.Println()
		pterm.DefaultSection.WithLevel(2).Println("Add a Key")

		// Key name
		var name string
		namePrompt := &survey.Input{
			Message: "Key name (e.g., openai-main):",
		}
		if err := survey.AskOne(namePrompt, &name); err != nil || name == "" {
			break
		}

		// Format
		var format string
		formatPrompt := &survey.Select{
			Message: "Choose format:",
			Options: []string{config.FormatOpenAI, config.FormatAnthropic, config.FormatGemini, config.FormatCohere},
			Default: config.FormatOpenAI,
		}
		survey.AskOne(formatPrompt, &format)

		// Base URL with validation
		defaultURL := getDefaultURL(format)
		var baseURL string
		for {
			urlPrompt := &survey.Input{
				Message: "Base URL:",
				Default: defaultURL,
			}
			if err := survey.AskOne(urlPrompt, &baseURL); err != nil {
				return err
			}
			u, err := url.Parse(baseURL)
			if err != nil || u.Scheme == "" || u.Host == "" {
				pterm.Error.Println("Invalid URL format. Please enter a valid URL (e.g., https://api.openai.com/v1)")
				continue
			}
			break
		}

		// Secret
		var secret string
		secretPrompt := &survey.Password{
			Message: "API Key:",
		}
		if err := survey.AskOne(secretPrompt, &secret); err != nil || secret == "" {
			pterm.Warning.Println("No API key provided, skipping this key")
			continue
		}

		// Models
		var models []config.ModelConfig
		for {
			var modelName string
			modelPrompt := &survey.Input{
				Message: "Model name (e.g., gpt-4, or empty to finish):",
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
		}

		if len(models) == 0 {
			pterm.Warning.Println("No models added, skipping this key")
			continue
		}

		keys = append(keys, config.KeyConfig{
			Name:    name,
			Format:  format,
			BaseURL: baseURL,
			Secret:  secret,
			Enabled: true,
			Models:  models,
		})

		// Ask if user wants to add another key
		var continueAdd bool
		continuePrompt := &survey.Confirm{
			Message: "Add another key?",
			Default: false,
		}
		survey.AskOne(continuePrompt, &continueAdd)
		if !continueAdd {
			break
		}
	}

	if len(keys) == 0 {
		return fmt.Errorf("no keys configured, aborting")
	}

	// Create config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: port,
		},
		Keys: keys,
		Router: config.RouterConfig{
			Retry: config.RetryConfig{
				MaxRetries: 2,
				Timeout:    "30s",
				Backoff:    "exponential",
			},
		},
		Stats: config.StatsConfig{
			Enabled: true,
			DBPath:  "./data/usage.db",
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		Trace: config.TraceConfig{
			Enabled: true,
			Header:  "x-request-id",
		},
	}

	// Show configuration summary and confirm
	pterm.Println()
	pterm.DefaultSection.Println("Configuration Summary")
	pterm.Printf("  Server: 127.0.0.1:%d\n", port)
	pterm.Printf("  Keys: %d\n", len(keys))
	for _, k := range keys {
		pterm.Printf("    - %s (%s): %d models\n", k.Name, k.Format, len(k.Models))
	}
	pterm.Println()

	var confirm bool
	confirmPrompt := &survey.Confirm{
		Message: "Save this configuration?",
		Default: true,
	}
	if err := survey.AskOne(confirmPrompt, &confirm); err != nil {
		return err
	}
	if !confirm {
		pterm.Info.Println("Configuration not saved")
		return nil
	}

	// Write config
	configPath := getConfigPath(c)
	if err := config.Save(configPath, cfg); err != nil {
		return err
	}

	pterm.Println()
	pterm.Success.Printf("Configuration saved to %s\n", configPath)
	pterm.Println()
	pterm.Info.Println("Next steps:")
	pterm.Info.Println("  1. Review and edit the config: tokrouter config show")
	pterm.Info.Println("  2. Start the server: tokrouter serve")
	return nil
}

func getDefaultURL(format string) string {
	switch format {
	case config.FormatOpenAI:
		return "https://api.openai.com/v1"
	case config.FormatAnthropic:
		return "https://api.anthropic.com/v1"
	case config.FormatGemini:
		return "https://generativelanguage.googleapis.com/v1"
	case config.FormatCohere:
		return "https://api.cohere.ai/v1"
	default:
		return ""
	}
}
