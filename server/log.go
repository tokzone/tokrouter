package server

import (
	"log/slog"
	"os"
	"strings"
)

// SlogLogger wraps slog.Logger
type SlogLogger struct {
	*slog.Logger
}

// NewSlogLogger creates a new SlogLogger
func NewSlogLogger(level string) *SlogLogger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
	slog.SetDefault(logger)
	return &SlogLogger{logger}
}

// Debug logs at debug level
func (l *SlogLogger) Debug(msg string, fields map[string]interface{}) {
	if fields == nil {
		l.Logger.Debug(msg)
		return
	}
	l.Logger.Debug(msg, toArgs(fields)...)
}

// Info logs at info level
func (l *SlogLogger) Info(msg string, fields map[string]interface{}) {
	if fields == nil {
		l.Logger.Info(msg)
		return
	}
	l.Logger.Info(msg, toArgs(fields)...)
}

// Warn logs at warn level
func (l *SlogLogger) Warn(msg string, fields map[string]interface{}) {
	if fields == nil {
		l.Logger.Warn(msg)
		return
	}
	l.Logger.Warn(msg, toArgs(fields)...)
}

// Error logs at error level
func (l *SlogLogger) Error(msg string, fields map[string]interface{}) {
	if fields == nil {
		l.Logger.Error(msg)
		return
	}
	l.Logger.Error(msg, toArgs(fields)...)
}

// Global logger for backward compatibility
var defaultLogger *SlogLogger

// SetLogLevel sets the global log level
func SetLogLevel(level string) {
	defaultLogger = NewSlogLogger(level)
}

func init() {
	// Default logger
	defaultLogger = NewSlogLogger("info")
}

// Debug logs at debug level (convenience function)
func Debug(msg string, fields map[string]interface{}) {
	defaultLogger.Debug(msg, fields)
}

// Info logs at info level (convenience function)
func Info(msg string, fields map[string]interface{}) {
	defaultLogger.Info(msg, fields)
}

// Warn logs at warn level (convenience function)
func Warn(msg string, fields map[string]interface{}) {
	defaultLogger.Warn(msg, fields)
}

// Error logs at error level (convenience function)
func Error(msg string, fields map[string]interface{}) {
	defaultLogger.Error(msg, fields)
}

// toArgs converts map to slog args
func toArgs(fields map[string]interface{}) []any {
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		// Redact sensitive fields
		if isSensitive(k) {
			if s, ok := v.(string); ok {
				v = redactKey(s)
			}
		}
		args = append(args, k, v)
	}
	return args
}

// isSensitive checks if field name contains sensitive keywords
func isSensitive(key string) bool {
	keyLower := strings.ToLower(key)
	return strings.Contains(keyLower, "key") ||
		strings.Contains(keyLower, "token") ||
		strings.Contains(keyLower, "secret")
}

// redactKey redacts API keys in strings
func redactKey(s string) string {
	if len(s) > 8 {
		return s[:4] + "..." + s[len(s)-4:]
	}
	return "[REDACTED]"
}

// LogRequest logs an incoming request
func LogRequest(method, path, model string, headers map[string][]string) {
	// Redact sensitive headers
	safeHeaders := make(map[string]string)
	for k, v := range headers {
		keyLower := strings.ToLower(k)
		if keyLower == "authorization" || keyLower == "x-api-key" {
			if len(v) > 0 {
				safeHeaders[k] = redactKey(v[0])
			}
		} else {
			if len(v) > 0 {
				safeHeaders[k] = v[0]
			}
		}
	}

	Info("request", map[string]interface{}{
		"method":  method,
		"path":    path,
		"model":   model,
		"headers": safeHeaders,
	})
}
