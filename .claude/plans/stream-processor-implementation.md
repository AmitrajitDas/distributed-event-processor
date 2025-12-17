# Stream Processor Implementation Plan

## Overview
Build the Stream Processor service - the core real-time event processing engine that consumes events from Kafka, applies windowing operations, maintains state in Redis, and forwards processed events.

## Architecture

### Service Flow
```
Kafka (events topic) → Consumer → Processor (Windowing/Aggregation) → Redis (State) → Output
                                        ↓
                                   Metrics/Logging
```

### Core Components

1. **Kafka Consumer**
   - Consume from 'events' topic
   - Consumer group for scalability
   - Exactly-once semantics with offset management
   - Backpressure handling

2. **Event Processor**
   - Windowing operations (tumbling, sliding, session)
   - Event-time processing with watermarks
   - Aggregations and transformations
   - Pattern detection (basic CEP)

3. **State Manager (Redis)**
   - Window state persistence
   - TTL-based state expiration
   - Checkpointing for fault tolerance
   - State recovery on restart

4. **Output Handler**
   - Forward processed events (to Kafka or next stage)
   - Dead letter queue for failures
   - Metrics emission

## Directory Structure

```
services/stream-processor/
├── cmd/
│   └── processor/
│       └── main.go                 # Entry point
├── internal/
│   ├── config/
│   │   ├── config.go              # Configuration structs & loading
│   │   └── config_test.go
│   ├── kafka/
│   │   ├── consumer.go            # Kafka consumer
│   │   ├── consumer_test.go
│   │   └── offset_manager.go      # Offset management
│   ├── storage/
│   │   ├── redis.go               # Redis client wrapper
│   │   ├── redis_test.go
│   │   └── state_store.go         # State management interface
│   ├── processor/
│   │   ├── processor.go           # Main processing logic
│   │   ├── processor_test.go
│   │   ├── windowing.go           # Window implementations
│   │   ├── windowing_test.go
│   │   ├── aggregator.go          # Aggregation logic
│   │   └── watermark.go           # Watermark handling
│   ├── models/
│   │   ├── event.go               # Event models (can reuse from gateway)
│   │   ├── window.go              # Window models
│   │   └── state.go               # State models
│   └── metrics/
│       ├── metrics.go             # Prometheus metrics
│       └── metrics_test.go
├── config.yaml                     # Default configuration
├── env.example                     # Environment variables example
├── Dockerfile                      # Container definition
├── go.mod
├── go.sum
└── README.md                       # Documentation

```

## Implementation Details

### 1. Configuration (internal/config/config.go)

```go
type Config struct {
    Environment string
    Kafka       KafkaConfig
    Redis       RedisConfig
    Processing  ProcessingConfig
    Metrics     MetricsConfig
}

type KafkaConfig struct {
    Brokers       []string
    Topic         string
    ConsumerGroup string
    AutoCommit    bool
    StartOffset   string // earliest, latest
}

type RedisConfig struct {
    Address      string
    Password     string
    DB           int
    PoolSize     int
    MaxRetries   int
}

type ProcessingConfig struct {
    Workers           int
    WindowType        string // tumbling, sliding, session
    WindowSize        int    // seconds
    WindowSlide       int    // seconds (for sliding windows)
    SessionGap        int    // seconds (for session windows)
    WatermarkDelay    int    // seconds
    MaxOutOfOrder     int    // seconds
}
```

**Environment prefix:** `PROCESSOR_`

### 2. Kafka Consumer (internal/kafka/consumer.go)

**Features:**
- Consumer group support
- Manual offset management for exactly-once
- Graceful shutdown
- Error handling with retries
- Backpressure handling (pause/resume)

**Key methods:**
- `NewConsumer(config, logger) (*Consumer, error)`
- `Start(ctx context.Context, handler EventHandler) error`
- `Close() error`
- `Commit(ctx context.Context) error`

### 3. Redis State Store (internal/storage/redis.go)

**Features:**
- Connection pooling
- Automatic reconnection
- TTL-based state expiration
- Batch operations for efficiency

**Key methods:**
- `NewRedisStore(config, logger) (*RedisStore, error)`
- `SaveWindowState(windowID string, state *WindowState, ttl time.Duration) error`
- `GetWindowState(windowID string) (*WindowState, error)`
- `DeleteWindowState(windowID string) error`
- `SaveCheckpoint(consumerGroup string, partition int32, offset int64) error`
- `GetCheckpoint(consumerGroup string, partition int32) (int64, error)`

### 4. Windowing (internal/processor/windowing.go)

**Window Types:**

1. **Tumbling Window**
   - Fixed size (e.g., 5 minutes)
   - Non-overlapping
   - Start: event_timestamp - (event_timestamp % window_size)
   - End: Start + window_size

2. **Sliding Window**
   - Fixed size with slide interval
   - Overlapping
   - Multiple windows per event
   - Example: 10-minute window, 5-minute slide

3. **Session Window**
   - Dynamic size based on inactivity gap
   - Closes after no events for gap duration
   - Per-key windows (grouped by tenant_id or subject)

**Key methods:**
- `AssignToWindows(event *Event) []Window`
- `IsWindowClosed(window Window, watermark time.Time) bool`
- `TriggerWindow(window Window) error`

### 5. Event Processor (internal/processor/processor.go)

