package router

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/tokzone/fluxcore/endpoint"
	"github.com/tokzone/fluxcore/errors"
	"github.com/tokzone/fluxcore/flux"
	"github.com/tokzone/fluxcore/message"
	"github.com/tokzone/fluxcore/provider"

	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/usage"
)

// Service is the router application service
type Service struct {
	mu            sync.RWMutex
	state         *serviceState // Atomic state container
	usageSvc      *usage.Service
	healthLogger  *slog.Logger
}

// serviceState holds mutable state swapped atomically on reload.
type serviceState struct {
	clients   map[string]*flux.Client
	aliasMap  map[string]string
	cfg       *config.Config
	retryMax  int
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

// NewService creates a router service from user endpoints
func NewService(userEndpoints []*flux.UserEndpoint, usageSvc *usage.Service, retryMax int) *Service {
	return &Service{
		state: &serviceState{
			clients:  buildClients(userEndpoints, retryMax),
			aliasMap: make(map[string]string),
			retryMax: retryMax,
		},
		usageSvc:     usageSvc,
		healthLogger: slog.Default().With("component", "router"),
	}
}

// buildClients groups user endpoints by model and creates a flux.Client for each.
func buildClients(userEndpoints []*flux.UserEndpoint, retryMax int) map[string]*flux.Client {
	modelEndpoints := make(map[string][]*flux.UserEndpoint)
	for _, ue := range userEndpoints {
		modelEndpoints[ue.Model()] = append(modelEndpoints[ue.Model()], ue)
	}

	clients := make(map[string]*flux.Client)
	for model, ues := range modelEndpoints {
		clients[model] = flux.NewClient(ues, flux.WithRetryMax(retryMax))
	}
	return clients
}

// NewServiceFromConfig creates a router service from config (factory method)
func NewServiceFromConfig(cfg *config.Config) (*Service, error) {
	userEndpoints := cfg.ToUserEndpoints()

	var usageSvc *usage.Service
	if cfg.Stats.Enabled {
		store, err := usage.NewStore(cfg.Stats.DBPath)
		if err != nil {
			return nil, err
		}
		usageSvc = usage.NewService(store)
	}

	svc := NewService(userEndpoints, usageSvc, cfg.Router.Retry.MaxRetries)
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
func (s *Service) Forward(ctx context.Context, rawReq []byte, clientFormat provider.Protocol) ([]byte, *message.Usage, error) {
	client, model, providerURL, modifiedReq, err := s.prepareRequestWithDetails(rawReq)
	if err != nil {
		return nil, nil, err
	}

	s.healthLogger.Debug("forward starting", "model", model, "provider", providerURL)

	resp, usage, err := client.Do(ctx, modifiedReq, clientFormat)
	if err != nil {
		s.healthLogger.Error("forward failed", "model", model, "provider", providerURL, "error", err.Error())
		return nil, nil, err
	}

	// Record usage with provider info
	if s.usageSvc != nil && usage != nil {
		s.usageSvc.RecordWithModelAndProvider(usage, model, providerURL, false)
	}

	return resp, usage, nil
}

// ForwardStream forwards request to upstream provider (streaming).
// Returns the stream result, model name, and provider URL (for usage recording after stream completes).
func (s *Service) ForwardStream(ctx context.Context, rawReq []byte, clientFormat provider.Protocol) (*flux.StreamResult, string, string, error) {
	client, model, providerURL, modifiedReq, err := s.prepareRequestWithDetails(rawReq)
	if err != nil {
		return nil, "", "", err
	}

	s.healthLogger.Debug("forward stream starting", "model", model, "provider", providerURL)

	result, err := client.DoStream(ctx, modifiedReq, clientFormat)
	if err != nil {
		s.healthLogger.Error("forward stream failed", "model", model, "provider", providerURL, "error", err.Error())
		return nil, "", "", err
	}

	return result, model, providerURL, nil
}

// RecordStreamUsage records usage after stream completes
func (s *Service) RecordStreamUsage(usage *message.Usage, model string, providerURL string) {
	if s.usageSvc == nil || usage == nil {
		return
	}
	s.usageSvc.RecordWithModelAndProvider(usage, model, providerURL, true)
}

func (s *Service) prepareRequestWithDetails(rawReq []byte) (*flux.Client, string, string, []byte, error) {
	model := parseModelFromRequest(rawReq)

	s.mu.RLock()
	state := s.state
	s.mu.RUnlock()

	if actualModel, ok := state.aliasMap[model]; ok {
		model = actualModel
		rawReq = rewriteModelInRequest(rawReq, model)
	}

	client, ok := state.clients[model]
	if !ok {
		return nil, "", "", nil, errors.Wrap(errors.CodeNoEndpoint, fmt.Sprintf("no endpoint for model: %s", model), nil)
	}

	ue := client.Next()
	if ue == nil {
		return nil, "", "", nil, errors.Wrap(errors.CodeNoEndpoint, fmt.Sprintf("no available endpoint for model: %s", model), nil)
	}
	providerURL := ue.BaseURL()

	return client, model, providerURL, rawReq, nil
}

func (s *Service) getClient(model string) (*flux.Client, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.state.clients == nil {
		return nil, false
	}
	client, ok := s.state.clients[model]
	return client, ok
}

func (s *Service) ProviderStatuses() []ProviderStatus {
	endpoints := endpoint.GlobalRegistry().GetAll()

	providerMap := make(map[string]*ProviderStatus)
	for _, ep := range endpoints {
		prov := ep.Provider()
		providerKey := prov.BaseURL
		pStatus, ok := providerMap[providerKey]
		if !ok {
			pStatus = &ProviderStatus{
				Name:     providerKey,
				Protocol: prov.Protocol.String(),
			}
			providerMap[providerKey] = pStatus
		}
		modelHealthy := !ep.IsCircuitBreakerOpen()
		pStatus.Models = append(pStatus.Models, ModelStatus{
			Name:    ep.Model,
			Healthy: modelHealthy,
		})
		if modelHealthy {
			pStatus.Healthy = true
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

func (s *Service) Close() error {
	if s.usageSvc == nil {
		return nil
	}
	return s.usageSvc.Close()
}

func (s *Service) Reload(cfg *config.Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	endpoint.GlobalRegistry().Clear()

	userEndpoints := cfg.ToUserEndpoints()

	s.mu.RLock()
	retryMax := s.state.retryMax
	s.mu.RUnlock()

	newState := &serviceState{
		cfg:      cfg,
		aliasMap: cfg.AliasMap(),
		clients:  buildClients(userEndpoints, retryMax),
		retryMax: retryMax,
	}

	s.mu.Lock()
	s.state = newState
	s.mu.Unlock()

	s.healthLogger.Info("config reloaded", "models", len(newState.clients))
	return nil
}

func parseModelFromRequest(rawReq []byte) string {
	var req struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(rawReq, &req); err != nil {
		return ""
	}
	return req.Model
}

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
