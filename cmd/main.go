package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Lucascluz/reverse/internal/config"
	"github.com/Lucascluz/reverse/internal/proxy"
)

func main() {
	// Load configuration
	cfg, err := config.Load("./config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// Initialize proxy with config
	setup, err := proxy.NewSetup(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Build the complete handler with middleware
	handler, err := setup.Handler()
	if err != nil {
		log.Fatal(err)
	}

	// Get the proxy instance
	p := setup.Proxy()

	// Setup servers
	proxySrv := &http.Server{
		Addr:         ":" + p.Port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  90 * time.Second,
	}

	probeSrv := &http.Server{
		Addr:         ":" + p.ProbePort,
		Handler:      p.ProbeMux(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
	}

	// Run servers
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		log.Printf("Proxy server listening on %s", proxySrv.Addr)
		if err := proxySrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("proxy server error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		log.Printf("Probe server listening on %s", probeSrv.Addr)
		if err := probeSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("probe server error: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("Gracefully shutting down")

	p.SetReady(false)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := probeSrv.Shutdown(ctx); err != nil {
		log.Printf("probe server shutdown error: %v", err)
	}

	if err := proxySrv.Shutdown(ctx); err != nil {
		log.Printf("proxy server shutdown error: %v", err)
	}

	wg.Wait()
	log.Print("Servers stopped")
}
