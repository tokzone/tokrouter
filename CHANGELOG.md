# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.7.1] - 2026-04-28

### Changed — Multi-Protocol Architecture (fluxcore)
- **Provider is now protocol-agnostic**: `Provider` struct only holds `ID` + `BaseURL` + health state. Protocol is no longer a Provider property.
- **Endpoint holds Protocols as capability list**: `Endpoint.Protocols []Protocol` declares supported protocols. One endpoint can serve multiple protocols (e.g., DeepSeek = OpenAI + Anthropic).
- **SelectProtocol for protocol resolution**: `SelectProtocol(input Protocol) Protocol` — direct pass-through when supported, fallback to `Protocols[0]` with auto-translation otherwise.
- **Prepare/Do separation**: `DoFuncGen(client, inputProtocol)` pre-computes target protocol mapping for every endpoint, returns `DoFunc` closure. Hot path has zero protocol decision overhead.
- **StreamDoFuncGen**: Same pre-computation pattern for streaming requests.
- **`Client.Do()` / `Client.DoStream()` deprecated**: Use `DoFuncGen` / `StreamDoFuncGen` instead.

### Changed — Router Interface
- **`RouterService` → `Router`**: Interface renamed per Go convention. Four protocol-specific forwarding methods replace generic `Forward`/`ForwardStream`:
  - `ForwardOpenAI(ctx, body, model)`
  - `ForwardAnthropic(ctx, body, model)`
  - `ForwardStreamOpenAI(ctx, body, model)`
  - `ForwardStreamAnthropic(ctx, body, model)`
- **Eliminated `clientFormat` parameter**: Protocol is bound at the HTTP handler level, not threaded through every call.
- **DoFunc index tables**: `buildDoFuncs()` generates 4 maps (`openAIDoFuncs`, `anthropicDoFuncs`, `openAIStreamDoFuncs`, `anthropicStreamDoFuncs`) keyed by model name. O(1) lookup on hot path.
- **Model alias with body rewriting**: `resolveAlias` + `rewriteModelInRequest` modify the request body so upstream receives the actual model name.
- **Error classification**: Forward failures classified as `errors.CodeNoEndpoint` for proper 503 upstream_error responses.

### Changed — Config
- **`KeyConfig.Protocols` removed**: Protocol list is derived from the provider preset (via `Provider` field), not stored redundantly in each key.
- **`ProtocolList()` derives from preset**: If `Provider` is set, looks up `GetPreset(k.Provider).Protocols`. Falls back to `Format` for custom services.
- **`ProviderPreset.Protocols` field**: DeepSeek and OpenRouter presets declare `Protocols: [openai, anthropic]`.
- **`Save()` writes `provider` field**: Config YAML now includes `provider` when set, no longer writes `protocols` on keys.

### Changed — CLI
- **Restructured commands**: Removed `init`, `keys`, `models`, `serve`, `status`, `summary`. New streamlined command set:
  - `add`, `remove`, `list`, `show`, `config`, `assistant`, `start`, `stop`
- **`add` command**: Preset mode automatically inherits protocols from the provider preset. Custom mode uses single `Format`. No `--protocols` flag needed.
- **`show` command**: New subcommands — `service`, `preset`, `health --watch`, `usage --today/--week/--chart/--export`.
- **`assistant` command** (new): Configure AI coding assistants (`claude-code`, `cursor`, `aider`, etc.) to use tokrouter. Supports `auto` detection and individual configuration.
- **Shell completion**: Enabled via `tr completion bash|zsh|fish`.

### Changed — Server
- **Protocol-specific handlers**: `HandleOpenAI` → `POST /v1/chat/completions`, `HandleAnthropic` → `POST /v1/messages`. Protocol is compile-time constant in each handler.
- **`HandleShutdown`** (new): `POST /shutdown` for graceful shutdown.
- **Signal handling**: `signal.Stop()` deferred in Run() goroutine, preventing leaks.

### Added — Presets
- **`Protocols` field on presets**: DeepSeek and OpenRouter presets now declare `[openai, anthropic]` protocol capability.

### Updated — Documentation
- **USER_GUIDE.md**: New-user workflow, full CLI reference with output examples, config reflects provider-based protocol derivation.
- **API.md**: Multi-protocol architecture section, URL-based protocol routing docs.
- **fluxcore README** (EN/CN): Updated API signatures, Prepare/Do separation docs, protocol selection docs.

### Migration Guide (v0.7.0 → v0.7.1)

**Config**: Remove `protocols` field from keys. Use `provider` to inherit preset protocols.

```yaml
# Before (v0.7.0)
keys:
  - name: deepseek
    base_url: "https://api.deepseek.com"
    format: openai
    protocols: [openai, anthropic]
    secret: "${DEEPSEEK_API_KEY}"
    models:
      - name: deepseek-chat

# After (v0.7.1)
keys:
  - name: deepseek
    provider: deepseek
    secret: "${DEEPSEEK_API_KEY}"
    models:
      - name: deepseek-chat
```

**CLI**: Old subcommands replaced:
| v0.7.0 | v0.7.1 |
|--------|--------|
| `tr keys add openai --secret sk-xxx` | `tr add openai --secret sk-xxx` |
| `tr keys list` | `tr list services` |
| `tr keys remove <name>` | `tr remove <name>` |
| `tr serve` | `tr start` |
| `tr status [--watch]` | `tr show health [--watch]` |
| `tr summary` | `tr show usage` |
| `tr init` | (removed, use `tr add` interactively) |
| `tr models` | `tr list models` |

## [0.7.0] - 2026-04-26

