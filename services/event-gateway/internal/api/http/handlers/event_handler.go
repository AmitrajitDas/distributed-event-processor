package handlers

import (
	"net/http"

	"github.com/distributed-event-processor/services/event-gateway/internal/kafka"
	"github.com/distributed-event-processor/services/event-gateway/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type EventHandler struct {
	producer  *kafka.Producer
	logger    *zap.Logger
	validator *validator.Validate
}

func NewEventHandler(producer *kafka.Producer, logger *zap.Logger) *EventHandler {
	return &EventHandler{
		producer:  producer,
		logger:    logger,
		validator: validator.New(),
	}
}

// IngestEvent handles single event ingestion
func (h *EventHandler) IngestEvent(c *gin.Context) {
	var req models.EventRequest

	// Bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid JSON in event request",
			zap.String("request_id", getRequestID(c)),
			zap.Error(err))

		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "invalid_json",
			"message":    "Invalid JSON format",
			"details":    err.Error(),
			"request_id": getRequestID(c),
		})
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		h.logger.Warn("Event validation failed",
			zap.String("request_id", getRequestID(c)),
			zap.Error(err))

		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "validation_failed",
			"message":    "Event validation failed",
			"details":    formatValidationErrors(err),
			"request_id": getRequestID(c),
		})
		return
	}

	// Convert to event
	event := req.ToEvent()

	// Add request metadata
	if event.Metadata == nil {
		event.Metadata = make(map[string]string)
	}
	event.Metadata["request_id"] = getRequestID(c)
	event.Metadata["client_ip"] = c.ClientIP()
	event.Metadata["user_agent"] = c.GetHeader("User-Agent")

	// Send to Kafka
	if err := h.producer.SendEvent(event); err != nil {
		h.logger.Error("Failed to send event to Kafka",
			zap.String("event_id", event.ID),
			zap.String("request_id", getRequestID(c)),
			zap.Error(err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "ingestion_failed",
			"message":    "Failed to ingest event",
			"event_id":   event.ID,
			"request_id": getRequestID(c),
		})
		return
	}

	h.logger.Info("Event ingested successfully",
		zap.String("event_id", event.ID),
		zap.String("event_type", event.Type),
		zap.String("source", event.Source),
		zap.String("request_id", getRequestID(c)))

	// Return success response
	response := models.EventResponse{
		EventID:   event.ID,
		Status:    "accepted",
		Timestamp: event.Timestamp,
		Message:   "Event ingested successfully",
	}

	c.Header("X-Event-ID", event.ID)
	c.JSON(http.StatusAccepted, response)
}

// IngestBatch handles batch event ingestion
func (h *EventHandler) IngestBatch(c *gin.Context) {
	var req models.BatchEventRequest

	// Bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid JSON in batch request",
			zap.String("request_id", getRequestID(c)),
			zap.Error(err))

		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "invalid_json",
			"message":    "Invalid JSON format",
			"details":    err.Error(),
			"request_id": getRequestID(c),
		})
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		h.logger.Warn("Batch validation failed",
			zap.String("request_id", getRequestID(c)),
			zap.Error(err))

		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "validation_failed",
			"message":    "Batch validation failed",
			"details":    formatValidationErrors(err),
			"request_id": getRequestID(c),
		})
		return
	}

	// Process events
	response := models.BatchEventResponse{
		Results: make([]models.BatchEventResult, len(req.Events)),
	}

	events := make([]*models.Event, 0, len(req.Events))

	for i, eventReq := range req.Events {
		// Validate individual event
		if err := h.validator.Struct(&eventReq); err != nil {
			response.FailedCount++
			response.Results[i] = models.BatchEventResult{
				Status: "failed",
				Error:  formatValidationErrors(err),
			}
			response.Errors = append(response.Errors,
				formatValidationErrors(err))
			continue
		}

		// Convert to event
		event := eventReq.ToEvent()

		// Add request metadata
		if event.Metadata == nil {
			event.Metadata = make(map[string]string)
		}
		event.Metadata["request_id"] = getRequestID(c)
		event.Metadata["client_ip"] = c.ClientIP()
		event.Metadata["user_agent"] = c.GetHeader("User-Agent")
		event.Metadata["batch_index"] = string(rune(i))

		events = append(events, event)
		response.Results[i] = models.BatchEventResult{
			EventID: event.ID,
			Status:  "accepted",
		}
		response.ProcessedCount++
	}

	// Send events to Kafka
	if len(events) > 0 {
		if err := h.producer.SendBatchEvents(events); err != nil {
			h.logger.Error("Failed to send batch events to Kafka",
				zap.String("request_id", getRequestID(c)),
				zap.Int("event_count", len(events)),
				zap.Error(err))

			c.JSON(http.StatusInternalServerError, gin.H{
				"error":      "ingestion_failed",
				"message":    "Failed to ingest batch events",
				"request_id": getRequestID(c),
			})
			return
		}
	}

	h.logger.Info("Batch events processed",
		zap.String("request_id", getRequestID(c)),
		zap.Int("total_events", len(req.Events)),
		zap.Int("processed", response.ProcessedCount),
		zap.Int("failed", response.FailedCount))

	status := http.StatusAccepted
	if response.FailedCount > 0 {
		status = http.StatusMultiStatus
	}

	c.JSON(status, response)
}

// ValidateEvent validates an event without ingesting it (dry-run)
func (h *EventHandler) ValidateEvent(c *gin.Context) {
	var req models.EventRequest

	// Bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid":      false,
			"error":      "invalid_json",
			"message":    "Invalid JSON format",
			"details":    err.Error(),
			"request_id": getRequestID(c),
		})
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid":      false,
			"error":      "validation_failed",
			"message":    "Event validation failed",
			"details":    formatValidationErrors(err),
			"request_id": getRequestID(c),
		})
		return
	}

	// Convert to event to test transformation
	event := req.ToEvent()

	c.JSON(http.StatusOK, gin.H{
		"valid":      true,
		"message":    "Event is valid",
		"event_id":   event.ID,
		"timestamp":  event.Timestamp,
		"request_id": getRequestID(c),
	})
}

// Helper functions

func getRequestID(c *gin.Context) string {
	if id, exists := c.Get("request_id"); exists {
		return id.(string)
	}
	return "unknown"
}

func formatValidationErrors(err error) string {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var errors []string
		for _, e := range validationErrors {
			errors = append(errors,
				"Field '"+e.Field()+"' failed validation: "+e.Tag())
		}
		if len(errors) > 0 {
			return errors[0] // Return first error for simplicity
		}
	}
	return err.Error()
}
