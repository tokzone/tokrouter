package server

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNewSlogLogger(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		wantLevel slog.Level
	}{
		{"debug", "debug", slog.LevelDebug},
		{"info", "info", slog.LevelInfo},
		{"warn", "warn", slog.LevelWarn},
		{"error", "error", slog.LevelError},
		{"default", "", slog.LevelInfo},
		{"uppercase", "DEBUG", slog.LevelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewSlogLogger(tt.level)
			if logger == nil {
				t.Fatal("NewSlogLogger returned nil")
			}
		})
	}
}

func TestSlogLoggerMethods(t *testing.T) {
	// Create a buffer to capture output
	buf := &bytes.Buffer{}
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	logger := slog.New(slog.NewJSONHandler(buf, opts))

	wrapped := &SlogLogger{logger}

	// Test Debug
	wrapped.Debug("debug message", map[string]interface{}{"key": "value"})
	if !strings.Contains(buf.String(), "debug message") {
		t.Error("Debug message not logged")
	}
	buf.Reset()

	// Test Info
	wrapped.Info("info message", map[string]interface{}{"key": "value"})
	if !strings.Contains(buf.String(), "info message") {
		t.Error("Info message not logged")
	}
	buf.Reset()

	// Test Warn
	wrapped.Warn("warn message", map[string]interface{}{"key": "value"})
	if !strings.Contains(buf.String(), "warn message") {
		t.Error("Warn message not logged")
	}
	buf.Reset()

	// Test Error
	wrapped.Error("error message", map[string]interface{}{"key": "value"})
	if !strings.Contains(buf.String(), "error message") {
		t.Error("Error message not logged")
	}
	buf.Reset()

	// Test nil fields
	wrapped.Info("info nil", nil)
	if !strings.Contains(buf.String(), "info nil") {
		t.Error("Info nil fields message not logged")
	}
}

func TestSetLogLevel(t *testing.T) {
	SetLogLevel("debug")
	if defaultLogger == nil {
		t.Error("defaultLogger is nil after SetLogLevel")
	}
}

func TestGlobalLogFunctions(t *testing.T) {
	// Set up a buffer to capture output
	buf := &bytes.Buffer{}
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	logger := slog.New(slog.NewJSONHandler(buf, opts))
	defaultLogger = &SlogLogger{logger}

	Debug("global debug", map[string]interface{}{"key": "value"})
	if !strings.Contains(buf.String(), "global debug") {
		t.Error("Global Debug not logged")
	}
	buf.Reset()

	Info("global info", map[string]interface{}{"key": "value"})
	if !strings.Contains(buf.String(), "global info") {
		t.Error("Global Info not logged")
	}
	buf.Reset()

	Warn("global warn", map[string]interface{}{"key": "value"})
	if !strings.Contains(buf.String(), "global warn") {
		t.Error("Global Warn not logged")
	}
	buf.Reset()

	Error("global error", map[string]interface{}{"key": "value"})
	if !strings.Contains(buf.String(), "global error") {
		t.Error("Global Error not logged")
	}
}

func TestToArgs(t *testing.T) {
	fields := map[string]interface{}{
		"normal":    "value",
		"api_key":   "sk-1234567890abcd",
		"token":     "tok-abcdefgh1234",
		"secret":    "secret123",
	}

	args := toArgs(fields)

	// Should have 8 args (4 keys + 4 values)
	if len(args) != 8 {
		t.Errorf("args count = %d, want 8", len(args))
	}

	// Check sensitive keys are redacted
	for i := 1; i < len(args); i += 2 {
		val := args[i]
		if s, ok := val.(string); ok {
			if strings.Contains(s, "sk-1234") && !strings.Contains(s, "...") {
				t.Errorf("api_key not redacted: %s", s)
			}
		}
	}
}

func TestIsSensitive(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"api_key", true},
		{"API_KEY", true},
		{"apiKey", true},
		{"token", true},
		{"access_token", true},
		{"secret", true},
		{"client_secret", true},
		{"normal_field", false},
		{"model", false},
		{"provider", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := isSensitive(tt.key)
			if got != tt.want {
				t.Errorf("isSensitive(%s) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestRedactKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"sk-1234567890abcdef", "sk-1...cdef"},
		{"short", "[REDACTED]"},
		{"exactly8", "[REDACTED]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := redactKey(tt.input)
			if got != tt.want {
				t.Errorf("redactKey(%s) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

func TestLogRequest(t *testing.T) {
	// Set up a buffer to capture output
	buf := &bytes.Buffer{}
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	logger := slog.New(slog.NewJSONHandler(buf, opts))
	defaultLogger = &SlogLogger{logger}

	headers := map[string][]string{
		"Content-Type":   {"application/json"},
		"Authorization":  {"Bearer secret-token-123"},
		"X-Api-Key":      {"sk-1234567890abcdef"},
		"User-Agent":     {"test-client"},
	}

	LogRequest("POST", "/v1/chat/completions", "gpt-4", headers)

	output := buf.String()
	if !strings.Contains(output, "POST") {
		t.Error("Method not logged")
	}
	if !strings.Contains(output, "gpt-4") {
		t.Error("Model not logged")
	}
	// Authorization and X-Api-Key should be redacted
	if strings.Contains(output, "secret-token-123") {
		t.Error("Authorization not redacted")
	}
	if strings.Contains(output, "sk-1234567890abcdef") {
		t.Error("X-Api-Key not redacted")
	}
}

func TestLogRequestEmptyHeaders(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	logger := slog.New(slog.NewJSONHandler(buf, opts))
	defaultLogger = &SlogLogger{logger}

	LogRequest("GET", "/health", "", map[string][]string{})

	if !strings.Contains(buf.String(), "GET") {
		t.Error("Method not logged with empty headers")
	}
}