### Added
- **RouterService interface**: Abstract router operations for testing and extensibility
  - `MockRouterService` implementation for unit tests
- **Provider presets**: Simplified configuration with preset providers
  - 20+ presets: openai, anthropic, deepseek, qwen, zhipu, moonshot, etc.
  - Just specify `provider` and `secret`, other fields auto-filled
- **HTTP client configuration**: Optional `http` config section
  - Customize timeout, connection pool settings
- **Rate limiting middleware**: Global + per-provider rate limiting (not yet enabled)
- **OpenAPI documentation**: `/openapi.yaml` and `/docs` endpoints
- **CLI improvements**: 
  - `tr config service <name>` - configure service settings
  - `tr config assistant <name>` - configure AI assistants to use tokrouter
  - `tr config assistant --auto` - auto-detect and configure installed assistants
  - `tr add <provider>` - add service with preset
  - `tr list services/assistants` - list configured items
  - `tr show <service>` - show service details
  - `tr start/stop` - manage tokrouter daemon

### Changed
- **RouterService interface signature**: `Forward`/`ForwardStream` now accept `model string` parameter
  - Eliminates redundant JSON parsing (was 3x, now max 2x)
- **Version**: Updated to v0.7.0

### Migration Guide (v0.6.0 → v0.7.0)

```yaml
# Before (v0.6.0)
keys:
  - name: openai-main
    base_url: "https://api.openai.com/v1"
    format: openai
    secret: "${OPENAI_API_KEY}"
    enabled: true
    models:
      - name: gpt-4

# After (v0.7.0) - Simplified with preset
keys:
  - provider: openai
    secret: "${OPENAI_API_KEY}"
    # models auto-filled: gpt-4o, gpt-4o-mini, gpt-4-turbo, o1, o3-mini...

# Traditional config still works (backward compatible)
```

## [0.6.0] - 2026-04-24

### Added
- **AvgLatency and SuccessRate in summary**: CLI summary now displays average latency and success rate statistics
- **Comprehensive test suite**: Added tests for SQLite storage, query functions, request parsing, error handling, logging, and concurrent operations
- **Test helper functions**: Added `newTestKey`, `newTestEndpoint`, `newTestService` to reduce test boilerplate
- **Package documentation**: Added `doc.go` for all packages (usage, router, server, config, cli)

### Changed
- **Go naming convention fix**: `GetAliasMap()` renamed to `AliasMap()` (Go getters should not use "Get" prefix)
- **Abbreviation consistency**: `LatencyMs` renamed to `LatencyMS` for consistency with other abbreviations (ID, URL, TLS)
- **Version sync**: Health endpoint version updated from v0.1.0 to v0.6.0
- **Test reliability**: Fixed fragile tests using channel synchronization instead of time.Sleep
- **Test coverage**: Improved from ~24% to ~75% average coverage across packages

### Removed
- **Unused error codes**: Removed ErrCodeNoEndpoint, ErrCodeUnauthorized, ErrCodeRateLimited, ErrCodeModelNotFound
- **Unused struct field**: Removed ModelStatus.Latency field (never populated)
- **Test-only exported methods**: Removed PrepareRequest, CurrentEndpoint, DroppedCount (only used in tests)
- **Deleted helpers.go**: Removed cli/helpers.go (functionality moved to specific command files)

### Added (CLI improvements)
- **Environment variable warning**: Logs warning when env var (e.g., `${OPENAI_API_KEY}`) is not set
- **Interactive mode for `keys add`**: Run without flags to enter interactive key creation
- **`models` command**: New `tokrouter models` command to list all available models
- **`serve --host` flag**: Specify server host address via CLI
- **Configuration confirmation in `init`**: Shows summary before saving, allows cancel
- **Input validation in `init`**: Validates port range (1-65535) and URL format

### Changed (CLI improvements)
- **Improved CLI error messages**: All errors now include context, suggestions, and usage examples
- **Improved config validation errors**: Friendly error messages with YAML configuration examples
- **Improved usage errors**: `ErrDisabled`, `ErrInvalidFilter`, `ErrRecordNotFound` now include suggestions
- **Improved `keys ping` output**: Shows latency per model and test summary (passed/failed count)
- **Config constants**: CLI now uses config.FormatOpenAI constants instead of hardcoded strings

### Breaking Changes
- CLI `keys add` flags are no longer required - interactive mode available when no flags provided
- Configuration validation errors format changed (now includes suggestions)

## [0.5.1] - 2026-04-23

### Added
- CI/CD workflow with golangci-lint
- Package documentation (doc.go) for all packages
- User guide and API documentation
- `keys enable/disable/ping` subcommands
- `status --watch` flag for real-time status
- `summary --chart` flag for ASCII bar chart
- **Model-level routing**: Requests are now routed to endpoints matching the requested model
- **Model alias support**: Map request model names to actual model names via `alias` field
- **Hot reload**: Send SIGHUP to reload config without restart
- **Latency-aware routing**: EWMA-based latency tracking for smarter endpoint selection

### Changed
- Simplified CLI command structure
- Unified configuration types across packages
- Improved circuit breaker health detection
- Renamed `providers` to `keys` in config
- Renamed `run` command to `serve`
- **Path resolution**: Relative paths now based on config file directory (not working directory)
- **Routing strategy**: Price-first, then EWMA latency (was static latency)

### Fixed
- **Data race**: Fixed concurrent read/write on modelMap/aliasMap during hot reload
- **ForwardStream**: Added latency update and health check consistency with Forward

### Removed
- Unused HealthChangeListener interface
- Redundant CLI commands (switch, ping as top-level)
- Duplicate type definitions

