package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"tokflux/tokrouter/config"
	"tokflux/tokrouter/router"
	"tokflux/tokrouter/usage"

	"github.com/tokzone/fluxcore/routing"
)

// TraceIDKey is the context key for trace ID
type ctxKey string

const TraceIDKey ctxKey = "trace_id"

// HandleRequest handles all requests using fluxcore
func HandleRequest(routerSvc *router.Service, clientFormat routing.Protocol) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read request body with size limit
		body, err := io.ReadAll(io.LimitReader(r.Body, MaxRequestBodySize))
		if err != nil {
			WriteErrorResponse(w, http.StatusBadRequest, NewErrorResponseWithCode(ErrCodeInvalidRequest, err.Error()))
			return
		}
		r.Body.Close()

		// Parse model for logging (ignore errors - not critical)
		var reqMap map[string]interface{}
		if err := json.Unmarshal(body, &reqMap); err != nil {
			WriteErrorResponse(w, http.StatusBadRequest, NewErrorResponseWithCode(ErrCodeInvalidRequest, "invalid JSON body"))
			return
		}
		model, _ := reqMap["model"].(string)
		stream, _ := reqMap["stream"].(bool)

		LogRequest(r.Method, r.URL.Path, model, r.Header)

		// Handle streaming vs non-streaming
		if stream {
			handleStreaming(w, r, routerSvc, body, clientFormat, model)
		} else {
			handleNonStreaming(w, r, routerSvc, body, clientFormat, model)
		}
	}
}

// handleStreaming handles streaming requests
func handleStreaming(w http.ResponseWriter, r *http.Request, routerSvc *router.Service, body []byte, clientFormat routing.Protocol, model string) {
	result, err := routerSvc.ForwardStream(r.Context(), body, clientFormat)
	if err != nil {
		WriteErrorResponse(w, http.StatusServiceUnavailable, NewErrorResponseWithCode(ErrCodeUpstreamFailed, err.Error()))
		Warn("proxy stream failed", map[string]interface{}{
			"model": model,
			"error": err.Error(),
		})
		return
	}

	// Write SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteErrorResponse(w, http.StatusInternalServerError, NewErrorResponseWithCode(ErrCodeInternalError, "streaming not supported"))
		return
	}

	// Read from channel and write SSE
	for chunk := range result.Ch {
		w.Write(chunk)
		flusher.Flush()
	}

	usage := result.Usage()
	Info("stream completed", map[string]interface{}{
		"model":         model,
		"input_tokens":  usage.InputTokens,
		"output_tokens": usage.OutputTokens,
	})
}

// handleNonStreaming handles non-streaming requests
func handleNonStreaming(w http.ResponseWriter, r *http.Request, routerSvc *router.Service, body []byte, clientFormat routing.Protocol, model string) {
	resp, usage, err := routerSvc.Forward(r.Context(), body, clientFormat)
	if err != nil {
		WriteErrorResponse(w, http.StatusServiceUnavailable, NewErrorResponseWithCode(ErrCodeUpstreamFailed, err.Error()))
		Error("proxy failed", map[string]interface{}{
			"model": model,
			"error": err.Error(),
		})
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)

	Info("request completed", map[string]interface{}{
		"model":         model,
		"input_tokens":  usage.InputTokens,
		"output_tokens": usage.OutputTokens,
	})
}

// HandleStatus handles status endpoint
func HandleStatus(routerSvc *router.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statuses := routerSvc.GetProviderStatuses()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(statuses)
	}
}

// HandleHealth handles health endpoint with dependency checks
func HandleHealth(routerSvc *router.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		details := make(map[string]interface{})

		// Check if at least one healthy endpoint exists
		if routerSvc != nil {
			statuses := routerSvc.GetProviderStatuses()
			healthyCount := 0
			for _, ps := range statuses {
				if ps.Healthy {
					healthyCount++
				}
			}
			details["endpoints"] = map[string]interface{}{
				"total":   len(statuses),
				"healthy": healthyCount,
			}
			if healthyCount == 0 && len(statuses) > 0 {
				status = "degraded"
			}

			// Check usage service
			_, err := routerSvc.GetStats(usage.QueryFilter{})
			if err != nil {
				details["usage"] = "disabled"
			} else {
				details["usage"] = "ok"
			}
		} else {
			details["endpoints"] = map[string]interface{}{
				"total":   0,
				"healthy": 0,
			}
			details["usage"] = "disabled"
		}

		w.Header().Set("Content-Type", "application/json")
		if status == "degraded" {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  status,
			"version": "v0.1.0",
			"details": details,
		})
	}
}

// WriteErrorResponse writes error response
func WriteErrorResponse(w http.ResponseWriter, statusCode int, errResp *ErrorResponse) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errResp)
}

// WithTraceID wraps handler with trace ID
func WithTraceID(next http.HandlerFunc, traceCfg config.TraceConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get(traceCfg.Header)
		if traceID == "" {
			traceID = generateTraceID()
		}
		w.Header().Set(traceCfg.Header, traceID)
		ctx := context.WithValue(r.Context(), TraceIDKey, traceID)
		next(w, r.WithContext(ctx))
	}
}

func generateTraceID() string {
	return fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
}
