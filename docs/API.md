# TokRouter API 文档

## 端点概览

| 端点 | 方法 | 说明 |
|------|------|------|
| /v1/chat/completions | POST | OpenAI 兼容聊天接口 |
| /v1/messages | POST | Anthropic 兼容消息接口 |
| /status | GET | Key 状态 |
| /health | GET | 健康检查（含依赖状态）

---

## POST /v1/chat/completions

OpenAI 兼容的聊天补全接口，支持流式和非流式响应。

### 请求

```json
{
  "model": "gpt-4",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello!"}
  ],
  "stream": false,
  "temperature": 0.7,
  "max_tokens": 1024
}
```

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| model | string | 是 | 模型名称 |
| messages | array | 是 | 消息数组 |
| stream | boolean | 否 | 是否流式响应，默认 false |
| temperature | number | 否 | 温度参数，0-2 |
| max_tokens | integer | 否 | 最大输出 token 数 |

### 非流式响应

```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "gpt-4",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "Hello! How can I help you today?"
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 8,
    "total_tokens": 18
  }
}
```

### 流式响应

设置 `"stream": true` 后，返回 SSE 格式:

```
data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","choices":[{"delta":{"content":"Hello"},"index":0}]}

data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","choices":[{"delta":{"content":"!"},"index":0}]}

data: [DONE]
```

### curl 示例

```bash
# 非流式
curl -X POST http://localhost:8765/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello"}]
  }'

# 流式
curl -X POST http://localhost:8765/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'
```

---

## POST /v1/messages

Anthropic 兼容的消息接口。

### 请求

```json
{
  "model": "claude-3-opus",
  "max_tokens": 1024,
  "messages": [
    {"role": "user", "content": "Hello!"}
  ],
  "stream": false
}
```

### 响应

```json
{
  "id": "msg_xxx",
  "type": "message",
  "role": "assistant",
  "content": [{"type": "text", "text": "Hello! How can I help you?"}],
  "model": "claude-3-opus",
  "usage": {
    "input_tokens": 10,
    "output_tokens": 8
  }
}
```

### curl 示例

```bash
curl -X POST http://localhost:8765/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-opus",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

---

## GET /status

返回所有提供者的运行时状态。

### 响应

```json
[
  {
    "name": "https://api.openai.com/v1",
    "protocol": "openai",
    "healthy": true,
    "models": [
      {"name": "gpt-4", "healthy": true},
      {"name": "gpt-3.5-turbo", "healthy": true}
    ]
  }
]
```

### curl 示例

```bash
curl http://localhost:8765/status
```

---

## GET /health

健康检查端点，包含依赖状态。

### 响应 (正常)

```json
{
  "status": "ok",
  "version": "v0.6.0",
  "details": {
    "endpoints": {"total": 2, "healthy": 2},
    "usage": "ok"
  }
}
```

### 响应 (降级)

当没有健康的端点时:

```json
{
  "status": "degraded",
  "version": "v0.6.0",
  "details": {
    "endpoints": {"total": 2, "healthy": 0},
    "usage": "ok"
  }
}
```

### curl 示例

```bash
curl http://localhost:8765/health
```

---

## 错误响应

所有错误响应格式:

```json
{
  "error": {
    "type": "invalid_request_error",
    "message": "invalid JSON body"
  }
}
```

### 错误类型

| 类型 | HTTP 状态码 | 说明 |
|------|-------------|------|
| invalid_request_error | 400 | 请求格式错误 |
| upstream_error | 503 | 上游服务不可用 |
| internal_error | 500 | 内部错误 |