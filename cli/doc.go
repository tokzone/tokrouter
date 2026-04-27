// Package cli provides command-line interface for tokrouter.
//
// Quick start for new users:
//
//	tr list presets           # Browse available provider presets
//	tr add openai --secret sk-xxx   # Add a service by preset
//	tr list services          # List configured services
//	tr start                  # Start the router
//	tr show health --watch    # Monitor endpoint health
//	tr show usage             # View usage statistics
//
// Command reference:
//
//	tr add        [<preset>] [--secret] [--name] [--base-url] [--format] [--model]...
//	tr remove     <name>
//	tr list       [services|models|presets|assistants]
//	tr show       [service <name>|preset <name>|config|health [--watch]|usage [...]]
//	tr config     <name> [--secret|--enable|--disable|--add-model|--remove-model|--alias]
//	tr assistant  [list|auto [--url]|<name> [--url]]
//	tr start      [--host] [--port]
//	tr stop
//
// All commands accept --config / -c flag to specify config file path (default: ./config.yaml).
// Shell completion: tr completion bash|zsh|fish
package cli