**Core Processing Loop:**
```
1. Consume event from Kafka
2. Parse and validate event
3. Assign to window(s)
4. Update window state in Redis
5. Check watermark
6. Trigger closed windows
7. Emit aggregated results
8. Commit offset
```

**Key methods:**
- `NewProcessor(config, consumer, stateStore, logger) *Processor`
- `Start(ctx context.Context) error`
- `Stop() error`
- `ProcessEvent(event *Event) error`
- `ProcessWindow(window Window) error`

### 6. Metrics (internal/metrics/metrics.go)

**Prometheus Metrics:**
- `events_consumed_total` - Total events consumed by type
- `events_processed_total` - Total events processed successfully
- `events_failed_total` - Total failed events by reason
- `processing_duration_seconds` - Event processing latency histogram
- `window_triggers_total` - Total window triggers by type
- `window_size_events` - Number of events per window
- `redis_operations_total` - Redis operation count by operation type
- `redis_errors_total` - Redis errors by operation type
- `consumer_lag` - Consumer lag per partition
- `watermark_delay_seconds` - Current watermark delay

### 7. Main Entry Point (cmd/processor/main.go)

**Initialization Flow:**
1. Load configuration
2. Initialize logger
3. Initialize Redis client
4. Initialize Kafka consumer
5. Initialize processor
6. Start HTTP server for health/metrics
7. Start processing loop
8. Handle graceful shutdown

## Configuration Files

### config.yaml
```yaml
environment: development

kafka:
  brokers:
    - "localhost:9092"
  topic: "events"
  consumer_group: "stream-processor"
  auto_commit: false
  start_offset: "latest"

redis:
  address: "localhost:6379"
  password: ""
  db: 0
  pool_size: 10
  max_retries: 3

processing:
  workers: 4
  window_type: "tumbling"
  window_size: 300 # 5 minutes
  window_slide: 60 # 1 minute (for sliding windows)
  session_gap: 600 # 10 minutes (for session windows)
  watermark_delay: 10 # 10 seconds
  max_out_of_order: 60 # 1 minute

metrics:
  enabled: true
  port: 9091
  path: "/metrics"

health:
  port: 8091
  path: "/health"
```

### env.example
```bash
PROCESSOR_ENVIRONMENT=development
PROCESSOR_KAFKA_BROKERS=localhost:9092
PROCESSOR_KAFKA_TOPIC=events
PROCESSOR_KAFKA_CONSUMER_GROUP=stream-processor
PROCESSOR_REDIS_ADDRESS=localhost:6379
PROCESSOR_PROCESSING_WINDOW_TYPE=tumbling
PROCESSOR_PROCESSING_WINDOW_SIZE=300
```

## Testing Strategy

### Unit Tests
- Configuration loading and validation
- Window assignment logic
- Watermark calculations
- State serialization/deserialization
- Metrics recording

### Integration Tests
- Kafka consumer with test containers
- Redis state persistence
- End-to-end event processing
- Window triggering and aggregation
- Graceful shutdown

### Test Coverage Target
- Minimum 80% code coverage
- All critical paths tested
- Error scenarios covered

## Dependencies (go.mod)

```
- github.com/IBM/sarama (Kafka client)
- github.com/redis/go-redis/v9 (Redis client)
- github.com/spf13/viper (Configuration)
- go.uber.org/zap (Logging)
- github.com/prometheus/client_golang (Metrics)
- github.com/stretchr/testify (Testing)
- github.com/google/uuid (UUID generation)
```

## Implementation Phases

### Phase 1: Foundation (Core Infrastructure)
1. Project structure setup
2. Configuration management (config.go, config.yaml)
3. Models (event.go, window.go, state.go)
4. Redis client wrapper (redis.go)
5. Basic tests

### Phase 2: Kafka Consumer
1. Consumer implementation with consumer group
2. Offset management
3. Error handling and retries
4. Consumer tests

### Phase 3: Windowing Logic
1. Tumbling window implementation
2. Sliding window implementation
3. Session window implementation
4. Watermark handling
5. Window tests

### Phase 4: Event Processing
1. Main processor implementation
2. Event to window assignment
3. State management integration
4. Window triggering and aggregation
5. Processor tests

### Phase 5: Observability
1. Prometheus metrics
2. Health check endpoints
3. Structured logging
4. Metrics tests

### Phase 6: Main Application
1. Main entry point (main.go)
2. HTTP server for health/metrics
3. Graceful shutdown
4. Integration tests

### Phase 7: Documentation & Deployment
1. README with usage instructions
2. Dockerfile
3. Docker Compose integration
4. Testing guide

## Success Criteria

- ✅ Service successfully consumes events from Kafka
- ✅ Events are correctly assigned to windows
- ✅ Window state is persisted in Redis
- ✅ Watermarks correctly trigger window closures
- ✅ All three window types (tumbling, sliding, session) work
- ✅ Exactly-once semantics with manual offset commits
- ✅ Graceful shutdown without data loss
- ✅ Comprehensive metrics exposed
- ✅ Health checks functional
- ✅ 80%+ test coverage
- ✅ Integration with existing infrastructure (Kafka, Redis, monitoring)

## Future Enhancements (Not in Initial Scope)

- Rule Engine integration (will be separate service)
- Advanced CEP patterns (complex event correlation)
- Multiple output sinks
- State snapshots and recovery
- Horizontal scalability with partition assignment
- Performance benchmarking and optimization
