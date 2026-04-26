package server

import (
	"context"
	"net/http"
	"sync"

	"golang.org/x/time/rate"

	"github.com/tokzone/tokrouter/config"
)

// RateLimiter implements global and per-provider rate limiting.
type RateLimiter struct {
	globalLimiter    *rate.Limiter
	providerLimiters sync.Map // providerURL -> *rate.Limiter
	perProviderRPS   rate.Limit
	perProviderBurst int
	enabled          bool
}

// NewRateLimiter creates a new rate limiter from config.
func NewRateLimiter(cfg config.RateLimitConfig) *RateLimiter {
	if !cfg.Enabled {
		return &RateLimiter{enabled: false}
	}

	globalRPS := rate.Limit(cfg.Global.RequestsPerSecond)
	if globalRPS == 0 {
		globalRPS = rate.Inf // No limit if not set
	}

	perProviderRPS := rate.Limit(cfg.PerProvider.RequestsPerSecond)
	if perProviderRPS == 0 {
		perProviderRPS = rate.Inf
	}

	return &RateLimiter{
		globalLimiter:    rate.NewLimiter(globalRPS, cfg.Global.Burst),
		perProviderRPS:   perProviderRPS,
		perProviderBurst: cfg.PerProvider.Burst,
		enabled:          true,
	}
}

// Wait waits for rate limit approval before processing a request.
// Returns error if the request exceeds the rate limit.
func (rl *RateLimiter) Wait(ctx context.Context, providerURL string) error {
	if !rl.enabled {
		return nil
	}

	// First check global limit
	if err := rl.globalLimiter.Wait(ctx); err != nil {
		return err
	}

	// Then check per-provider limit
	if rl.perProviderRPS == rate.Inf {
		return nil
	}

	limiter := rl.getProviderLimiter(providerURL)
	return limiter.Wait(ctx)
}

// getProviderLimiter gets or creates a rate limiter for a specific provider.
func (rl *RateLimiter) getProviderLimiter(providerURL string) *rate.Limiter {
	if v, ok := rl.providerLimiters.Load(providerURL); ok {
		return v.(*rate.Limiter)
	}

	limiter := rate.NewLimiter(rl.perProviderRPS, rl.perProviderBurst)
	rl.providerLimiters.Store(providerURL, limiter)
	return limiter
}

// Clear removes all per-provider limiters. Called during config reload.
func (rl *RateLimiter) Clear() {
	rl.providerLimiters.Clear()
}

// RateLimitMiddleware wraps a handler with rate limiting.
// Provider URL is extracted from request context if available.
func RateLimitMiddleware(rl *RateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get provider URL from context (set during routing)
		providerURL := ""
		if v := r.Context().Value(providerURLCtxKey); v != nil {
			providerURL = v.(string)
		}

		if err := rl.Wait(r.Context(), providerURL); err != nil {
			WriteErrorResponse(w, http.StatusTooManyRequests,
				NewErrorResponseWithCode("RATE_LIMITED", "Too many requests. Please try again later."))
			return
		}

		next(w, r)
	}
}

const providerURLCtxKey ctxKey = "provider_url"