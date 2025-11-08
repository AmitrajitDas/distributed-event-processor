# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **Distributed Event Processing Platform** - a high-performance, microservices-based system designed for real-time event stream processing with complex event pattern detection capabilities. It's built as an interview-ready project showcasing enterprise-level system design patterns.

**Key capabilities:**
- Process 100K+ events/second with sub-millisecond latency
- Support complex event processing (CEP) with windowing and pattern matching
- Multi-tenant isolation with exactly-once processing guarantees
- Comprehensive observability with Prometheus, Grafana, Loki, and Jaeger

## Technology Stack

**Go Services (High-Performance):**
- `event-gateway` - Multi-protocol event ingestion (HTTP, gRPC, WebSocket)
- `stream-processor` - Core real-time processing engine
- `rule-engine` - Dynamic CEP rules and pattern matching
- `event-store` - Event persistence and replay

**Spring Boot Services (Enterprise Features):**
- `schema-registry` - Schema versioning and compatibility
- `output-manager` - Multiple sink connectors
- `notification-service` - Multi-channel notifications (Email, SMS, Slack)

**Infrastructure:**
- Apache Kafka - Event streaming backbone
- Redis - Low-latency state storage
- PostgreSQL - Structured event history
- MongoDB - Configuration storage
- MinIO - S3-compatible cold storage

## Essential Commands

### Development Setup
```bash
# Initial setup - creates directories and downloads dependencies
make setup-dev

# Start all infrastructure (Kafka, Redis, Postgres, MongoDB, monitoring stack)
make infra-up

# Start only infrastructure services without monitoring
make infra-only

# Start only monitoring services (Prometheus, Grafana, Loki, Jaeger)
make monitoring-only

# Stop all services
make infra-down
```

### Building Services
```bash
# Build all Go services
make build-all

# Build individual services
make build-gateway          # Event Gateway
make build-processor        # Stream Processor
make build-store           # Event Store
make build-rule-engine     # Rule Engine

# Build Spring Boot services
make build-schema-registry
make build-output-manager
make build-notification-service

# Build Docker images for all services
make docker-build
```

### Testing
```bash
# Run all tests (Go + Java)
make test-all

# Run Go tests only
make test-go

# Run Java tests only
make test-java

# Test individual Go service (must be in service directory)
cd services/event-gateway && go test -v ./...

# Run specific test
cd services/event-gateway && go test -v ./internal/api/http/handlers -run TestEventHandler
```

### Linting and Formatting
```bash
# Run all linters
make lint-all

# Format Go code
make format

# Run golangci-lint for specific service
cd services/event-gateway && golangci-lint run
```

### Infrastructure Testing
```bash
# Test Kafka setup and connectivity
make test-kafka

# Test database connections
make test-databases

# Health check all services
make health-check

# Show all service URLs
make show-urls
```

## Architecture and Data Flow

### Service Communication Pattern

1. **Ingestion Flow:**
   ```
   Client → Event Gateway (HTTP/gRPC) → Kafka Topics → Stream Processor
   ```

2. **Processing Flow:**
   ```
   Stream Processor → Rule Engine → Transformations → Windowing/CEP → Output Manager
                   ↓
              Redis (State)
   ```

3. **Storage Tiers:**
   - **Hot (7 days):** Kafka - Real-time streaming
   - **Warm (90 days):** PostgreSQL - Queryable history
   - **Cold (indefinite):** S3/MinIO - Long-term archive
   - **State:** Redis - Processing windows and aggregations

### Key Architectural Patterns

- **Event Sourcing:** Immutable event log as source of truth
- **CQRS:** Separate read/write data models
- **Exactly-Once Processing:** Idempotent operations with Kafka transactions
- **Backpressure Handling:** Flow control when consumers lag
- **Circuit Breakers:** Prevent cascade failures
- **Multi-tenancy:** Isolation at partition and processing levels

## Project Structure

```
services/
├── event-gateway/       # Go - Event ingestion (port 8090)
├── stream-processor/    # Go - Core processing engine
├── schema-registry/     # Spring Boot - Schema management (port 8081)
├── rule-engine/        # Go - Dynamic rules and CEP
├── event-store/        # Go - Event persistence
├── output-manager/     # Spring Boot - Output connectors
└── notification-service/ # Spring Boot - Multi-channel notifications

infrastructure/
├── docker/             # Docker Compose files
├── kubernetes/         # K8s manifests
└── monitoring/         # Prometheus, Grafana, Loki configs
    ├── prometheus/     # Metrics collection config
    ├── grafana/        # Dashboards and datasources
    ├── loki/          # Log aggregation config
    └── alertmanager/  # Alert routing config
```

### Go Service Structure (Standard Pattern)
Each Go service follows this internal structure:
```
cmd/[service]/          # Entry point (main.go)
internal/
  ├── api/             # API layer (HTTP/gRPC)
  │   ├── handlers/    # Request handlers
  │   ├── middleware/  # Middleware components
  │   └── server/      # Server initialization
  ├── config/          # Configuration management
  ├── kafka/           # Kafka integration
  └── models/          # Data models
```

