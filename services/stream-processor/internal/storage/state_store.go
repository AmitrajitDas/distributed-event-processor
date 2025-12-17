package storage

import (
	"context"
	"time"

	"github.com/distributed-event-processor/services/stream-processor/internal/models"
)

// StateStore defines the interface for state management
type StateStore interface {
	// Window state management
	SaveWindowState(ctx context.Context, windowID string, state *models.WindowState, ttl time.Duration) error
	GetWindowState(ctx context.Context, windowID string) (*models.WindowState, error)
	DeleteWindowState(ctx context.Context, windowID string) error
	GetAllWindowStates(ctx context.Context) ([]*models.WindowState, error)

	// Checkpoint management
	SaveCheckpoint(ctx context.Context, consumerGroup string, partition int32, offset int64) error
	GetCheckpoint(ctx context.Context, consumerGroup string, partition int32) (int64, error)

	// Watermark management
	SaveWatermark(ctx context.Context, partition int32, watermark *models.WatermarkState) error
	GetWatermark(ctx context.Context, partition int32) (*models.WatermarkState, error)

	// Health check
	IsHealthy(ctx context.Context) bool

	// Cleanup
	Close() error
}
