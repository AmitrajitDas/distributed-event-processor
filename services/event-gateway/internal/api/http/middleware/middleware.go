package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/distributed-event-processor/services/event-gateway/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// Prometheus metrics
var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	activeConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_active_connections",
			Help: "Number of active HTTP connections",
		},
	)

	eventsIngested = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_ingested_total",
			Help: "Total number of events ingested",
		},
		[]string{"event_type", "source"},
	)

	eventsIngestedFailed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_ingested_failed_total",
			Help: "Total number of failed event ingestions",
		},
		[]string{"event_type", "source", "reason"},
	)
)

// RequestID middleware adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// Logger middleware provides structured logging
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			logger.Error("Panic recovered",
				zap.String("error", err),
				zap.String("request_id", getRequestID(c)),
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path))
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}

// CORS middleware handles Cross-Origin Resource Sharing
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID, X-Event-ID")
		c.Header("Access-Control-Allow-Credentials", "false")
		c.Header("Access-Control-Max-Age", "3600")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RateLimit middleware implements token bucket rate limiting
func RateLimit(cfg config.RateLimitConfig) gin.HandlerFunc {
	limiter := rate.NewLimiter(
		rate.Limit(cfg.RequestsPerSecond),
		cfg.BurstSize,
	)

	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate_limit_exceeded",
				"message":     "Rate limit exceeded",
				"retry_after": "1s",
				"request_id":  getRequestID(c),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// Metrics middleware collects Prometheus metrics
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Track active connections
		activeConnections.Inc()
		defer activeConnections.Dec()

		c.Next()

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Get route pattern (if available)
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = c.Request.URL.Path
		}

		// Record metrics
		status := strconv.Itoa(c.Writer.Status())
		httpRequestsTotal.WithLabelValues(
			c.Request.Method,
			endpoint,
			status,
		).Inc()

		httpRequestDuration.WithLabelValues(
			c.Request.Method,
			endpoint,
		).Observe(duration)
	}
}

// RequestSizeLimit middleware limits request body size
func RequestSizeLimit(maxSize string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var maxBytes int64

		// Parse max size (e.g., "10MB", "1KB")
		switch maxSize {
		case "1KB":
			maxBytes = 1024
		case "1MB":
			maxBytes = 1024 * 1024
		case "10MB":
			maxBytes = 10 * 1024 * 1024
		default:
			maxBytes = 10 * 1024 * 1024 // Default 10MB
		}

		if c.Request.ContentLength > maxBytes {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":      "request_too_large",
				"message":    fmt.Sprintf("Request body too large. Maximum allowed: %s", maxSize),
				"request_id": getRequestID(c),
			})
			c.Abort()
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// Timeout middleware sets a timeout for request processing
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set context timeout
		ctx := c.Request.Context()
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// Security middleware adds security headers
func Security() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}

// Helper functions

func getRequestID(c *gin.Context) string {
	if id, exists := c.Get("request_id"); exists {
		return id.(string)
	}
	return "unknown"
}
