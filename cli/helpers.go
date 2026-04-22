package cli

import (
	"tokflux/tokrouter/config"
	"tokflux/tokrouter/router"
	"tokflux/tokrouter/usage"
)

func createRouter(cfg *config.Config) (*router.Service, error) {
	// Delegate to router layer - no cross-layer dependency
	return router.NewServiceFromConfig(cfg)
}

func queryStats(routerSvc *router.Service, filter usage.QueryFilter) ([]usage.StatRow, error) {
	return routerSvc.GetStats(filter)
}