## Configuration Management

### Go Services
- Use **Viper** for configuration management
- Config precedence: Environment variables > config.yaml > defaults
- Environment variable prefix: `GATEWAY_`, `PROCESSOR_`, etc.
- Example: `GATEWAY_KAFKA_BROKERS=localhost:9092`
- Config files expected in: `.` or `/etc/[service-name]/`

### Spring Boot Services
- Standard `application.properties` or `application.yml`
- Profile-based configuration: `application-{profile}.yml`
- Environment variables override application properties

## Important Service Ports

```
Event Gateway:        8090
Kafka:               9092
Kafka UI:            8080
Schema Registry:     8081
Redis:               6379
PostgreSQL:          5432
MongoDB:            27017
Prometheus:          9090
Grafana:             3000 (admin/event123)
Loki:               3100
Jaeger:             16686
MinIO Console:       9001 (minioadmin/minioadmin)
AlertManager:        9093
```

## Development Workflow

### Adding a New Event Type
1. Define schema in `shared/schemas/`
2. Register schema with Schema Registry
3. Update event models in relevant services
4. Add processing rules in Rule Engine
5. Configure output routing in Output Manager

### Working with Kafka
- Default topic: `events`
- Partitions: 12 (configurable via `KAFKA_NUM_PARTITIONS`)
- Replication factor: 1 (development)
- Use Kafka UI (http://localhost:8080) for topic management and message inspection

### Monitoring and Debugging
- **Grafana dashboards:** http://localhost:3000 - Pre-configured dashboards for all services
- **Prometheus queries:** http://localhost:9090 - Query metrics directly
- **Jaeger traces:** http://localhost:16686 - Distributed tracing
- **Loki logs:** Query via Grafana - Centralized log aggregation
- **Application logs:** Services use zap (Go) and logback (Spring Boot) with JSON formatting

### Performance Tuning
Key configuration parameters:
- `kafka.batch_size`: Controls Kafka producer batching (default: 100)
- `kafka.required_acks`: Acknowledgment level - 0 (none), 1 (leader), -1 (all replicas)
- `rate_limit.requests_per_second`: Gateway rate limiting threshold
- `rate_limit.burst_size`: Burst allowance for rate limiting

## Common Patterns and Conventions

### Error Handling
- Go services: Return errors up the stack, log at boundaries
- Use structured logging with correlation IDs for request tracing
- Dead letter queues for unprocessable events

### Testing
- **Unit tests:** Focus on business logic, mock external dependencies
- **Integration tests:** Test service boundaries with test containers
- **Load tests:** Use tools like k6 or Gatling for performance validation
- Target: 80%+ code coverage for all services

### Git Workflow
- Feature branches: `feature/description`
- Bug fixes: `fix/description`
- Commit message format: `type(scope): description`
- Types: feat, fix, refactor, test, docs, chore

## Important Implementation Details

### Exactly-Once Processing
- Kafka transactions enabled for producer/consumer
- Idempotent producer configuration: `enable.idempotence=true`
- Consumer group management with offset commits
- Deduplication at event ID level

### State Management
- Redis used for windowing state (tumbling, sliding, session windows)
- State persisted with TTL based on window duration
- Checkpointing for fault tolerance
- State recovery on processor restart

### Event Schema Evolution
- Backward compatibility required for all schema changes
- Version every schema change
- Schema Registry validates compatibility before registration
- Use Avro or Protobuf for schema-defined events

### Windowing Operations
- **Tumbling windows:** Fixed-size, non-overlapping time windows
- **Sliding windows:** Fixed-size, overlapping windows
- **Session windows:** Dynamic size based on inactivity gap
- All windows use event time, not processing time
- Watermarks handle out-of-order events (configurable delay)

## Known Limitations

- Current implementation uses single Kafka broker (for development)
- Redis is single-node (not clustered)
- No authentication on infrastructure services (development mode)
- Schema Registry is Confluent Community edition (not custom Spring Boot implementation yet)
- Cold storage uses MinIO instead of AWS S3 (for local development)

## Debugging Tips

### Kafka Issues
```bash
# Check Kafka logs
docker logs kafka

# List topics
docker exec kafka kafka-topics --list --bootstrap-server localhost:9092

# Consume from topic
docker exec kafka kafka-console-consumer --topic events --from-beginning --bootstrap-server localhost:9092
```

### Database Connection Issues
```bash
# Test PostgreSQL
docker exec postgres psql -U eventuser -d event_processor -c "SELECT 1;"

# Test Redis
docker exec redis redis-cli ping

# Test MongoDB
docker exec mongodb mongosh --eval "db.runCommand('ismaster')" --quiet
```

### Service Not Starting
1. Check if infrastructure services are running: `docker compose ps`
2. Verify configuration: Check environment variables and config files
3. Check logs: `docker compose logs [service-name]`
4. Verify port availability: `lsof -i :[port]`
