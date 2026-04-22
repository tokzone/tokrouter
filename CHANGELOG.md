# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- CI/CD workflow with golangci-lint
- Package documentation (doc.go) for all packages
- User guide and API documentation
- `keys enable/disable/ping` subcommands
- `status --watch` flag for real-time status
- `summary --chart` flag for ASCII bar chart

### Changed
- Simplified CLI command structure
- Unified configuration types across packages
- Improved circuit breaker health detection
- Renamed `providers` to `keys` in config
- Renamed `run` command to `serve`

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