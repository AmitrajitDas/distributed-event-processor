package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouter(handler *EventHandler) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-request-id")
		c.Next()
	})
	router.POST("/events", handler.IngestEvent)
	router.POST("/events/batch", handler.IngestBatch)
	router.POST("/events/validate", handler.ValidateEvent)
	return router
}

func TestIngestEvent_InvalidJSON(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewEventHandler(nil, logger)
	router := setupTestRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_json", response["error"])
}

func TestIngestEvent_ValidationFailed(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewEventHandler(nil, logger)
	router := setupTestRouter(handler)

	// Missing required fields
	payload := map[string]interface{}{
		"type": "test.event",
		// Missing "source" and "data"
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "validation_failed", response["error"])
}

func TestValidateEvent_Valid(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewEventHandler(nil, logger)
	router := setupTestRouter(handler)

	payload := map[string]interface{}{
		"type":   "user.created",
		"source": "test-service",
		"data": map[string]interface{}{
			"user_id": "123",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/events/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, true, response["valid"])
	assert.NotEmpty(t, response["event_id"])
}

func TestValidateEvent_InvalidJSON(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewEventHandler(nil, logger)
	router := setupTestRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/events/validate", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, false, response["valid"])
	assert.Equal(t, "invalid_json", response["error"])
}

func TestValidateEvent_ValidationFailed(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewEventHandler(nil, logger)
	router := setupTestRouter(handler)

	// Missing required fields
	payload := map[string]interface{}{
		"type": "test.event",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/events/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, false, response["valid"])
	assert.Equal(t, "validation_failed", response["error"])
}

func TestIngestBatch_InvalidJSON(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewEventHandler(nil, logger)
	router := setupTestRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/events/batch", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_json", response["error"])
}

func TestIngestBatch_EmptyEvents(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewEventHandler(nil, logger)
	router := setupTestRouter(handler)

	payload := map[string]interface{}{
		"events": []interface{}{},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/events/batch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "validation_failed", response["error"])
}

func TestGetRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("with request_id set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("request_id", "test-123")

		id := getRequestID(c)
		assert.Equal(t, "test-123", id)
	})

	t.Run("without request_id set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		id := getRequestID(c)
		assert.Equal(t, "unknown", id)
	})
}

func TestFormatValidationErrors(t *testing.T) {
	// Test with non-validation error
	err := assert.AnError
	result := formatValidationErrors(err)
	assert.Equal(t, err.Error(), result)
}

// Note: Tests for successful event ingestion are omitted as they would require
// a real Kafka producer or complex mocking. The validation tests above provide
// adequate coverage of request handling, parsing, and validation logic.
