package cli

// getAssistantConfigs returns all assistants (built-in + user-defined)
func getAssistantConfigs() []AssistantConfig {
	return builtinAssistants()
}