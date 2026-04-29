# tokrouter

**Your LLM Aggregator**

One config file. All your LLM APIs.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)
[![Build](https://img.shields.io/github/actions/workflow/status/tokflux/tokrouter/release.yml?style=flat-square)](https://github.com/tokflux/tokrouter/actions)
[![Version](https://img.shields.io/badge/Version-v0.7.3-blue?style=flat-square)]()

[中文文档](README_CN.md)

---

## 4 Lines to Production

```yaml
# config.yaml - Simplified with preset
keys:
  - provider: openai
    secret: "${OPENAI_API_KEY}"

# Start
tr start
# Your gateway is ready at http://127.0.0.1:8765
```

**Provider Presets** — Just specify `provider` and `secret`, everything else auto-filled:
- International: openai, anthropic, google, mistral, cohere, groq, deepseek
- Chinese: zhipu, qwen, tencent, baidu, moonshot, minimax, siliconflow, yi...
- Platforms: together, replicate, openrouter

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
tr --version
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
| `v0.6.0` | Specific version |
| `main` | Latest build from main branch |

---

## Quick Start

```bash
# Install
git clone https://github.com/tokflux/tokrouter.git
cd tokrouter
go build ./cmd/tokrouter

# Simplified config with presets
cat > config.yaml << 'EOF'
keys:
  - provider: openai
    secret: "${OPENAI_API_KEY}"
  - provider: anthropic
    secret: "${ANTHROPIC_API_KEY}"
  - provider: deepseek
    secret: "${DEEPSEEK_API_KEY}"
EOF

tr start
```

**Or use interactive init:**

```bash
tr add       # Interactive: add services (presets or custom)
tr start
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
│  Router             Domain service          │
│  ServiceEndpoint    Network CB aggregate    │
│  Route              Model CB aggregate      │
│  RouteTable         Pre-computed snapshot   │
│  RouteRepository    CB state persistence    │
│  message/           Request/Response IR     │
│  errors/            Error classification    │
│  translate/         Protocol conversion     │
│                                             │
└─────────────────────────────────────────────┘

1 binary. 1 config file. 0 dependencies.
```

---

## How It Works

```go
// router/router.go - The aggregation service
type routerCtx struct {
    oaTables   map[fluxcore.Model]*fluxcore.RouteTable
    anthTables map[fluxcore.Model]*fluxcore.RouteTable
    retryMax   int
}

type router struct {
    mu        sync.RWMutex
    ctx       *routerCtx           // atomic.Pointer swapped on reload
    oaRouter  *fluxcore.Router
    anthRouter *fluxcore.Router
    svcEPs    map[string]*fluxcore.ServiceEndpoint
    routeRepo *fluxcore.RouteRepository  // CB state survives reload
    usageSvc  *usage.Service
}

func (r *router) ForwardOpenAI(ctx context.Context, body []byte, model string) ([]byte, *message.Usage, error) {
    r.mu.RLock()
    table := r.ctx.oaTables[fluxcore.Model(model)]
    retryMax := r.ctx.retryMax
    r.mu.RUnlock()

    if table == nil {
        return nil, nil, errors.New("no route for model")
    }

    route, resp, usage, err := r.oaRouter.Execute(ctx, table, body, retryMax)
    if err != nil {
        return nil, nil, err
    }
    r.usageSvc.RecordWithModelAndProvider(usage, model, route.SvcEP().Service().Name, false)
    return resp, usage, nil
}
```

All routing logic (route selection, retry, failover, health feedback) handled by fluxcore's `Router.Execute`.

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

Two-layer circuit breaker in fluxcore:

```
ServiceEndpoint layer (network):
  DNS / Connection refused / Timeout → Immediate trip (threshold=1)
  Recovery: 120s

Route layer (model):
  429 Rate Limit → Trip (threshold=3 cumulative)
  500 Server Error → Trip (threshold=3 cumulative)
  Recovery: 60s

tokrouter auto-switches to next healthy route.
CB state survives config reload (SIGHUP) via RouteRepository.
```

---

## Model-Level Routing

Requests are routed to endpoints matching the requested model:

```yaml
keys:
  - name: openai-main
    models:
      - name: gpt-4
        priority: 100
      - name: gpt-3.5-turbo
        priority: 10
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
        priority: 50
```

---

## Hot Reload

Reload config without restart. Circuit breaker state is preserved across reloads via `RouteRepository.FindOrCreate()`.

```bash
kill -SIGHUP $(pidof tokrouter)
```

---

## Latency-Aware Routing

Endpoints are selected by:
1. **Priority** (lower is better) - initial selection
2. **EWMA latency** (recent latency weighted higher) - runtime adjustment

This avoids slow endpoints automatically.

---

## Cost Tracking

```bash
tr show health

Key            Format    Healthy    Models
openai-main    openai    OK         2/2
anthropic-main anthropic OK         1/1

tr show usage --month
Key            Input      Output     Requests   Avg Latency  Success
openai-main    152340     45678     1234       245ms        98.5%
anthropic-main 23456      12345     567        189ms        99.2%
```

---

## CLI Commands

```bash
# Quick start
tr add openai --secret sk-xxx        # Add a service with preset
tr start                             # Start server at 127.0.0.1:8765
tr start --port 8080                 # Custom port

# Service management
tr add                               # Interactive: select preset or custom
tr add deepseek --secret sk-xxx      # Add with preset, auto-fill config
tr add --name my --base-url ... --format openai --secret sk-xxx --model gpt-4
tr remove <name>                     # Remove a service
tr config <name> --enable            # Enable/disable a service
tr config <name> --secret sk-new     # Update API key
tr config <name> --add-model gpt-4   # Add model to service
tr config <name> --remove-model old  # Remove model from service

# Viewing
tr list services                     # List all configured services (default)
tr list models                       # List all available models
tr list presets                      # List provider presets (26 built-in)
tr list assistants                   # List supported AI tools
tr show service <name>               # Service details
tr show preset <name>                # Preset details
tr show config                       # Current configuration
tr show health                       # Endpoint health status
tr show health --watch               # Real-time refresh (2s)
tr show usage --month                # Monthly usage statistics
tr show usage --chart                # Token distribution chart
tr show usage --export csv           # Export as CSV

# AI assistant integration
tr assistant list                    # List supported AI tools
tr assistant auto                    # Auto-detect and configure all
tr assistant cursor                  # Configure specific tool

# Server lifecycle
tr start [--host HOST] [--port PORT] # Start server
tr stop                              # Stop server gracefully

# Shell completion
tr completion bash|zsh|fish          # Generate completion script
```

---

## API Endpoints

| Endpoint | Format | Description |
|----------|--------|-------------|
| `POST /v1/chat/completions` | OpenAI | OpenAI compatible, streaming supported |
| `POST /v1/messages` | Anthropic | Anthropic compatible, streaming supported |
| `GET /status` | JSON | Key status |
| `GET /health` | JSON | Health check with dependency status |
| `POST /shutdown` | - | Graceful server shutdown |
| `GET /openapi.yaml` | YAML | OpenAPI 3.0 specification |
| `GET /docs` | HTML | Swagger UI documentation |

---

## AI Tool Integration

Use `tr assistant` to auto-configure your AI tools:

```bash
tr assistant list          # See supported tools
tr assistant auto          # Auto-detect and configure all
tr assistant cursor        # Configure a specific tool
```

Manual configuration reference:

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
│   ├── record.go     # Usage record entity
│   ├── query.go      # QueryFilter, StatRow
│   ├── service.go    # Usage service
│   ├── sqlite.go     # SQLite store
│   ├── errors.go     # Usage errors
│   ├── doc.go        # Package documentation
│   └── *_test.go     # Tests (sqlite, query, service)
│
├── router/           # Aggregation service
│   ├── router.go     # Router service
│   ├── doc.go        # Package documentation
│   └── *_test.go     # Tests (unit + concurrent)
│
├── server/           # HTTP server
│   ├── handler.go    # API handlers
│   ├── server.go     # HTTP server
│   ├── log.go        # Request logging (with redaction)
│   ├── errors.go     # HTTP error response
│   ├── doc.go        # Package documentation
│   └── *_test.go     # Tests (handler, log, errors)
│
├── config/           # Configuration
│   ├── config.go     # YAML config loader
│   ├── doc.go        # Package documentation
│   └── *_test.go     # Tests
│
├── cli/              # Command line
│   ├── root.go       # CLI entry
│   ├── cmd_init.go   # init command
│   ├── cmd_serve.go  # serve command
│   ├── cmd_status.go # status command
│   ├── cmd_keys.go   # keys command
│   ├── cmd_summary.go # summary command
│   ├── cmd_config.go # config command
│   ├── cmd_models.go # models command
│   ├── doc.go        # Package documentation
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
tr add --name my-key --format openai --secret $MY_KEY --base-url https://api.example.com/v1
```

**Q: How do I test if my key works?**
```bash
tr show health          # Check endpoint health
```

**Q: How do I use Claude Code with tokrouter?**
```bash
tr assistant claude-code
# Or manually: export ANTHROPIC_BASE_URL=http://127.0.0.1:8765
```

**Q: How do I use aider with tokrouter?**
```bash
tr assistant aider
# Or manually: export OPENAI_API_BASE=http://127.0.0.1:8765/v1
```

**Q: Why port 8765?**

Port 7890 conflicts with Clash proxy. 8765 is uncommon and safe.

**Q: Does tokrouter support streaming?**

Yes, streaming is fully supported for both OpenAI and Anthropic formats.

**Q: How does automatic fallback work?**

Two-layer protection: network errors trip the service endpoint immediately (threshold=1, recovery=120s). Model errors (429/5xx) trip the route after 3 failures (recovery=60s). The router auto-switches to the next healthy route.

---

## Get Started

```bash
git clone https://github.com/tokflux/tokrouter.git
cd tokrouter
go build ./cmd/tokrouter

tr add
tr start
```

**Next steps:**
1. Star the repo
2. Run `tr add` to configure
3. Point your AI tools to `http://127.0.0.1:8765`

---

## Related Projects

| Project | Description |
|---------|-------------|
| **fluxcore** | DDD-based LLM routing engine (ServiceEndpoint, Route, RouteTable, Router) |

---

## License

MIT. Free forever.

---

**tokrouter - Your LLM Aggregator. One config, one binary, full control.**