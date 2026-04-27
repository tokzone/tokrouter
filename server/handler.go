package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/tokzone/fluxcore/flux"
	"github.com/tokzone/fluxcore/message"

	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/router"
	"github.com/tokzone/tokrouter/usage"
)

// TraceIDKey is the context key for trace ID
type ctxKey string

const TraceIDKey ctxKey = "trace_id"

// MaxRequestBodySize limits request body size (10MB)
const MaxRequestBodySize = 10 * 1024 * 1024

// requestMeta is a minimal struct for extracting model/stream from the request body
// without unmarshaling the entire payload (e.g., large message arrays).
type requestMeta struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
}

type forwardFunc func(context.Context, []byte, string) ([]byte, *message.Usage, error)
type streamForwardFunc func(context.Context, []byte, string) (*flux.StreamResult, string, string, error)

// HandleOpenAI handles requests on the OpenAI-compatible endpoint (POST /v1/chat/completions).
func HandleOpenAI(r router.Router) http.HandlerFunc {
	return handleRoute(r.ForwardOpenAI, r.ForwardStreamOpenAI, r)
}

// HandleAnthropic handles requests on the Anthropic-compatible endpoint (POST /v1/messages).
func HandleAnthropic(r router.Router) http.HandlerFunc {
	return handleRoute(r.ForwardAnthropic, r.ForwardStreamAnthropic, r)
}

// handleRoute is the common handler for all protocol endpoints.
func handleRoute(fwd forwardFunc, streamFwd streamForwardFunc, r router.Router) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		body, model, stream, ok := readAndParse(w, req)
		if !ok {
			return
		}

		LogRequest(req.Method, req.URL.Path, model, req.Header)

		if stream {
			result, actualModel, providerURL, err := streamFwd(req.Context(), body, model)
			if err != nil {
				writeStreamError(w, model, err)
				return
			}
			defer result.Close()

			writeSSE(w, req, result)

			usage := result.Usage()
			Info("stream completed", map[string]interface{}{
				"model":         actualModel,
				"provider":      providerURL,
				"input_tokens":  usage.InputTokens,
				"output_tokens": usage.OutputTokens,
			})
			r.RecordStreamUsage(usage, actualModel, providerURL)
		} else {
			resp, usage, err := fwd(req.Context(), body, model)
			if err != nil {
				WriteErrorResponse(w, http.StatusServiceUnavailable, NewErrorResponseWithCode(ErrCodeUpstreamFailed, err.Error()))
				Error("proxy failed", map[string]interface{}{
					"model": model,
					"error": err.Error(),
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(resp)

			Info("request completed", map[string]interface{}{
				"model":         model,
				"input_tokens":  usage.InputTokens,
				"output_tokens": usage.OutputTokens,
			})
		}
	}
}

// readAndParse reads the request body with size limit and parses model/stream fields.
func readAndParse(w http.ResponseWriter, r *http.Request) (body []byte, model string, stream bool, ok bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxRequestBodySize))
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, NewErrorResponseWithCode(ErrCodeInvalidRequest, err.Error()))
		return nil, "", false, false
	}
	r.Body.Close()

	var meta requestMeta
	if err := json.Unmarshal(body, &meta); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, NewErrorResponseWithCode(ErrCodeInvalidRequest, "invalid JSON body"))
		return nil, "", false, false
	}
	return body, meta.Model, meta.Stream, true
}

// writeStreamError writes an error response for streaming failures.
func writeStreamError(w http.ResponseWriter, model string, err error) {
	WriteErrorResponse(w, http.StatusServiceUnavailable, NewErrorResponseWithCode(ErrCodeUpstreamFailed, err.Error()))
	Warn("proxy stream failed", map[string]interface{}{
		"model": model,
		"error": err.Error(),
	})
}

// writeSSE writes SSE headers and streams chunks from the result channel.
func writeSSE(w http.ResponseWriter, r *http.Request, result *flux.StreamResult) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteErrorResponse(w, http.StatusInternalServerError, NewErrorResponseWithCode(ErrCodeInternalError, "streaming not supported"))
		return
	}

	for chunk := range result.Ch {
		w.Write(chunk)
		flusher.Flush()
	}
}

// HandleStatus handles status endpoint
func HandleStatus(routerSvc router.Router) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var statuses []router.ProviderStatus
		if routerSvc != nil {
			statuses = routerSvc.ProviderStatuses()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(statuses)
	}
}

// HandleHealth handles health endpoint with dependency checks
func HandleHealth(routerSvc router.Router) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		details := make(map[string]interface{})

		// Check if at least one healthy endpoint exists
		if routerSvc != nil {
			statuses := routerSvc.ProviderStatuses()
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
			_, err := routerSvc.Stats(usage.QueryFilter{})
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
			"version": "v0.7.0",
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
