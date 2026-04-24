// Package router provides the core routing service for tokrouter.
//
// The Service struct coordinates fluxcore's EndpointPool for request forwarding
// and usage tracking for cost statistics. It implements:
//   - Request forwarding (Forward, ForwardStream)
//   - Provider status tracking (ProviderStatuses)
//   - Circuit breaker health detection
//   - Hot reload via atomic state swapping (Reload)
//
// The service maintains model-specific endpoint pools and selects endpoints based on:
//  1. Priority (lower = preferred) - initial selection
//  2. EWMA latency - runtime adjustment
//
// The service is created from config using NewServiceFromConfig, which initializes
// the endpoint pool and optional usage tracking.
//
// Basic usage:
//
//	svc, err := router.NewServiceFromConfig(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer svc.Close()
//
//	// Forward a request
//	resp, usage, err := svc.Forward(ctx, body, routing.ProtocolOpenAI)
//
//	// Hot reload config
//	if err := svc.Reload(newCfg); err != nil {
//	    log.Printf("reload failed: %v", err)
//	}
package router
