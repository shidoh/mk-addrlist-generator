package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"time"

	"mk-addrlist-generator/pkg/config"
	"mk-addrlist-generator/pkg/generator"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Build information (set at compile time)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// Prometheus metrics
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mk_addrlist_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mk_addrlist_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	listEntriesCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mk_addrlist_list_entries_total",
			Help: "Total number of entries in each list",
		},
		[]string{"list_name", "source_type"},
	)

	listGenerationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mk_addrlist_list_generation_duration_seconds",
			Help:    "Duration of list generation in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"list_name", "format"},
	)

	listGenerationErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mk_addrlist_list_generation_errors_total",
			Help: "Total number of list generation errors",
		},
		[]string{"list_name"},
	)
)

func init() {
	// Register metrics
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(listEntriesCount)
	prometheus.MustRegister(listGenerationDuration)
	prometheus.MustRegister(listGenerationErrors)
}

// ServerConfig holds server configuration
type ServerConfig struct {
	EnableMetrics bool
	EnableHealth  bool
	LogLevel      slog.Level
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		EnableMetrics: true,
		EnableHealth:  true,
		LogLevel:      slog.LevelInfo,
	}
}

type Server struct {
	cfg       *config.Config
	serverCfg ServerConfig
	generator *generator.Generator
	router    *gin.Engine
	server    *http.Server
	logger    *slog.Logger
	startTime time.Time
}

func NewServer(cfg *config.Config) *Server {
	return NewServerWithConfig(cfg, DefaultServerConfig())
}

func NewServerWithConfig(cfg *config.Config, serverCfg ServerConfig) *Server {
	// Setup structured logging
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: serverCfg.LogLevel,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	// Set gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	s := &Server{
		cfg:       cfg,
		serverCfg: serverCfg,
		generator: generator.NewGenerator(cfg),
		router:    gin.New(),
		logger:    logger,
		startTime: time.Now(),
	}

	// Add middleware
	s.router.Use(s.loggingMiddleware())
	s.router.Use(s.metricsMiddleware())
	s.router.Use(gin.Recovery())

	// Register routes
	s.registerRoutes()

	return s
}

func (s *Server) registerRoutes() {
	// Health and readiness endpoints
	if s.serverCfg.EnableHealth {
		s.router.GET("/health", s.HandleHealth)
		s.router.GET("/healthz", s.HandleHealth)
		s.router.GET("/ready", s.HandleReady)
		s.router.GET("/readyz", s.HandleReady)
		s.router.GET("/live", s.HandleLive)
		s.router.GET("/livez", s.HandleLive)
	}

	// Metrics endpoint
	if s.serverCfg.EnableMetrics {
		s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// API info endpoint
	s.router.GET("/", s.HandleInfo)
	s.router.GET("/info", s.HandleInfo)

	// List endpoints
	s.router.GET("/lists/all", s.HandleGetAllLists)
	s.router.GET("/list/:name", s.HandleGetListByName)
	s.router.GET("/lists", s.HandleListNames)

	// Stats endpoint
	s.router.GET("/stats", s.HandleStats)
}

func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		s.logger.Info("http request",
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("duration", time.Since(start)),
			slog.String("client_ip", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
		)
	}
}

func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		c.Next()

		duration := time.Since(start).Seconds()
		status := fmt.Sprintf("%d", c.Writer.Status())

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

func (s *Server) Start(addr string) error {
	s.server = &http.Server{
		Addr:              addr,
		Handler:           s.router,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	s.logger.Info("starting server",
		slog.String("address", addr),
		slog.String("version", Version),
	)

	return s.server.ListenAndServe()
}

func (s *Server) Stop() error {
	if s.server != nil {
		s.logger.Info("stopping server")
		return s.server.Close()
	}
	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		s.logger.Info("gracefully shutting down server")
		return s.server.Shutdown(ctx)
	}
	return nil
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
	BuildTime string `json:"build_time,omitempty"`
	GitCommit string `json:"git_commit,omitempty"`
	Uptime    string `json:"uptime"`
	GoVersion string `json:"go_version"`
}

func (s *Server) HandleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   Version,
		BuildTime: BuildTime,
		GitCommit: GitCommit,
		Uptime:    time.Since(s.startTime).Round(time.Second).String(),
		GoVersion: runtime.Version(),
	})
}

func (s *Server) HandleReady(c *gin.Context) {
	// Check if we can generate lists (basic readiness check)
	if len(s.cfg.Lists) == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"reason": "no lists configured",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "ready",
		"list_count": len(s.cfg.Lists),
	})
}

