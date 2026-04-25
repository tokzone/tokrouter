package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tokzone/tokrouter/config"
	"github.com/tokzone/tokrouter/router"

	"github.com/tokzone/fluxcore/provider"
)

// Server represents HTTP server
type Server struct {
	cfg        config.ServerConfig
	traceCfg   config.TraceConfig
	router     *router.Service
	server     *http.Server
	configPath string
}

// NewServer creates a new HTTP server
func NewServer(routerSvc *router.Service, traceCfg config.TraceConfig, configPath string) *Server {
	cfg := routerSvc.ServerConfig()
	mux := http.NewServeMux()

	// Wrap handlers with trace middleware
	traceMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return WithTraceID(next, traceCfg)
	}

	mux.HandleFunc("/v1/chat/completions", traceMiddleware(HandleRequest(routerSvc, provider.ProtocolOpenAI)))
	mux.HandleFunc("/v1/messages", traceMiddleware(HandleRequest(routerSvc, provider.ProtocolAnthropic)))
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
		cfg:        cfg,
		traceCfg:   traceCfg,
		router:     routerSvc,
		server:     httpServer,
		configPath: configPath,
	}
}

// Run starts the HTTP server
func (s *Server) Run() {
	done := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		for {
			sig := <-sigCh
			switch sig {
			case syscall.SIGHUP:
				// Hot reload
				fmt.Println("Received SIGHUP, reloading config...")
				cfg, err := config.Load(s.configPath)
				if err != nil {
					fmt.Printf("Error loading config: %v\n", err)
					continue
				}
				if err := s.router.Reload(cfg); err != nil {
					fmt.Printf("Error applying config: %v\n", err)
					continue
				}
				fmt.Println("Config reloaded successfully")
			case syscall.SIGINT, syscall.SIGTERM:
				fmt.Println("\nShutting down...")
				// Graceful shutdown with timeout
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer shutdownCancel()
				if err := s.server.Shutdown(shutdownCtx); err != nil {
					fmt.Printf("Shutdown error: %v\n", err)
				}
				close(done)
				return
			}
		}
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

	<-done // Wait for shutdown signal handler
	s.router.Close()
}