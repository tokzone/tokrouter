# tokrouter

One binary to manage all LLM APIs. Configure everything from the command line — no manual file editing.

- **One binary**, zero dependencies, runs locally
- **One command to set up everything** — auto-detect AI tools, pick models, add keys, all in one wizard
- **26 provider presets** ready to use with automatic failover

## Installation

### Pre-built binary (recommended)

**Linux**
```bash
curl -sL https://github.com/tokzone/tokrouter/releases/latest/download/tokrouter-linux-amd64 -o tokrouter
chmod +x tokrouter
sudo mv tokrouter /usr/local/bin/
```

**macOS (Apple Silicon)**
```bash
curl -sL https://github.com/tokzone/tokrouter/releases/latest/download/tokrouter-darwin-arm64 -o tokrouter
chmod +x tokrouter
sudo mv tokrouter /usr/local/bin/
```

**Windows (PowerShell)**
```powershell
Invoke-WebRequest -Uri "https://github.com/tokzone/tokrouter/releases/latest/download/tokrouter-windows-amd64.exe" -OutFile "tokrouter.exe"
```

Verify:
```bash
tokrouter version
```

### Build from source

```bash
git clone https://github.com/tokzone/tokrouter.git && cd tokrouter
go build -o tokrouter ./cmd/tokrouter
```

## Get Started

Two steps from install to running:

### Step 1: One-command setup

```bash
./tokrouter assistant auto --url http://127.0.0.1:8765
```

Auto-detects installed AI coding tools (Claude Code, Cursor, Aider, Codex CLI, etc.), lists what it found. Then guides you through picking a default model from 26 presets — if you choose an unconfigured provider, it prompts for the API key and adds it automatically. Confirm and it writes all tool configs in one shot.

### Step 2: Start the gateway

```bash
./tokrouter start
# Gateway running at http://127.0.0.1:8765
```

Verify:
```bash
curl http://127.0.0.1:8765/health
```

Your AI tools are now pointed at tokrouter. Open them and start using. To add more providers later, use `./tokrouter add`.

---

## assistant auto

`assistant auto` is the core feature: auto-detects installed AI coding tools, guides model selection (auto-adds providers when needed), and writes all tool configs in one shot.

### Supported AI tools

| Tool | Protocol | Config Type | Written To |
|------|---------|-------------|------------|
| **Claude Code** | Anthropic Messages | env vars | Shell profile (`~/.zshrc` etc.) |
| **Codex CLI** | Responses API | TOML | `~/.codex/config.toml` |
| **Cursor** | OpenAI Chat | JSON | `~/.cursor/config.json` |
| **Aider** | OpenAI Chat | env vars | Shell profile (`~/.zshrc` etc.) |
| **Windsurf** | OpenAI Chat | JSON | `~/.windsurf/config.json` |
| **Cline** | OpenAI Chat | JSON | VS Code `settings.json` |
| **Continue.dev** | OpenAI Chat | JSON | `~/.continue/config.json` |

Note for Codex CLI: uses OpenAI Responses API. Generated TOML includes `wire_api = "responses"` for compatibility.

### Configure a single tool

```bash
./tokrouter assistant claude-code --url http://127.0.0.1:8765
./tokrouter assistant codex --url http://127.0.0.1:8765
./tokrouter assistant cursor --url http://127.0.0.1:8765
```

Restart the tool after configuration.

---

## Provider Presets

The `add` command and `assistant auto` support 26 built-in presets. Presets auto-fill base URL, protocol format, and default models:

```
openai         anthropic      google         mistral
cohere         groq           deepseek       zhipu
qwen           tencent        baidu          qianfan
huawei         moonshot       minimax        siliconflow
yi             stepfun        baichuan       xunfei
doubao         parallel       modelscope     together
replicate      openrouter
```

View preset details:

```bash
./tokrouter show preset deepseek    # shows base_url, format, default models
./tokrouter list presets            # list all presets
```

---

## Command Reference

### Manage services

```bash
./tokrouter add                              # interactive add
./tokrouter add <preset> --secret sk-xxx     # direct preset add
./tokrouter remove <name>                    # remove service
./tokrouter config <name> --enable           # enable
./tokrouter config <name> --disable          # disable
./tokrouter config <name> --secret sk-new    # update key
./tokrouter config <name> --add-model gpt-4  # add model
./tokrouter config <name> --remove-model old # remove model
```

### Run the gateway

```bash
./tokrouter start                     # default 127.0.0.1:8765
./tokrouter start --port 8080         # custom port
./tokrouter start --host 0.0.0.0      # listen on all interfaces
./tokrouter stop                      # graceful shutdown
kill -SIGHUP $(pidof tokrouter)       # hot reload config without downtime
```

### Inspect state

```bash
./tokrouter list services             # configured services
./tokrouter list models               # all available models
./tokrouter list presets              # all presets
./tokrouter list assistants           # supported AI tools
./tokrouter show health               # endpoint health
./tokrouter show health --watch       # live refresh
./tokrouter show usage --month        # monthly usage stats
./tokrouter show usage --chart        # token distribution chart
./tokrouter show usage --export csv   # export CSV
```

### Shell completion

```bash
./tokrouter completion bash           # Bash
./tokrouter completion zsh            # Zsh
./tokrouter completion fish           # Fish
```

---

## Configuration File

`./tokrouter add` and `assistant auto` write to `config.yaml`. Most operations can be done through commands; edit the file directly for bulk changes.

```yaml
server:
  host: "127.0.0.1"       # listen address
  port: 8765               # port

router:
  retry:
    max_retries: 2         # max retry attempts
    timeout: 30s
    backoff: exponential

keys:
  # Mode 1: Preset (recommended — auto-fills base_url and format)
  - name: my-deepseek     # unique name, defaults to provider name
    provider: deepseek
    secret: "${DEEPSEEK_API_KEY}"
    models:               # optional, uses all preset models if omitted
      - name: deepseek-v4-pro
        priority: 100
      - name: deepseek-chat
        priority: 200

  # Mode 2: Custom (manual configuration)
  - name: self-hosted
    base_url: https://llm.example.com/v1
    format: openai        # openai / anthropic / gemini / cohere
    secret: sk-xxx
    models:
      - name: local-model
```

### Priority and failover

With multiple keys for the same model, the one with lower `priority` is tried first. If a key fails, the next one takes over automatically.

### Model aliases

```yaml
models:
  - name: gpt-4-turbo
    alias: gpt-4-1106-preview   # tools see gpt-4-turbo, requests go to gpt-4-1106-preview
```

---

## API Endpoints

| Endpoint | Protocol | Streaming |
|----------|----------|:---------:|
| `POST /v1/chat/completions` | OpenAI Chat Completions | ✓ |
| `POST /v1/messages` | Anthropic Messages | ✓ |
| `POST /v1/responses` | OpenAI Responses (Codex CLI) | ✓ |
| `GET /v1/models` | Model list | - |
| `GET /health` | Health check | - |
| `GET /status` | Service status | - |
| `GET /openapi.yaml` | OpenAPI spec | - |
| `GET /docs` | Swagger UI | - |
