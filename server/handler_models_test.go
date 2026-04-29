package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleModels_ListReturnsCodexFields(t *testing.T) {
	handler := HandleModels()
	req := httptest.NewRequest("GET", "/v1/models", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var list modelList
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal model list: %v", err)
	}
	if list.Object != "list" {
		t.Errorf("expected object='list', got '%s'", list.Object)
	}
	if len(list.Data) == 0 {
		t.Fatal("expected non-empty model list")
	}

	// Verify each model has the Codex-expected fields
	for _, m := range list.Data {
		if m.ID == "" {
			t.Error("model missing id")
		}
		if m.Slug == "" {
			t.Error("model missing slug")
		}
		if m.Object != "model" {
			t.Errorf("model %s: expected object='model', got '%s'", m.ID, m.Object)
		}
		if m.Created == 0 {
			t.Error("model missing created")
		}
		if m.OwnedBy == "" {
			t.Error("model missing owned_by")
		}
		if len(m.InputModalities) == 0 {
			t.Errorf("model %s: missing input_modalities", m.ID)
		}
	}
}

func TestHandleModels_ListHasETag(t *testing.T) {
	handler := HandleModels()
	req := httptest.NewRequest("GET", "/v1/models", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	etag := w.Header().Get("X-Models-Etag")
	if etag == "" {
		t.Error("missing X-Models-Etag header")
	}
}

func TestHandleModels_IndividualModelFound(t *testing.T) {
	handler := HandleModels()
	req := httptest.NewRequest("GET", "/v1/models/gpt-5.4", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for known model, got %d", w.Code)
	}

	var m modelEntry
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("unmarshal model: %v", err)
	}
	if m.Slug == "" {
		t.Error("individual model missing slug")
	}
	if m.ContextWindow == 0 && m.MaxContextWindow == 0 {
		t.Errorf("model %s: context_window should not be 0", m.ID)
	}
}

func TestHandleModels_IndividualModelNotFound(t *testing.T) {
	handler := HandleModels()
	req := httptest.NewRequest("GET", "/v1/models/nonexistent-model-xyz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown model, got %d", w.Code)
	}
}

func TestHandleModels_NoDuplicateModels(t *testing.T) {
	handler := HandleModels()
	req := httptest.NewRequest("GET", "/v1/models", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var list modelList
	json.Unmarshal(w.Body.Bytes(), &list)

	seen := map[string]bool{}
	for _, m := range list.Data {
		if seen[m.ID] {
			t.Errorf("duplicate model id: %s", m.ID)
		}
		seen[m.ID] = true
	}
}
