package models

import (
	"encoding/json"
	"time"
)

// WindowState represents the state of a window stored in Redis
type WindowState struct {
	WindowID      string                 `json:"window_id"`
	WindowType    WindowType             `json:"window_type"`
	WindowStart   time.Time              `json:"window_start"`
	WindowEnd     time.Time              `json:"window_end"`
	Key           string                 `json:"key,omitempty"` // For session windows
	EventCount    int                    `json:"event_count"`
	EventIDs      []string               `json:"event_ids"`         // For deduplication
	EventTypes    map[string]int         `json:"event_types"`       // Count by event type
	Sources       map[string]int         `json:"sources"`           // Count by source
	FirstEvent    time.Time              `json:"first_event"`       // First event timestamp
	LastEvent     time.Time              `json:"last_event"`        // Last event timestamp
	Aggregations  map[string]interface{} `json:"aggregations"`      // Custom aggregations
	TenantID      string                 `json:"tenant_id,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// NewWindowState creates a new window state
func NewWindowState(window *Window) *WindowState {
	return &WindowState{
		WindowID:     window.ID,
		WindowType:   window.Type,
		WindowStart:  window.Start,
		WindowEnd:    window.End,
		Key:          window.Key,
		EventCount:   0,
		EventIDs:     make([]string, 0),
		EventTypes:   make(map[string]int),
		Sources:      make(map[string]int),
		Aggregations: make(map[string]interface{}),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// AddEvent adds an event to the window state
func (ws *WindowState) AddEvent(event *Event) {
	// Increment event count
	ws.EventCount++

	// Track event ID for deduplication
	ws.EventIDs = append(ws.EventIDs, event.ID)

	// Update event type counts
	ws.EventTypes[event.Type]++

	// Update source counts
	ws.Sources[event.Source]++

	// Update first/last event timestamps
	if ws.FirstEvent.IsZero() || event.Timestamp.Before(ws.FirstEvent) {
		ws.FirstEvent = event.Timestamp
	}
	if ws.LastEvent.IsZero() || event.Timestamp.After(ws.LastEvent) {
		ws.LastEvent = event.Timestamp
	}

	// Update tenant ID (use first event's tenant)
	if ws.TenantID == "" {
		ws.TenantID = event.TenantID
	}

	ws.UpdatedAt = time.Now()
}

// HasEvent checks if an event ID already exists in the window state (for deduplication)
func (ws *WindowState) HasEvent(eventID string) bool {
	for _, id := range ws.EventIDs {
		if id == eventID {
			return true
		}
	}
	return false
}

// ToProcessedEvent converts window state to a processed event
func (ws *WindowState) ToProcessedEvent() *ProcessedEvent {
	return &ProcessedEvent{
		WindowID:     ws.WindowID,
		WindowStart:  ws.WindowStart,
		WindowEnd:    ws.WindowEnd,
		EventCount:   ws.EventCount,
		EventTypes:   ws.EventTypes,
		Sources:      ws.Sources,
		Aggregations: ws.Aggregations,
		FirstEvent:   ws.FirstEvent,
		LastEvent:    ws.LastEvent,
		ProcessedAt:  time.Now(),
		TenantID:     ws.TenantID,
	}
}

// ToJSON serializes window state to JSON
func (ws *WindowState) ToJSON() ([]byte, error) {
	return json.Marshal(ws)
}

// FromJSON deserializes window state from JSON
func FromJSON(data []byte) (*WindowState, error) {
	var state WindowState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// Checkpoint represents a consumer offset checkpoint
type Checkpoint struct {
	ConsumerGroup string    `json:"consumer_group"`
	Partition     int32     `json:"partition"`
	Offset        int64     `json:"offset"`
	Timestamp     time.Time `json:"timestamp"`
}

// CheckpointKey generates a Redis key for a checkpoint
func CheckpointKey(consumerGroup string, partition int32) string {
	return "checkpoint:" + consumerGroup + ":" + string(rune(partition))
}

// ToJSON serializes checkpoint to JSON
func (c *Checkpoint) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// CheckpointFromJSON deserializes checkpoint from JSON
func CheckpointFromJSON(data []byte) (*Checkpoint, error) {
	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, err
	}
	return &checkpoint, nil
}

// WatermarkState represents the current watermark state
type WatermarkState struct {
	Partition       int32     `json:"partition"`
	MaxEventTime    time.Time `json:"max_event_time"`
	CurrentWatermark time.Time `json:"current_watermark"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ToJSON serializes watermark state to JSON
func (wm *WatermarkState) ToJSON() ([]byte, error) {
	return json.Marshal(wm)
}

// WatermarkFromJSON deserializes watermark state from JSON
func WatermarkFromJSON(data []byte) (*WatermarkState, error) {
	var state WatermarkState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}
