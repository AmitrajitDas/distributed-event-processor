# Distributed Event Processing Platform - Makefile

.PHONY: help build-all test-all lint-all clean docker-build docker-push run-local setup-dev

# Default target
help:
	@echo "Available targets:"
	@echo "  build-all       - Build all services"
	@echo "  test-all        - Run tests for all services"
	@echo "  lint-all        - Run linters for all services"
	@echo "  clean           - Clean build artifacts"
	@echo "  docker-build    - Build Docker images"
	@echo "  docker-push     - Push Docker images"
	@echo "  run-local       - Run all services locally"
	@echo "  setup-dev       - Setup development environment"
	@echo "  infra-up        - Start all infrastructure and monitoring services"
	@echo "  infra-down      - Stop all services"
	@echo "  infra-only      - Start only infrastructure services (no monitoring)"
	@echo "  monitoring-only - Start only monitoring services"

# Build targets
build-all: build-gateway build-processor build-store build-rule-engine

build-gateway:
	@echo "Building Event Gateway..."
	cd services/event-gateway && go build -o ../../bin/event-gateway ./cmd/gateway

build-processor:
	@echo "Building Stream Processor..."
	cd services/stream-processor && go build -o ../../bin/stream-processor ./cmd/processor

build-store:
	@echo "Building Event Store..."
	cd services/event-store && go build -o ../../bin/event-store ./cmd/store

build-rule-engine:
	@echo "Building Rule Engine..."
	cd services/rule-engine && go build -o ../../bin/rule-engine ./cmd/engine

build-schema-registry:
	@echo "Building Schema Registry..."
	cd services/schema-registry && ./mvnw clean package -DskipTests

build-output-manager:
	@echo "Building Output Manager..."
	cd services/output-manager && ./mvnw clean package -DskipTests

build-notification-service:
	@echo "Building Notification Service..."
	cd services/notification-service && ./mvnw clean package -DskipTests

# Test targets
test-all: test-go test-java

test-go:
	@echo "Running Go tests..."
	cd services/event-gateway && go test -v ./...
	cd services/stream-processor && go test -v ./...
	cd services/event-store && go test -v ./...
	cd services/rule-engine && go test -v ./...

test-java:
	@echo "Running Java tests..."
	cd services/schema-registry && ./mvnw test
	cd services/output-manager && ./mvnw test
	cd services/notification-service && ./mvnw test

# Lint targets
lint-all: lint-go lint-java

lint-go:
	@echo "Running Go linters..."
	cd services/event-gateway && golangci-lint run
	cd services/stream-processor && golangci-lint run
	cd services/event-store && golangci-lint run
	cd services/rule-engine && golangci-lint run

lint-java:
	@echo "Running Java linters..."
	cd services/schema-registry && ./mvnw checkstyle:check
	cd services/output-manager && ./mvnw checkstyle:check
	cd services/notification-service && ./mvnw checkstyle:check

# Docker targets
docker-build:
	@echo "Building Docker images..."
	docker build -t event-processor/gateway:latest -f services/event-gateway/Dockerfile .
	docker build -t event-processor/stream-processor:latest -f services/stream-processor/Dockerfile .
	docker build -t event-processor/event-store:latest -f services/event-store/Dockerfile .
	docker build -t event-processor/rule-engine:latest -f services/rule-engine/Dockerfile .
	docker build -t event-processor/schema-registry:latest -f services/schema-registry/Dockerfile .
	docker build -t event-processor/output-manager:latest -f services/output-manager/Dockerfile .
	docker build -t event-processor/notification-service:latest -f services/notification-service/Dockerfile .

docker-push:
	@echo "Pushing Docker images..."
	docker push event-processor/gateway:latest
	docker push event-processor/stream-processor:latest
	docker push event-processor/event-store:latest
	docker push event-processor/rule-engine:latest
	docker push event-processor/schema-registry:latest
	docker push event-processor/output-manager:latest
	docker push event-processor/notification-service:latest

# Infrastructure targets - Unified Docker Compose
infra-up:
	@echo "Starting all infrastructure and monitoring services..."
	docker compose up -d

infra-down:
	@echo "Stopping all services..."
	docker compose down

infra-restart:
	@echo "Restarting all services..."
	docker compose restart

infra-logs:
	@echo "Showing logs for all services..."
	docker compose logs -f

