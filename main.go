package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"mk-addrlist-generator/pkg/api"
	"mk-addrlist-generator/pkg/config"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	port := flag.String("port", "8080", "Port to listen on")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Setup HTTP server
	r := gin.Default()
	server := api.NewServer(cfg)
	server.SetupRoutes(r)

	// Start server
	log.Printf("Starting server on port %s", *port)
	if err := r.Run(fmt.Sprintf(":%s", *port)); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
