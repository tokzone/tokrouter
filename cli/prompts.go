package cli

import (
	"github.com/AlecAivazis/survey/v2"
)

// askConfirm prompts user for confirmation with a message.
// Returns true if user confirms, false if cancelled or error.
func askConfirm(message string, defaultVal bool) bool {
	var confirm bool
	prompt := &survey.Confirm{
		Message: message,
		Default: defaultVal,
	}
	if err := survey.AskOne(prompt, &confirm); err != nil || !confirm {
		return false
	}
	return true
}