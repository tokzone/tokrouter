package router

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/tokzone/fluxcore"
	"github.com/tokzone/fluxcore/errors"
	"github.com/tokzone/fluxcore/message"

	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/usage"
)

type Router interface {
	ForwardOpenAI(ctx context.Context, body []byte, model string) ([]byte, *message.Usage, error)
	ForwardAnthropic(ctx context.Context, body []byte, model string) ([]byte, *message.Usage, error)
	ForwardStreamOpenAI(ctx context.Context, body []byte, model string) (*fluxcore.StreamResult, string, string, error)
	ForwardStreamAnthropic(ctx context.Context, body []byte, model string) (*fluxcore.StreamResult, string, string, error)
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
	ctx          *routerCtx
	oaRouter     *fluxcore.Router
	anthRouter   *fluxcore.Router
	svcEPs       map[string]*fluxcore.ServiceEndpoint
	routeRepo    *fluxcore.RouteRepository
	usageSvc     *usage.Service
	healthLogger *slog.Logger
}

var _ Router = (*router)(nil)

// routerCtx holds the pre-prepared routing context, swapped atomically on reload.
type routerCtx struct {
	oaTables   map[fluxcore.Model]*fluxcore.RouteTable
	anthTables map[fluxcore.Model]*fluxcore.RouteTable
	aliases    map[string]string
	cfg        *config.Config
	retryMax   int
}

// ProviderStatus represents runtime status of a provider.
type ProviderStatus struct {
	Name     string
	Protocol string
	Healthy  bool
	Models   []ModelStatus
}

// ModelStatus represents status of a model.
type ModelStatus struct {
	Name    string
	Healthy bool
}

// New creates a Router from config.
func New(cfg *config.Config, usageSvc *usage.Service) (Router, error) {
	svcEPs := make(map[string]*fluxcore.ServiceEndpoint)
	repo := fluxcore.NewRouteRepository()

	httpClient := buildHTTPClient(&cfg.HTTP)

	oaRouter := fluxcore.NewRouter(fluxcore.ProtocolOpenAI, fluxcore.WithHTTPClient(httpClient))
	anthRouter := fluxcore.NewRouter(fluxcore.ProtocolAnthropic, fluxcore.WithHTTPClient(httpClient))

	ctx := buildRouterCtx(cfg, svcEPs, repo)

	return &router{
		ctx:          ctx,
		oaRouter:     oaRouter,
		anthRouter:   anthRouter,
		svcEPs:       svcEPs,
		routeRepo:    repo,
		usageSvc:     usageSvc,
		healthLogger: slog.Default().With("component", "router"),
	}, nil
}

// NewFromConfig creates a Router from config (factory method).
func NewFromConfig(cfg *config.Config) (Router, error) {
	var usageSvc *usage.Service
	if cfg.Stats.Enabled {
		store, err := usage.NewStore(cfg.Stats.DBPath)
		if err != nil {
			return nil, err
		}
		usageSvc = usage.NewService(store)
	}

	return New(cfg, usageSvc)
}

func buildRouterCtx(cfg *config.Config, svcEPs map[string]*fluxcore.ServiceEndpoint, repo *fluxcore.RouteRepository) *routerCtx {
	routes := cfg.ToRoutes(svcEPs, func(desc fluxcore.RouteDesc) *fluxcore.Route {
		return repo.FindOrCreate(desc.IdentityKey(), func() *fluxcore.Route {
			return fluxcore.NewRoute(desc)
		})
	})

	oaTables := make(map[fluxcore.Model]*fluxcore.RouteTable)
	anthTables := make(map[fluxcore.Model]*fluxcore.RouteTable)

	modelRoutes := make(map[fluxcore.Model][]*fluxcore.Route)
	for _, route := range routes {
		m := route.Desc().Model
		modelRoutes[m] = append(modelRoutes[m], route)
	}

	for model, rts := range modelRoutes {
		oaTables[model] = fluxcore.NewRouteTable(rts, fluxcore.ProtocolOpenAI)
		anthTables[model] = fluxcore.NewRouteTable(rts, fluxcore.ProtocolAnthropic)
	}

	retryMax := cfg.Router.Retry.MaxRetries

	return &routerCtx{
		oaTables:   oaTables,
		anthTables: anthTables,
		aliases:    cfg.AliasMap(),
		cfg:        cfg,
		retryMax:   retryMax,
	}
}

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

func (s *router) ForwardOpenAI(ctx context.Context, body []byte, model string) ([]byte, *message.Usage, error) {
	return s.forward(ctx, body, model, s.ctx.oaTables, s.oaRouter)
}

func (s *router) ForwardAnthropic(ctx context.Context, body []byte, model string) ([]byte, *message.Usage, error) {
	return s.forward(ctx, body, model, s.ctx.anthTables, s.anthRouter)
}

