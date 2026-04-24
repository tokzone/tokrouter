# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

