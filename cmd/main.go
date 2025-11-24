package main

import (
	"log"
	"net/http"

	"github.com/Lucascluz/reverse/internal/config"
	"github.com/Lucascluz/reverse/internal/proxy"
)

func main() {

	// Load configuration
	cfg, err := config.Load("./config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// Setup proxy
	p := proxy.NewProxy(cfg)

	// Start the server
	log.Println("Proxy server listening on :8080")
	if err := http.ListenAndServe(":8080", p); err != nil {
		log.Fatal(err)
	}
}