func (s *router) forward(ctx context.Context, body []byte, model string, tables map[fluxcore.Model]*fluxcore.RouteTable, r *fluxcore.Router) ([]byte, *message.Usage, error) {
	model, body = resolveAlias(model, body, s.ctx.aliases)

	table, ok := tables[fluxcore.Model(model)]
	if !ok {
		return nil, nil, errors.Wrap(errors.CodeNoEndpoint, fmt.Sprintf("no endpoint for model: %s", model), nil)
	}

	s.healthLogger.Debug("forward starting", "model", model)

	route, resp, usage, err := r.Execute(ctx, table, body, s.ctx.retryMax)
	if err != nil {
		if route != nil {
			s.healthLogger.Error("forward failed", "model", model, "provider", route.SvcEP().Service().Name, "error", err.Error())
		} else {
			s.healthLogger.Error("forward failed", "model", model, "error", err.Error())
		}
		return nil, nil, err
	}

	if s.usageSvc != nil && usage != nil {
		s.usageSvc.RecordWithModelAndProvider(usage, model, route.SvcEP().Service().Name, false)
	}

	return resp, usage, nil
}

func (s *router) ForwardStreamOpenAI(ctx context.Context, body []byte, model string) (*fluxcore.StreamResult, string, string, error) {
	return s.forwardStream(ctx, body, model, s.ctx.oaTables, s.oaRouter)
}

func (s *router) ForwardStreamAnthropic(ctx context.Context, body []byte, model string) (*fluxcore.StreamResult, string, string, error) {
	return s.forwardStream(ctx, body, model, s.ctx.anthTables, s.anthRouter)
}

func (s *router) forwardStream(ctx context.Context, body []byte, model string, tables map[fluxcore.Model]*fluxcore.RouteTable, r *fluxcore.Router) (*fluxcore.StreamResult, string, string, error) {
	model, body = resolveAlias(model, body, s.ctx.aliases)

	table, ok := tables[fluxcore.Model(model)]
	if !ok {
		return nil, "", "", errors.Wrap(errors.CodeNoEndpoint, fmt.Sprintf("no endpoint for model: %s", model), nil)
	}

	s.healthLogger.Debug("forward stream starting", "model", model)

	route, result, err := r.ExecuteStream(ctx, table, body, s.ctx.retryMax)
	if err != nil {
		if route != nil {
			s.healthLogger.Error("forward stream failed", "model", model, "provider", route.SvcEP().Service().Name, "error", err.Error())
		} else {
			s.healthLogger.Error("forward stream failed", "model", model, "error", err.Error())
		}
		return nil, "", "", err
	}

	providerURL := route.SvcEP().Service().Name
	return result, model, providerURL, nil
}

// RecordStreamUsage records usage after stream completes.
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
	grouped := s.routeRepo.RoutesByServiceEndpoint()
	statuses := make([]ProviderStatus, 0, len(grouped))
	for name, routes := range grouped {
		proto := servicePrimaryProtocol(routes[0].SvcEP().Service()).String()
		ps := ProviderStatus{Name: name, Protocol: proto}
		for _, route := range routes {
			available := route.IsAvailable()
			ps.Models = append(ps.Models, ModelStatus{
				Name:    string(route.Desc().Model),
				Healthy: available,
			})
			if available && route.SvcEP().IsAvailable() {
				ps.Healthy = true
			}
		}
		statuses = append(statuses, ps)
	}
	return statuses
}

// Stats returns usage statistics.
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

	newCtx := buildRouterCtx(cfg, s.svcEPs, s.routeRepo)

	s.mu.Lock()
	s.ctx = newCtx
	s.mu.Unlock()

	s.healthLogger.Info("config reloaded", "route_tables", len(newCtx.oaTables))
	return nil
}

func rewriteModelInRequest(rawReq []byte, newModel string) []byte {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(rawReq, &m); err != nil {
		return rawReq
	}
	modelJSON, err := json.Marshal(newModel)
	if err != nil {
		return rawReq
	}
	m["model"] = modelJSON
	result, err := json.Marshal(m)
	if err != nil {
		return rawReq
	}
	return result
}

func buildHTTPClient(cfg *config.HTTPConfig) *http.Client {
	timeout, _ := time.ParseDuration(cfg.Timeout)
	idleTimeout, _ := time.ParseDuration(cfg.IdleConnTimeout)
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	if idleTimeout == 0 {
		idleTimeout = 90 * time.Second
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        cfg.MaxIdleConns,
			MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
			IdleConnTimeout:     idleTimeout,
		},
	}
}

func servicePrimaryProtocol(svc fluxcore.Service) fluxcore.Protocol {
	for _, p := range fluxcore.ProtocolPriority() {
		if _, ok := svc.BaseURLs[p]; ok {
			return p
		}
	}
	return fluxcore.ProtocolOpenAI
}
