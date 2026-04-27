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

// Router defines the interface for routing operations.
type Router interface {
	ForwardOpenAI(ctx context.Context, body []byte, model string) ([]byte, *message.Usage, error)
	ForwardAnthropic(ctx context.Context, body []byte, model string) ([]byte, *message.Usage, error)
	ForwardStreamOpenAI(ctx context.Context, body []byte, model string) (*flux.StreamResult, string, string, error)
	ForwardStreamAnthropic(ctx context.Context, body []byte, model string) (*flux.StreamResult, string, string, error)
	RecordStreamUsage(usage *message.Usage, model string, providerURL string)
	ProviderStatuses() []ProviderStatus
	Stats(filter usage.QueryFilter) ([]usage.StatRow, error)
	ServerConfig() config.ServerConfig
	Reload(cfg *config.Config) error
	Close() error
}

// router is the unexported implementation of Router.
type router struct {
	mu           sync.RWMutex
	ctx          *routerCtx // Atomic state container (swapped on reload)
	usageSvc     *usage.Service
	healthLogger *slog.Logger
}

// Ensure router implements Router at compile time.
var _ Router = (*router)(nil)

// routerCtx holds the pre-prepared routing context, swapped atomically on reload.
type routerCtx struct {
	openAIDoFuncs       map[string]flux.DoFunc
	anthropicDoFuncs    map[string]flux.DoFunc
	openAIStreamDoFuncs map[string]flux.StreamDoFunc
	anthropicStreamDoFuncs map[string]flux.StreamDoFunc
	aliases             map[string]string
	cfg                 *config.Config
	retryMax            int
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

// New creates a Router from user endpoints.
func New(userEndpoints []*flux.UserEndpoint, usageSvc *usage.Service, retryMax int, httpCfg *config.HTTPConfig, aliases map[string]string, cfg *config.Config) Router {
	if aliases == nil {
		aliases = make(map[string]string)
	}
	openAIDoFuncs, anthropicDoFuncs, openAIStreamDoFuncs, anthropicStreamDoFuncs := buildDoFuncs(userEndpoints, retryMax, httpCfg)
	return &router{
		ctx: &routerCtx{
			openAIDoFuncs:          openAIDoFuncs,
			anthropicDoFuncs:       anthropicDoFuncs,
			openAIStreamDoFuncs:    openAIStreamDoFuncs,
			anthropicStreamDoFuncs: anthropicStreamDoFuncs,
			aliases:                aliases,
			cfg:                    cfg,
			retryMax:               retryMax,
		},
		usageSvc:     usageSvc,
		healthLogger: slog.Default().With("component", "router"),
	}
}

// buildDoFuncs groups user endpoints by model, creates a flux.Client for each model,
// and pre-generates DoFunc instances for both OpenAI and Anthropic input protocols.
func buildDoFuncs(userEndpoints []*flux.UserEndpoint, retryMax int, httpCfg *config.HTTPConfig) (map[string]flux.DoFunc, map[string]flux.DoFunc, map[string]flux.StreamDoFunc, map[string]flux.StreamDoFunc) {
	modelEndpoints := make(map[string][]*flux.UserEndpoint)
	for _, ue := range userEndpoints {
		modelEndpoints[ue.Model()] = append(modelEndpoints[ue.Model()], ue)
	}

	var httpOpts []flux.Option
	if httpCfg != nil && httpCfg.Timeout != "" {
		httpOpts = append(httpOpts, flux.WithHTTPClient(buildHTTPClient(httpCfg)))
	}

	openAIDoFuncs := make(map[string]flux.DoFunc)
	anthropicDoFuncs := make(map[string]flux.DoFunc)
	openAIStreamDoFuncs := make(map[string]flux.StreamDoFunc)
	anthropicStreamDoFuncs := make(map[string]flux.StreamDoFunc)

	for model, ues := range modelEndpoints {
		opts := append([]flux.Option{flux.WithRetryMax(retryMax)}, httpOpts...)
		client := flux.NewClient(ues, opts...)
		openAIDoFuncs[model] = flux.DoFuncGen(client, provider.ProtocolOpenAI)
		anthropicDoFuncs[model] = flux.DoFuncGen(client, provider.ProtocolAnthropic)
		openAIStreamDoFuncs[model] = flux.StreamDoFuncGen(client, provider.ProtocolOpenAI)
		anthropicStreamDoFuncs[model] = flux.StreamDoFuncGen(client, provider.ProtocolAnthropic)
	}
	return openAIDoFuncs, anthropicDoFuncs, openAIStreamDoFuncs, anthropicStreamDoFuncs
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

// NewFromConfig creates a Router from config (factory method).
func NewFromConfig(cfg *config.Config) (Router, error) {
	userEndpoints := cfg.ToUserEndpoints()

	var usageSvc *usage.Service
	if cfg.Stats.Enabled {
		store, err := usage.NewStore(cfg.Stats.DBPath)
		if err != nil {
			return nil, err
		}
		usageSvc = usage.NewService(store)
	}

	var httpCfg *config.HTTPConfig
	if cfg.HTTP.Timeout != "" {
		httpCfg = &cfg.HTTP
	}

	svc := New(userEndpoints, usageSvc, cfg.Router.Retry.MaxRetries, httpCfg, cfg.AliasMap(), cfg)

	return svc, nil
}

// ServerConfig returns server configuration.
func (s *router) ServerConfig() config.ServerConfig {
	s.mu.RLock()
	cfg := s.ctx.cfg
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

// ForwardOpenAI forwards a non-streaming request received via the OpenAI endpoint.
func (s *router) ForwardOpenAI(ctx context.Context, body []byte, model string) ([]byte, *message.Usage, error) {
	return s.forward(ctx, body, model, s.ctx.openAIDoFuncs)
}

// ForwardAnthropic forwards a non-streaming request received via the Anthropic endpoint.
func (s *router) ForwardAnthropic(ctx context.Context, body []byte, model string) ([]byte, *message.Usage, error) {
	return s.forward(ctx, body, model, s.ctx.anthropicDoFuncs)
}

func (s *router) forward(ctx context.Context, body []byte, model string, doFuncs map[string]flux.DoFunc) ([]byte, *message.Usage, error) {
	model, body = resolveAlias(model, body, s.ctx.aliases)

	doFunc, ok := doFuncs[model]
	if !ok {
		return nil, nil, errors.Wrap(errors.CodeNoEndpoint, fmt.Sprintf("no endpoint for model: %s", model), nil)
	}

	s.healthLogger.Debug("forward starting", "model", model)

	resp, usage, providerURL, err := doFunc(ctx, body)
	if err != nil {
		s.healthLogger.Error("forward failed", "model", model, "provider", providerURL, "error", err.Error())
		return nil, nil, err
	}

	if s.usageSvc != nil && usage != nil {
		s.usageSvc.RecordWithModelAndProvider(usage, model, providerURL, false)
	}

	return resp, usage, nil
}

// ForwardStreamOpenAI forwards a streaming request received via the OpenAI endpoint.
func (s *router) ForwardStreamOpenAI(ctx context.Context, body []byte, model string) (*flux.StreamResult, string, string, error) {
	return s.forwardStream(ctx, body, model, s.ctx.openAIStreamDoFuncs)
}

// ForwardStreamAnthropic forwards a streaming request received via the Anthropic endpoint.
func (s *router) ForwardStreamAnthropic(ctx context.Context, body []byte, model string) (*flux.StreamResult, string, string, error) {
	return s.forwardStream(ctx, body, model, s.ctx.anthropicStreamDoFuncs)
}

func (s *router) forwardStream(ctx context.Context, body []byte, model string, streamDoFuncs map[string]flux.StreamDoFunc) (*flux.StreamResult, string, string, error) {
	model, body = resolveAlias(model, body, s.ctx.aliases)

	streamDoFunc, ok := streamDoFuncs[model]
	if !ok {
		return nil, "", "", errors.Wrap(errors.CodeNoEndpoint, fmt.Sprintf("no endpoint for model: %s", model), nil)
	}

	s.healthLogger.Debug("forward stream starting", "model", model)

	result, _, providerURL, err := streamDoFunc(ctx, body)
	if err != nil {
		s.healthLogger.Error("forward stream failed", "model", model, "provider", providerURL, "error", err.Error())
		return nil, "", "", err
	}

	return result, model, providerURL, nil
}

// RecordStreamUsage records usage after stream completes
func (s *router) RecordStreamUsage(usage *message.Usage, model string, providerURL string) {
	if s.usageSvc == nil || usage == nil {
		return
	}
	s.usageSvc.RecordWithModelAndProvider(usage, model, providerURL, true)
}

// resolveAlias resolves model aliases and rewrites the model field in the request body.
func resolveAlias(model string, body []byte, aliases map[string]string) (string, []byte) {
	if actualModel, ok := aliases[model]; ok {
		model = actualModel
		body = rewriteModelInRequest(body, model)
	}
	return model, body
}

func (s *router) ProviderStatuses() []ProviderStatus {
	endpoints := endpoint.GlobalRegistry().GetAll()

	providerMap := make(map[string]*ProviderStatus)
	for _, ep := range endpoints {
		prov := ep.Provider()
		providerKey := prov.BaseURL
		pStatus, ok := providerMap[providerKey]
		if !ok {
			pStatus = &ProviderStatus{
				Name:     providerKey,
				Protocol: ep.Protocol().String(),
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
func (s *router) Stats(filter usage.QueryFilter) ([]usage.StatRow, error) {
	if s.usageSvc == nil {
		return nil, usage.ErrDisabled
	}
	return s.usageSvc.Query(filter)
}

func (s *router) Close() error {
	if s.usageSvc == nil {
		return nil
	}
	return s.usageSvc.Close()
}

func (s *router) Reload(cfg *config.Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	endpoint.GlobalRegistry().Clear()

	userEndpoints := cfg.ToUserEndpoints()

	s.mu.RLock()
	retryMax := s.ctx.retryMax
	s.mu.RUnlock()

	// Pass HTTP config if timeout is set
	var httpCfg *config.HTTPConfig
	if cfg.HTTP.Timeout != "" {
		httpCfg = &cfg.HTTP
	}

	openAIDoFuncs, anthropicDoFuncs, openAIStreamDoFuncs, anthropicStreamDoFuncs := buildDoFuncs(userEndpoints, retryMax, httpCfg)

	s.mu.Lock()
	s.ctx = &routerCtx{
		openAIDoFuncs:          openAIDoFuncs,
		anthropicDoFuncs:       anthropicDoFuncs,
		openAIStreamDoFuncs:    openAIStreamDoFuncs,
		anthropicStreamDoFuncs: anthropicStreamDoFuncs,
		aliases:                cfg.AliasMap(),
		cfg:                    cfg,
		retryMax:               retryMax,
	}
	s.mu.Unlock()

	s.healthLogger.Info("config reloaded", "models", len(openAIDoFuncs))
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
