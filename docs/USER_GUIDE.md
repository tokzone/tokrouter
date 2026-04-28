# TokRouter 用户手册

## 新用户上手指南

按以下步骤快速搭建 LLM API 路由服务：

```bash
# 1. 查看可用预置 Provider
tr list presets

# 2. 添加服务（预置模式：自动填充 BaseURLs、Format、默认模型）
tr add openai --secret sk-xxx

# 3. 查看已配置的服务
tr list services

# 4. 查看所有可用模型
tr list models

# 5. 启动路由服务
tr start

# 6. 查看实时健康状态
tr show health --watch

# 7. 查看用量统计
tr show usage
```

---

## CLI 命令参考

所有命令均支持 `--config` / `-c` 指定配置文件路径（默认 `./config.yaml`）。
Shell 补全：`tr completion bash|zsh|fish`

### tr add — 添加服务

添加 API 服务，支持**预置模式**（推荐）和**自定义模式**。

```bash
# 预置模式：使用内置 Provider 模板
tr add openai --secret sk-xxx
tr add deepseek --secret sk-xxx
tr add openrouter --secret sk-or-xxx

# 指定模型（覆盖预置默认模型）
tr add openai --secret sk-xxx --model gpt-4 --model gpt-4o

# 自定义模式：完全手动配置
tr add --name my-api --secret sk-xxx --base-url https://api.example.com/v1 --format openai --model gpt-4

# 交互式模式（无参数，无 --name）
tr add
```

| 参数 | 说明 |
|------|------|
| `<preset>` | 预置 Provider 名称（如 openai, deepseek, openrouter） |
| `--name` | 服务名称（自定义模式必需） |
| `--secret` | API 密钥 |
| `--base-url` | API 基础 URL（自定义模式） |
| `--format` | 协议格式：openai/anthropic/gemini/cohere |
| `--model` | 模型名称（可多次指定，空格分隔） |

### tr remove — 移除服务

```bash
tr remove <name>
tr remove openai-1
```

### tr list — 列出资源

```bash
# 子命令
tr list services       # 列出所有已配置的服务
tr list models         # 列出所有可用模型
tr list presets        # 列出所有内置 Provider 预置
tr list assistants     # 列出支持的 AI 助手

# 默认（不带子命令）= tr list services
tr list
```

**tr list services 输出示例：**
```
Services
┌──────────────┬────────────┬──────────┬────────┬─────────┐
│ NAME         │ PROVIDER   │ FORMAT   │ MODELS │ STATUS  │
├──────────────┼────────────┼──────────┼────────┼─────────┤
│ openai-main  │ custom     │ openai   │ 2      │ enabled │
│ deepseek-1   │ deepseek   │ openai   │ 1      │ enabled │
└──────────────┴────────────┴──────────┴────────┴─────────┘
```

**tr list models 输出示例：**
```
Available Models
┌─────────────────┬──────────────┬──────────┬──────────┬─────────┐
│ MODEL           │ SERVICE      │ FORMAT   │ PRIORITY │ STATUS  │
├─────────────────┼──────────────┼──────────┼──────────┼─────────┤
│ gpt-4           │ openai-main  │ openai   │ 0        │ enabled │
│ claude-3-opus   │ anthropic-1  │ anthropic│ 50       │ enabled │
└─────────────────┴──────────────┴──────────┴──────────┴─────────┘
```

### tr show — 查看详情

```bash
# 子命令
tr show service <name>     # 查看服务详情（模型列表、优先级、别名）
tr show preset <name>      # 查看预置详情（默认模型、BaseURLs、地区）
tr show config             # 查看当前完整配置
tr show health             # 查看端点健康状态
tr show health --watch     # 实时刷新（每 2 秒）
tr show usage              # 本月用量统计
tr show usage --today      # 今日用量
tr show usage --week       # 本周用量
tr show usage --chart      # 柱状图展示
tr show usage --export csv > stats.csv   # 导出 CSV
tr show usage --export json > stats.json # 导出 JSON

# 默认（不带子命令）= tr show config
tr show
```

**tr show service 输出示例：**
```
Service: openai-main
  Provider:  openai
  Format:    openai
  BaseURLs:  map[openai:https://api.openai.com/v1]
  Status:    enabled
  Models:    2

  ┌───────────────┬────────┬──────────┐
  │ NAME          │ ALIAS  │ PRIORITY │
  ├───────────────┼────────┼──────────┤
  │ gpt-4         │ -      │ 0        │
  │ gpt-4o        │ -      │ 0        │
  └───────────────┴────────┴──────────┘
```

**tr show health 输出示例：**
```
Endpoint Health
┌──────────────────────────────────┬──────────┬─────────┬────────┐
│ NAME                             │ PROTOCOL │ HEALTHY │ MODELS │
├──────────────────────────────────┼──────────┼─────────┼────────┤
│ https://api.openai.com/v1        │ openai   │ ✓       │ 2/2    │
│ https://api.anthropic.com/v1     │ anthropic│ ✓       │ 1/1    │
└──────────────────────────────────┴──────────┴─────────┴────────┘
```

