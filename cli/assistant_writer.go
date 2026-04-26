package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/pterm/pterm"
)

// AssistantConfig defines an AI assistant configuration template
type AssistantConfig struct {
	Name         string
	DisplayName  string
	ConfigType   string // "env", "json", "yaml"
	CheckCommand string // 命令检测（空则用配置目录检测）
	ConfigPaths  []string
	EnvVars      []EnvVarConfig
	JSONConfig   map[string]interface{}
	JSONKeyPath  string // dot-separated path for JSON merge
}

// EnvVarConfig defines an environment variable
type EnvVarConfig struct {
	Name  string
	Value string
}

// findAssistant finds an assistant by name
func findAssistant(name string) *AssistantConfig {
	for i, a := range getAssistantConfigs() {
		if a.Name == name {
			return &getAssistantConfigs()[i]
		}
	}
	return nil
}

// checkAssistantInstalled checks if an assistant is installed
// Priority: 1) CheckCommand (if set) 2) ConfigPaths (directory existence)
func checkAssistantInstalled(a *AssistantConfig) bool {
	// 优先检测命令是否存在
	if a.CheckCommand != "" {
		_, err := exec.LookPath(a.CheckCommand)
		if err == nil {
			return true
		}
	}

	// 如果没有命令检测，则检测配置目录是否存在
	for _, path := range a.ConfigPaths {
		expanded := expandPath(path)
		if _, err := os.Stat(expanded); err == nil {
			return true
		}
		// 检测目录是否存在（针对 ~/.cursor/ 这类目录）
		dir := filepath.Dir(expanded)
		if _, err := os.Stat(dir); err == nil {
			return true
		}
	}
	return false
}

// getAssistantConfigPath returns the config file path for an assistant
func getAssistantConfigPath(a *AssistantConfig) string {
	// Return first existing path or first default path
	for _, path := range a.ConfigPaths {
		expanded := expandPath(path)
		if _, err := os.Stat(expanded); err == nil {
			return expanded
		}
	}
	return expandPath(a.ConfigPaths[0])
}

// expandPath expands ~ and environment variables in a path
func expandPath(path string) string {
	// Expand ~ first
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		path = home + path[1:]
	}
	// Expand environment variables (basic support for ${VAR})
	if runtime.GOOS == "windows" {
		if path == "${PROFILE}" {
			profile := os.Getenv("USERPROFILE")
			return profile + "\\Documents\\PowerShell\\Microsoft.PowerShell_profile.ps1"
		}
		path = os.ExpandEnv(path)
	} else {
		path = os.ExpandEnv(path)
	}
	return path
}

// generateAssistantConfig generates config content for display
func generateAssistantConfig(a *AssistantConfig, url string) string {
	if a.ConfigType == "env" {
		result := ""
		for _, env := range a.EnvVars {
			value := replaceURL(env.Value, url)
			if runtime.GOOS == "windows" {
				result += fmt.Sprintf("$env:%s = \"%s\"\n", env.Name, value)
			} else {
				result += fmt.Sprintf("export %s=\"%s\"\n", env.Name, value)
			}
		}
		return result
	}
	if a.ConfigType == "json" {
		config := deepCopyMap(a.JSONConfig)
		replaceURLInMap(config, url)
		data, _ := json.MarshalIndent(config, "", "  ")
		return string(data)
	}
	return ""
}

// writeAssistantConfig writes config for an assistant
func writeAssistantConfig(a *AssistantConfig, url string) error {
	configPath := getAssistantConfigPath(a)

	if a.ConfigType == "env" {
		return writeEnvConfig(configPath, a, url)
	}
	if a.ConfigType == "json" {
		return writeJSONConfig(configPath, a, url)
	}
	return fmt.Errorf("unsupported config type: %s", a.ConfigType)
}

