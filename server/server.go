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
)

// Server represents HTTP server
type Server struct {
	cfg        config.ServerConfig
	traceCfg   config.TraceConfig
	router     router.Router
	server     *http.Server
	configPath string
	shutdownCh chan struct{}
}

// NewServer creates a new HTTP server
func NewServer(routerSvc router.Router, traceCfg config.TraceConfig, configPath string) *Server {
	cfg := routerSvc.ServerConfig()
	mux := http.NewServeMux()

	// Wrap handlers with trace middleware
	traceMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return WithTraceID(next, traceCfg)
	}

	mux.HandleFunc("/v1/chat/completions", traceMiddleware(HandleOpenAI(routerSvc)))
	mux.HandleFunc("/v1/messages", traceMiddleware(HandleAnthropic(routerSvc)))
	mux.HandleFunc("/v1/responses", traceMiddleware(HandleResponses(routerSvc)))
	mux.HandleFunc("/v1/models", HandleModels())
	mux.HandleFunc("/v1/models/", HandleModels())
	mux.HandleFunc("/status", traceMiddleware(HandleStatus(routerSvc)))
	mux.HandleFunc("/health", traceMiddleware(HandleHealth(routerSvc)))

	// OpenAPI documentation endpoints
	mux.HandleFunc("/openapi.yaml", HandleOpenAPISpec)
	mux.HandleFunc("/docs", HandleSwaggerUI)
	mux.HandleFunc("/docs/", HandleSwaggerUI)

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

	srv := &Server{
		cfg:        cfg,
		traceCfg:   traceCfg,
		router:     routerSvc,
		server:     httpServer,
		configPath: configPath,
		shutdownCh: make(chan struct{}),
	}

	// Register shutdown endpoint after server struct is created
	mux.HandleFunc("/shutdown", HandleShutdown(srv))

	return srv
}

// shutdown triggers graceful shutdown.
func (s *Server) shutdown() {
	close(s.shutdownCh)
}

// gracefulShutdown performs a graceful HTTP server shutdown with a 30s timeout.
func (s *Server) gracefulShutdown(done chan struct{}) {
	fmt.Println("\nShutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := s.server.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("Shutdown error: %v\n", err)
	}
	close(done)
}

// HandleShutdown returns an HTTP handler that triggers server shutdown.
func HandleShutdown(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		go s.shutdown()
	}
}

// Run starts the HTTP server
func (s *Server) Run() {
	done := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		defer signal.Stop(sigCh)
		for {
			select {
			case sig := <-sigCh:
				switch sig {
				case syscall.SIGHUP:
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
					s.gracefulShutdown(done)
					return
				}
			case <-s.shutdownCh:
				s.gracefulShutdown(done)
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
