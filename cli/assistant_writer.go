package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/pterm/pterm"

	"github.com/tokzone/tokrouter/config"
)

// AssistantConfig defines an AI assistant configuration template
type AssistantConfig struct {
	Name         string
	DisplayName  string
	ConfigType   string // "env", "json", "yaml"
	CheckCommand string // command to detect installation (empty = use config dirs)
	ConfigPaths  []string
	EnvVars      []EnvVarConfig
	JSONConfig   map[string]interface{}
	JSONKeyPath  string // dot-separated path for JSON merge
	TOMLConfig   map[string]string // TOML key-value pairs (for "toml" ConfigType)
	ModelNote    string // hint shown after config, for assistants where we can't set the model programmatically
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
func checkAssistantInstalled(a *AssistantConfig) bool {
	if a.CheckCommand != "" {
		_, err := exec.LookPath(a.CheckCommand)
		if err == nil {
			return true
		}
	}
	for _, path := range a.ConfigPaths {
		expanded := expandPath(path)
		if _, err := os.Stat(expanded); err == nil {
			return true
		}
		dir := filepath.Dir(expanded)
		if _, err := os.Stat(dir); err == nil {
			return true
		}
	}
	return false
}

// getAssistantConfigPath returns the config file path for an assistant
func getAssistantConfigPath(a *AssistantConfig) string {
	for _, path := range a.ConfigPaths {
		expanded := expandPath(path)
		if _, err := os.Stat(expanded); err == nil {
			return expanded
		}
	}
	return expandPath(a.ConfigPaths[0])
}

// getShellProfilePath returns the first existing shell profile path.
func getShellProfilePath() string {
	paths := getShellProfilePaths()
	for _, p := range paths {
		expanded := expandPath(p)
		if _, err := os.Stat(expanded); err == nil {
			return expanded
		}
	}
	if len(paths) > 0 {
		return expandPath(paths[0])
	}
	return ""
}

// expandPath expands ~ and environment variables in a path
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		path = home + path[1:]
	}
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
func generateAssistantConfig(a *AssistantConfig, url, model string) string {
	if a.ConfigType == "env" {
		result := ""
		for _, env := range a.EnvVars {
			value := resolveTemplate(env.Value, url, model)
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
		resolveTemplateInMap(config, url, model)
		data, _ := json.MarshalIndent(config, "", "  ")
		return string(data)
	}
	if a.ConfigType == "toml" {
		result := ""
		for k, v := range a.TOMLConfig {
			val := resolveTemplate(v, url, model)
			if val == "" {
				continue
			}
			result += fmt.Sprintf("%s = %s\n", k, tomlLiteral(val))
		}
		if len(a.EnvVars) > 0 {
			result += "\n# Also set in shell profile:\n"
			for _, env := range a.EnvVars {
				val := resolveTemplate(env.Value, url, model)
				if runtime.GOOS == "windows" {
					result += fmt.Sprintf("$env:%s = \"%s\"\n", env.Name, val)
				} else {
					result += fmt.Sprintf("export %s=\"%s\"\n", env.Name, val)
				}
			}
		}
		return result
	}
	return ""
}

// writeAssistantConfig writes config for an assistant
func writeAssistantConfig(a *AssistantConfig, url, model string) error {
	configPath := getAssistantConfigPath(a)

	if a.ConfigType == "env" {
		return writeEnvConfig(configPath, a, url, model)
	}
	if a.ConfigType == "json" {
		return writeJSONConfig(configPath, a, url, model)
	}
	if a.ConfigType == "toml" {
		if err := writeTOMLConfig(configPath, a, url, model); err != nil {
			return err
		}
		// Also write env vars to shell profile if present
		if len(a.EnvVars) > 0 {
			shellPath := getShellProfilePath()
			if shellPath != "" {
				shellA := &AssistantConfig{
					Name:       a.Name,
					ConfigType: "env",
					EnvVars:    a.EnvVars,
				}
				return writeEnvConfig(shellPath, shellA, url, model)
			}
		}
		return nil
	}
	return fmt.Errorf("unsupported config type: %s", a.ConfigType)
}

// writeEnvConfig writes environment variable config to shell profile.
// Existing assignments are replaced in place; new vars are appended.
func writeEnvConfig(path string, a *AssistantConfig, url, model string) error {
	existing, _ := os.ReadFile(path)
	content := string(existing)

	var toAppend []string
	for _, env := range a.EnvVars {
		value := resolveTemplate(env.Value, url, model)
		var pattern *regexp.Regexp
		var newLine string
		if runtime.GOOS == "windows" {
			pattern = regexp.MustCompile(`(?m)^\$env:` + regexp.QuoteMeta(env.Name) + `\s*=\s*".*"$`)
			newLine = fmt.Sprintf(`$env:%s = "%s"`, env.Name, value)
		} else {
			pattern = regexp.MustCompile(`(?m)^export ` + regexp.QuoteMeta(env.Name) + `=.*$`)
			newLine = fmt.Sprintf(`export %s="%s"`, env.Name, value)
		}
		if pattern.MatchString(content) {
			content = pattern.ReplaceAllString(content, newLine)
			pterm.Info.Printf("  Updated %s in %s\n", env.Name, path)
		} else {
			toAppend = append(toAppend, newLine)
		}
	}

	if len(toAppend) > 0 {
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += "\n# Added by tokrouter\n"
		for _, line := range toAppend {
			content += line + "\n"
		}
	}

	return os.WriteFile(path, []byte(content), 0644)
}

// writeTOMLConfig writes TOML config file.
// Existing keys are replaced in place; new keys are appended.
func writeTOMLConfig(path string, a *AssistantConfig, url, model string) error {
	existing, _ := os.ReadFile(path)
	content := string(existing)

	var toAppend []string
	for k, v := range a.TOMLConfig {
		value := resolveTemplate(v, url, model)
		if value == "" {
			continue
		}
		newLine := fmt.Sprintf("%s = %s", k, tomlLiteral(value))
		pattern := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(k) + `\s*=\s*.*$`)
		if pattern.MatchString(content) {
			content = pattern.ReplaceAllString(content, newLine)
			pterm.Info.Printf("  Updated %s in %s\n", k, path)
		} else {
			toAppend = append(toAppend, newLine)
		}
	}

	if len(toAppend) > 0 {
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += "\n# Added by tokrouter\n"
		for _, line := range toAppend {
			content += line + "\n"
		}
	}

	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)

	return os.WriteFile(path, []byte(content), 0644)
}

