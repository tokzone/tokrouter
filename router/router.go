package router

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/tokzone/fluxcore/call"
	"github.com/tokzone/fluxcore/message"
	"github.com/tokzone/fluxcore/routing"

	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/usage"
)

// Service is the router application service
type Service struct {
	mu            sync.RWMutex
	state         *serviceState // Atomic state container
	usageSvc      *usage.Service
	healthLogger  *slog.Logger
	lastHealthMap map[string]bool // Track last health state for change detection
}

// serviceState holds the mutable state of the service (swapped atomically on reload)
type serviceState struct {
	modelPools map[string]*routing.EndpointPool // Model -> EndpointPool
	aliasMap   map[string]string                // Model alias mapping (request model -> actual model)
	cfg        *config.Config                   // Store config for ServerConfig extraction
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
}

// NewService creates a router service from loaded endpoints
func NewService(endpoints []*routing.Endpoint, usageSvc *usage.Service, retryMax int) *Service {
	return &Service{
		state: &serviceState{
			modelPools: buildModelPools(endpoints, retryMax),
			aliasMap:   make(map[string]string),
		},
		usageSvc:      usageSvc,
		healthLogger:  slog.Default().With("component", "router"),
		lastHealthMap: make(map[string]bool),
	}
}

// buildModelPools groups endpoints by model and creates a pool for each
func buildModelPools(endpoints []*routing.Endpoint, retryMax int) map[string]*routing.EndpointPool {
	modelEndpoints := make(map[string][]*routing.Endpoint)
	for _, ep := range endpoints {
		modelEndpoints[ep.Model] = append(modelEndpoints[ep.Model], ep)
	}

	pools := make(map[string]*routing.EndpointPool)
	for model, eps := range modelEndpoints {
		pools[model] = routing.NewEndpointPool(eps, retryMax)
	}
	return pools
}

// NewServiceFromConfig creates a router service from config (factory method)
func NewServiceFromConfig(cfg *config.Config) (*Service, error) {
	endpoints := cfg.ToEndpoints()

	var usageSvc *usage.Service
	if cfg.Stats.Enabled {
		store, err := usage.NewStore(cfg.Stats.DBPath)
		if err != nil {
			return nil, err
		}
		usageSvc = usage.NewService(store)
	}

	svc := NewService(endpoints, usageSvc, cfg.Router.Retry.MaxRetries)
	svc.state.cfg = cfg

	// Build alias map from config
	svc.state.aliasMap = cfg.AliasMap()

	return svc, nil
}

// ServerConfig returns server configuration
func (s *Service) ServerConfig() config.ServerConfig {
	s.mu.RLock()
	cfg := s.state.cfg
	s.mu.RUnlock()

	if cfg == nil {
		return config.ServerConfig{Host: "127.0.0.1", Port: 8765}
	}
	return config.ServerConfig{
		Host:     cfg.Server.Host,
		Port:     cfg.Server.Port,
		TLSCert:  cfg.Server.TLSCert,
		TLSKey:   cfg.Server.TLSKey,
		LogLevel: cfg.Log.Level,
	}
}

// Forward forwards request to upstream provider (non-streaming)
func (s *Service) Forward(ctx context.Context, rawReq []byte, clientFormat routing.Protocol) ([]byte, *message.Usage, error) {
	pool, epBefore, modifiedReq, err := s.prepareRequest(rawReq)
	if err != nil {
		return nil, nil, err
	}

	resp, usage, err := call.Request(ctx, pool, modifiedReq, clientFormat)
	if err != nil {
		s.checkHealthChange(epBefore)
		return nil, nil, err
	}

	// Record usage
	if s.usageSvc != nil {
		ep := pool.CurrentEp()
		s.usageSvc.RecordWithEndpoint(usage, ep, false) // ignore dropped status for non-streaming
	}

	return resp, usage, nil
}

