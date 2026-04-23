package cli

import (
	"context"
	"fmt"

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

	// Ask for server port
	var port int
	portPrompt := &survey.Input{
		Message: "Server port:",
		Default: "8765",
	}
	survey.AskOne(portPrompt, &port)

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
			Options: []string{"openai", "anthropic", "gemini", "cohere"},
			Default: "openai",
		}
		survey.AskOne(formatPrompt, &format)

		// Base URL
		defaultURL := getDefaultURL(format)
		var baseURL string
		urlPrompt := &survey.Input{
			Message: "Base URL:",
			Default: defaultURL,
		}
		survey.AskOne(urlPrompt, &baseURL)

		// Secret
		var secret string
		secretPrompt := &survey.Password{
			Message: "API Key:",
		}
		survey.AskOne(secretPrompt, &secret)

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

			var inputPrice float64
			inputPricePrompt := &survey.Input{
				Message: "Input price per 1K tokens:",
				Default: "0.03",
			}
			survey.AskOne(inputPricePrompt, &inputPrice)

			var outputPrice float64
			outputPricePrompt := &survey.Input{
				Message: "Output price per 1K tokens:",
				Default: "0.06",
			}
			survey.AskOne(outputPricePrompt, &outputPrice)

			models = append(models, config.ModelConfig{
				Name: modelName,
				Pricing: config.PricingConfig{
					Input:  inputPrice,
					Output: outputPrice,
				},
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
	case "openai":
		return "https://api.openai.com/v1"
	case "anthropic":
		return "https://api.anthropic.com/v1"
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1"
	case "cohere":
		return "https://api.cohere.ai/v1"
	default:
		return ""
	}
}
