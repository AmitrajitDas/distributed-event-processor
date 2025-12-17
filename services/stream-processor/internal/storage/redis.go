package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/distributed-event-processor/services/stream-processor/internal/config"
	"github.com/distributed-event-processor/services/stream-processor/internal/models"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisStore handles Redis operations for state management
type RedisStore struct {
	client *redis.Client
	config config.RedisConfig
	logger *zap.Logger
}

// NewRedisStore creates a new Redis store instance
func NewRedisStore(cfg config.RedisConfig, logger *zap.Logger) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Address,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Connected to Redis",
		zap.String("address", cfg.Address),
		zap.Int("db", cfg.DB))

	return &RedisStore{
		client: client,
		config: cfg,
		logger: logger,
	}, nil
}

// SaveWindowState saves window state to Redis with TTL
func (r *RedisStore) SaveWindowState(ctx context.Context, windowID string, state *models.WindowState, ttl time.Duration) error {
	key := r.windowKey(windowID)

	// Serialize state to JSON
	data, err := state.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize window state: %w", err)
	}

	// Save to Redis with TTL
	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		r.logger.Error("Failed to save window state",
			zap.String("window_id", windowID),
			zap.Error(err))
		return fmt.Errorf("failed to save window state: %w", err)
	}

	r.logger.Debug("Saved window state",
		zap.String("window_id", windowID),
		zap.Duration("ttl", ttl))

	return nil
}

// GetWindowState retrieves window state from Redis
func (r *RedisStore) GetWindowState(ctx context.Context, windowID string) (*models.WindowState, error) {
	key := r.windowKey(windowID)

	// Get from Redis
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			// Key doesn't exist
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get window state: %w", err)
	}

	// Deserialize from JSON
	state, err := models.FromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize window state: %w", err)
	}

	return state, nil
}

// DeleteWindowState deletes window state from Redis
func (r *RedisStore) DeleteWindowState(ctx context.Context, windowID string) error {
	key := r.windowKey(windowID)

	if err := r.client.Del(ctx, key).Err(); err != nil {
		r.logger.Error("Failed to delete window state",
			zap.String("window_id", windowID),
			zap.Error(err))
		return fmt.Errorf("failed to delete window state: %w", err)
	}

	r.logger.Debug("Deleted window state", zap.String("window_id", windowID))
	return nil
}

// SaveCheckpoint saves a consumer offset checkpoint
func (r *RedisStore) SaveCheckpoint(ctx context.Context, consumerGroup string, partition int32, offset int64) error {
	key := r.checkpointKey(consumerGroup, partition)

	checkpoint := models.Checkpoint{
		ConsumerGroup: consumerGroup,
		Partition:     partition,
		Offset:        offset,
		Timestamp:     time.Now(),
	}

	// Serialize checkpoint
	data, err := checkpoint.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize checkpoint: %w", err)
	}

	// Save to Redis (no TTL for checkpoints)
	if err := r.client.Set(ctx, key, data, 0).Err(); err != nil {
		r.logger.Error("Failed to save checkpoint",
			zap.String("consumer_group", consumerGroup),
			zap.Int32("partition", partition),
			zap.Int64("offset", offset),
			zap.Error(err))
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	r.logger.Debug("Saved checkpoint",
		zap.String("consumer_group", consumerGroup),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset))

	return nil
}

// GetCheckpoint retrieves a consumer offset checkpoint
func (r *RedisStore) GetCheckpoint(ctx context.Context, consumerGroup string, partition int32) (int64, error) {
	key := r.checkpointKey(consumerGroup, partition)

	// Get from Redis
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			// No checkpoint exists
			return -1, nil
		}
		return -1, fmt.Errorf("failed to get checkpoint: %w", err)
	}

	// Deserialize checkpoint
	checkpoint, err := models.CheckpointFromJSON(data)
	if err != nil {
		return -1, fmt.Errorf("failed to deserialize checkpoint: %w", err)
	}

	return checkpoint.Offset, nil
}

// SaveWatermark saves the current watermark state
func (r *RedisStore) SaveWatermark(ctx context.Context, partition int32, watermark *models.WatermarkState) error {
	key := r.watermarkKey(partition)

	// Serialize watermark
	data, err := watermark.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize watermark: %w", err)
	}

	// Save to Redis
	if err := r.client.Set(ctx, key, data, 24*time.Hour).Err(); err != nil {
		r.logger.Error("Failed to save watermark",
			zap.Int32("partition", partition),
			zap.Error(err))
		return fmt.Errorf("failed to save watermark: %w", err)
	}

	r.logger.Debug("Saved watermark",
		zap.Int32("partition", partition),
		zap.Time("watermark", watermark.CurrentWatermark))

	return nil
}

// GetWatermark retrieves the current watermark state
func (r *RedisStore) GetWatermark(ctx context.Context, partition int32) (*models.WatermarkState, error) {
	key := r.watermarkKey(partition)

	// Get from Redis
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			// No watermark exists
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get watermark: %w", err)
	}

	// Deserialize watermark
	watermark, err := models.WatermarkFromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize watermark: %w", err)
	}

	return watermark, nil
}

// GetAllWindowStates retrieves all window states (for recovery)
func (r *RedisStore) GetAllWindowStates(ctx context.Context) ([]*models.WindowState, error) {
	pattern := r.windowKey("*")

	// Scan for all window keys
	var cursor uint64
	var keys []string

	for {
		var err error
		var batch []string
		batch, cursor, err = r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan window keys: %w", err)
		}

		keys = append(keys, batch...)

		if cursor == 0 {
			break
		}
	}

	// Retrieve all states
	states := make([]*models.WindowState, 0, len(keys))
	for _, key := range keys {
		data, err := r.client.Get(ctx, key).Bytes()
		if err != nil {
			if err == redis.Nil {
				continue // Key expired between scan and get
			}
			r.logger.Warn("Failed to get window state", zap.String("key", key), zap.Error(err))
			continue
		}

		state, err := models.FromJSON(data)
		if err != nil {
			r.logger.Warn("Failed to deserialize window state", zap.String("key", key), zap.Error(err))
			continue
		}

		states = append(states, state)
	}

	return states, nil
}

// IsHealthy checks if Redis connection is healthy
func (r *RedisStore) IsHealthy(ctx context.Context) bool {
	if err := r.client.Ping(ctx).Err(); err != nil {
		r.logger.Error("Redis health check failed", zap.Error(err))
		return false
	}
	return true
}

// Close closes the Redis connection
func (r *RedisStore) Close() error {
	r.logger.Info("Closing Redis connection")
	return r.client.Close()
}

// Helper methods for key generation
func (r *RedisStore) windowKey(windowID string) string {
	return fmt.Sprintf("window:%s", windowID)
}

func (r *RedisStore) checkpointKey(consumerGroup string, partition int32) string {
	return fmt.Sprintf("checkpoint:%s:%d", consumerGroup, partition)
}

func (r *RedisStore) watermarkKey(partition int32) string {
	return fmt.Sprintf("watermark:%d", partition)
}
