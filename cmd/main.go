package main

import (
	"log"
	"net/http"

	"github.com/Lucascluz/reverse/internal/proxy"
)

func main() {

	// Setup proxy
	p := proxy.NewProxy()

	// Start the server
	log.Println("Proxy server listening on :8080")
	if err := http.ListenAndServe(":8080", p.ServeHTTP()); err != nil {
		log.Fatal(err)
	}
}
