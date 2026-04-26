package router

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/tokzone/fluxcore/endpoint"
	"github.com/tokzone/fluxcore/errors"
	"github.com/tokzone/fluxcore/flux"
	"github.com/tokzone/fluxcore/message"
	"github.com/tokzone/fluxcore/provider"

	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/usage"
)

// RouterService defines the interface for router operations.
type RouterService interface {
	Forward(ctx context.Context, rawReq []byte, model string, clientFormat provider.Protocol) ([]byte, *message.Usage, error)
	ForwardStream(ctx context.Context, rawReq []byte, model string, clientFormat provider.Protocol) (*flux.StreamResult, string, string, error)
	RecordStreamUsage(usage *message.Usage, model string, providerURL string)
	ProviderStatuses() []ProviderStatus
	Stats(filter usage.QueryFilter) ([]usage.StatRow, error)
	ServerConfig() config.ServerConfig
	Reload(cfg *config.Config) error
	Close() error
}

// Service is the router application service (implements RouterService)
type Service struct {
	mu            sync.RWMutex
	state         *serviceState // Atomic state container
	usageSvc      *usage.Service
	healthLogger  *slog.Logger
}

// Ensure Service implements RouterService at compile time
var _ RouterService = (*Service)(nil)

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
func NewService(userEndpoints []*flux.UserEndpoint, usageSvc *usage.Service, retryMax int, httpCfg *config.HTTPConfig) *Service {
	return &Service{
		state: &serviceState{
			clients:  buildClients(userEndpoints, retryMax, httpCfg),
			aliasMap: make(map[string]string),
			retryMax: retryMax,
		},
		usageSvc:     usageSvc,
		healthLogger: slog.Default().With("component", "router"),
	}
}

// buildClients groups user endpoints by model and creates a flux.Client for each.
// If httpCfg is provided, creates a custom HTTP client with those settings.
func buildClients(userEndpoints []*flux.UserEndpoint, retryMax int, httpCfg *config.HTTPConfig) map[string]*flux.Client {
	modelEndpoints := make(map[string][]*flux.UserEndpoint)
	for _, ue := range userEndpoints {
		modelEndpoints[ue.Model()] = append(modelEndpoints[ue.Model()], ue)
	}

	// Build HTTP client options if config provided
	var httpOpts []flux.Option
	if httpCfg != nil && httpCfg.Timeout != "" {
		httpOpts = append(httpOpts, flux.WithHTTPClient(buildHTTPClient(httpCfg)))
	}

	clients := make(map[string]*flux.Client)
	for model, ues := range modelEndpoints {
		opts := append([]flux.Option{flux.WithRetryMax(retryMax)}, httpOpts...)
		clients[model] = flux.NewClient(ues, opts...)
	}
	return clients
}

// buildHTTPClient creates a custom HTTP client from HTTPConfig.
func buildHTTPClient(cfg *config.HTTPConfig) *http.Client {
	timeout, _ := time.ParseDuration(cfg.Timeout)
	idleTimeout, _ := time.ParseDuration(cfg.IdleConnTimeout)

	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        cfg.MaxIdleConns,
			MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
			IdleConnTimeout:     idleTimeout,
		},
	}
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

	// Pass HTTP config if timeout is set
	var httpCfg *config.HTTPConfig
	if cfg.HTTP.Timeout != "" {
		httpCfg = &cfg.HTTP
	}

	svc := NewService(userEndpoints, usageSvc, cfg.Router.Retry.MaxRetries, httpCfg)
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
func (s *Service) Forward(ctx context.Context, rawReq []byte, model string, clientFormat provider.Protocol) ([]byte, *message.Usage, error) {
	client, actualModel, providerURL, modifiedReq, err := s.prepareRequest(rawReq, model)
	if err != nil {
		return nil, nil, err
	}

	s.healthLogger.Debug("forward starting", "model", actualModel, "provider", providerURL)

	resp, usage, err := client.Do(ctx, modifiedReq, clientFormat)
	if err != nil {
		s.healthLogger.Error("forward failed", "model", actualModel, "provider", providerURL, "error", err.Error())
		return nil, nil, err
	}

	if s.usageSvc != nil && usage != nil {
		s.usageSvc.RecordWithModelAndProvider(usage, actualModel, providerURL, false)
	}

	return resp, usage, nil
}

// ForwardStream forwards request to upstream provider (streaming).
func (s *Service) ForwardStream(ctx context.Context, rawReq []byte, model string, clientFormat provider.Protocol) (*flux.StreamResult, string, string, error) {
	client, actualModel, providerURL, modifiedReq, err := s.prepareRequest(rawReq, model)
	if err != nil {
		return nil, "", "", err
	}

	s.healthLogger.Debug("forward stream starting", "model", actualModel, "provider", providerURL)

	result, err := client.DoStream(ctx, modifiedReq, clientFormat)
	if err != nil {
		s.healthLogger.Error("forward stream failed", "model", actualModel, "provider", providerURL, "error", err.Error())
		return nil, "", "", err
	}

	return result, actualModel, providerURL, nil
}

// RecordStreamUsage records usage after stream completes
func (s *Service) RecordStreamUsage(usage *message.Usage, model string, providerURL string) {
	if s.usageSvc == nil || usage == nil {
		return
	}
	s.usageSvc.RecordWithModelAndProvider(usage, model, providerURL, true)
}

func (s *Service) prepareRequest(rawReq []byte, model string) (*flux.Client, string, string, []byte, error) {
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

	// Pass HTTP config if timeout is set
	var httpCfg *config.HTTPConfig
	if cfg.HTTP.Timeout != "" {
		httpCfg = &cfg.HTTP
	}

	newState := &serviceState{
		cfg:      cfg,
		aliasMap: cfg.AliasMap(),
		clients:  buildClients(userEndpoints, retryMax, httpCfg),
		retryMax: retryMax,
	}

	s.mu.Lock()
	s.state = newState
	s.mu.Unlock()

	s.healthLogger.Info("config reloaded", "models", len(newState.clients))
	return nil
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