**tr show usage 输出示例：**
```
Usage Summary (month)
┌────────────────┬────────┬────────┬──────────┬────────────┬─────────┐
│ SERVICE        │ INPUT  │ OUTPUT │ REQUESTS │ AVG LATENCY│ SUCCESS │
├────────────────┼────────┼────────┼──────────┼────────────┼─────────┤
│ openai-main    │ 15234  │ 45678  │ 1234     │ 245ms      │ 98.5%   │
│ anthropic-1    │ 23456  │ 12345  │ 567      │ 189ms      │ 99.2%   │
└────────────────┴────────┴────────┴──────────┴────────────┴─────────┘
```

### tr config — 修改服务配置

```bash
# 更新 API 密钥
tr config <name> --secret sk-new-key

# 启用/禁用服务
tr config <name> --enable
tr config <name> --disable

# 添加/移除模型
tr config <name> --add-model gpt-4o
tr config <name> --add-model my-model --alias gpt-4
tr config <name> --remove-model gpt-3.5-turbo

# 可以组合多个操作
tr config <name> --enable --add-model gpt-4o --secret sk-new
```

### tr assistant — 配置 AI 助手

将 AI 编码助手（Claude Code, Cursor, Aider 等）指向 tokrouter。

```bash
# 子命令
tr assistant list                # 列出支持的助手
tr assistant auto                # 自动检测并配置所有已安装的助手
tr assistant auto --url http://localhost:9000

# 配置指定助手
tr assistant claude-code         # 配置 Claude Code
tr assistant cursor              # 配置 Cursor
tr assistant aider               # 配置 Aider
tr assistant windsurf            # 配置 Windsurf
tr assistant cline               # 配置 Cline
tr assistant continue            # 配置 Continue

# 指定自定义 URL
tr assistant claude-code --url http://myhost:8765
```

### tr start — 启动服务

```bash
tr start                          # 默认 (127.0.0.1:8765)
tr start --host 0.0.0.0           # 监听所有接口
tr start --port 9000              # 自定义端口
tr start --host 0.0.0.0 --port 9000
```

启动后服务在**前台运行**，按 `Ctrl+C` 停止。服务端口会写入 `/tmp/tokrouter.port` 供 `tr stop` 读取。

### tr stop — 停止服务

```bash
tr stop
```

通过 HTTP `POST /shutdown` 优雅关闭后台运行的 tokrouter。前台运行的服务按 `Ctrl+C` 即可。

---

## 配置详解

### 基础配置

```yaml
# config.yaml
server:
  host: "127.0.0.1"
  port: 8765

keys:
  - name: openai-main
    base_urls:
      openai: "https://api.openai.com/v1"
    secret: "${OPENAI_API_KEY}"
    format: openai                    # 单协议（向后兼容）
    enabled: true
    models:
      - name: gpt-4
      - name: gpt-4o

  - name: deepseek-main
    provider: deepseek              # 预置模板，自动获取 BaseURLs、Format、协议列表
    secret: "${DEEPSEEK_API_KEY}"
    enabled: true
    models:
      - name: deepseek-chat
      - name: deepseek-reasoner
```

**说明**：以上配置已足够运行。server、router、stats、log、trace 均有默认值，按需覆盖即可。

### 必需配置 — keys

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| name | string | ✓ | 服务标识 |
| provider | string | - | 预置 Provider 标识（如 openai, deepseek）。设置后 BaseURLs、Format、协议列表自动从预置获取 |
| base_urls | map | - | API 基础 URL 映射（自定义服务必需，预置模式自动填充）。格式: `{openai: "https://...", anthropic: "https://..."}` |
| secret | string | ✓ | API 密钥（支持 `${VAR}` 环境变量） |
| format | string | - | 单协议：openai/anthropic/gemini/cohere（预置模式自动填充） |
| enabled | bool | - | 是否启用（默认 true） |
| models | []ModelConfig | ✓ | 模型列表（至少一个） |

**协议列表说明：**
- 协议列表由 Provider 预置决定，不需在 KeyConfig 中单独配置
- DeepSeek、OpenRouter 等预置已声明 `Protocols: [openai, anthropic]`
- 自定义服务（无 provider）使用 `format` 单协议
- `/v1/chat/completions` 入口使用 OpenAI 协议处理
- `/v1/messages` 入口使用 Anthropic 协议处理

**Model 配置：**

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| name | string | ✓ | 模型名称 |
| alias | string | - | 模型别名（请求模型名映射到实际模型名） |
| priority | int64 | - | 端点优先级（越低越优先，默认 0） |

### 可选配置（均有默认值）

**Server：**

| 字段 | 默认值 | 说明 |
|------|--------|------|
| host | "127.0.0.1" | 监听地址 |
| port | 8765 | 监听端口 |
| tls_cert | "" | TLS 证书路径 |
| tls_key | "" | TLS 密钥路径 |