# Selective service management
infra-only:
	@echo "Starting only infrastructure services..."
	docker compose up -d zookeeper kafka redis postgres mongodb schema-registry kafka-ui minio createbuckets

monitoring-only:
	@echo "Starting only monitoring services..."
	docker compose up -d prometheus grafana loki promtail jaeger node-exporter cadvisor alertmanager redis-exporter postgres-exporter kafka-exporter

# Local development
run-local: infra-up
	@echo "Starting all services locally..."
	./bin/event-gateway &
	./bin/stream-processor &
	./bin/event-store &
	./bin/rule-engine &
	cd services/schema-registry && java -jar target/schema-registry-*.jar &
	cd services/output-manager && java -jar target/output-manager-*.jar &
	cd services/notification-service && java -jar target/notification-service-*.jar &

setup-dev:
	@echo "Setting up development environment..."
	mkdir -p bin
	mkdir -p logs
	go mod download
	cd services/schema-registry && ./mvnw dependency:resolve
	cd services/output-manager && ./mvnw dependency:resolve
	cd services/notification-service && ./mvnw dependency:resolve

# Utility targets
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf logs/
	cd services/event-gateway && go clean
	cd services/stream-processor && go clean
	cd services/event-store && go clean
	cd services/rule-engine && go clean
	cd services/schema-registry && ./mvnw clean
	cd services/output-manager && ./mvnw clean
	cd services/notification-service && ./mvnw clean

# Development workflow
dev-setup: setup-dev infra-up

# Format code
format:
	@echo "Formatting code..."
	cd services/event-gateway && go fmt ./...
	cd services/stream-processor && go fmt ./...
	cd services/event-store && go fmt ./...
	cd services/rule-engine && go fmt ./...

# Generate protobuf files
generate-proto:
	@echo "Generating protobuf files..."
	protoc --go_out=. --go-grpc_out=. shared/proto/*.proto

# Database migrations
migrate-up:
	@echo "Running database migrations..."
	# Add migration commands here

migrate-down:
	@echo "Rolling back database migrations..."
	# Add rollback commands here

# Testing and debugging
test-kafka:
	@echo "Testing Kafka setup..."
	docker exec kafka kafka-topics --create --topic test-events --bootstrap-server localhost:9092 --partitions 3 --replication-factor 1 --if-not-exists
	echo 'Hello Event Processing World!' | docker exec -i kafka kafka-console-producer --topic test-events --bootstrap-server localhost:9092
	docker exec kafka kafka-console-consumer --topic test-events --from-beginning --bootstrap-server localhost:9092 --max-messages 1

test-databases:
	@echo "Testing database connections..."
	docker exec postgres psql -U eventuser -d event_processor -c "SELECT version();"
	docker exec redis redis-cli ping
	docker exec mongodb mongosh --eval "db.runCommand('ismaster')" --quiet

# Health checks
health-check:
	@echo "Checking service health..."
	@echo "Kafka:" && curl -s http://localhost:8080 > /dev/null && echo "✅ Kafka UI accessible" || echo "❌ Kafka UI failed"
	@echo "Prometheus:" && curl -s http://localhost:9090/-/healthy > /dev/null && echo "✅ Prometheus healthy" || echo "❌ Prometheus failed"
	@echo "Grafana:" && curl -s http://localhost:3000/api/health > /dev/null && echo "✅ Grafana healthy" || echo "❌ Grafana failed"
	@echo "Loki:" && curl -s http://localhost:3100/ready > /dev/null && echo "✅ Loki ready" || echo "❌ Loki failed"
	@echo "Jaeger:" && curl -s http://localhost:16686/ > /dev/null && echo "✅ Jaeger accessible" || echo "❌ Jaeger failed"

# Service URLs
show-urls:
	@echo "🌐 Service URLs:"
	@echo "  Kafka UI:        http://localhost:8080"
	@echo "  Grafana:         http://localhost:3000 (admin/admin)"
	@echo "  Prometheus:      http://localhost:9090"
	@echo "  Loki:            http://localhost:3100"
	@echo "  Jaeger:          http://localhost:16686"
	@echo "  MinIO Console:   http://localhost:9001 (minioadmin/minioadmin)"
	@echo "  AlertManager:    http://localhost:9093"


