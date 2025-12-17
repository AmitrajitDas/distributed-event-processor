package models

import (
	"fmt"
	"time"
)

// WindowType defines the type of window
type WindowType string

const (
	WindowTypeTumbling WindowType = "tumbling"
	WindowTypeSliding  WindowType = "sliding"
	WindowTypeSession  WindowType = "session"
)

// Window represents a time window for event processing
type Window struct {
	ID        string     `json:"id"`
	Type      WindowType `json:"type"`
	Start     time.Time  `json:"start"`
	End       time.Time  `json:"end"`
	Key       string     `json:"key,omitempty"` // For session windows (tenant_id:subject)
	IsClosed  bool       `json:"is_closed"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Duration returns the duration of the window
func (w *Window) Duration() time.Duration {
	return w.End.Sub(w.Start)
}

// Contains checks if a timestamp falls within the window
func (w *Window) Contains(timestamp time.Time) bool {
	return !timestamp.Before(w.Start) && timestamp.Before(w.End)
}

// String returns a string representation of the window
func (w *Window) String() string {
	if w.Key != "" {
		return fmt.Sprintf("%s[%s](%s-%s)", w.Type, w.Key, w.Start.Format(time.RFC3339), w.End.Format(time.RFC3339))
	}
	return fmt.Sprintf("%s(%s-%s)", w.Type, w.Start.Format(time.RFC3339), w.End.Format(time.RFC3339))
}

// TumblingWindow creates a tumbling window for a given timestamp
func TumblingWindow(timestamp time.Time, windowSize time.Duration) *Window {
	// Align window start to window size boundary
	windowStart := timestamp.Truncate(windowSize)
	windowEnd := windowStart.Add(windowSize)

	return &Window{
		ID:        fmt.Sprintf("tumbling-%d", windowStart.Unix()),
		Type:      WindowTypeTumbling,
		Start:     windowStart,
		End:       windowEnd,
		IsClosed:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// SlidingWindows creates all sliding windows that contain a given timestamp
func SlidingWindows(timestamp time.Time, windowSize, slideSize time.Duration) []*Window {
	windows := make([]*Window, 0)

	// Calculate how many windows this event belongs to
	numWindows := int(windowSize / slideSize)

	for i := 0; i < numWindows; i++ {
		// Calculate window start by going back i*slideSize from the aligned time
		alignedTime := timestamp.Truncate(slideSize)
		windowStart := alignedTime.Add(-time.Duration(i) * slideSize)
		windowEnd := windowStart.Add(windowSize)

		// Only include if timestamp falls within this window
		if !timestamp.Before(windowStart) && timestamp.Before(windowEnd) {
			windows = append(windows, &Window{
				ID:        fmt.Sprintf("sliding-%d-%d", windowStart.Unix(), windowEnd.Unix()),
				Type:      WindowTypeSliding,
				Start:     windowStart,
				End:       windowEnd,
				IsClosed:  false,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			})
		}
	}

	return windows
}

// SessionWindow creates a session window for a given key and timestamp
func SessionWindow(key string, timestamp time.Time, sessionGap time.Duration) *Window {
	// Session windows are created dynamically and extended as events arrive
	// Initial window starts at event time and ends at event time + gap
	return &Window{
		ID:        fmt.Sprintf("session-%s-%d", key, timestamp.Unix()),
		Type:      WindowTypeSession,
		Start:     timestamp,
		End:       timestamp.Add(sessionGap),
		Key:       key,
		IsClosed:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ExtendSessionWindow extends a session window with a new event timestamp
func ExtendSessionWindow(window *Window, eventTime time.Time, sessionGap time.Duration) {
	// Extend window start if event is earlier
	if eventTime.Before(window.Start) {
		window.Start = eventTime
	}

	// Extend window end to be event time + gap
	newEnd := eventTime.Add(sessionGap)
	if newEnd.After(window.End) {
		window.End = newEnd
	}

	window.UpdatedAt = time.Now()
}

// IsWindowClosed checks if a window should be closed based on watermark
func IsWindowClosed(window *Window, watermark time.Time) bool {
	// A window is closed when the watermark passes the window end time
	return watermark.After(window.End) || watermark.Equal(window.End)
}