// writeJSONConfig writes JSON config with merge
func writeJSONConfig(path string, a *AssistantConfig, url, model string) error {
	var existing map[string]interface{}
	data, err := os.ReadFile(path)
	if err == nil {
		json.Unmarshal(data, &existing)
	} else {
		existing = map[string]interface{}{}
	}

	newConfig := deepCopyMap(a.JSONConfig)
	resolveTemplateInMap(newConfig, url, model)

	merged := mergeMaps(existing, newConfig)

	result, _ := json.MarshalIndent(merged, "", "  ")

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
	case "codex":
		if runtime.GOOS == "windows" {
			return "Restart PowerShell or run: $env:OPENAI_API_KEY = ..."
		}
		return "Run: source ~/.zshrc (or ~/.bashrc) for API key"
	default:
		return "Restart the assistant"
	}
}

// resolveTemplate replaces {{URL}}, {{MODEL}}, {{CONTEXT}} placeholders in a value.
func resolveTemplate(value, url, model string) string {
	value = strings.ReplaceAll(value, "{{URL}}", url)
	value = strings.ReplaceAll(value, "{{MODEL}}", model)
	if strings.Contains(value, "{{CONTEXT}}") {
		value = strings.ReplaceAll(value, "{{CONTEXT}}", lookupModelContext(model))
	}
	return value
}

// lookupModelContext finds the context window size for a model from built-in presets.
func lookupModelContext(model string) string {
	for _, preset := range getPresetsFromConfig() {
		for _, m := range preset.DefaultModels {
			if m.Name == model && m.Context > 0 {
				return fmt.Sprintf("%d", m.Context)
			}
		}
	}
	return ""
}

// getPresetsFromConfig returns all available presets.
func getPresetsFromConfig() []config.ProviderPreset {
	return config.ListPresets()
}

// tomlLiteral formats a value for TOML: bare for integers, quoted for strings.
func tomlLiteral(v string) string {
	if isNumeric(v) {
		return v
	}
	return `"` + v + `"`
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
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

func resolveTemplateInMap(m map[string]interface{}, url, model string) {
	for k, v := range m {
		if sub, ok := v.(map[string]interface{}); ok {
			resolveTemplateInMap(sub, url, model)
		} else if str, ok := v.(string); ok {
			m[k] = resolveTemplate(str, url, model)
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
