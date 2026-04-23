# tokrouter

**Your LLM Aggregator**

One config file. All your LLM APIs.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)
[![Build](https://img.shields.io/github/actions/workflow/status/tokflux/tokrouter/ci.yml?style=flat-square)](https://github.com/tokflux/tokrouter/actions)

---

## 4 Lines to Production

```yaml
# config.yaml
keys:
  - name: openai-main
    base_url: "https://api.openai.com/v1"
    format: openai
    secret: "${OPENAI_API_KEY}"
    models:
      - name: "gpt-4"
        pricing: {input: 0.03, output: 0.06}

# Start
tokrouter serve
# Your gateway is ready at http://127.0.0.1:8765
```

---

## Challenges

Managing multiple LLM providers presents these challenges:

| Challenge | Impact |
|------|---------|
| **Key Scattered** | Keys scattered across tools, no centralized control |
| **Cost Invisible** | No visibility into actual spend per provider |
| **Manual Switching** | Change endpoints manually when quota exhausted |
| **Format Lock-in** | OpenAI SDK can't call Anthropic, Anthropic SDK can't call OpenAI |
| **No Fallback** | One provider fails, your entire workflow stops |

---

## The Solution

tokrouter gives you unified control:

| tokrouter | Manual Management |
|-----------|-------------------|
| Single YAML config | Keys scattered everywhere |
| Automatic cost tracking | Blind to actual spend |
| Auto fallback chain | Manual provider switching |
| Anthropic ↔ OpenAI transparent | SDK lock-in |
| Circuit breaker + retry | Single point of failure |

---

## Installation

### Pre-built Binaries (Recommended)

Download from [GitHub Releases](https://github.com/tokflux/tokrouter/releases):

**Linux (amd64)**
```bash
curl -sL https://github.com/tokflux/tokrouter/releases/latest/download/tokrouter-linux-amd64 -o tokrouter
chmod +x tokrouter
sudo mv tokrouter /usr/local/bin/
```

**Linux (arm64 / Raspberry Pi)**
```bash
curl -sL https://github.com/tokflux/tokrouter/releases/latest/download/tokrouter-linux-arm64 -o tokrouter
chmod +x tokrouter
sudo mv tokrouter /usr/local/bin/
```

**macOS (Intel)**
```bash
curl -sL https://github.com/tokflux/tokrouter/releases/latest/download/tokrouter-darwin-amd64 -o tokrouter
chmod +x tokrouter
sudo mv tokrouter /usr/local/bin/
```

**macOS (Apple Silicon)**
```bash
curl -sL https://github.com/tokflux/tokrouter/releases/latest/download/tokrouter-darwin-arm64 -o tokrouter
chmod +x tokrouter
sudo mv tokrouter /usr/local/bin/
```

**Windows (PowerShell)**
```powershell
Invoke-WebRequest -Uri "https://github.com/tokflux/tokrouter/releases/latest/download/tokrouter-windows-amd64.exe" -OutFile "tokrouter.exe"
```

Verify:
```bash
tokrouter version
```

### Build from Source

```bash
git clone https://github.com/tokflux/tokrouter.git
cd tokrouter
go build ./cmd/tokrouter
```

---

## Docker

### Quick Start

```bash
mkdir tokrouter-deploy && cd tokrouter-deploy

curl -O https://raw.githubusercontent.com/tokflux/tokrouter/main/docker-compose.yaml
curl -O https://raw.githubusercontent.com/tokflux/tokrouter/main/config.example.yaml
mv config.example.yaml config.yaml

vim config.yaml  # Set your API keys

export OPENAI_API_KEY=sk-xxx
export ANTHROPIC_API_KEY=sk-xxx
docker compose up -d

curl http://localhost:8765/health
```

### Manual Docker Run

```bash
docker run -d \
  --name tokrouter \
  -p 8765:8765 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v tokrouter_data:/app/data \
  -e OPENAI_API_KEY=sk-xxx \
  ghcr.io/tokflux/tokrouter:latest
```

### Available Tags

| Tag | Description |
|-----|-------------|
| `latest` | Most recent release |
| `v0.1.0` | Specific version |
| `main` | Latest build from main branch |

---

## Quick Start

```bash
git clone https://github.com/tokflux/tokrouter.git
cd tokrouter
go build ./cmd/tokrouter

cat > config.yaml << 'EOF'
server:
  host: "127.0.0.1"
  port: 8765

keys:
  - name: openai-main
    enabled: true
    format: openai
    secret: "${OPENAI_API_KEY}"
    base_url: "https://api.openai.com/v1"
    models:
      - name: "gpt-4"
        pricing:
          input: 0.03
          output: 0.06

  - name: anthropic-main
    enabled: true
    format: anthropic
    secret: "${ANTHROPIC_API_KEY}"
    base_url: "https://api.anthropic.com/v1"
    models:
      - name: "claude-3-opus"
        pricing:
          input: 0.015
          output: 0.075

router:
  retry:
    max_retries: 2
    timeout: "30s"

stats:
  enabled: true
  db_path: "./data/usage.db"
EOF

tokrouter serve
```

---

## Who Uses This

| User | Use Case |
|------|----------|
| **Indie Developers** | Single config for all AI tools (Claude Code, aider, Cursor) |
| **AI Researchers** | Experiment across providers without code changes |
| **Power Users** | Maximize quota, minimize cost, automatic fallback |
| **Self-hosters** | No cloud dependency, full control over data |

---

## Architecture

```
┌─────────────────────────────────────────────┐
│                 tokrouter                    │
│           "Your LLM Aggregator"              │
├─────────────────────────────────────────────┤
│                                             │
│  usage/             Cost tracking (SQLite)  │
│  router/            Application service     │
│  server/            HTTP server + handlers  │
│  config/            YAML loading            │
│  cli/               Command line entry      │
│                                             │
├─────────────────────────────────────────────┤
│                                             │
│              fluxcore (domain layer)        │
│                                             │
│  routing/           Endpoint + Pool         │
│  call/              HTTP + Retry + Fallback │
│  message/           Request/Response IR     │
│  translate/         Anthropic ↔ OpenAI      │
│                                             │
└─────────────────────────────────────────────┘

1 binary. 1 config file. 0 dependencies.
```

---

## How It Works

```go
// router/router.go - The aggregation service
func NewService(endpoints []*routing.Endpoint, usageSvc *usage.Service, retryMax int) *Service {
    pool := routing.NewEndpointPool(endpoints, retryMax)
    return &Service{pool: pool, usageSvc: usageSvc}
}

func (s *Service) Forward(ctx context.Context, rawReq []byte, clientFormat string) {
    resp, usage, _ := call.Request(ctx, s.pool, rawReq, clientFormat)
    s.usageSvc.Record(usage)  // Track cost
    return resp
}
```

All routing handled by fluxcore domain layer.

---

## Protocol Conversion

```bash
# Use Anthropic SDK with OpenAI backend
export ANTHROPIC_BASE_URL=http://127.0.0.1:8765

# Use OpenAI SDK with Anthropic backend
export OPENAI_API_BASE=http://127.0.0.1:8765/v1

# Format conversion happens automatically
# Your SDK doesn't know the difference
```

---

## Circuit Breaker

```
Model state machine (fluxcore):

Healthy → Fail(1) → Fail(2) → Fail(3) → Unhealthy
                                         ↓
                                   60s auto recovery

tokrouter auto-switches to next healthy model.
```

---

## Model-Level Routing

Requests are routed to endpoints matching the requested model:

```yaml
keys:
  - name: openai-main
    models:
      - name: gpt-4
        pricing: {input: 0.03, output: 0.06}
      - name: gpt-3.5-turbo
        pricing: {input: 0.001, output: 0.002}
```

Request for `gpt-4` → only routes to gpt-4 endpoints (not gpt-3.5-turbo).

### Model Alias

Map request model names to actual model names:

```yaml
keys:
  - name: openai-main
    models:
      - name: gpt-4-turbo
        alias: gpt-4-1106-preview  # Request "gpt-4-turbo" → uses "gpt-4-1106-preview"
        pricing: {input: 0.01, output: 0.03}
```

---

## Hot Reload

Reload config without restart:

```bash
kill -SIGHUP $(pidof tokrouter)
```

---

## Latency-Aware Routing

Endpoints are selected by:
1. **Price** (lower is better)
2. **EWMA latency** (recent latency weighted higher)

This avoids slow endpoints automatically.

---

## Cost Tracking

```bash
tokrouter status

Key            Format    Healthy    Models
openai-main    openai    OK         2/2
anthropic-main anthropic OK         1/1

tokrouter summary --month
Key            Input      Output     Cost       Requests
openai-main    152340     45678      $4.56      1234
anthropic-main 23456      12345      $1.23      567
```

---

## CLI Commands

```bash
tokrouter init                       # Interactive configuration
tokrouter serve                      # Start server (127.0.0.1:8765)
tokrouter serve --port 9000          # Custom port
tokrouter status                     # Show key status
tokrouter status --watch             # Real-time refresh
tokrouter keys                       # List all keys
tokrouter keys add                   # Add a new key
tokrouter keys remove <name>         # Remove a key
tokrouter keys enable <name>         # Enable a key
tokrouter keys disable <name>        # Disable a key
tokrouter keys ping <name>           # Test connectivity
tokrouter summary --month            # Monthly statistics
tokrouter summary --today            # Today's usage
tokrouter summary --chart            # ASCII bar chart
tokrouter summary --export csv       # Export as CSV
tokrouter summary --export json      # Export as JSON
tokrouter config                     # Show configuration
```

---

## API Endpoints

| Endpoint | Format | Description |
|----------|--------|-------------|
| `POST /v1/chat/completions` | OpenAI | OpenAI compatible, streaming supported |
| `POST /v1/messages` | Anthropic | Anthropic compatible, streaming supported |
| `GET /status` | JSON | Key status |
| `GET /health` | JSON | Health check with dependency status |

---

## AI Tool Integration

**Claude Code** (Anthropic format):
```json
// ~/.claude/settings.json
{
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:8765"
  }
}
```

**aider** (OpenAI format):
```yaml
# ~/.aider.conf.yml
openai-api-base: http://127.0.0.1:8765/v1
model: gpt-4
```

**Cursor / VS Code Copilot**:
```json
// Settings
"openai.apiBase": "http://127.0.0.1:8765/v1"
```

---

## Protocol Support

| Format | Endpoint | Conversion |
|--------|----------|------------|
| **OpenAI** | `/v1/chat/completions` | Native |
| **Anthropic** | `/v1/messages` | Full bidirectional |
| **Gemini** | Provider config | Auto via fluxcore |
| **Cohere** | Provider config | Auto via fluxcore |

**OpenAI-Compatible Providers**:

| Provider | Base URL |
|----------|----------|
| Zhipu GLM-4 | `https://open.bigmodel.cn/api/paas/v4` |
| Alibaba Qwen | DashScope API |
| DeepSeek | `https://api.deepseek.com` |
| Mistral | `https://api.mistral.ai` |
| Groq | `https://api.groq.com` |

---

## Directory Structure

```
tokrouter/
├── usage/            # Cost tracking (SQLite)
│   ├── record.go     # UsageRecord DTO
│   ├── query.go      # QueryFilter, StatRow
│   ├── service.go    # StatsService
│   ├── sqlite.go     # SQLite storage
│   └── errors.go
│
├── router/           # Aggregation service
│   ├── router.go     # RouterService
│   └── router_test.go
│
├── server/           # HTTP server
│   ├── handler.go    # API handlers
│   ├── handler_test.go
│   ├── server.go     # HTTP server
│   ├── log.go        # Request logging
│   └── errors.go     # HTTP error response
│
├── config/           # Configuration
│   ├── config.go     # YAML config loader
│   └── config_test.go
│
├── cli/              # Command line
│   ├── root.go       # CLI entry
│   ├── cmd_init.go   # init command
│   ├── cmd_serve.go  # serve command
│   ├── cmd_status.go # status command
│   ├── cmd_keys.go   # keys command
│   ├── cmd_summary.go # summary command
│   ├── cmd_config.go # config command
│   └── helpers.go    # helper functions
│
├── cmd/tokrouter/    # Entry point
│   └── main.go
│
└── config.example.yaml
```

---

## FAQ

**Q: How do I add a new API key?**
```bash
tokrouter keys add --name my-key --format openai --secret $MY_KEY --base-url https://api.example.com/v1
```

**Q: How do I test if my key works?**
```bash
tokrouter keys ping openai-main
```

**Q: Pricing unit - what does `input: 0.03` mean?**

It means **$0.03 per 1,000 tokens**. Example: 100K input tokens = $0.03 × 100 = $3.00.

**Q: How do I use Claude Code with tokrouter?**
```bash
export ANTHROPIC_BASE_URL=http://127.0.0.1:8765
claude
```

**Q: How do I use aider with tokrouter?**
```bash
export OPENAI_API_BASE=http://127.0.0.1:8765/v1
aider --model gpt-4
```

**Q: Why port 8765?**

Port 7890 conflicts with Clash proxy. 8765 is uncommon and safe.

**Q: Does tokrouter support streaming?**

Yes, streaming is fully supported for both OpenAI and Anthropic formats.

**Q: How does automatic fallback work?**

Model fails 3 times → marked unhealthy → auto-switch to next healthy model. 60 seconds later, retry unhealthy model.

---

## Get Started

```bash
git clone https://github.com/tokflux/tokrouter.git
cd tokrouter
go build ./cmd/tokrouter

tokrouter init
tokrouter serve
```

**Next steps:**
1. Star the repo
2. Run `tokrouter init` to configure
3. Point your AI tools to `http://127.0.0.1:8765`

---

## Related Projects

| Project | Description |
|---------|-------------|
| **fluxcore** | LLM API Router Library (core routing engine) |

---

## License

MIT. Free forever.

---

**tokrouter - Your LLM Aggregator. One config, one binary, full control.**

---

# 中文说明

## tokrouter

**你的 LLM 聚合器**

一个配置文件，聚合所有 LLM API。

---

## 4 行配置上线

```yaml
# config.yaml
keys:
  - name: openai-main
    base_url: "https://api.openai.com/v1"
    format: openai
    secret: "${OPENAI_API_KEY}"
    models:
      - name: "gpt-4"
        pricing: {input: 0.03, output: 0.06}

# 启动
tokrouter serve
# 网关就绪：http://127.0.0.1:8765
```

---

## 问题背景

多 LLM 提供商管理面临的挑战：

| 挑战 | 影响 |
|------|------|
| **密钥分散** | 密钥散落在各工具，无统一管理 |
| **成本不可见** | 不知道每个提供商花了多少钱 |
| **手动切换** | 配额用完需手动换提供商 |
| **格式锁定** | OpenAI SDK 无法调用 Anthropic，反之亦然 |
| **无降级** | 一个提供商失败，整个流程中断 |

---

## 解决方案

tokrouter 给你统一掌控：

| tokrouter | 手动管理 |
|-----------|----------|
| 单一 YAML 配置 | 密钥散落各处 |
| 自动成本追踪 | 看不到实际花费 |
| 自动故障转移 | 手动切换提供商 |
| Anthropic ↔ OpenAI 透明转换 | SDK 锁定 |
| 熔断器 + 重试 | 单点故障 |

---

## 安装

### 下载预编译二进制（推荐）

从 [GitHub Releases](https://github.com/tokflux/tokrouter/releases) 下载：

**Linux**
```bash
curl -sL https://github.com/tokflux/tokrouter/releases/latest/download/tokrouter-linux-amd64 -o tokrouter
chmod +x tokrouter
sudo mv tokrouter /usr/local/bin/
```

**macOS (M1/M2)**
```bash
curl -sL https://github.com/tokflux/tokrouter/releases/latest/download/tokrouter-darwin-arm64 -o tokrouter
chmod +x tokrouter
sudo mv tokrouter /usr/local/bin/
```

**Windows**
```powershell
Invoke-WebRequest -Uri "https://github.com/tokflux/tokrouter/releases/latest/download/tokrouter-windows-amd64.exe" -OutFile "tokrouter.exe"
```

验证安装：
```bash
tokrouter version
```

### 从源码构建

```bash
git clone https://github.com/tokflux/tokrouter.git
cd tokrouter
go build ./cmd/tokrouter
```

---

## Docker 部署

### 快速开始

```bash
mkdir tokrouter-deploy && cd tokrouter-deploy

curl -O https://raw.githubusercontent.com/tokflux/tokrouter/main/docker-compose.yaml
curl -O https://raw.githubusercontent.com/tokflux/tokrouter/main/config.example.yaml
mv config.example.yaml config.yaml

vim config.yaml  # 设置 API Keys

export OPENAI_API_KEY=sk-xxx
export ANTHROPIC_API_KEY=sk-xxx
docker compose up -d

curl http://localhost:8765/health
```

### 手动 Docker 运行

```bash
docker run -d \
  --name tokrouter \
  -p 8765:8765 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v tokrouter_data:/app/data \
  -e OPENAI_API_KEY=sk-xxx \
  ghcr.io/tokflux/tokrouter:latest
```

---

## 快速开始

```bash
git clone https://github.com/tokflux/tokrouter.git
cd tokrouter
go build ./cmd/tokrouter

cat > config.yaml << 'EOF'
server:
  host: "127.0.0.1"
  port: 8765

keys:
  - name: openai-main
    enabled: true
    format: openai
    secret: "${OPENAI_API_KEY}"
    base_url: "https://api.openai.com/v1"
    models:
      - name: "gpt-4"
        pricing:
          input: 0.03
          output: 0.06

  - name: anthropic-main
    enabled: true
    format: anthropic
    secret: "${ANTHROPIC_API_KEY}"
    base_url: "https://api.anthropic.com/v1"
    models:
      - name: "claude-3-opus"
        pricing:
          input: 0.015
          output: 0.075

router:
  retry:
    max_retries: 2
    timeout: "30s"

stats:
  enabled: true
  db_path: "./data/usage.db"
EOF

tokrouter serve
```

---

## 适用人群

| 用户 | 用途 |
|------|------|
| **独立开发者** | 所有 AI 工具共用一套配置（Claude Code、aider、Cursor） |
| **AI 研究者** | 跨提供商实验，无需改代码 |
| **重度用户** | 最大化配额、最小化成本、自动降级 |
| **自托管者** | 无云依赖，数据完全掌控 |

---

## 架构

```
┌─────────────────────────────────────────────┐
│                 tokrouter                    │
│           "你的 LLM 聚合器"                  │
├─────────────────────────────────────────────┤
│                                             │
│  usage/             成本追踪 (SQLite)       │
│  router/            应用服务                │
│  server/            HTTP 服务器 + 处理器    │
│  config/            YAML 加载               │
│  cli/               命令行入口              │
│                                             │
├─────────────────────────────────────────────┤
│                                             │
│              fluxcore (领域层)              │
│                                             │
│  routing/           端点 + 池               │
│  call/              HTTP + 重试 + 降级      │
│  message/           请求/响应中间表示       │
│  translate/         Anthropic ↔ OpenAI      │
│                                             │
└─────────────────────────────────────────────┘

1 个二进制。1 个配置文件。0 依赖。
```

---

## 工作原理

```go
// router/router.go - 聚合服务
func NewService(endpoints []*routing.Endpoint, usageSvc *usage.Service, retryMax int) *Service {
    pool := routing.NewEndpointPool(endpoints, retryMax)
    return &Service{pool: pool, usageSvc: usageSvc}
}

func (s *Service) Forward(ctx context.Context, rawReq []byte, clientFormat string) {
    resp, usage, _ := call.Request(ctx, s.pool, rawReq, clientFormat)
    s.usageSvc.Record(usage)  // 记录成本
    return resp
}
```

所有路由逻辑由 fluxcore 领域层处理。

---

## 协议转换

```bash
# 用 Anthropic SDK 调用 OpenAI 后端
export ANTHROPIC_BASE_URL=http://127.0.0.1:8765

# 用 OpenAI SDK 调用 Anthropic 后端
export OPENAI_API_BASE=http://127.0.0.1:8765/v1

# 格式自动转换
# 你的 SDK 无感知
```

---

## 熔断器

```
模型状态机 (fluxcore):

健康 → 失败(1) → 失败(2) → 失败(3) → 不健康
                                         ↓
                                   60秒自动恢复

tokrouter 自动切换到下一个健康模型。
```

---

## 模型级路由

请求只路由到匹配模型的端点：

```yaml
keys:
  - name: openai-main
    models:
      - name: gpt-4
        pricing: {input: 0.03, output: 0.06}
      - name: gpt-3.5-turbo
        pricing: {input: 0.001, output: 0.002}
```

请求 `gpt-4` → 只路由到 gpt-4 端点（不会路由到 gpt-3.5-turbo）。

### 模型别名

将请求模型名映射到实际模型名：

```yaml
keys:
  - name: openai-main
    models:
      - name: gpt-4-turbo
        alias: gpt-4-1106-preview  # 请求 "gpt-4-turbo" → 实际用 "gpt-4-1106-preview"
        pricing: {input: 0.01, output: 0.03}
```

---

## 热重载

无需重启即可重载配置：

```bash
kill -SIGHUP $(pidof tokrouter)
```

---

## 延迟感知路由

端点选择策略：
1. **价格**（低优先）
2. **EWMA 延迟**（近期延迟权重更高）

自动避开慢端点。

---

## 成本追踪

```bash
tokrouter status

Key            Format    Healthy    Models
openai-main    openai    OK         2/2
anthropic-main anthropic OK         1/1

tokrouter summary --month
Key            Input      Output     Cost       Requests
openai-main    152340     45678      $4.56      1234
anthropic-main 23456      12345      $1.23      567
```

---

## CLI 命令

```bash
tokrouter init                       # 交互式配置
tokrouter serve                      # 启动服务器 (127.0.0.1:8765)
tokrouter serve --port 9000          # 自定义端口
tokrouter status                     # 查看密钥状态
tokrouter status --watch             # 实时刷新
tokrouter keys                       # 列出所有密钥
tokrouter keys add                   # 添加密钥
tokrouter keys remove <name>         # 删除密钥
tokrouter keys enable <name>         # 启用密钥
tokrouter keys disable <name>        # 禁用密钥
tokrouter keys ping <name>           # 测试连通性
tokrouter summary --month            # 月度统计
tokrouter summary --today            # 今日统计
tokrouter summary --chart            # ASCII 图表
tokrouter summary --export csv       # 导出 CSV
tokrouter summary --export json      # 导出 JSON
tokrouter config                     # 显示配置
```

---

## API 端点

| 端点 | 格式 | 说明 |
|------|------|------|
| `POST /v1/chat/completions` | OpenAI | OpenAI 兼容，支持流式 |
| `POST /v1/messages` | Anthropic | Anthropic 兼容，支持流式 |
| `GET /status` | JSON | 密钥状态 |
| `GET /health` | JSON | 健康检查（含依赖状态） |

---

## AI 工具集成

**Claude Code**（Anthropic 格式）：
```json
// ~/.claude/settings.json
{
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:8765"
  }
}
```

**aider**（OpenAI 格式）：
```yaml
# ~/.aider.conf.yml
openai-api-base: http://127.0.0.1:8765/v1
model: gpt-4
```

**Cursor / VS Code Copilot**：
```json
// 设置
"openai.apiBase": "http://127.0.0.1:8765/v1"
```

---

## 协议支持

| 格式 | 端点 | 转换 |
|------|------|------|
| **OpenAI** | `/v1/chat/completions` | 原生 |
| **Anthropic** | `/v1/messages` | 双向转换 |
| **Gemini** | 提供商配置 | fluxcore 自动处理 |
| **Cohere** | 提供商配置 | fluxcore 自动处理 |

**OpenAI 兼容提供商**：

| 提供商 | Base URL |
|------|----------|
| 智谱 GLM-4 | `https://open.bigmodel.cn/api/paas/v4` |
| 阿里通义 | DashScope API |
| DeepSeek | `https://api.deepseek.com` |
| Mistral | `https://api.mistral.ai` |
| Groq | `https://api.groq.com` |

---

## 目录结构

```
tokrouter/
├── usage/            # 成本追踪 (SQLite)
│   ├── record.go     # UsageRecord DTO
│   ├── query.go      # QueryFilter, StatRow
│   ├── service.go    # StatsService
│   ├── sqlite.go     # SQLite 存储
│   └── errors.go
│
├── router/           # 聚合服务
│   ├── router.go     # RouterService
│   └ router_test.go
│
├── server/           # HTTP 服务器
│   ├── handler.go    # API 处理器
│   ├── handler_test.go
│   ├── server.go     # HTTP 服务器
│   ├── log.go        # 请求日志
│   └ errors.go       # HTTP 错误响应
│
├── config/           # 配置
│   ├── config.go     # YAML 配置加载器
│   └ config_test.go
│
├── cli/              # 命令行
│   ├── root.go       # CLI 入口
│   ├── cmd_init.go   # init 命令
│   ├── cmd_serve.go  # serve 命令
│   ├── cmd_status.go # status 命令
│   ├── cmd_keys.go   # keys 命令
│   ├── cmd_summary.go # summary 命令
│   ├── cmd_config.go # config 命令
│   └── helpers.go    # 辅助函数
│
├── cmd/tokrouter/    # 入口点
│   └ main.go
│
└── config.example.yaml
```

---

## 常见问题

**Q: 如何添加新的 API 密钥？**
```bash
tokrouter keys add --name my-key --format openai --secret $MY_KEY --base-url https://api.example.com/v1
```

**Q: 如何测试密钥是否可用？**
```bash
tokrouter keys ping openai-main
```

**Q: pricing 单位是什么？**

`input: 0.03` 表示**每 1000 tokens $0.03**。例如：100K 输入 tokens = $0.03 × 100 = $3.00。

**Q: 如何用 Claude Code 连接 tokrouter？**
```bash
export ANTHROPIC_BASE_URL=http://127.0.0.1:8765
claude
```

**Q: 如何用 aider 连接 tokrouter？**
```bash
export OPENAI_API_BASE=http://127.0.0.1:8765/v1
aider --model gpt-4
```

**Q: 为什么端口是 8765？**

端口 7890 与 Clash 代理冲突。8765 不常用，更安全。

**Q: 支持流式响应吗？**

支持，OpenAI 和 Anthropic 格式都完全支持流式。

**Q: 自动降级如何工作？**

模型失败 3 次 → 标记不健康 → 自动切换下一个健康模型。60 秒后重试不健康模型。

---

## 快速上手

```bash
git clone https://github.com/tokflux/tokrouter.git
cd tokrouter
go build ./cmd/tokrouter

tokrouter init
tokrouter serve
```

**下一步：**
1. Star 本仓库
2. 运行 `tokrouter init` 配置
3. 将 AI 工具指向 `http://127.0.0.1:8765`

---

## 相关项目

| 项目 | 说明 |
|------|------|
| **fluxcore** | LLM API 路由库（核心路由引擎） |

---

## 许可证

MIT。永久免费。

---

**tokrouter - 你的 LLM 聚合器。一个配置，一个二进制，完全掌控。**
