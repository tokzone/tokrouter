# tokrouter

**Your LLM Aggregator**

One config file. All your LLM APIs.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)
[![Build](https://img.shields.io/github/actions/workflow/status/tokflux/tokrouter/release.yml?style=flat-square)](https://github.com/tokflux/tokrouter/actions)
[![Version](https://img.shields.io/badge/Version-v0.7.0-blue?style=flat-square)]()

[中文文档](README_CN.md)

---

## 4 Lines to Production

```yaml
# config.yaml - Simplified with preset
keys:
  - provider: openai
    secret: "${OPENAI_API_KEY}"

# Start
tokrouter serve
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

tokrouter serve
```

**Or use interactive init:**

```bash
tokrouter init  # Interactive configuration wizard
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
│  flux/              Client + UserEndpoint   │
│  endpoint/          Global registry         │
│  provider/          Provider abstraction    │
│  message/           Request/Response IR     │
│  errors/            Error classification    │
│  translate/         Anthropic ↔ OpenAI      │
│                                             │
└─────────────────────────────────────────────┘

1 binary. 1 config file. 0 dependencies.
```

---

## How It Works

```go
// router/router.go - The aggregation service
type Service struct {
    mu            sync.RWMutex
    state         *serviceState  // Atomic state container (swapped on reload)
    usageSvc      *usage.Service
    healthLogger  *slog.Logger
}

type serviceState struct {
    clients   map[string]*flux.Client  // Model -> flux.Client
    aliasMap  map[string]string         // Model alias mapping
    cfg       *config.Config            // Current configuration
    retryMax  int                       // Retry configuration
}

func NewService(userEndpoints []*flux.UserEndpoint, usageSvc *usage.Service, retryMax int) *Service {
    // Build clients - each model has its own flux.Client
    clients := buildClients(userEndpoints, retryMax)
    return &Service{
        state:        &serviceState{clients: clients, aliasMap: make(map[string]string), retryMax: retryMax},
        usageSvc:     usageSvc,
        healthLogger: slog.Default().With("component", "router"),
    }
}

func (s *Service) Forward(ctx context.Context, rawReq []byte, clientFormat provider.Protocol) ([]byte, *message.Usage, error) {
    client, model, providerURL, req, err := s.prepareRequestWithDetails(rawReq)
    s.healthLogger.Debug("forward starting", "model", model, "provider", providerURL)
    resp, usage, err := client.Do(ctx, req, clientFormat)
    if err != nil {
        s.healthLogger.Error("forward failed", "model", model, "provider", providerURL, "error", err.Error())
        return nil, nil, err
    }
    s.usageSvc.RecordWithModelAndProvider(usage, model, providerURL, false)  // Track cost with provider
    return resp, usage, nil
}
```

All routing logic (endpoint selection, retry, fallback) handled by fluxcore's `flux.Client`.

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

Reload config without restart:

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
tokrouter status

Key            Format    Healthy    Models
openai-main    openai    OK         2/2
anthropic-main anthropic OK         1/1

tokrouter summary --month
Key            Input      Output     Requests   Avg Latency  Success
openai-main    152340     45678     1234       245ms        98.5%
anthropic-main 23456      12345     567        189ms        99.2%
```

---

## CLI Commands

```bash
tokrouter init                       # Interactive configuration wizard
tokrouter serve                      # Start server (127.0.0.1:8765)
tokrouter serve --host 0.0.0.0       # Listen on all interfaces

# New: Service management
tokrouter add <provider>             # Add service with preset (interactive)
tokrouter list services              # List all configured services
tokrouter list assistants            # List supported AI assistants
tokrouter show <service>             # Show service details
tokrouter remove <service>           # Remove a service
tokrouter start                      # Start tokrouter daemon
tokrouter stop                       # Stop tokrouter daemon

# New: Configuration
tokrouter config service <name> --enable/--disable/--secret/--add-model/--remove-model
tokrouter config assistant <name>    # Configure AI assistant to use tokrouter
tokrouter config assistant --auto    # Auto-configure all installed assistants

# Status & monitoring
tokrouter status                     # Show key status
tokrouter status --watch             # Real-time refresh
tokrouter models                     # List all available models
tokrouter keys                       # List all keys (legacy)
tokrouter keys ping <name>           # Test connectivity
tokrouter summary --month            # Monthly statistics
tokrouter summary --chart            # ASCII bar chart

# OpenAPI documentation (new)
curl http://localhost:8765/openapi.yaml  # OpenAPI spec
curl http://localhost:8765/docs          # Swagger UI
```

---

## API Endpoints

| Endpoint | Format | Description |
|----------|--------|-------------|
| `POST /v1/chat/completions` | OpenAI | OpenAI compatible, streaming supported |
| `POST /v1/messages` | Anthropic | Anthropic compatible, streaming supported |
| `GET /status` | JSON | Key status |
| `GET /health` | JSON | Health check with dependency status |
| `GET /openapi.yaml` | YAML | OpenAPI 3.0 specification |
| `GET /docs` | HTML | Swagger UI documentation |

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
tokrouter keys add --name my-key --format openai --secret $MY_KEY --base-url https://api.example.com/v1
```

**Q: How do I test if my key works?**
```bash
tokrouter keys ping openai-main
```

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