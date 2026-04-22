// Package server provides HTTP server functionality for tokrouter.
//
// The server exposes the following endpoints:
//   - POST /v1/chat/completions - OpenAI-compatible chat endpoint
//   - POST /v1/messages - Anthropic-compatible messages endpoint
//   - GET /status - Provider status endpoint
//   - GET /health - Health check with dependency status
//
// The server uses the router.Service for request forwarding and supports
// both streaming (SSE) and non-streaming responses.
//
// Basic usage:
//
//	svc, _ := router.NewServiceFromConfig(cfg)
//	server := server.NewServer(svc, traceCfg)
//	server.Run()
package server
