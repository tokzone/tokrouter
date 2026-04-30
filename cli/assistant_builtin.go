package cli

import "runtime"

// builtinAssistants returns the built-in assistant configuration templates.
// This file contains ONLY configuration data, no logic code.
// To add a new assistant, simply add it to this list.
func builtinAssistants() []AssistantConfig {
	return []AssistantConfig{
		{
			Name:         "claude-code",
			DisplayName:  "Claude Code",
			ConfigType:   "env",
			CheckCommand: "claude",
			ConfigPaths:  getShellProfilePaths(),
			EnvVars: []EnvVarConfig{
				{Name: "ANTHROPIC_BASE_URL", Value: "{{URL}}"},
				{Name: "ANTHROPIC_API_KEY", Value: "tokrouter-key"},
			},
			ModelNote: "Use /model in Claude Code to select your preferred model.",
		},
		{
			Name:         "cursor",
			DisplayName:  "Cursor",
			ConfigType:   "json",
			CheckCommand: "",
			ConfigPaths: []string{
				"~/.cursor/config.json",
				"${APPDATA}/Cursor/User/globalStorage/settings.json",
			},
			JSONKeyPath: "models.baseURL",
			JSONConfig: map[string]interface{}{
				"models": map[string]interface{}{
					"baseURL": "{{URL}}/v1",
				},
			},
			ModelNote: "Open Cursor Settings > Models and select your model from the list.",
		},
		{
			Name:         "aider",
			DisplayName:  "Aider",
			ConfigType:   "env",
			CheckCommand: "aider",
			ConfigPaths:  getShellProfilePaths(),
			EnvVars: []EnvVarConfig{
				{Name: "OPENAI_API_BASE", Value: "{{URL}}/v1"},
				{Name: "OPENAI_API_KEY", Value: "tokrouter-key"},
			},
			ModelNote: "Run aider with: aider --model {{MODEL}}",
		},
		{
			Name:         "windsurf",
			DisplayName:  "Windsurf",
			ConfigType:   "json",
			CheckCommand: "",
			ConfigPaths: []string{
				"~/.windsurf/config.json",
				"${APPDATA}/Windsurf/User/globalStorage/settings.json",
			},
			JSONKeyPath: "models.baseURL",
			JSONConfig: map[string]interface{}{
				"models": map[string]interface{}{
					"baseURL": "{{URL}}/v1",
				},
			},
			ModelNote: "Open Windsurf Settings > Models and select your model from the list.",
		},
		{
			Name:         "cline",
			DisplayName:  "Cline (VS Code)",
			ConfigType:   "json",
			CheckCommand: "code",
			ConfigPaths: []string{
				"~/.vscode/settings.json",
				"${APPDATA}/Code/User/settings.json",
			},
			JSONKeyPath: "cline.baseUrl",
			JSONConfig: map[string]interface{}{
				"cline": map[string]interface{}{
					"baseUrl": "{{URL}}/v1",
				},
			},
			ModelNote: "Open Cline settings in VS Code and configure your model.",
		},
		{
			Name:         "continue",
			DisplayName:  "Continue.dev",
			ConfigType:   "json",
			CheckCommand: "code",
			ConfigPaths: []string{
				"~/.continue/config.json",
			},
			JSONKeyPath: "models.custom.baseURL",
			JSONConfig: map[string]interface{}{
				"models": map[string]interface{}{
					"custom": map[string]interface{}{
						"baseURL": "{{URL}}/v1",
					},
				},
			},
			ModelNote: "Edit ~/.continue/config.json and set the model field to your preferred model.",
		},
		{
			Name:         "codex",
			DisplayName:  "OpenAI Codex",
			ConfigType:   "toml",
			CheckCommand: "codex",
			ConfigPaths: []string{
				"~/.codex/config.toml",
			},
			TOMLConfig: map[string]string{
				"model_provider":                      "tokrouter",
				"model":                               "{{MODEL}}",
				"model_context_window":                "{{CONTEXT}}",
				"model_providers.tokrouter.name":      "tokrouter",
				"model_providers.tokrouter.base_url":  "{{URL}}/v1",
				"model_providers.tokrouter.wire_api":  "responses",
			},
			EnvVars: []EnvVarConfig{
				{Name: "OPENAI_API_KEY", Value: "tokrouter-key"},
			},
		},
	}
}

// getShellProfilePaths returns shell profile paths based on OS
func getShellProfilePaths() []string {
	if runtime.GOOS == "windows" {
		return []string{"${PROFILE}"}
	}
	return []string{
		"~/.zshrc",
		"~/.bashrc",
		"~/.bash_profile",
		"~/.profile",
	}
}
