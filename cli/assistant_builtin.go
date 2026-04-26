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
			CheckCommand: "claude", // 命令检测
			ConfigPaths:  getShellProfilePaths(),
			EnvVars: []EnvVarConfig{
				{Name: "ANTHROPIC_BASE_URL", Value: "{{URL}}"},
				{Name: "ANTHROPIC_API_KEY", Value: "tokrouter-key"},
			},
		},
		{
			Name:         "cursor",
			DisplayName:  "Cursor",
			ConfigType:   "json",
			CheckCommand: "", // IDE 无命令，用目录检测
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
		},
		{
			Name:         "aider",
			DisplayName:  "Aider",
			ConfigType:   "env",
			CheckCommand: "aider", // 命令检测
			ConfigPaths:  getShellProfilePaths(),
			EnvVars: []EnvVarConfig{
				{Name: "OPENAI_API_BASE", Value: "{{URL}}/v1"},
				{Name: "OPENAI_API_KEY", Value: "tokrouter-key"},
			},
		},
		{
			Name:         "windsurf",
			DisplayName:  "Windsurf",
			ConfigType:   "json",
			CheckCommand: "", // IDE 无命令，用目录检测
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
		},
		{
			Name:         "cline",
			DisplayName:  "Cline (VS Code)",
			ConfigType:   "json",
			CheckCommand: "code", // VS Code 命令检测（插件依赖 VS Code）
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
		},
		{
			Name:         "continue",
			DisplayName:  "Continue.dev",
			ConfigType:   "json",
			CheckCommand: "code", // VS Code 命令检测（插件依赖 VS Code）
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
		},
	}
}

// getShellProfilePaths returns shell profile paths based on OS
// This is configuration data, not logic.
func getShellProfilePaths() []string {
	if runtime.GOOS == "windows" {
		return []string{"${PROFILE}"} // PowerShell profile
	}
	return []string{
		"~/.zshrc",
		"~/.bashrc",
		"~/.bash_profile",
		"~/.profile",
	}
}