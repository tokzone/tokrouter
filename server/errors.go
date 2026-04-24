package server

import "fmt"

// Error codes for structured error handling
const (
	ErrCodeInvalidRequest = "INVALID_REQUEST"
	ErrCodeUpstreamFailed = "UPSTREAM_FAILED"
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
	ErrCodeUpstreamFailed: {
		message:    "Upstream provider request failed",
		suggestion: "Check your network connection and API key validity. Use 'tokrouter test --key <name>' to verify",
	},
	ErrCodeInternalError: {
		message:    "Internal server error",
		suggestion: "Check server logs for details. If persistent, report issue at github.com/github.com/tokzone/tokrouter",
	},
}

// NewErrorResponse creates an error response with code and suggestion
func NewErrorResponse(err error) *ErrorResponse {
	msg := "unknown error"
	if err != nil {
		msg = err.Error()
	}
	return &ErrorResponse{
		Type:    "error",
		Code:    ErrCodeInternalError,
		Message: msg,
	}
}

// NewErrorResponseWithCode creates a structured error response
func NewErrorResponseWithCode(code string, details string) *ErrorResponse {
	def, ok := errorDefinitions[code]
	if !ok {
		// Unknown code defaults to internal error
		code = ErrCodeInternalError
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
