// Package router provides the core routing service for tokrouter.
//
// The Service struct coordinates fluxcore's EndpointPool for request forwarding
// and usage tracking for cost statistics. It implements:
//   - Request forwarding (Forward, ForwardStream)
//   - Provider status tracking (GetProviderStatuses)
//   - Circuit breaker health detection
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
package router
