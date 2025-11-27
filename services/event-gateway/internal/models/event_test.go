package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventRequest_ToEvent(t *testing.T) {
	req := &EventRequest{
		Type:    "user.created",
		Source:  "user-service",
		Subject: "user-123",
		Data: map[string]interface{}{
			"user_id": "123",
			"email":   "test@example.com",
		},
		Version: "1.0",
		Metadata: map[string]string{
			"correlation_id": "corr-456",
		},
	}

	event := req.ToEvent()

	assert.NotEmpty(t, event.ID, "Event ID should be generated")
	assert.Equal(t, req.Type, event.Type)
	assert.Equal(t, req.Source, event.Source)
	assert.Equal(t, req.Subject, event.Subject)
	assert.Equal(t, req.Data, event.Data)
	assert.Equal(t, req.Version, event.Version)
	assert.Equal(t, req.Metadata, event.Metadata)
	assert.False(t, event.Timestamp.IsZero(), "Timestamp should be set")
	assert.True(t, event.Timestamp.Before(time.Now().Add(time.Second)), "Timestamp should be recent")
}

func TestEventRequest_ToEvent_MinimalFields(t *testing.T) {
	req := &EventRequest{
		Type:   "test.event",
		Source: "test-service",
		Data: map[string]interface{}{
			"key": "value",
		},
	}

	event := req.ToEvent()

	assert.NotEmpty(t, event.ID)
	assert.Equal(t, "test.event", event.Type)
	assert.Equal(t, "test-service", event.Source)
	assert.Empty(t, event.Subject)
	assert.Empty(t, event.Version)
	assert.Nil(t, event.Metadata)
}

func TestEventRequest_ToEvent_GeneratesUniqueIDs(t *testing.T) {
	req := &EventRequest{
		Type:   "test.event",
		Source: "test-service",
		Data:   map[string]interface{}{"key": "value"},
	}

	event1 := req.ToEvent()
	event2 := req.ToEvent()

	assert.NotEqual(t, event1.ID, event2.ID, "Each event should have a unique ID")
}

func TestEvent_AllFields(t *testing.T) {
	now := time.Now().UTC()
	event := Event{
		ID:            "event-123",
		Type:          "order.placed",
		Source:        "order-service",
		Subject:       "order-456",
		TenantID:      "tenant-789",
		Data:          map[string]interface{}{"amount": 100.50},
		Timestamp:     now,
		Version:       "2.0",
		SchemaVersion: "1.0.0",
		Metadata:      map[string]string{"env": "production"},
		CorrelationID: "corr-123",
		Priority:      1,
	}

	assert.Equal(t, "event-123", event.ID)
	assert.Equal(t, "order.placed", event.Type)
	assert.Equal(t, "order-service", event.Source)
	assert.Equal(t, "order-456", event.Subject)
	assert.Equal(t, "tenant-789", event.TenantID)
	assert.Equal(t, 100.50, event.Data["amount"])
	assert.Equal(t, now, event.Timestamp)
	assert.Equal(t, "2.0", event.Version)
	assert.Equal(t, "1.0.0", event.SchemaVersion)
	assert.Equal(t, "production", event.Metadata["env"])
	assert.Equal(t, "corr-123", event.CorrelationID)
	assert.Equal(t, 1, event.Priority)
}

func TestEventResponse(t *testing.T) {
	now := time.Now().UTC()
	response := EventResponse{
		EventID:   "event-123",
		Status:    "accepted",
		Timestamp: now,
		Message:   "Event processed successfully",
	}

	assert.Equal(t, "event-123", response.EventID)
	assert.Equal(t, "accepted", response.Status)
	assert.Equal(t, now, response.Timestamp)
	assert.Equal(t, "Event processed successfully", response.Message)
}

func TestBatchEventRequest(t *testing.T) {
	req := BatchEventRequest{
		Events: []EventRequest{
			{
				Type:   "event.type1",
				Source: "source1",
				Data:   map[string]interface{}{"key1": "value1"},
			},
			{
				Type:   "event.type2",
				Source: "source2",
				Data:   map[string]interface{}{"key2": "value2"},
			},
		},
	}

	assert.Len(t, req.Events, 2)
	assert.Equal(t, "event.type1", req.Events[0].Type)
	assert.Equal(t, "event.type2", req.Events[1].Type)
}

func TestBatchEventResponse(t *testing.T) {
	response := BatchEventResponse{
		ProcessedCount: 3,
		FailedCount:    1,
		Results: []BatchEventResult{
			{EventID: "event-1", Status: "accepted"},
			{EventID: "event-2", Status: "accepted"},
			{EventID: "event-3", Status: "accepted"},
			{EventID: "", Status: "failed", Error: "validation error"},
		},
		Errors: []string{"validation error"},
	}

	assert.Equal(t, 3, response.ProcessedCount)
	assert.Equal(t, 1, response.FailedCount)
	assert.Len(t, response.Results, 4)
	assert.Len(t, response.Errors, 1)
}

func TestBatchEventResult(t *testing.T) {
	t.Run("successful result", func(t *testing.T) {
		result := BatchEventResult{
			EventID: "event-123",
			Status:  "accepted",
		}

		assert.Equal(t, "event-123", result.EventID)
		assert.Equal(t, "accepted", result.Status)
		assert.Empty(t, result.Error)
	})

	t.Run("failed result", func(t *testing.T) {
		result := BatchEventResult{
			Status: "failed",
			Error:  "validation failed: missing required field",
		}

		assert.Empty(t, result.EventID)
		assert.Equal(t, "failed", result.Status)
		assert.Equal(t, "validation failed: missing required field", result.Error)
	})
}

func TestHealthCheck(t *testing.T) {
	now := time.Now().UTC()
	health := HealthCheck{
		Status:    "healthy",
		Timestamp: now,
		Version:   "1.0.0",
		Services: map[string]string{
			"kafka":    "healthy",
			"redis":    "healthy",
			"postgres": "healthy",
		},
	}

	assert.Equal(t, "healthy", health.Status)
	assert.Equal(t, now, health.Timestamp)
	assert.Equal(t, "1.0.0", health.Version)
	assert.Len(t, health.Services, 3)
	assert.Equal(t, "healthy", health.Services["kafka"])
}
