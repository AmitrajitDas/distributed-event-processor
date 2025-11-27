package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/distributed-event-processor/services/event-gateway/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestID_GeneratesID(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		id := c.GetString("request_id")
		c.String(http.StatusOK, id)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestRequestID_UsesExistingID(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		id := c.GetString("request_id")
		c.String(http.StatusOK, id)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "custom-request-id")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "custom-request-id", w.Body.String())
	assert.Equal(t, "custom-request-id", w.Header().Get("X-Request-ID"))
}

func TestCORS(t *testing.T) {
	router := gin.New()
	router.Use(CORS())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("sets CORS headers on GET", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
	})

	t.Run("handles OPTIONS preflight", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestRateLimit(t *testing.T) {
	cfg := config.RateLimitConfig{
		RequestsPerSecond: 2,
		BurstSize:         2,
	}

	router := gin.New()
	router.Use(RateLimit(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// First two requests should succeed (within burst)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
	}

	// Third request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRequestSizeLimit(t *testing.T) {
	router := gin.New()
	router.Use(RequestSizeLimit("1KB"))
	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("allows small requests", func(t *testing.T) {
		body := bytes.NewBufferString("small body")
		req := httptest.NewRequest(http.MethodPost, "/test", body)
		req.Header.Set("Content-Length", "10")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("rejects large requests", func(t *testing.T) {
		// Create a body larger than 1KB
		largeBody := bytes.Repeat([]byte("x"), 2048)
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBuffer(largeBody))
		req.Header.Set("Content-Length", "2048")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})
}

func TestTimeout(t *testing.T) {
	router := gin.New()
	router.Use(Timeout(100 * time.Millisecond))
	router.GET("/test", func(c *gin.Context) {
		// Check if context has deadline
		_, ok := c.Request.Context().Deadline()
		if ok {
			c.String(http.StatusOK, "has_deadline")
		} else {
			c.String(http.StatusOK, "no_deadline")
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "has_deadline", w.Body.String())
}

func TestSecurity(t *testing.T) {
	router := gin.New()
	router.Use(Security())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
}

func TestLogger(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	router := gin.New()
	router.Use(Logger(logger))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMetrics(t *testing.T) {
	router := gin.New()
	router.Use(Metrics())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetRequestID_Helper(t *testing.T) {
	t.Run("returns request_id when set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("request_id", "test-id-123")

		id := getRequestID(c)
		assert.Equal(t, "test-id-123", id)
	})

	t.Run("returns unknown when not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		id := getRequestID(c)
		assert.Equal(t, "unknown", id)
	})
}
