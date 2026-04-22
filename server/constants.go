package server

// Constants for server configuration
const (
	// Request limits
	MaxRequestBodySize = 10 * 1024 * 1024 // 10MB max request body size

	// Default timeouts
	DefaultReadTimeout  = 30 // seconds
	DefaultWriteTimeout = 30 // seconds

	// SSE streaming
	SSEChannelBuffer = 100 // buffer size for SSE channel
)
