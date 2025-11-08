# Distributed Event Processing Platform

A high-performance, distributed event processing platform designed to handle real-time event streams with complex pattern detection and analytics capabilities.

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.21+
- Java 17+ (for Spring Boot services)
- Node.js 18+ (for monitoring tools)

### Local Development Setup

```bash
# Clone and setup
git clone <repository-url>
cd distributed-event-processor

# Start infrastructure services
docker-compose up -d kafka redis postgres mongodb

# Build and run services
make build-all
make run-local
```

## 🏗️ Architecture

This platform implements a microservices architecture with the following components:

- **Event Gateway** (Go): Multi-protocol event ingestion
- **Stream Processor** (Go): Real-time event processing engine
- **Schema Registry** (Spring Boot): Schema management and evolution
- **Rule Engine** (Go): Dynamic processing rules and CEP
- **Event Store** (Go): Event persistence and replay
- **Output Manager** (Spring Boot): Multiple sink connectors
- **Notification Service** (Spring Boot): Multi-channel notification system

## 📊 Performance

- **Throughput**: 100,000+ events/second
- **Latency**: Sub-millisecond processing (p99 < 1ms)
- **Availability**: 99.9% uptime target
- **Scaling**: Auto-scale based on event volume

## 🎯 Key Features

### Real-time Processing

- Stream transformations and filtering
- Windowing operations (time, count, session)
- Complex event pattern detection
- Stream joins and aggregations

### Enterprise Features

- Multi-tenant isolation
- Schema evolution support
- Exactly-once processing guarantees
- Dead letter queue handling

### Observability

- Comprehensive metrics (Prometheus)
- Distributed tracing
- Centralized logging (Loki)
- Real-time dashboards (Grafana)

## 📁 Project Structure

```
├── services/              # Microservices
│   ├── event-gateway/     # Event ingestion service
│   ├── stream-processor/  # Core processing engine
│   ├── schema-registry/   # Schema management
│   ├── rule-engine/       # CEP and dynamic rules
│   ├── event-store/       # Event persistence
│   ├── output-manager/    # Output connectors
│   └── notification-service/ # Multi-channel notifications
├── infrastructure/        # Deployment & ops
│   ├── docker/           # Docker compositions
│   ├── kubernetes/       # K8s manifests
│   └── monitoring/       # Observability stack
├── shared/               # Common libraries
│   ├── proto/           # gRPC definitions
│   ├── schemas/         # Event schemas
│   └── libraries/       # Shared utilities
└── docs/                # Documentation
    ├── architecture/    # System design
    ├── api/            # API docs
    └── deployment/     # Operations guides
```

## 🔧 Development

### Build Commands

```bash
make build-all          # Build all services
make test-all           # Run all tests
make lint-all           # Run linters
make docker-build       # Build Docker images
```

### Running Services

```bash
make run-gateway        # Start event gateway
make run-processor      # Start stream processor
make run-schema         # Start schema registry
make run-monitoring     # Start observability stack
```

## 📖 Documentation

- [Architecture Overview](docs/architecture/README.md)
- [API Documentation](docs/api/README.md)
- [Deployment Guide](docs/deployment/README.md)
- [Development Setup](docs/development.md)

## 🎤 Interview Highlights

This project demonstrates advanced distributed systems concepts:

- **System Design**: Microservices, event sourcing, CQRS
- **Performance**: High throughput, low latency processing
- **Reliability**: Fault tolerance, exactly-once processing
- **Scalability**: Horizontal scaling, auto-scaling
- **Observability**: Comprehensive monitoring and alerting

Perfect for showcasing enterprise-level system design skills in technical interviews.

## 📜 License

MIT License - see [LICENSE](LICENSE) for details.
