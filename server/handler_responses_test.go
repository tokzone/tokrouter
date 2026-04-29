package server

import (
	"encoding/json"
	"testing"
)

func TestChatChunkUsageParsing(t *testing.T) {
	// Simulate DeepSeek final SSE chunk with usage
	data := []byte(`{
		"id": "chatcmpl-123",
		"object": "chat.completion.chunk",
		"model": "deepseek-v4-pro",
		"choices": [{"index": 0, "delta": {}, "finish_reason": "stop"}],
		"usage": {"prompt_tokens": 1234, "completion_tokens": 567, "total_tokens": 1801}
	}`)
	var cc chatChunk
	if err := json.Unmarshal(data, &cc); err != nil {
		t.Fatalf("chatChunk unmarshal failed: %v", err)
	}
	if cc.Usage == nil {
		t.Fatal("expected usage to be non-nil")
	}
	if cc.Usage.PromptTokens != 1234 {
		t.Errorf("expected prompt_tokens=1234, got %d", cc.Usage.PromptTokens)
	}
	if cc.Usage.CompletionTokens != 567 {
		t.Errorf("expected completion_tokens=567, got %d", cc.Usage.CompletionTokens)
	}
	total := cc.Usage.PromptTokens + cc.Usage.CompletionTokens
	if total != 1801 {
		t.Errorf("expected total_tokens=1801, got %d", total)
	}
}

func TestChatChunkUsageParsing_NoUsage(t *testing.T) {
	data := []byte(`{
		"id": "chatcmpl-123",
		"object": "chat.completion.chunk",
		"choices": [{"index": 0, "delta": {"content": "hello"}}]
	}`)
	var cc chatChunk
	if err := json.Unmarshal(data, &cc); err != nil {
		t.Fatalf("chatChunk unmarshal failed: %v", err)
	}
	if cc.Usage != nil {
		t.Error("expected usage to be nil when absent")
	}
	if len(cc.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(cc.Choices))
	}
	if cc.Choices[0].Delta.Content != "hello" {
		t.Errorf("expected delta.content='hello', got '%s'", cc.Choices[0].Delta.Content)
	}
}

func TestResponsesToChat_String(t *testing.T) {
	body := []byte(`{"model":"gpt-5.4","input":"Hello world"}`)
	chat, err := responsesToChat(body)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]interface{}
	json.Unmarshal(chat, &result)

	msgs := result["messages"].([]interface{})
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	msg := msgs[0].(map[string]interface{})
	if msg["role"] != "user" {
		t.Errorf("expected role user, got %s", msg["role"])
	}
	if msg["content"] != "Hello world" {
		t.Errorf("expected content 'Hello world', got %v", msg["content"])
	}
}

func TestResponsesToChat_MultiMessage(t *testing.T) {
	body := []byte(`{
		"model": "deepseek-v4-pro",
		"input": [
			{"type": "message", "role": "user", "content": [{"type": "input_text", "text": "Hello"}]},
			{"type": "message", "role": "assistant", "content": [{"type": "output_text", "text": "Hi there!"}]},
			{"type": "message", "role": "user", "content": [{"type": "input_text", "text": "How are you?"}]}
		],
		"stream": true
	}`)
	chat, err := responsesToChat(body)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]interface{}
	json.Unmarshal(chat, &result)

	msgs := result["messages"].([]interface{})
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	m0 := msgs[0].(map[string]interface{})
	if m0["role"] != "user" || m0["content"] != "Hello" {
		t.Errorf("msg[0]: got role=%v content=%v", m0["role"], m0["content"])
	}
	m1 := msgs[1].(map[string]interface{})
	if m1["role"] != "assistant" || m1["content"] != "Hi there!" {
		t.Errorf("msg[1]: got role=%v content=%v", m1["role"], m1["content"])
	}
	if result["stream"] != true {
		t.Error("expected stream: true")
	}
}

func TestResponsesToChat_WithInstructions(t *testing.T) {
	body := []byte(`{
		"model": "gpt-5.4",
		"instructions": "You are a helpful assistant.",
		"input": "Hello"
	}`)
	chat, err := responsesToChat(body)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]interface{}
	json.Unmarshal(chat, &result)

	msgs := result["messages"].([]interface{})
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	m0 := msgs[0].(map[string]interface{})
	if m0["role"] != "system" {
		t.Errorf("expected system role for instructions, got %s", m0["role"])
	}
}

func TestChatToResponses(t *testing.T) {
	chat := []byte(`{
		"id": "chatcmpl-123",
		"model": "gpt-5.4",
		"choices": [{"message": {"content": "Hello from AI"}}],
		"usage": {"prompt_tokens": 10, "completion_tokens": 5}
	}`)
	resp, err := chatToResponses(chat, "gpt-5.4")
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]interface{}
	json.Unmarshal(resp, &result)

	if result["object"] != "response" {
		t.Errorf("expected object=response, got %v", result["object"])
	}
	output := result["output"].([]interface{})
	if len(output) != 1 {
		t.Fatalf("expected 1 output, got %d", len(output))
	}
	o0 := output[0].(map[string]interface{})
	content := o0["content"].([]interface{})
	c0 := content[0].(map[string]interface{})
	if c0["text"] != "Hello from AI" {
		t.Errorf("expected text 'Hello from AI', got %v", c0["text"])
	}
}

func TestParseInputItems_String(t *testing.T) {
	_, ok := parseInputItems(json.RawMessage(`"hello"`))
	if ok {
		t.Error("string should not be multi-message")
	}
}

func TestParseInputItems_MessageItems(t *testing.T) {
	items, ok := parseInputItems(json.RawMessage(`[
		{"type": "message", "role": "user", "content": "hi"}
	]`))
	if !ok {
		t.Fatal("should detect message items")
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Type != "message" {
		t.Errorf("expected type message, got %s", items[0].Type)
	}
}

func TestParseInputItems_ContentBlocks(t *testing.T) {
	_, ok := parseInputItems(json.RawMessage(`[
		{"type": "input_text", "text": "hello"},
		{"type": "input_image", "image_url": "http://x"}
	]`))
	if ok {
		t.Error("content blocks should not be treated as message items")
	}
}
