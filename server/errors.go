package server

import "fmt"

// Error codes for structured error handling
const (
	ErrCodeInvalidRequest = "INVALID_REQUEST"
	ErrCodeNoEndpoint     = "NO_ENDPOINT"
	ErrCodeUpstreamFailed = "UPSTREAM_FAILED"
	ErrCodeUnauthorized   = "UNAUTHORIZED"
	ErrCodeRateLimited    = "RATE_LIMITED"
	ErrCodeModelNotFound  = "MODEL_NOT_FOUND"
	ErrCodeInternalError  = "INTERNAL_ERROR"
)

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Type       string `json:"type"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// Error definitions with recovery hints
var errorDefinitions = map[string]struct {
	message    string
	suggestion string
}{
	ErrCodeInvalidRequest: {
		message:    "Invalid request format",
		suggestion: "Check that your request body is valid JSON and matches the expected format",
	},
	ErrCodeNoEndpoint: {
		message:    "No healthy endpoint available",
		suggestion: "Check that at least one key is enabled and has healthy models. Use 'tokrouter status' to check key health",
	},
	ErrCodeUpstreamFailed: {
		message:    "Upstream provider request failed",
		suggestion: "Check your network connection and API key validity. Use 'tokrouter test --key <name>' to verify",
	},
	ErrCodeUnauthorized: {
		message:    "Authentication failed with upstream provider",
		suggestion: "Verify your API key is correct and has not expired. Check the key in config.yaml",
	},
	ErrCodeRateLimited: {
		message:    "Rate limit exceeded",
		suggestion: "Wait a moment and retry, or add additional keys for this provider",
	},
	ErrCodeModelNotFound: {
		message:    "Model not found",
		suggestion: "Check that the model name is correct and supported by the provider",
	},
	ErrCodeInternalError: {
		message:    "Internal server error",
		suggestion: "Check server logs for details. If persistent, report issue at github.com/tokflux/tokrouter",
	},
}

// NewErrorResponse creates an error response with code and suggestion
func NewErrorResponse(err error) *ErrorResponse {
	return &ErrorResponse{
		Type:    "error",
		Code:    ErrCodeInternalError,
		Message: err.Error(),
	}
}

// NewErrorResponseWithCode creates a structured error response
func NewErrorResponseWithCode(code string, details string) *ErrorResponse {
	def, ok := errorDefinitions[code]
	if !ok {
		def = errorDefinitions[ErrCodeInternalError]
	}

	msg := def.message
	if details != "" {
		msg = def.message + ": " + details
	}

	return &ErrorResponse{
		Type:       "error",
		Code:       code,
		Message:    msg,
		Suggestion: def.suggestion,
	}
}

// Error implements error interface
func (e *ErrorResponse) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("[%s] %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}
