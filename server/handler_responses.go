package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tokzone/fluxcore"
	fluxerrors "github.com/tokzone/fluxcore/errors"

	"github.com/tokzone/tokrouter/router"
)

// --- OpenAI Responses API types ---

type responsesRequest struct {
	Model        string          `json:"model"`
	Input        json.RawMessage `json:"input"`
	Instructions string          `json:"instructions,omitempty"`
	Stream       bool            `json:"stream,omitempty"`
}

// responsesInputItem represents a single item in the input array.
// Items with type "message" carry role+content; other types are treated as content blocks.
type responsesInputItem struct {
	Type    string          `json:"type"`
	Role    string          `json:"role,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`
	Text    string          `json:"text,omitempty"`
}

type responsesOutput struct {
	Type    string              `json:"type"`
	Content []responsesContent  `json:"content"`
}

type responsesContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responsesSuccess struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Model   string             `json:"model"`
	Output  []responsesOutput  `json:"output"`
	Usage   *responsesUsage    `json:"usage,omitempty"`
}

type responsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// chatCompletionMsg mirrors the Chat Completions response message.
type chatCompletionChoice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

type chatCompletionResp struct {
	ID    string                  `json:"id"`
	Model string                  `json:"model"`
	Choices []chatCompletionChoice `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// --- Chat Completions SSE chunk types ---

type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type chatChunk struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
	Usage *chatUsage `json:"usage"`
}

// HandleResponses handles requests on the OpenAI Responses API endpoint (POST /v1/responses).
func HandleResponses(r router.Router) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		body, model, stream, ok := readAndParseResponses(w, req)
		if !ok {
			return
		}

		chatBody, err := responsesToChat(body)
		if err != nil {
			WriteErrorResponse(w, http.StatusBadRequest, NewErrorResponseWithCode(fluxerrors.CodeInvalidRequest, err.Error()))
			return
		}

		LogRequest(req.Method, req.URL.Path, model, req.Header)

		// Log conversion details for debugging
		var reqParsed responsesRequest
		json.Unmarshal(body, &reqParsed)
		inputItems, isMsg := parseInputItems(reqParsed.Input)
		itemTypes := "string"
		firstType := ""
		if isMsg {
			itemTypes = "message-items"
			if len(inputItems) > 0 {
				firstType = inputItems[0].Type + "/" + inputItems[0].Role
			}
		} else {
			var s string
			if json.Unmarshal(reqParsed.Input, &s) != nil {
				itemTypes = "content-blocks"
			}
		}
		var msgCount int
		var chatPreview map[string]interface{}
		json.Unmarshal(chatBody, &chatPreview)
		if msgs, ok := chatPreview["messages"].([]interface{}); ok {
			msgCount = len(msgs)
		}
		Info("responses converted", map[string]interface{}{
			"orig_bytes":   len(body),
			"chat_bytes":   len(chatBody),
			"input_type":   itemTypes,
			"input_items":  len(inputItems),
			"first_item":   firstType,
			"msg_count":    msgCount,
		})

		if stream {
			streamCtx, streamCancel := context.WithTimeout(req.Context(), 10*time.Minute)
			defer streamCancel()
			result, actualModel, providerURL, err := r.ForwardStreamOpenAI(streamCtx, chatBody, model)
			if err != nil {
				writeStreamError(w, model, err)
				return
			}
			defer result.Close()

			writeResponsesSSE(w, req, result)

			streamErr := result.Error()
			if streamErr != nil {
				Error("responses stream had error", map[string]interface{}{
					"error": streamErr.Error(),
				})
			}
			usage := result.Usage()
			Info("responses stream completed", map[string]interface{}{
				"model":         actualModel,
				"provider":      providerURL,
				"input_tokens":  usage.InputTokens,
				"output_tokens": usage.OutputTokens,
			})
			r.RecordStreamUsage(usage, actualModel, providerURL)
		} else {
			resp, usage, err := r.ForwardOpenAI(req.Context(), chatBody, model)
			if err != nil {
				ClassifyAndWriteError(w, err)
				Error("responses proxy failed", map[string]interface{}{
					"model": model,
					"error": err.Error(),
				})
				return
			}

			responsesResp, err := chatToResponses(resp, model)
			if err != nil {
				WriteErrorResponse(w, http.StatusInternalServerError, NewErrorResponseWithCode(fluxerrors.CodeServerError, "failed to convert response"))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(responsesResp)

			Info("responses request completed", map[string]interface{}{
				"model":         model,
				"input_tokens":  usage.InputTokens,
				"output_tokens": usage.OutputTokens,
			})
		}
	}
}

// readAndParseResponses reads the body and parses the Responses API request fields.
func readAndParseResponses(w http.ResponseWriter, r *http.Request) (body []byte, model string, stream bool, ok bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxRequestBodySize))
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, NewErrorResponseWithCode(fluxerrors.CodeInvalidRequest, err.Error()))
		return nil, "", false, false
	}
	r.Body.Close()

	var req responsesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, NewErrorResponseWithCode(fluxerrors.CodeInvalidRequest, "invalid JSON body"))
		return nil, "", false, false
	}
	return body, req.Model, req.Stream, true
}

// responsesToChat converts a Responses API request body to Chat Completions format.
func responsesToChat(responsesBody []byte) ([]byte, error) {
	var req responsesRequest
	if err := json.Unmarshal(responsesBody, &req); err != nil {
		return nil, fmt.Errorf("parse responses request: %w", err)
	}

	var messages []map[string]interface{}

	if req.Instructions != "" {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": req.Instructions,
		})
	}

	// input can be a string, a content-block array, or a message-item array.
	items, isMultiMsg := parseInputItems(req.Input)
	if isMultiMsg {
		for _, item := range items {
			msg := convertMessageItem(item)
			if msg != nil {
				messages = append(messages, msg)
			}
		}
	} else {
		userContent := convertInputToContent(req.Input)
		messages = append(messages, map[string]interface{}{
			"role":    "user",
			"content": userContent,
		})
	}

	chatReq := map[string]interface{}{
		"model":    req.Model,
		"messages": messages,
	}
	if req.Stream {
		chatReq["stream"] = true
	}

	return json.Marshal(chatReq)
}

// parseInputItems tries to parse input as an array of message items.
// Returns (items, true) if items have message semantics, (nil, false) otherwise.
func parseInputItems(input json.RawMessage) ([]responsesInputItem, bool) {
	// Try string first — not a message array
	var s string
	if json.Unmarshal(input, &s) == nil {
		return nil, false
	}

	var items []responsesInputItem
	if json.Unmarshal(input, &items) != nil {
		return nil, false
	}

	// If any item has type "message", treat the whole array as message items.
	for _, item := range items {
		if item.Type == "message" {
			return items, true
		}
	}
	return nil, false
}

// convertMessageItem converts a Responses API message item to a Chat Completions message.
func convertMessageItem(item responsesInputItem) map[string]interface{} {
	role := item.Role
	if role == "" {
		role = "user"
	}
	// Map "developer" role to "system" for Chat Completions compatibility
	if role == "developer" {
		role = "system"
	}

	content := extractTextContent(item.Content)
	return map[string]interface{}{
		"role":    role,
		"content": content,
	}
}

// extractTextContent extracts the primary text from a Responses API content array.
// Content can be a string or an array of content blocks.
func extractTextContent(content json.RawMessage) interface{} {
	if len(content) == 0 {
		return ""
	}

	// Try string
	var s string
	if json.Unmarshal(content, &s) == nil {
		return s
	}

	// Try array of content blocks
	var blocks []map[string]interface{}
	if json.Unmarshal(content, &blocks) != nil {
		return string(content)
	}

	// Collect text from all blocks
	var texts []string
	var images []map[string]interface{}
	for _, block := range blocks {
		blockType, _ := block["type"].(string)
		switch blockType {
		case "input_text", "output_text":
			if t, ok := block["text"].(string); ok {
				texts = append(texts, t)
			}
		case "input_image":
			if img, ok := block["image_url"]; ok {
				images = append(images, map[string]interface{}{
					"type":      "image_url",
					"image_url": img,
				})
			}
		default:
			// Pass through unknown block types
		}
	}

	if len(images) > 0 && len(texts) > 0 {
		// Multimodal: return content array
		var result []map[string]interface{}
		for _, t := range texts {
			result = append(result, map[string]interface{}{
				"type": "text",
				"text": t,
			})
		}
		result = append(result, images...)
		return result
	}
	if len(images) > 0 {
		return images
	}
	if len(texts) == 1 {
		return texts[0]
	}
	if len(texts) > 1 {
		return strings.Join(texts, "\n")
	}
	return ""
}

// convertInputToContent converts a non-message-item input array to Chat Completions content.
func convertInputToContent(input json.RawMessage) interface{} {
	var s string
	if json.Unmarshal(input, &s) == nil {
		return s
	}

	var blocks []map[string]interface{}
	if json.Unmarshal(input, &blocks) != nil {
		return string(input)
	}

	var result []map[string]interface{}
	for _, block := range blocks {
		blockType, _ := block["type"].(string)
		switch blockType {
		case "input_text":
			result = append(result, map[string]interface{}{
				"type": "text",
				"text": block["text"],
			})
		case "input_image":
			result = append(result, map[string]interface{}{
				"type":      "image_url",
				"image_url": block["image_url"],
			})
		default:
			result = append(result, block)
		}
	}
	return result
}

// chatToResponses converts a Chat Completions response to Responses API format.
func chatToResponses(chatBody []byte, model string) ([]byte, error) {
	var chat chatCompletionResp
	if err := json.Unmarshal(chatBody, &chat); err != nil {
		return nil, fmt.Errorf("parse chat response: %w", err)
	}

	content := ""
	if len(chat.Choices) > 0 {
		content = chat.Choices[0].Message.Content
	}

	resp := responsesSuccess{
		ID:     chat.ID,
		Object: "response",
		Model:  model,
		Output: []responsesOutput{
			{
				Type: "message",
				Content: []responsesContent{
					{Type: "output_text", Text: content},
				},
			},
		},
		Usage: &responsesUsage{
			InputTokens:  chat.Usage.PromptTokens,
			OutputTokens: chat.Usage.CompletionTokens,
		},
	}

	return json.Marshal(resp)
}

// writeResponsesSSE converts Chat Completions SSE stream to Responses API SSE format.
func writeResponsesSSE(w http.ResponseWriter, r *http.Request, result *fluxcore.StreamResult) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteErrorResponse(w, http.StatusInternalServerError, NewErrorResponseWithCode(fluxerrors.CodeServerError, "streaming not supported"))
		return
	}

	responseID := ""
	responseModel := ""
	createdSent := false
	itemAdded := false
	contentPartAdded := false
	seq := 0
	var accumulatedUsage *chatUsage

	for chunk := range result.Ch {
		scanner := bufio.NewScanner(bytes.NewReader(chunk))
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				continue
			}

			var cc chatChunk
			if json.Unmarshal([]byte(data), &cc) != nil {
				continue
			}

			// Capture response ID and model from first chunk
			if responseID == "" && cc.ID != "" {
				responseID = cc.ID
				responseModel = cc.Model
			}

			// Send response.created + response.in_progress on first chunk
			if !createdSent && responseID != "" {
				now := time.Now().Unix()
				seq++
				createdData, _ := json.Marshal(map[string]interface{}{
					"type": "response.created",
					"sequence_number": seq,
					"response": map[string]interface{}{
						"id":         responseID,
						"object":     "response",
						"created_at": now,
						"status":     "in_progress",
						"model":      responseModel,
						"output":     []interface{}{},
					},
				})
				fmt.Fprintf(w, "event: response.created\ndata: %s\n\n", createdData)
				flusher.Flush()

				seq++
				inProgressData, _ := json.Marshal(map[string]interface{}{
					"type": "response.in_progress",
					"sequence_number": seq,
					"response": map[string]interface{}{
						"id":     responseID,
						"object": "response",
						"status": "in_progress",
					},
				})
				fmt.Fprintf(w, "event: response.in_progress\ndata: %s\n\n", inProgressData)
				flusher.Flush()
				createdSent = true
			}

			// Send output_item.added before first delta
			if !itemAdded && responseID != "" {
				seq++
				itemID := responseID + "_item_0"
				itemData, _ := json.Marshal(map[string]interface{}{
					"type":            "response.output_item.added",
					"sequence_number": seq,
					"output_index":    0,
					"item": map[string]interface{}{
						"id":      itemID,
						"type":    "message",
						"status":  "in_progress",
						"role":    "assistant",
						"content": []interface{}{},
					},
				})
				fmt.Fprintf(w, "event: response.output_item.added\ndata: %s\n\n", itemData)
				flusher.Flush()
				itemAdded = true
			}

			// Send content_part.added before first delta
			if !contentPartAdded && responseID != "" {
				seq++
				itemID := responseID + "_item_0"
				partData, _ := json.Marshal(map[string]interface{}{
					"type":            "response.content_part.added",
					"sequence_number": seq,
					"output_index":    0,
					"content_index":   0,
					"item_id":         itemID,
					"part": map[string]interface{}{
						"type":        "output_text",
						"text":        "",
						"annotations": []interface{}{},
					},
				})
				fmt.Fprintf(w, "event: response.content_part.added\ndata: %s\n\n", partData)
				flusher.Flush()
				contentPartAdded = true
			}

			if len(cc.Choices) > 0 && cc.Choices[0].Delta.Content != "" {
				delta := cc.Choices[0].Delta.Content
				seq++
				itemID := responseID + "_item_0"
				deltaData, _ := json.Marshal(map[string]interface{}{
					"type":            "response.output_text.delta",
					"sequence_number": seq,
					"item_id":         itemID,
					"output_index":    0,
					"content_index":   0,
					"delta":           delta,
				})
				fmt.Fprintf(w, "event: response.output_text.delta\ndata: %s\n\n", deltaData)
				flusher.Flush()
			}

			if cc.Usage != nil {
				accumulatedUsage = cc.Usage
			}
		}
	}

	// Send closing events: output_text.done → content_part.done → output_item.done → response.completed
	if contentPartAdded {
		itemID := responseID + "_item_0"

		seq++
		textDoneData, _ := json.Marshal(map[string]interface{}{
			"type":            "response.output_text.done",
			"sequence_number": seq,
			"item_id":         itemID,
			"output_index":    0,
			"content_index":   0,
			"text":            "",
		})
		fmt.Fprintf(w, "event: response.output_text.done\ndata: %s\n\n", textDoneData)
		flusher.Flush()

		seq++
		partDoneData, _ := json.Marshal(map[string]interface{}{
			"type":            "response.content_part.done",
			"sequence_number": seq,
			"output_index":    0,
			"content_index":   0,
			"item_id":         itemID,
			"part": map[string]interface{}{
				"type":        "output_text",
				"text":        "",
				"annotations": []interface{}{},
			},
		})
		fmt.Fprintf(w, "event: response.content_part.done\ndata: %s\n\n", partDoneData)
		flusher.Flush()

		seq++
		seq++
		itemDoneData, _ := json.Marshal(map[string]interface{}{
			"type":            "response.output_item.done",
			"sequence_number": seq,
			"output_index":    0,
			"item": map[string]interface{}{
				"id":      itemID,
				"type":    "message",
				"status":  "completed",
				"role":    "assistant",
				"content": []interface{}{},
			},
		})
		fmt.Fprintf(w, "event: response.output_item.done\ndata: %s\n\n", itemDoneData)
		flusher.Flush()
	}

	// Send response.completed after stream ends
	seq++
	completedData := map[string]interface{}{
		"type":            "response.completed",
		"sequence_number": seq,
		"response": map[string]interface{}{
			"id":     responseID,
			"object": "response",
			"status": "completed",
			"model":  responseModel,
		},
	}
	if accumulatedUsage != nil {
		completedData["response"].(map[string]interface{})["usage"] = map[string]int{
			"input_tokens":  accumulatedUsage.PromptTokens,
			"output_tokens": accumulatedUsage.CompletionTokens,
			"total_tokens":  accumulatedUsage.PromptTokens + accumulatedUsage.CompletionTokens,
		}
	}
	sseData, _ := json.Marshal(completedData)
	fmt.Fprintf(w, "event: response.completed\ndata: %s\n\n", sseData)
	flusher.Flush()
}