// ForwardStream forwards request to upstream provider (streaming).
// Returns the stream result and the pool (for usage recording after stream completes).
func (s *Service) ForwardStream(ctx context.Context, rawReq []byte, clientFormat routing.Protocol) (*call.StreamResult, *routing.EndpointPool, error) {
	pool, epBefore, modifiedReq, err := s.prepareRequest(rawReq)
	if err != nil {
		return nil, nil, err
	}

	result, err := call.RequestStream(ctx, pool, modifiedReq, clientFormat)
	if err != nil {
		s.checkHealthChange(epBefore)
		return nil, nil, err
	}

	return result, pool, nil
}

// RecordStreamUsage records usage after stream completes
func (s *Service) RecordStreamUsage(usage *message.Usage, pool *routing.EndpointPool) {
	if s.usageSvc == nil || usage == nil || pool == nil {
		return
	}
	ep := pool.CurrentEp()
	s.usageSvc.RecordWithEndpoint(usage, ep, true)
}

// prepareRequest extracts model, gets pool, and applies alias mapping.
// Returns pool, endpoint before request, modified request, and error.
func (s *Service) prepareRequest(rawReq []byte) (*routing.EndpointPool, *routing.Endpoint, []byte, error) {
	model := parseModelFromRequest(rawReq)

	// Apply alias mapping
	s.mu.RLock()
	state := s.state
	s.mu.RUnlock()

	if actualModel, ok := state.aliasMap[model]; ok {
		model = actualModel
		rawReq = rewriteModelInRequest(rawReq, model)
	}
	pool, ok := state.modelPools[model]
	if !ok {
		return nil, nil, nil, fmt.Errorf("no endpoint for model: %s", model)
	}

	return pool, pool.CurrentEp(), rawReq, nil
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

// getPool returns the endpoint pool for a model
func (s *Service) getPool(model string) (*routing.EndpointPool, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.state.modelPools == nil {
		return nil, false
	}
	pool, ok := s.state.modelPools[model]
	return pool, ok
}

// ProviderStatuses returns runtime status of all providers
func (s *Service) ProviderStatuses() []ProviderStatus {
	s.mu.RLock()
	state := s.state
	s.mu.RUnlock()

	// Group by BaseURL (provider identifier)
	providerMap := make(map[string]*ProviderStatus)
	for _, pool := range state.modelPools {
		ep := pool.CurrentEp()
		if ep == nil || ep.Key == nil {
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
		modelHealthy := !ep.IsCircuitBreakerOpen()
		provider.Models = append(provider.Models, ModelStatus{
			Name:    ep.Model,
			Healthy: modelHealthy,
		})
		if modelHealthy {
			provider.Healthy = true
		}
	}

	statuses := make([]ProviderStatus, 0, len(providerMap))
	for _, status := range providerMap {
		statuses = append(statuses, *status)
	}

	return statuses
}

// Stats returns usage statistics
func (s *Service) Stats(filter usage.QueryFilter) ([]usage.StatRow, error) {
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

// Reload reloads endpoints from config (for hot reload)
func (s *Service) Reload(cfg *config.Config) error {
	// Validate config before applying
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	endpoints := cfg.ToEndpoints()

	// Get retryMax from existing pool
	s.mu.RLock()
	retryMax := 2 // default
	for _, pool := range s.state.modelPools {
		retryMax = pool.RetryMax()
		break
	}
	s.mu.RUnlock()

	// Create new state (atomic swap)
	newState := &serviceState{
		cfg:        cfg,
		aliasMap:   cfg.AliasMap(),
		modelPools: buildModelPools(endpoints, retryMax),
	}

	s.mu.Lock()
	s.state = newState
	s.mu.Unlock()

	s.healthLogger.Info("config reloaded", "models", len(newState.modelPools))
	return nil
}

// parseModelFromRequest extracts model field from raw JSON request
func parseModelFromRequest(rawReq []byte) string {
	var req struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(rawReq, &req); err != nil {
		return ""
	}
	return req.Model
}

// rewriteModelInRequest rewrites the model field in raw JSON request
func rewriteModelInRequest(rawReq []byte, newModel string) []byte {
	var req map[string]interface{}
	if err := json.Unmarshal(rawReq, &req); err != nil {
		return rawReq
	}
	req["model"] = newModel
	result, err := json.Marshal(req)
	if err != nil {
		return rawReq
	}
	return result
}
