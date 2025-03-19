package main

import (
	"fmt"
	"log"
	"net/http"

	"gihan9a/braidmock/internal/config"
	"gihan9a/braidmock/internal/server"
	"gihan9a/braidmock/internal/tls"
)

func main() {
	// Parse command line flags and get configuration
	cfg, err := config.ParseFlags()
	if err != nil {
		log.Fatalf("Error parsing configuration: %v", err)
	}

	// Set up the TLS certificate if needed
	if cfg.TLS.Enabled && cfg.TLS.GenerateCert {
		if err := tls.EnsureCertificate(cfg.TLS.CertFile, cfg.TLS.KeyFile); err != nil {
			log.Fatalf("Failed to set up TLS certificate: %v", err)
		}
	}

	// Create server
	braidServer, err := server.NewBraidMockServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer braidServer.Close()

	// Set up watchers for the directory
	if err := braidServer.SetupWatchers(); err != nil {
		log.Fatalf("Failed to set up file watchers: %v", err)
	}

	// Set up HTTP router
	router := braidServer.SetupRoutes()

	// Start server with or without TLS
	addr := fmt.Sprintf(":%d", cfg.Port)
	if cfg.TLS.Enabled {
		log.Printf("Braid mock server running at https://localhost%s", addr)
		log.Printf("Serving .braid files from directory: %s", cfg.RootDir)
		log.Printf("Using TLS certificate: %s", cfg.TLS.CertFile)
		log.Printf("Using TLS key: %s", cfg.TLS.KeyFile)
		log.Fatal(http.ListenAndServeTLS(addr, cfg.TLS.CertFile, cfg.TLS.KeyFile, router))
	} else {
		log.Printf("Braid mock server running at http://localhost%s", addr)
		log.Printf("Serving .braid files from directory: %s", cfg.RootDir)
		log.Fatal(http.ListenAndServe(addr, router))
	}
}
