package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"tokflux/tokrouter/config"
	"tokflux/tokrouter/router"

	"github.com/tokzone/fluxcore/routing"
)

// Server represents HTTP server
type Server struct {
	cfg      config.ServerConfig
	traceCfg config.TraceConfig
	router   *router.Service
	server   *http.Server
}

// NewServer creates a new HTTP server
func NewServer(routerSvc *router.Service, traceCfg config.TraceConfig) *Server {
	cfg := routerSvc.ServerConfig()
	mux := http.NewServeMux()

	// Wrap handlers with trace middleware
	traceMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return WithTraceID(next, traceCfg)
	}

	mux.HandleFunc("/v1/chat/completions", traceMiddleware(HandleRequest(routerSvc, routing.ProtocolOpenAI)))
	mux.HandleFunc("/v1/messages", traceMiddleware(HandleRequest(routerSvc, routing.ProtocolAnthropic)))
	mux.HandleFunc("/status", traceMiddleware(HandleStatus(routerSvc)))
	mux.HandleFunc("/health", traceMiddleware(HandleHealth(routerSvc)))

	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler: mux,
	}

	// Configure TLS if certificates provided
	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(cfg.TLSCert, cfg.TLSKey)
		if err != nil {
			fmt.Printf("Error loading TLS certificates: %v\n", err)
			os.Exit(1)
		}
		httpServer.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
	}

	return &Server{
		cfg:      cfg,
		traceCfg: traceCfg,
		router:   routerSvc,
		server:   httpServer,
	}
}

// Run starts the HTTP server
func (s *Server) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Println("\nShutting down...")
		s.server.Shutdown(ctx)
		cancel()
	}()

	// Set log level from config
	SetLogLevel(s.cfg.LogLevel)

	fmt.Printf("tokrouter listening on %s:%d\n", s.cfg.Host, s.cfg.Port)
	if s.cfg.TLSCert != "" && s.cfg.TLSKey != "" {
		fmt.Println("TLS enabled")
		if err := s.server.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
			os.Exit(1)
		}
	}

	s.router.Close()
}
