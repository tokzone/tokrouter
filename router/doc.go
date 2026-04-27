// Package router provides the core routing service for tokrouter.
//
// Router coordinates protocol-specific DoFunc maps for request forwarding
// and usage tracking for cost statistics. It implements:
//   - Request forwarding (ForwardOpenAI, ForwardAnthropic, and stream variants)
//   - Provider status tracking (ProviderStatuses)
//   - Circuit breaker health detection
//   - Hot reload via atomic state swapping (Reload)
//
// Routing selects endpoints based on priority (lower = preferred) and
// health. Protocol selection is orthogonal — input protocol is baked into
// DoFunc closures at startup, output protocol is pre-computed per endpoint.
//
// Basic usage:
//
//	svc, err := router.NewFromConfig(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer svc.Close()
//
//	// Forward an OpenAI request
//	resp, usage, err := svc.ForwardOpenAI(ctx, body, "gpt-4")
//
//	// Hot reload config
//	if err := svc.Reload(newCfg); err != nil {
//	    log.Printf("reload failed: %v", err)
//	}
package router