// writeEnvConfig writes environment variable config to shell profile
func writeEnvConfig(path string, a *AssistantConfig, url string) error {
	// Read existing content
	existing, _ := os.ReadFile(path)
	content := string(existing)

	// Check if already configured
	for _, env := range a.EnvVars {
		if runtime.GOOS == "windows" {
			if contains(content, "$env:"+env.Name) {
				pterm.Info.Printf("  %s already configured in %s\n", env.Name, path)
				continue
			}
		} else {
			if contains(content, "export "+env.Name) {
				pterm.Info.Printf("  %s already configured in %s\n", env.Name, path)
				continue
			}
		}
	}

	// Add new exports
	lines := "\n# Added by tokrouter\n"
	for _, env := range a.EnvVars {
		value := replaceURL(env.Value, url)
		if runtime.GOOS == "windows" {
			lines += fmt.Sprintf("$env:%s = \"%s\"\n", env.Name, value)
		} else {
			lines += fmt.Sprintf("export %s=\"%s\"\n", env.Name, value)
		}
	}

	// Append to file
	newContent := content + lines
	return os.WriteFile(path, []byte(newContent), 0644)
}

// writeJSONConfig writes JSON config with merge
func writeJSONConfig(path string, a *AssistantConfig, url string) error {
	// Check if file exists
	var existing map[string]interface{}
	data, err := os.ReadFile(path)
	if err == nil {
		json.Unmarshal(data, &existing)
	} else {
		existing = map[string]interface{}{}
	}

	// Deep copy template and replace URL
	newConfig := deepCopyMap(a.JSONConfig)
	replaceURLInMap(newConfig, url)

	// Merge
	merged := mergeMaps(existing, newConfig)

	// Write back
	result, _ := json.MarshalIndent(merged, "", "  ")

	// Ensure directory exists
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)

	return os.WriteFile(path, result, 0644)
}

// getAssistantRestartHint returns restart hint for an assistant
func getAssistantRestartHint(a *AssistantConfig) string {
	switch a.Name {
	case "claude-code":
		if runtime.GOOS == "windows" {
			return "Restart PowerShell or run: $env:ANTHROPIC_BASE_URL = ..."
		}
		return "Run: source ~/.zshrc (or ~/.bashrc)"
	case "cursor":
		return "Restart Cursor IDE"
	case "aider":
		if runtime.GOOS == "windows" {
			return "Restart PowerShell or run: $env:OPENAI_API_BASE = ..."
		}
		return "Run: source ~/.zshrc (or ~/.bashrc)"
	case "windsurf":
		return "Restart Windsurf IDE"
	case "cline":
		return "Restart VS Code"
	case "continue":
		return "Restart Continue.dev extension"
	default:
		return "Restart the assistant"
	}
}

// Helper functions (pure logic, no config data)

func replaceURL(value, url string) string {
	if value == "{{URL}}" {
		return url
	}
	if value == "{{URL}}/v1" {
		return url + "/v1"
	}
	return value
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	for k, v := range m {
		if sub, ok := v.(map[string]interface{}); ok {
			result[k] = deepCopyMap(sub)
		} else {
			result[k] = v
		}
	}
	return result
}

func replaceURLInMap(m map[string]interface{}, url string) {
	for k, v := range m {
		if sub, ok := v.(map[string]interface{}); ok {
			replaceURLInMap(sub, url)
		} else if str, ok := v.(string); ok {
			m[k] = replaceURL(str, url)
		}
	}
}

func mergeMaps(existing, new map[string]interface{}) map[string]interface{} {
	result := deepCopyMap(existing)
	for k, v := range new {
		if existingSub, ok := result[k].(map[string]interface{}); ok {
			if newSub, ok := v.(map[string]interface{}); ok {
				result[k] = mergeMaps(existingSub, newSub)
			} else {
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}
	return result
}

// checkServerStatus checks if tokrouter server is running
// Returns "running" or "not running"
func checkServerStatus(url string) string {
	// TODO: Implement actual HTTP check
	return "not running"
}