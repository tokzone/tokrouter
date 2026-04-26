package cli

// getAssistantConfigs returns all assistants (built-in + user-defined)
// User config can override built-in configs with same name
func getAssistantConfigs() []AssistantConfig {
	builtins := builtinAssistants()

	// TODO: Load user-defined assistants from ~/.tokrouter/assistants.yaml
	// For now, return built-in only
	// users, _ := loadUserAssistants()
	// return mergeAssistants(builtins, users)

	return builtins
}

// loadUserAssistants loads user-defined assistants from YAML file
// Returns empty list if file doesn't exist
// TODO: Implement YAML loading
// func loadUserAssistants() ([]AssistantConfig, error) {
//     path := "~/.tokrouter/assistants.yaml"
//     // Check if file exists
//     // Load and parse YAML
//     // Return configs
// }

// mergeAssistants merges built-in and user configs
// User configs override built-in configs with same name
func mergeAssistants(builtins, users []AssistantConfig) []AssistantConfig {
	result := make([]AssistantConfig, len(builtins))
	copy(result, builtins)

	// Add or override with user configs
	for _, user := range users {
		found := false
		for i, b := range result {
			if b.Name == user.Name {
				result[i] = user // Override
				found = true
				break
			}
		}
		if !found {
			result = append(result, user) // Add new
		}
	}

	return result
}