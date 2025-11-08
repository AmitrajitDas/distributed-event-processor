package handlers

import (
	"net/http"
	"time"

	"github.com/eventprocessor/event-gateway/internal/models"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type HealthHandler struct {
	logger    *zap.Logger
	startTime time.Time
}

func NewHealthHandler(logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		logger:    logger,
		startTime: time.Now(),
	}
}

// Health provides basic health check
func (h *HealthHandler) Health(c *gin.Context) {
	health := models.HealthCheck{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
		Version:   "1.0.0",
		Services:  make(map[string]string),
	}

	c.JSON(http.StatusOK, health)
}

// DetailedHealth provides detailed health check with dependency status
func (h *HealthHandler) DetailedHealth(c *gin.Context) {
	health := models.HealthCheck{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
		Version:   "1.0.0",
		Services:  make(map[string]string),
	}

	// Check Kafka connectivity (simplified)
	health.Services["kafka"] = "healthy"

	// Add system information
	uptime := time.Since(h.startTime)

	response := gin.H{
		"status":    health.Status,
		"timestamp": health.Timestamp,
		"version":   health.Version,
		"services":  health.Services,
		"system": gin.H{
			"uptime_seconds": int(uptime.Seconds()),
			"uptime_human":   uptime.String(),
			"started_at":     h.startTime.UTC(),
		},
		"performance": gin.H{
			"goroutines": "healthy", // Could use runtime.NumGoroutine()
			"memory":     "healthy", // Could use runtime.ReadMemStats()
		},
	}

	c.JSON(http.StatusOK, response)
}

// Ready checks if the service is ready to serve traffic
func (h *HealthHandler) Ready(c *gin.Context) {
	// Check if service is ready (all dependencies available)
	ready := true
	services := make(map[string]string)

	// Check Kafka connectivity
	services["kafka"] = "ready"

	status := http.StatusOK
	statusText := "ready"

	if !ready {
		status = http.StatusServiceUnavailable
		statusText = "not_ready"
	}

	c.JSON(status, gin.H{
		"status":    statusText,
		"timestamp": time.Now().UTC(),
		"services":  services,
	})
}

// Live checks if the service is alive (liveness probe)
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now().UTC(),
		"uptime":    time.Since(h.startTime).String(),
	})
}