func (s *Server) HandleLive(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}

// InfoResponse represents the API info response
type InfoResponse struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Endpoints   []string `json:"endpoints"`
	Formats     []string `json:"formats"`
	ListCount   int      `json:"list_count"`
}

func (s *Server) HandleInfo(c *gin.Context) {
	formats := make([]string, 0)
	for _, f := range generator.AllFormats() {
		formats = append(formats, string(f))
	}

	c.JSON(http.StatusOK, InfoResponse{
		Name:        "MikroTik Address List Generator",
		Version:     Version,
		Description: "Generates MikroTik address lists from various sources",
		Endpoints: []string{
			"/health", "/ready", "/live", "/metrics",
			"/lists/all", "/list/:name", "/lists", "/stats",
		},
		Formats:   formats,
		ListCount: len(s.cfg.Lists),
	})
}

func (s *Server) HandleGetAllLists(c *gin.Context) {
	format := generator.ParseFormat(c.Query("format"))
	aggregate := c.Query("aggregate") == "true"
	deduplicate := c.DefaultQuery("deduplicate", "true") == "true"

	// Update generator options
	options := s.generator.GetOptions()
	options.Aggregate = aggregate
	options.Deduplicate = deduplicate
	s.generator.SetOptions(options)

	start := time.Now()
	script, err := s.generator.GenerateAllWithFormat(format)
	duration := time.Since(start).Seconds()

	if err != nil {
		s.logger.Error("failed to generate all lists",
			slog.String("format", string(format)),
			slog.String("error", err.Error()),
		)
		listGenerationErrors.WithLabelValues("all").Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	listGenerationDuration.WithLabelValues("all", string(format)).Observe(duration)

	// Set appropriate content type
	contentType := s.getContentType(format)
	c.Header("Content-Type", contentType)
	c.String(http.StatusOK, script)
}

func (s *Server) HandleGetListByName(c *gin.Context) {
	name := c.Param("name")
	format := generator.ParseFormat(c.Query("format"))
	aggregate := c.Query("aggregate") == "true"
	deduplicate := c.DefaultQuery("deduplicate", "true") == "true"

	list, exists := s.cfg.Lists[name]
	if !exists {
		s.logger.Warn("list not found",
			slog.String("list_name", name),
		)
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("list %s not found", name)})
		return
	}

	// Update generator options
	options := s.generator.GetOptions()
	options.Aggregate = aggregate
	options.Deduplicate = deduplicate
	s.generator.SetOptions(options)

	start := time.Now()
	script, err := s.generator.GenerateListWithFormat(name, list, format)
	duration := time.Since(start).Seconds()

	if err != nil {
		s.logger.Error("failed to generate list",
			slog.String("list_name", name),
			slog.String("format", string(format)),
			slog.String("error", err.Error()),
		)
		listGenerationErrors.WithLabelValues(name).Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	listGenerationDuration.WithLabelValues(name, string(format)).Observe(duration)

	// Set appropriate content type
	contentType := s.getContentType(format)
	c.Header("Content-Type", contentType)
	c.String(http.StatusOK, script)
}

func (s *Server) HandleListNames(c *gin.Context) {
	names := make([]string, 0, len(s.cfg.Lists))
	for name := range s.cfg.Lists {
		names = append(names, name)
	}

	c.JSON(http.StatusOK, gin.H{
		"lists": names,
		"count": len(names),
	})
}

func (s *Server) HandleStats(c *gin.Context) {
	stats := s.generator.GetStats()

	// Update Prometheus metrics
	for name, stat := range stats {
		listEntriesCount.WithLabelValues(name, "url").Set(float64(stat.URLEntries))
		listEntriesCount.WithLabelValues(name, "file").Set(float64(stat.FileEntries))
		listEntriesCount.WithLabelValues(name, "static").Set(float64(stat.StaticEntries))
		listEntriesCount.WithLabelValues(name, "total").Set(float64(stat.TotalEntries))
	}

	c.JSON(http.StatusOK, stats)
}

func (s *Server) getContentType(format generator.OutputFormat) string {
	switch format {
	case generator.FormatJSON:
		return "application/json"
	case generator.FormatPlain, generator.FormatMikrotik, generator.FormatNftables:
		return "text/plain; charset=utf-8"
	default:
		return "text/plain; charset=utf-8"
	}
}

// GetRouter returns the gin router for testing
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}

// SetGenerator allows setting a custom generator for testing
func (s *Server) SetGenerator(gen *generator.Generator) {
	s.generator = gen
}
