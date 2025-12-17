package models

import (
	"time"
)

// Event represents an event consumed from Kafka
// This structure matches the Event produced by the event-gateway
type Event struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Source        string                 `json:"source"`
	Subject       string                 `json:"subject,omitempty"`
	TenantID      string                 `json:"tenant_id,omitempty"`
	Data          map[string]interface{} `json:"data"`
	Timestamp     time.Time              `json:"timestamp"`
	Version       string                 `json:"version,omitempty"`
	SchemaVersion string                 `json:"schema_version,omitempty"`
	Metadata      map[string]string      `json:"metadata,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Priority      int                    `json:"priority,omitempty"`
}

// ProcessedEvent represents an event after processing and aggregation
type ProcessedEvent struct {
	WindowID      string                 `json:"window_id"`
	WindowStart   time.Time              `json:"window_start"`
	WindowEnd     time.Time              `json:"window_end"`
	EventCount    int                    `json:"event_count"`
	EventTypes    map[string]int         `json:"event_types"`    // Count by event type
	Sources       map[string]int         `json:"sources"`        // Count by source
	Aggregations  map[string]interface{} `json:"aggregations"`   // Custom aggregations
	FirstEvent    time.Time              `json:"first_event"`    // Timestamp of first event in window
	LastEvent     time.Time              `json:"last_event"`     // Timestamp of last event in window
	ProcessedAt   time.Time              `json:"processed_at"`   // When the window was processed
	TenantID      string                 `json:"tenant_id,omitempty"`
	Metadata      map[string]string      `json:"metadata,omitempty"`
}

// EventKey represents a key for grouping events (used in session windows)
type EventKey struct {
	TenantID string
	Subject  string
}

// String returns a string representation of the event key
func (k EventKey) String() string {
	if k.TenantID != "" && k.Subject != "" {
		return k.TenantID + ":" + k.Subject
	}
	if k.TenantID != "" {
		return k.TenantID
	}
	if k.Subject != "" {
		return k.Subject
	}
	return "default"
}

// GetEventKey extracts the grouping key from an event
func GetEventKey(event *Event) EventKey {
	return EventKey{
		TenantID: event.TenantID,
		Subject:  event.Subject,
	}
}
