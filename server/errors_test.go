package server

import (
	"testing"
)

func TestNewErrorResponse(t *testing.T) {
	err := NewErrorResponse(nil)
	if err == nil {
		t.Fatal("NewErrorResponse returned nil")
	}
	if err.Type != "error" {
		t.Errorf("Type = %s, want error", err.Type)
	}
	if err.Code != ErrCodeInternalError {
		t.Errorf("Code = %s, want %s", err.Code, ErrCodeInternalError)
	}
}

func TestNewErrorResponseWithCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		details  string
		wantCode string
		wantMsg  bool
	}{
		{
			name:     "invalid request",
			code:     ErrCodeInvalidRequest,
			details:  "missing model field",
			wantCode: ErrCodeInvalidRequest,
			wantMsg:  true,
		},
		{
			name:     "upstream failed",
			code:     ErrCodeUpstreamFailed,
			details:  "connection timeout",
			wantCode: ErrCodeUpstreamFailed,
			wantMsg:  true,
		},
		{
			name:     "internal error",
			code:     ErrCodeInternalError,
			details:  "",
			wantCode: ErrCodeInternalError,
			wantMsg:  true,
		},
		{
			name:     "unknown code defaults to internal",
			code:     "UNKNOWN_CODE",
			details:  "",
			wantCode: ErrCodeInternalError,
			wantMsg:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewErrorResponseWithCode(tt.code, tt.details)
			if err == nil {
				t.Fatal("NewErrorResponseWithCode returned nil")
			}
			if err.Code != tt.wantCode {
				t.Errorf("Code = %s, want %s", err.Code, tt.wantCode)
			}
			if tt.wantMsg && err.Message == "" {
				t.Error("Message should not be empty")
			}
			if err.Suggestion == "" {
				t.Error("Suggestion should not be empty")
			}
		})
	}
}

func TestErrorResponseError(t *testing.T) {
	tests := []struct {
		name    string
		err     *ErrorResponse
		wantStr string
	}{
		{
			name: "with code",
			err: &ErrorResponse{
				Type:    "error",
				Code:    ErrCodeInvalidRequest,
				Message: "test message",
			},
			wantStr: "[INVALID_REQUEST] test message",
		},
		{
			name: "without code",
			err: &ErrorResponse{
				Type:    "error",
				Code:    "",
				Message: "test message",
			},
			wantStr: "error: test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.wantStr {
				t.Errorf("Error() = %s, want %s", got, tt.wantStr)
			}
		})
	}
}