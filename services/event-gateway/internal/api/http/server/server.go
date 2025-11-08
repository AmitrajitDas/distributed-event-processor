package server

import (
	"net/http"
	"os"
	"time"

	"github.com/eventprocessor/event-gateway/internal/api/http/handlers"
	"github.com/eventprocessor/event-gateway/internal/api/http/middleware"
	"github.com/eventprocessor/event-gateway/internal/config"
	"github.com/eventprocessor/event-gateway/internal/kafka"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Server struct {
	config   *config.Config
	producer *kafka.Producer
	logger   *zap.Logger
	router   *gin.Engine
}

func New(cfg *config.Config, producer *kafka.Producer, logger *zap.Logger) *Server {
	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	server := &Server{
		config:   cfg,
		producer: producer,
		logger:   logger,
		router:   router,
	}

	server.setupMiddleware()
	server.setupRoutes()

	return server
}

func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Request ID middleware
	s.router.Use(middleware.RequestID())

	// Logging middleware
	s.router.Use(middleware.Logger(s.logger))

	// CORS middleware
	s.router.Use(middleware.CORS())

	// Rate limiting middleware
	s.router.Use(middleware.RateLimit(s.config.RateLimit))

	// Metrics middleware
	s.router.Use(middleware.Metrics())

	// Request size limit middleware
	s.router.Use(middleware.RequestSizeLimit("10MB"))
}

func (s *Server) setupRoutes() {
	// Create handlers
	eventHandler := handlers.NewEventHandler(s.producer, s.logger)
	healthHandler := handlers.NewHealthHandler(s.logger)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Event ingestion endpoints
		v1.POST("/events", eventHandler.IngestEvent)
		v1.POST("/events/batch", eventHandler.IngestBatch)

		// Event validation endpoint (dry-run)
		v1.POST("/events/validate", eventHandler.ValidateEvent)
	}

	// Health check endpoints
	s.router.GET("/health", healthHandler.Health)
	s.router.GET("/health/detailed", healthHandler.DetailedHealth)
	s.router.GET("/health/ready", healthHandler.Ready)
	s.router.GET("/health/live", healthHandler.Live)

	// Metrics endpoint
	if s.config.Metrics.Enabled {
		s.router.GET(s.config.Metrics.Path, gin.WrapH(promhttp.Handler()))
	}

	// API documentation endpoint
	s.router.GET("/api/docs", s.apiDocs)

	// Root endpoint
	s.router.GET("/", s.root)
}

func (s *Server) GetRouter() http.Handler {
	return s.router
}

// API documentation endpoint
func (s *Server) apiDocs(c *gin.Context) {
	docs := gin.H{
		"service": "Event Gateway",
		"version": "1.0.0",
		"endpoints": gin.H{
			"POST /api/v1/events": gin.H{
				"description":  "Ingest a single event",
				"content_type": "application/json",
				"example": gin.H{
					"type":    "user.created",
					"source":  "user-service",
					"subject": "user-123",
					"data": gin.H{
						"user_id": "123",
						"email":   "user@example.com",
					},
				},
			},
			"POST /api/v1/events/batch": gin.H{
				"description":  "Ingest multiple events in a single request",
				"content_type": "application/json",
				"max_events":   100,
			},
			"POST /api/v1/events/validate": gin.H{
				"description":  "Validate event without ingesting (dry-run)",
				"content_type": "application/json",
			},
			"GET /health": gin.H{
				"description": "Basic health check",
			},
			"GET /health/detailed": gin.H{
				"description": "Detailed health check with dependencies",
			},
			"GET /metrics": gin.H{
				"description": "Prometheus metrics",
			},
		},
	}

	c.JSON(http.StatusOK, docs)
}

// Root endpoint
func (s *Server) root(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service":   "Event Gateway",
		"version":   "1.0.0",
		"status":    "running",
		"timestamp": time.Now().UTC(),
		"docs":      "/api/docs",
		"health":    "/health",
		"metrics":   s.config.Metrics.Path,
	})
}
