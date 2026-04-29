package server

import (
	"testing"

	"github.com/tokzone/fluxcore/errors"
)

func TestNewErrorResponseWithCode(t *testing.T) {
	tests := []struct {
		name     string
		code     errors.ErrorCode
		details  string
		wantCode string
		wantMsg  bool
	}{
		{
			name:     "invalid request",
			code:     errors.CodeInvalidRequest,
			details:  "missing model field",
			wantCode: string(errors.CodeInvalidRequest),
			wantMsg:  true,
		},
		{
			name:     "server error",
			code:     errors.CodeServerError,
			details:  "connection timeout",
			wantCode: string(errors.CodeServerError),
			wantMsg:  true,
		},
		{
			name:     "no endpoint",
			code:     errors.CodeNoEndpoint,
			details:  "",
			wantCode: string(errors.CodeNoEndpoint),
			wantMsg:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errResp := NewErrorResponseWithCode(tt.code, tt.details)
			if errResp == nil {
				t.Fatal("NewErrorResponseWithCode returned nil")
			}
			if errResp.Code != tt.wantCode {
				t.Errorf("Code = %s, want %s", errResp.Code, tt.wantCode)
			}
			if tt.wantMsg && errResp.Message == "" {
				t.Error("Message should not be empty")
			}
			if errResp.Suggestion == "" {
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
				Code:    string(errors.CodeInvalidRequest),
				Message: "test message",
			},
			wantStr: "[invalid_request] test message",
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
