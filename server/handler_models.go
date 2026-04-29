package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/tokzone/tokrouter/config"
)

type modelEntry struct {
	ID                       string   `json:"id"`
	Slug                     string   `json:"slug"`
	Object                   string   `json:"object"`
	Created                  int64    `json:"created"`
	OwnedBy                  string   `json:"owned_by"`
	ContextWindow            int      `json:"context_window"`
	MaxContextWindow         int      `json:"max_context_window"`
	InputModalities          []string `json:"input_modalities"`
	SupportsReasoningSummary bool     `json:"supports_reasoning_summaries"`
}

type modelList struct {
	Object string       `json:"object"`
	Data   []modelEntry `json:"data"`
}

// HandleModels returns a handler for GET /v1/models that lists all available models
// with metadata fields Codex expects (slug, context_window, input_modalities, etc.).
func HandleModels() http.HandlerFunc {
	presets := config.BuiltinPresets
	seen := map[string]bool{}
	var models []modelEntry

	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	for _, preset := range presets {
		for _, m := range preset.DefaultModels {
			if seen[m.Name] {
				continue
			}
			seen[m.Name] = true
			models = append(models, modelEntry{
				ID:                       m.Name,
				Slug:                     m.Name,
				Object:                   "model",
				Created:                  created,
				OwnedBy:                  preset.Name,
				ContextWindow:            m.Context,
				MaxContextWindow:         m.Context,
				InputModalities:          []string{"text"},
				SupportsReasoningSummary: false,
			})
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Support GET /v1/models/{model} for individual model lookup
		path := strings.TrimPrefix(r.URL.Path, "/v1/models/")
		if path != "" && path != r.URL.Path {
			// Individual model request
			w.Header().Set("Content-Type", "application/json")
			for _, m := range models {
				if m.ID == path {
					json.NewEncoder(w).Encode(m)
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "model not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Models-Etag", "tokrouter-models-v1")
		json.NewEncoder(w).Encode(modelList{
			Object: "list",
			Data:   models,
		})
	}
}
