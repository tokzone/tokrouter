# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

## [0.1.0] - 2024-01-01

### Added
- Initial release
- OpenAI-compatible chat completions endpoint
- Anthropic-compatible messages endpoint
- Provider status endpoint
- Health check endpoint
- Usage statistics tracking
- CLI for server management
- Configuration via YAML files
- Circuit breaker for fault tolerance
- Automatic retry on failures
- Protocol translation (OpenAI ↔ Anthropic)