**Router：**

| 字段 | 默认值 | 说明 |
|------|--------|------|
| retry.max_retries | 2 | 最大重试次数 |
| retry.timeout | "30s" | 请求超时 |
| retry.backoff | "exponential" | 退避策略 |

**Stats：**

| 字段 | 默认值 | 说明 |
|------|--------|------|
| enabled | true | 是否启用用量统计 |
| db_path | "./data/usage.db" | 统计数据库路径 |

**Log：**

| 字段 | 默认值 | 说明 |
|------|--------|------|
| level | "info" | 日志级别：debug/info/warn/error |
| format | "json" | 日志格式：json/text |
| output | "stdout" | 输出位置 |

**Trace：**

| 字段 | 默认值 | 说明 |
|------|--------|------|
| enabled | true | 是否启用链路追踪 |
| header | "x-request-id" | 追踪请求头字段名 |

---

## 使用场景

### 预设快速添加

62+ 内置 Provider 预置，自动填充 BaseURLs 和默认模型：

```bash
tr list presets                # 浏览预置列表
tr add openai --secret sk-xxx
tr add deepseek --secret sk-xxx
tr add openrouter --secret sk-or-xxx
```

### 多协议支持

DeepSeek、OpenRouter 等 Provider 同时支持 OpenAI 和 Anthropic 协议，预置模板已声明协议列表，无需手动配置：

```yaml
keys:
  - name: deepseek
    provider: deepseek          # 预置已含 protocols: [openai, anthropic]
    secret: "${DEEPSEEK_API_KEY}"
    enabled: true
    models:
      - name: deepseek-chat
```

此时：
- `POST /v1/chat/completions` → OpenAI 格式直传 DeepSeek
- `POST /v1/messages` → Anthropic 请求翻译为 OpenAI 格式发往 DeepSeek

### 负载均衡

配置多个相同模型的端点，按优先级自动轮询：

```yaml
keys:
  - name: openai-1
    base_urls:
      openai: "https://api.openai.com/v1"
    secret: "sk-xxx1"
    format: openai
    models:
      - name: gpt-4
        priority: 100       # 优先使用
  - name: openai-2
    base_urls:
      openai: "https://api.openai.com/v1"
    secret: "sk-xxx2"
    format: openai
    models:
      - name: gpt-4
        priority: 200       # 次选
```

### 故障转移

某个端点连续失败 3 次后自动熔断，60 秒后恢复。同时自动切换到健康的备用端点。

### 模型别名

将请求模型名映射到实际模型名：

```yaml
keys:
  - name: openai
    models:
      - name: gpt-4-1106-preview
        alias: gpt-4-turbo    # 请求 gpt-4-turbo → 实际调 gpt-4-1106-preview
```

### 延迟感知路由

端点选择策略：
1. 优先级优先（低优先级优先）
2. 同优先级按 EWMA 延迟（近期延迟权重更高）选择最快端点

### AI 助手集成

将 tokrouter 设置为 AI 编码助手的 API 代理：

```bash
# 自动检测并配置所有已安装的 AI 助手
tr assistant auto

# 或手动配置单个
tr assistant claude-code
tr assistant cursor
tr assistant aider
```

### 热重载

无需重启即可重载配置：

```bash
kill -SIGHUP $(pidof tokrouter)
```

或在运行 `tr start` 的终端中按 `Ctrl+C`，修改配置后重新 `tr start`。

### 成本监控

```bash
tr show usage --today          # 今日用量
tr show usage --week           # 本周用量
tr show usage --chart          # 柱状图
tr show usage --export csv > stats.csv
```

---

## API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/v1/chat/completions` | POST | OpenAI 兼容聊天接口 |
| `/v1/messages` | POST | Anthropic 兼容消息接口 |
| `/status` | GET | 端点运行时状态 |
| `/health` | GET | 健康检查（含依赖状态） |
| `/shutdown` | POST | 优雅关闭服务器 |

### 健康检查

```bash
curl http://localhost:8765/health
curl http://localhost:8765/status
```

### 错误类型

| 类型 | HTTP | 说明 |
|------|------|------|
| invalid_request_error | 400 | 请求格式错误 |
| upstream_error | 503 | 上游服务不可用 |
| internal_error | 500 | 内部错误 |

---

## 故障排查

| 错误 | 原因 | 解决方案 |
|------|------|----------|
| connection refused | 服务未启动 | `tr start` 启动服务 |
| invalid API key | 密钥无效 | `tr config <name> --secret sk-correct` |
| timeout | 上游响应慢 | 增加 config.yaml 中 `retry.timeout` |
| no healthy endpoints | 所有端点不可用 | `tr show health` 检查状态 |
| config invalid: no keys | 未配置服务 | `tr add` 添加服务 |

调试模式：

```yaml
log:
  level: "debug"
```

```bash
tr show health --watch           # 实时监控端点健康
tr show service <name>           # 检查特定服务配置
tr show config                   # 检查完整配置
```
