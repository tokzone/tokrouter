package server

import (
	stderrors "errors"
	"fmt"
	"net/http"

	"github.com/tokzone/fluxcore/errors"
)

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Type       string `json:"type"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// Error definitions with recovery hints (UX enhancement, not in fluxcore scope).
var errorDefinitions = map[errors.ErrorCode]struct {
	message    string
	suggestion string
}{
	errors.CodeInvalidRequest: {
		message:    "Invalid request format",
		suggestion: "Check that your request body is valid JSON and matches the expected format",
	},
	errors.CodeNoEndpoint: {
		message:    "No available endpoint for the requested model",
		suggestion: "Check that the model name is correct and at least one provider is configured",
	},
	errors.CodeRateLimit: {
		message:    "Rate limit exceeded on upstream provider",
		suggestion: "Reduce request frequency or add more provider endpoints",
	},
	errors.CodeServerError: {
		message:    "Upstream provider returned an error",
		suggestion: "The provider may be experiencing issues. Check 'tr show health' for details",
	},
	errors.CodeNetworkError: {
		message:    "Network error connecting to upstream provider",
		suggestion: "Check your network connection and API key validity. Use 'tr test --key <name>' to verify",
	},
	errors.CodeTimeout: {
		message:    "Request to upstream provider timed out",
		suggestion: "The provider may be slow or unreachable. Check your network or try again later",
	},
	errors.CodeDNSError: {
		message:    "DNS resolution failed for upstream provider",
		suggestion: "Check your DNS configuration and network connectivity",
	},
	errors.CodeAuthError: {
		message:    "Authentication failed with upstream provider",
		suggestion: "Check your API key is valid and has not expired",
	},
	errors.CodeModelError: {
		message:    "Model error from upstream provider",
		suggestion: "The model may be overloaded or unavailable. Try again later or use a different model",
	},
}

// NewErrorResponseWithCode creates a structured error response.
func NewErrorResponseWithCode(code errors.ErrorCode, details string) *ErrorResponse {
	def, ok := errorDefinitions[code]
	if !ok {
		code = errors.CodeServerError
		def = errorDefinitions[code]
	}

	msg := def.message
	if details != "" {
		msg = def.message + ": " + details
	}

	return &ErrorResponse{
		Type:       "error",
		Code:       string(code),
		Message:    msg,
		Suggestion: def.suggestion,
	}
}

// Error implements error interface.
func (e *ErrorResponse) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("[%s] %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// ClassifyAndWriteError writes an error response with proper HTTP status code derived from the error.
func ClassifyAndWriteError(w http.ResponseWriter, err error) {
	code, httpStatus := classifyError(err)
	WriteErrorResponse(w, httpStatus, NewErrorResponseWithCode(code, err.Error()))
}

func classifyError(err error) (errors.ErrorCode, int) {
	var classified *errors.ClassifiedError
	if stderrors.As(err, &classified) {
		status := httpStatusCode(classified.Code)
		if classified.StatusCode > 0 {
			status = classified.StatusCode
		}
		return classified.Code, status
	}
	return errors.CodeServerError, http.StatusServiceUnavailable
}

func httpStatusCode(code errors.ErrorCode) int {
	switch code {
	case errors.CodeInvalidRequest:
		return http.StatusBadRequest
	case errors.CodeAuthError:
		return http.StatusUnauthorized
	case errors.CodeRateLimit:
		return http.StatusTooManyRequests
	case errors.CodeTimeout:
		return http.StatusGatewayTimeout
	case errors.CodeNetworkError, errors.CodeDNSError:
		return http.StatusBadGateway
	case errors.CodeNoEndpoint, errors.CodeModelError:
		return http.StatusServiceUnavailable
	default:
		return http.StatusServiceUnavailable
	}
}
