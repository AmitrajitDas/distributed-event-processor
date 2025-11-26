package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupHealthRouter(handler *HealthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", handler.Health)
	router.GET("/health/detailed", handler.DetailedHealth)
	router.GET("/health/ready", handler.Ready)
	router.GET("/health/live", handler.Live)
	return router
}

func TestHealth_Basic(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewHealthHandler(logger, nil)
	router := setupHealthRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "1.0.0", response["version"])
	assert.NotEmpty(t, response["timestamp"])
}

func TestDetailedHealth_WithoutProducer(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewHealthHandler(logger, nil)
	router := setupHealthRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/health/detailed", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "degraded", response["status"])

	services := response["services"].(map[string]interface{})
	assert.Equal(t, "unavailable", services["kafka"])

	// Check performance metrics are present
	performance := response["performance"].(map[string]interface{})
	assert.NotNil(t, performance["goroutines"])
	assert.NotNil(t, performance["memory_alloc_mb"])
	assert.NotNil(t, performance["memory_sys_mb"])
	assert.NotNil(t, performance["memory_heap_mb"])
	assert.NotNil(t, performance["gc_cycles"])

	// Check system metrics
	system := response["system"].(map[string]interface{})
	assert.NotNil(t, system["uptime_seconds"])
	assert.NotNil(t, system["uptime_human"])
	assert.NotNil(t, system["started_at"])
}

func TestReady_WithoutProducer(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewHealthHandler(logger, nil)
	router := setupHealthRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "not_ready", response["status"])

	services := response["services"].(map[string]interface{})
	assert.Equal(t, "unavailable", services["kafka"])
}

func TestLive(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewHealthHandler(logger, nil)
	router := setupHealthRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "alive", response["status"])
	assert.NotEmpty(t, response["timestamp"])
	assert.NotEmpty(t, response["uptime"])
}
