package handlers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/distributed-event-processor/services/event-gateway/internal/kafka"
	"github.com/distributed-event-processor/services/event-gateway/internal/models"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type HealthHandler struct {
	logger    *zap.Logger
	producer  *kafka.Producer
	startTime time.Time
}

func NewHealthHandler(logger *zap.Logger, producer *kafka.Producer) *HealthHandler {
	return &HealthHandler{
		logger:    logger,
		producer:  producer,
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

	// Check Kafka connectivity
	kafkaStatus := "healthy"
	if h.producer == nil {
		kafkaStatus = "unavailable"
		health.Status = "degraded"
	} else if !h.producer.IsHealthy() {
		kafkaStatus = "unhealthy"
		health.Status = "degraded"
	}
	health.Services["kafka"] = kafkaStatus

	// Add system information
	uptime := time.Since(h.startTime)

	// Get runtime stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

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
			"goroutines":       runtime.NumGoroutine(),
			"memory_alloc_mb":  float64(memStats.Alloc) / 1024 / 1024,
			"memory_sys_mb":    float64(memStats.Sys) / 1024 / 1024,
			"memory_heap_mb":   float64(memStats.HeapAlloc) / 1024 / 1024,
			"gc_cycles":        memStats.NumGC,
			"gc_pause_total_ms": float64(memStats.PauseTotalNs) / 1e6,
		},
	}

	c.JSON(http.StatusOK, response)
}

// Ready checks if the service is ready to serve traffic
func (h *HealthHandler) Ready(c *gin.Context) {
	ready := true
	services := make(map[string]string)

	// Check Kafka connectivity
	if h.producer == nil {
		services["kafka"] = "unavailable"
		ready = false
	} else if !h.producer.IsHealthy() {
		services["kafka"] = "not_ready"
		ready = false
	} else {
		services["kafka"] = "ready"
	}

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
