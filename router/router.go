package router

import (
	"context"
	"log/slog"
	"sync"

	"tokflux/tokrouter/config"
	"tokflux/tokrouter/usage"

	"github.com/tokzone/fluxcore/call"
	"github.com/tokzone/fluxcore/message"
	"github.com/tokzone/fluxcore/routing"
)

// Service is the router application service
type Service struct {
	mu            sync.RWMutex
	pool          *routing.EndpointPool // Single source of truth for routing
	endpoints     []*routing.Endpoint   // Keep reference for status queries
	usageSvc      *usage.Service
	cfg           *config.Config // Store config for ServerConfig extraction
	healthLogger  *slog.Logger
	lastHealthMap map[string]bool // Track last health state for change detection
}

// ProviderStatus represents runtime status of a provider (separated from config)
type ProviderStatus struct {
	Name     string // Provider identifier (BaseURL)
	Protocol string // Protocol format
	Healthy  bool   // Overall health status
	Models   []ModelStatus
}

// ModelStatus represents status of a model (DTO)
type ModelStatus struct {
	Name    string
	Healthy bool
	Latency int
}

// NewService creates a router service from loaded endpoints
func NewService(endpoints []*routing.Endpoint, usageSvc *usage.Service, retryMax int) (*Service, error) {
	// Build EndpointPool from endpoints (pool is the single source of truth)
	pool := routing.NewEndpointPool(endpoints, retryMax)

	return &Service{
		pool:          pool,
		endpoints:     endpoints,
		usageSvc:      usageSvc,
		healthLogger:  slog.Default().With("component", "router"),
		lastHealthMap: make(map[string]bool),
	}, nil
}

// NewServiceFromConfig creates a router service from config (factory method)
// This eliminates cross-layer dependency in cli/helpers.go
func NewServiceFromConfig(cfg *config.Config) (*Service, error) {
	endpoints := cfg.ToEndpoints()

	var usageSvc *usage.Service
	if cfg.Stats.Enabled {
		storage, err := usage.NewStorage(cfg.Stats.DBPath)
		if err != nil {
			return nil, err
		}
		usageSvc = usage.NewService(storage)
	}

	svc, err := NewService(endpoints, usageSvc, cfg.Router.Retry.MaxRetries)
	if err != nil {
		return nil, err
	}
	svc.cfg = cfg
	return svc, nil
}

// ServerConfig returns server configuration
func (s *Service) ServerConfig() config.ServerConfig {
	if s.cfg == nil {
		return config.ServerConfig{Host: "127.0.0.1", Port: 8765}
	}
	return config.ServerConfig{
		Host:     s.cfg.Server.Host,
		Port:     s.cfg.Server.Port,
		TLSCert:  s.cfg.Server.TLSCert,
		TLSKey:   s.cfg.Server.TLSKey,
		LogLevel: s.cfg.Log.Level,
	}
}

// Forward forwards request to upstream provider (non-streaming)
func (s *Service) Forward(ctx context.Context, rawReq []byte, clientFormat routing.Protocol) ([]byte, *message.Usage, error) {
	// Get endpoint before request for health tracking
	epBefore := s.pool.CurrentEp()

	resp, usage, err := call.Request(ctx, s.pool, rawReq, clientFormat)
	if err != nil {
		// Check health state change after failed request
		s.checkHealthChange(epBefore)
		return nil, nil, err
	}

	// Record usage with current endpoint info
	if s.usageSvc != nil {
		ep := s.pool.CurrentEp()
		s.usageSvc.RecordWithEndpoint(usage, ep, false) // non-streaming
	}

	return resp, usage, nil
}

// ForwardStream forwards request to upstream provider (streaming)
func (s *Service) ForwardStream(ctx context.Context, rawReq []byte, clientFormat routing.Protocol) (*call.StreamResult, error) {
	result, err := call.RequestStream(ctx, s.pool, rawReq, clientFormat)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// checkHealthChange logs health state changes in real-time
func (s *Service) checkHealthChange(ep *routing.Endpoint) {
	if ep == nil || ep.Key == nil {
		return
	}

	provider := ep.Key.BaseURL
	model := ep.Model
	key := provider + "/" + model

	s.mu.Lock()
	defer s.mu.Unlock()

	// Use IsCircuitBreakerOpen to determine health (circuit breaker open means unhealthy)
	currentHealthy := !ep.IsCircuitBreakerOpen()
	if lastHealthy, ok := s.lastHealthMap[key]; ok && lastHealthy != currentHealthy {
		if currentHealthy {
			s.healthLogger.Info("endpoint recovered", "provider", provider, "model", model)
		} else {
			s.healthLogger.Warn("endpoint became unhealthy", "provider", provider, "model", model)
		}
	}
	s.lastHealthMap[key] = currentHealthy
}

// CurrentEndpoint returns the current best endpoint
func (s *Service) CurrentEndpoint() *routing.Endpoint {
	return s.pool.CurrentEp()
}

// GetProviderStatuses returns runtime status of all providers
func (s *Service) GetProviderStatuses() []ProviderStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Group by BaseURL (provider identifier)
	providerMap := make(map[string]*ProviderStatus)
	for _, ep := range s.endpoints {
		if ep.Key == nil {
			continue
		}
		providerKey := ep.Key.BaseURL
		if _, ok := providerMap[providerKey]; !ok {
			providerMap[providerKey] = &ProviderStatus{
				Name:     providerKey,
				Protocol: ep.Key.Protocol.String(),
			}
		}
		provider := providerMap[providerKey]
		modelHealthy := !ep.IsCircuitBreakerOpen() // Healthy if circuit breaker not open
		provider.Models = append(provider.Models, ModelStatus{
			Name:    ep.Model,
			Healthy: modelHealthy,
		})
		if modelHealthy {
			provider.Healthy = true
		}
	}

	var statuses []ProviderStatus
	for _, status := range providerMap {
		statuses = append(statuses, *status)
	}

	return statuses
}

// GetStats returns usage statistics
func (s *Service) GetStats(filter usage.QueryFilter) ([]usage.StatRow, error) {
	if s.usageSvc == nil {
		return nil, usage.ErrDisabled
	}
	return s.usageSvc.Query(filter)
}

// Close closes the router service
func (s *Service) Close() error {
	if s.usageSvc == nil {
		return nil
	}
	return s.usageSvc.Close()
}
