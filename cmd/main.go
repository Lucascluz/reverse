package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Lucascluz/reverse/internal/config"
	"github.com/Lucascluz/reverse/internal/observability"
	"github.com/Lucascluz/reverse/internal/proxy"
)

const (
	shutdownTimeout = 10 * time.Second
	logPrefix       = "[reverse]"
)

func main() {
	logger := log.New(os.Stdout, logPrefix+" ", log.LstdFlags|log.Lmicroseconds)

	// Initialize application
	app, err := initialize(logger)
	if err != nil {
		logger.Fatalf("initialization failed: %v", err)
	}

	// Start servers
	if err := app.start(logger); err != nil {
		logger.Fatalf("failed to start servers: %v", err)
	}

	// Wait for shutdown signal
	app.waitForShutdown(logger)

	// Graceful shutdown
	if err := app.shutdown(logger); err != nil {
		logger.Fatalf("shutdown error: %v", err)
	}
}

// app represents the running application with all its components
type app struct {
	config         *config.Config
	observability  *observability.Observability
	proxy          *proxy.Proxy
	proxySrv       *http.Server
	probeSrv       *http.Server
	shutdownSignal chan os.Signal
	serverErrors   chan error
	serverWg       sync.WaitGroup
}

// initialize sets up all application components and returns an app instance
func initialize(logger *log.Logger) (*app, error) {
	logger.Println("initializing application...")

	// Load configuration

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "/etc/config/config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	logger.Println("configuration loaded successfully")

	// Initialize proxy with config
	setup, err := proxy.NewSetup(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy setup: %w", err)
	}

	// Get proxy instance
	p := setup.Proxy()

	// Build the complete handler with middleware
	handler, err := setup.Handler()
	if err != nil {
		return nil, fmt.Errorf("failed to build handler: %w", err)
	}
	logger.Println("proxy handler configured")

	// Get backends from load balancer for health checking
	backendInterfaces := make([]observability.HealthAware, 0)
	for _, b := range p.LoadBalancer().Pool().Backends() {
		backendInterfaces = append(backendInterfaces, b)
	}

	// Create observability hub
	obs, err := observability.NewObservability(cfg, p, backendInterfaces)
	if err != nil {
		return nil, fmt.Errorf("failed to create observability: %w", err)
	}
	logger.Println("observability hub initialized")

	// Start health checks
	if err := obs.StartHealthChecks(backendInterfaces, func() {
		// Update load balancer's ready flag based on current pool health status
		p.LoadBalancer().SetReady(p.LoadBalancer().Pool().IsReady())
	}); err != nil {
		return nil, fmt.Errorf("failed to start health checks: %w", err)
	}
	logger.Println("health checks started")

	// Setup proxy servers
	proxySrv := createProxyServer(cfg, handler)
	probeSrv := createProbeServer(cfg, obs.Probe())

	app := &app{
		config:         cfg,
		observability:  obs,
		proxy:          p,
		proxySrv:       proxySrv,
		probeSrv:       probeSrv,
		shutdownSignal: make(chan os.Signal, 1),
		serverErrors:   make(chan error, 2),
	}

	logger.Println("application initialized successfully")
	return app, nil
}

// start begins listening on proxy and probe servers
func (a *app) start(logger *log.Logger) error {
	a.serverWg.Add(2)

	// Start proxy server
	go func() {
		defer a.serverWg.Done()
		logger.Printf("proxy server listening on %s", a.proxySrv.Addr)
		if err := a.proxySrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.serverErrors <- fmt.Errorf("proxy server error: %w", err)
		}
	}()

	// Start probe server
	go func() {
		defer a.serverWg.Done()
		logger.Printf("probe server listening on %s", a.probeSrv.Addr)
		if err := a.probeSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.serverErrors <- fmt.Errorf("probe server error: %w", err)
		}
	}()

	// Check for immediate startup errors
	select {
	case err := <-a.serverErrors:
		return err
	case <-time.After(100 * time.Millisecond):
		// Give servers time to start; if they crash, we'll catch it on shutdown
		return nil
	}
}

// waitForShutdown blocks until a shutdown signal is received
func (a *app) waitForShutdown(logger *log.Logger) {
	signal.Notify(a.shutdownSignal, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-a.shutdownSignal:
		logger.Printf("received signal: %v, shutting down gracefully", sig)
	case err := <-a.serverErrors:
		logger.Printf("server error: %v, shutting down", err)
	}
}

// shutdown gracefully stops all servers and cleans up resources
func (a *app) shutdown(logger *log.Logger) error {
	logger.Println("graceful shutdown initiated")

	// Step 1: Mark proxy as not ready to prevent new requests
	// This signals orchestrators to stop sending traffic
	logger.Println("marking proxy as not ready (draining connections)")
	a.proxy.SetReady(false)

	// Step 2: Stop observability components (health checker)
	logger.Println("stopping observability components")
	if err := a.observability.Stop(); err != nil {
		logger.Printf("error stopping observability: %v", err)
	}

	// Step 3: Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Step 4: Shutdown servers concurrently
	logger.Println("shutting down HTTP servers")
	shutdownErrs := make(chan error, 2)

	go func() {
		if err := a.probeSrv.Shutdown(ctx); err != nil {
			shutdownErrs <- fmt.Errorf("probe server shutdown: %w", err)
		}
	}()

	go func() {
		if err := a.proxySrv.Shutdown(ctx); err != nil {
			shutdownErrs <- fmt.Errorf("proxy server shutdown: %w", err)
		}
	}()

	// Step 5: Wait for servers to finish
	done := make(chan struct{})
	go func() {
		a.serverWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Println("servers shut down successfully")
	case <-time.After(shutdownTimeout):
		logger.Println("warning: server shutdown timeout exceeded")
	}

	// Step 6: Collect any errors that occurred
	close(shutdownErrs)
	for err := range shutdownErrs {
		if err != nil {
			logger.Printf("shutdown error: %v", err)
		}
	}

	return nil
}

// createProxyServer configures the main proxy HTTP server
func createProxyServer(cfg *config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         ":" + cfg.Proxy.Port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  90 * time.Second,
	}
}

// createProbeServer configures the probe/health check HTTP server
func createProbeServer(cfg *config.Config, probe *observability.Probe) *http.Server {
	return &http.Server{
		Addr:         ":" + cfg.Proxy.ProbePort,
		Handler:      probe.Handler(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
	}
}
