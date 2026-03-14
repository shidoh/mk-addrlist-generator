package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mk-addrlist-generator/pkg/api"
	"mk-addrlist-generator/pkg/config"

	"github.com/spf13/cobra"
)

var (
	listenAddr      string
	shutdownTimeout time.Duration
	enableMetrics   bool
	enableHealth    bool
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	Long: `Start the HTTP server that provides address lists via HTTP API.

The server provides the following endpoints:
  /health, /healthz     - Health check endpoint
  /ready, /readyz       - Readiness check endpoint
  /live, /livez         - Liveness check endpoint
  /metrics              - Prometheus metrics endpoint
  /lists/all            - Get all address lists
  /list/<name>          - Get a specific list by name
  /lists                - List all available list names
  /stats                - Get statistics for all lists

Query parameters:
  format=<format>       - Output format (mikrotik, plain, json, nftables)
  aggregate=true        - Enable CIDR aggregation
  deduplicate=true      - Enable deduplication (default: true)`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringVarP(&listenAddr, "listen", "l", ":8080", "Address to listen on")
	serveCmd.Flags().DurationVar(&shutdownTimeout, "shutdown-timeout", 30*time.Second, "Graceful shutdown timeout")
	serveCmd.Flags().BoolVar(&enableMetrics, "metrics", true, "Enable Prometheus metrics endpoint")
	serveCmd.Flags().BoolVar(&enableHealth, "health", true, "Enable health check endpoints")
}

func runServe(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := config.ValidateConfig(cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Setup server configuration
	serverCfg := api.ServerConfig{
		EnableMetrics: enableMetrics,
		EnableHealth:  enableHealth,
		LogLevel:      slog.LevelInfo,
	}
	if verbose {
		serverCfg.LogLevel = slog.LevelDebug
	}

	// Set version info
	api.Version = Version
	api.BuildTime = BuildTime
	api.GitCommit = GitCommit

	// Create and start HTTP server
	server := api.NewServerWithConfig(cfg, serverCfg)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := server.Start(listenAddr); err != nil {
			errChan <- err
		}
	}()

	fmt.Printf("Server started on %s\n", listenAddr)
	fmt.Printf("Version: %s, Build: %s, Commit: %s\n", Version, BuildTime, GitCommit)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		fmt.Printf("\nReceived signal %s, initiating graceful shutdown...\n", sig)
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Error during shutdown: %v\n", err)
		return err
	}

	fmt.Println("Server stopped gracefully")
	return nil
}
