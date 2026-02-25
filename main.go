package main

import (
	"flag"
	"fmt"
	"log"
	"mk-addrlist-generator/pkg/api"
	"mk-addrlist-generator/pkg/config"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	listenAddr := flag.String("listen", ":8080", "Address to listen on")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := config.ValidateConfig(cfg); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Create and start HTTP server
	server := api.NewServer(cfg)
	go func() {
		if err := server.Start(*listenAddr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	fmt.Printf("Server started on %s\n", *listenAddr)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down server...")
	if err := server.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}
