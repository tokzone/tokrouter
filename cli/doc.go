// Package cli provides command-line interface for tokrouter.
//
// Commands:
//   - tokrouter init     - Interactive configuration setup
//   - tokrouter serve    - Start the HTTP server
//   - tokrouter status   - Show key status (--watch for real-time)
//   - tokrouter summary  - Show usage statistics (--today, --week, --month, --chart)
//   - tokrouter keys     - Manage API keys (list, add, remove, enable, disable, ping)
//   - tokrouter config   - Show current configuration
//
// All commands accept --config flag to specify config file path.
package cli
