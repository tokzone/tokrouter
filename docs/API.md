# TokRouter API 文档

## 端点概览

TokRouter 提供两个协议的 API 入口，分别处理不同协议格式的请求。

| 端点 | 方法 | 说明 | 协议 |
|------|------|------|------|
| /v1/chat/completions | POST | OpenAI 兼容聊天接口 | OpenAI |
| /v1/messages | POST | Anthropic 兼容消息接口 | Anthropic |
| /status | GET | 端点运行时状态 | - |
| /health | GET | 健康检查（含依赖状态） | - |
| /shutdown | POST | 优雅关闭服务器 | - |

---

## 多协议架构

TokRouter 根据 **URL 路径** 决定输入协议：

- `POST /v1/chat/completions` → **输入协议 = OpenAI**。调用 `ForwardOpenAI`。
- `POST /v1/messages` → **输入协议 = Anthropic**。调用 `ForwardAnthropic`。

每个 Endpoint 声明其**协议能力列表** (`protocols`)。路由时：
- 若 endpoint 支持输入协议 → **直传**（无转换）
- 若 endpoint 不支持输入协议 → 回退到 `protocols[0]`，自动进行协议转换

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

返回所有 Provider 的运行时状态。

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

| 字段 | 说明 |
|------|------|
| name | Provider 的 PrimaryBaseURL |
| protocol | 默认协议（protocols[0]） |
| healthy | 至少一个模型健康则为 true |
| models | 各模型的健康状态 |

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
  "version": "v0.7.2",
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
  "version": "v0.7.2",
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
