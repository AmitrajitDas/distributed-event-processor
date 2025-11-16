package models

import (
	"time"

	"github.com/google/uuid"
)

// Event represents an incoming event
type Event struct {
	ID            string                 `json:"id" validate:"required"`
	Type          string                 `json:"type" validate:"required"`
	Source        string                 `json:"source" validate:"required"`
	Subject       string                 `json:"subject,omitempty"`
	TenantID      string                 `json:"tenant_id,omitempty"`
	Data          map[string]interface{} `json:"data" validate:"required"`
	Timestamp     time.Time              `json:"timestamp"`
	Version       string                 `json:"version,omitempty"`
	SchemaVersion string                 `json:"schema_version,omitempty"`
	Metadata      map[string]string      `json:"metadata,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Priority      int                    `json:"priority,omitempty"`
}

// EventRequest represents the request payload for event ingestion
type EventRequest struct {
	Type     string                 `json:"type" validate:"required"`
	Source   string                 `json:"source" validate:"required"`
	Subject  string                 `json:"subject,omitempty"`
	Data     map[string]interface{} `json:"data" validate:"required"`
	Version  string                 `json:"version,omitempty"`
	Metadata map[string]string      `json:"metadata,omitempty"`
}

// ToEvent converts EventRequest to Event with generated fields
func (er *EventRequest) ToEvent() *Event {
	return &Event{
		ID:        uuid.New().String(),
		Type:      er.Type,
		Source:    er.Source,
		Subject:   er.Subject,
		Data:      er.Data,
		Timestamp: time.Now().UTC(),
		Version:   er.Version,
		Metadata:  er.Metadata,
	}
}

// EventResponse represents the response after successful event ingestion
type EventResponse struct {
	EventID   string    `json:"event_id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message,omitempty"`
}

// BatchEventRequest represents multiple events in a single request
type BatchEventRequest struct {
	Events []EventRequest `json:"events" validate:"required,min=1,max=100,dive"`
}

// BatchEventResponse represents response for batch event ingestion
type BatchEventResponse struct {
	ProcessedCount int                `json:"processed_count"`
	FailedCount    int                `json:"failed_count"`
	Results        []BatchEventResult `json:"results"`
	Errors         []string           `json:"errors,omitempty"`
}

// BatchEventResult represents individual event result in batch
type BatchEventResult struct {
	EventID string `json:"event_id"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

// HealthCheck represents health check response
type HealthCheck struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version"`
	Services  map[string]string `json:"services"`
}
