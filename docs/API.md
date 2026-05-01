# TokRouter API 文档

## 端点概览

TokRouter 提供多协议 API 入口，支持 Chat Completions、Responses API 和 Anthropic Messages。

| 端点 | 方法 | 说明 | 协议 |
|------|------|------|------|
| /v1/chat/completions | POST | OpenAI 兼容聊天接口 | OpenAI |
| /v1/responses | POST | OpenAI Responses API 接口 | OpenAI |
| /v1/messages | POST | Anthropic 兼容消息接口 | Anthropic |
| /v1/models | GET | 模型列表（含上下文长度等元数据） | - |
| /v1/models/{model} | GET | 单个模型详情 | - |
| /status | GET | 端点运行时状态 | - |
| /health | GET | 健康检查（含依赖状态） | - |
| /shutdown | POST | 优雅关闭服务器 | -

---

## 多协议架构

TokRouter 根据 **URL 路径** 决定输入协议：

- `POST /v1/chat/completions` → **输入协议 = OpenAI**。调用 `ForwardOpenAI`。
- `POST /v1/responses` → **输入协议 = OpenAI**。内置 Responses → Chat Completions 转换，调用 `ForwardOpenAI`。
- `POST /v1/messages` → **输入协议 = Anthropic**。调用 `ForwardAnthropic`。

路由时 fluxcore 自动处理协议匹配：
- 若 Service 的 BaseURLs 包含输入协议 → **直传**（无转换）
- 若不包含 → 按 `ProtocolPriority()`（OpenAI > Anthropic > Gemini > Cohere）选择可用协议，自动进行协议转换

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

## POST /v1/responses

OpenAI Responses API 兼容接口。TokRouter 自动将 Responses API 格式转换为 Chat Completions 格式，
转发到上游，再将响应转换回 Responses API 格式。

支持功能：
- `input` 支持字符串、content blocks 数组、message items 数组三种格式
- `instructions` → system message
- `developer` role → `system` role（Chat Completions 兼容）
- 多模态（text + image_url）
- 流式 SSE（完整生命周期事件：created → in_progress → output_item.added → content_part.added → delta → done → completed）

### 请求

```json
{
  "model": "gpt-4",
  "input": [{"type": "message", "role": "user", "content": "Hello!"}],
  "instructions": "You are a helpful assistant.",
  "stream": false
}
```

### 响应

```json
{
  "id": "resp_abc123",
  "object": "response",
  "model": "gpt-4",
  "output": [
    {
      "type": "message",
      "content": [{"type": "output_text", "text": "Hello! How can I help?"}]
    }
  ],
  "usage": {
    "input_tokens": 10,
    "output_tokens": 20
  }
}
```

### curl 示例（非流式）

```bash
curl -X POST http://localhost:8765/v1/responses \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "input": [{"type": "message", "role": "user", "content": "Hello"}],
    "instructions": "Be helpful."
  }'
```

### curl 示例（流式 SSE）

```bash
curl -X POST http://localhost:8765/v1/responses \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "input": "Hello world",
    "stream": true
  }'
```

---

## GET /v1/models

返回所有可用模型列表，包含上下文长度等元数据（兼容 Codex CLI 等 OpenAI SDK 工具）。

### 响应

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4",
      "slug": "gpt-4",
      "object": "model",
      "created": 1767196800,
      "owned_by": "openai",
      "context_window": 8192,
      "max_context_window": 8192,
      "input_modalities": ["text"],
      "supports_reasoning_summaries": false
    }
  ]
}
```

### 获取单个模型

```bash
curl http://localhost:8765/v1/models/gpt-4
```

---

## GET /status

返回所有 ServiceEndpoint 的运行时状态（含双层熔断信息）。

### 响应

```json
[
  {
    "name": "openai",
    "protocol": "openai",
    "healthy": true,
    "models": [
      {"name": "gpt-4", "healthy": true},
      {"name": "gpt-3.5-turbo", "healthy": true}
    ]
  }
]
```

| 字段 | 说明 |
|------|------|
| name | ServiceEndpoint 名称 |
| protocol | 主协议（BaseURLs 中优先级最高的） |
| healthy | ServiceEndpoint 网络熔断 + 至少一个 Route 模型熔断均健康 |
| models | 各 Route 的健康状态（双层：网络 + 模型熔断） |

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
  "version": "v0.7.4",
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
  "version": "v0.7.4",
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

## POST /shutdown

优雅关闭 tokrouter 服务器。

### 响应

```
200 OK
```

### curl 示例

```bash
curl -X POST http://localhost:8765/shutdown
```

---

## 错误响应

所有错误响应格式:

```json
{
  "type": "error",
  "code": "INVALID_REQUEST",
  "message": "Invalid request format: missing model field",
  "suggestion": "Check that your request body is valid JSON and matches the expected format"
}
```

### 错误码

| 错误码 | HTTP 状态码 | 说明 |
|--------|-------------|------|
| INVALID_REQUEST | 400 | 请求格式错误 |
| AUTH_ERROR | 401 | 上游认证失败 |
| RATE_LIMIT | 429 | 上游限流 |
| TIMEOUT | 504 | 上游超时 |
| NETWORK_ERROR | 502 | 网络连接失败 |
| DNS_ERROR | 502 | DNS 解析失败 |
| NO_ENDPOINT | 503 | 无可用端点 |
| MODEL_ERROR | 503 | 模型不可用 |
| SERVER_ERROR | 503 | 服务器内部错误 |

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| type | string | 固定 `"error"` |
| code | string | 机器可读错误码 |
| message | string | 人类可读错误描述 |
| suggestion | string | 可选的恢复建议 |
