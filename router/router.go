package router

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/usage"

	"github.com/tokzone/fluxcore/call"
	"github.com/tokzone/fluxcore/message"
	"github.com/tokzone/fluxcore/routing"
)

// Service is the router application service
type Service struct {
	mu            sync.RWMutex
	pool          *routing.EndpointPool // Single source of truth for routing
	endpoints     []*routing.Endpoint   // Keep reference for status queries
	modelMap      map[string][]*routing.Endpoint // Model to endpoints mapping
	aliasMap      map[string]string                // Model alias mapping (request model -> actual model)
	pricingMap    map[uint]config.PricingConfig    // Endpoint ID -> Pricing (for O(1) lookup)
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

	// Build model to endpoints mapping
	modelMap := make(map[string][]*routing.Endpoint)
	for _, ep := range endpoints {
		modelMap[ep.Model] = append(modelMap[ep.Model], ep)
	}

	return &Service{
		pool:          pool,
		endpoints:     endpoints,
		modelMap:      modelMap,
		aliasMap:      make(map[string]string),
		pricingMap:    make(map[uint]config.PricingConfig),
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

	// Build alias map from config
	svc.aliasMap = cfg.GetAliasMap()

	// Build pricing map for O(1) lookup
	svc.pricingMap = buildPricingMap(cfg, endpoints)

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
	// Parse model from request
	model := parseModelFromRequest(rawReq)

	// Get alias and model maps under read lock
	s.mu.RLock()
	aliasMap := s.aliasMap
	modelMap := s.modelMap
	s.mu.RUnlock()

	// Apply alias mapping if exists
	if actualModel, ok := aliasMap[model]; ok {
		model = actualModel
		rawReq = rewriteModelInRequest(rawReq, model)
	}

	// Select endpoint for model (no lock needed, using local modelMap)
	epBefore := s.selectEndpointForModel(model, modelMap)

	// Use global pool if no model-specific endpoint
	pool := s.pool
	if epBefore == nil {
		epBefore = pool.CurrentEp()
	}

	resp, usage, err := call.Request(ctx, pool, rawReq, clientFormat)
	if err != nil {
		s.checkHealthChange(epBefore)
		return nil, nil, err
	}

	// Update endpoint latency
	if ep := pool.CurrentEp(); ep != nil && usage != nil {
		ep.UpdateLatency(usage.LatencyMs)
	}

	// Record usage
	if s.usageSvc != nil {
		ep := pool.CurrentEp()
		inputPrice, outputPrice := s.getPricing(ep)
		s.usageSvc.RecordWithEndpoint(usage, ep, false, inputPrice, outputPrice)
	}

	return resp, usage, nil
}

// ForwardStream forwards request to upstream provider (streaming)
func (s *Service) ForwardStream(ctx context.Context, rawReq []byte, clientFormat routing.Protocol) (*call.StreamResult, error) {
	// Parse model from request
	model := parseModelFromRequest(rawReq)

	// Get alias and model maps under read lock
	s.mu.RLock()
	aliasMap := s.aliasMap
	modelMap := s.modelMap
	s.mu.RUnlock()

	// Apply alias mapping if exists
	if actualModel, ok := aliasMap[model]; ok {
		model = actualModel
		rawReq = rewriteModelInRequest(rawReq, model)
	}

	// Select endpoint for model (no lock needed, using local modelMap)
	epBefore := s.selectEndpointForModel(model, modelMap)

	// Use global pool if no model-specific endpoint
	pool := s.pool
	if epBefore == nil {
		epBefore = pool.CurrentEp()
	}

	result, err := call.RequestStream(ctx, pool, rawReq, clientFormat)
	if err != nil {
		s.checkHealthChange(epBefore)
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

// Reload reloads endpoints from config (for hot reload)
func (s *Service) Reload(cfg *config.Config) {
	endpoints := cfg.ToEndpoints()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Update endpoints
	s.endpoints = endpoints
	s.cfg = cfg
	s.aliasMap = cfg.GetAliasMap()
	s.pricingMap = buildPricingMap(cfg, endpoints)

	// Rebuild model map
	s.modelMap = make(map[string][]*routing.Endpoint)
	for _, ep := range endpoints {
		s.modelMap[ep.Model] = append(s.modelMap[ep.Model], ep)
	}

	// Rebuild pool
	s.pool = routing.NewEndpointPool(endpoints, s.pool.RetryMax())

	s.healthLogger.Info("config reloaded", "endpoints", len(endpoints))
}

// getPricing returns the pricing for an endpoint from config (O(1) lookup)
func (s *Service) getPricing(ep *routing.Endpoint) (inputPrice, outputPrice float64) {
	if ep == nil {
		return 0, 0
	}
	if pricing, ok := s.pricingMap[ep.ID]; ok {
		return pricing.Input, pricing.Output
	}
	return 0, 0
}

// buildPricingMap creates endpoint ID -> pricing mapping for O(1) lookup
func buildPricingMap(cfg *config.Config, endpoints []*routing.Endpoint) map[uint]config.PricingConfig {
	pricingMap := make(map[uint]config.PricingConfig)
	for _, ep := range endpoints {
		for _, kc := range cfg.Keys {
			if kc.BaseURL == ep.Key.BaseURL {
				for _, mc := range kc.Models {
					if mc.Name == ep.Model {
						pricingMap[ep.ID] = mc.Pricing
					}
				}
			}
		}
	}
	return pricingMap
}

// selectEndpointForModel selects the best healthy endpoint for a specific model.
func (s *Service) selectEndpointForModel(model string, modelMap map[string][]*routing.Endpoint) *routing.Endpoint {
	endpoints, ok := modelMap[model]
	if !ok || len(endpoints) == 0 {
		return nil
	}

	// Find best healthy endpoint (zero allocation)
	var best *routing.Endpoint
	for _, ep := range endpoints {
		if ep.IsCircuitBreakerOpen() {
			continue
		}
		if best == nil || ep.Priority < best.Priority {
			best = ep
		}
	}
	return best
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
