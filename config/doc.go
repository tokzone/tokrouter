// Package config provides configuration loading for tokrouter.
//
// Configuration is loaded from YAML files and supports:
//   - Server settings (host, port, TLS)
//   - Key configurations (API keys, base URLs, protocols, models)
//   - Router settings (retry, timeout)
//   - Usage statistics settings
//   - Logging settings
//
// Basic usage:
//
//	cfg, err := config.Load("config.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Access config values
//	fmt.Println(cfg.Server.Host, cfg.Server.Port)
//	fmt.Println(len(cfg.Keys), "keys configured")
package config
