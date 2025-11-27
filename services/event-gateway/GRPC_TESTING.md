# gRPC Testing Guide

This guide explains how to test the Event Gateway gRPC service.

## Prerequisites

### 1. Install gRPC Tools

#### grpcurl (for command-line testing)
```bash
# macOS
brew install grpcurl

# Linux
wget https://github.com/fullstorydev/grpcurl/releases/download/v1.8.9/grpcurl_1.8.9_linux_x86_64.tar.gz
tar -xvf grpcurl_1.8.9_linux_x86_64.tar.gz
sudo mv grpcurl /usr/local/bin/

# Verify installation
grpcurl --version
```

#### Evans (interactive gRPC client)
```bash
# macOS
brew install evans

# Linux
wget https://github.com/ktr0731/evans/releases/download/v0.10.11/evans_linux_amd64.tar.gz
tar -xvf evans_linux_amd64.tar.gz
sudo mv evans /usr/local/bin/

# Verify installation
evans --version
```

### 2. Start Infrastructure

```bash
# Start Kafka and other dependencies
make infra-up

# Or just Kafka
docker compose -f infrastructure/docker/docker-compose.yml up -d kafka zookeeper
```

### 3. Build and Start the Service

```bash
# Build the service
cd services/event-gateway
go build -o bin/event-gateway cmd/gateway/main.go

# Run the service
./bin/event-gateway
```

The gRPC server will start on port **9090** by default.

---

## Testing Methods

### Method 1: Using grpcurl (Command-Line)

**grpcurl** is like `curl` but for gRPC.

#### List Available Services

```bash
grpcurl -plaintext localhost:9090 list
```

**Expected output:**
```
events.v1.EventGateway
grpc.reflection.v1alpha.ServerReflection
```

#### List Available Methods

```bash
grpcurl -plaintext localhost:9090 list events.v1.EventGateway
```

**Expected output:**
```
events.v1.EventGateway.HealthCheck
events.v1.EventGateway.IngestEvent
events.v1.EventGateway.IngestEventBatch
events.v1.EventGateway.StreamEvents
events.v1.EventGateway.ValidateEvent
```

#### Describe a Method

```bash
grpcurl -plaintext localhost:9090 describe events.v1.EventGateway.IngestEvent
```

#### Test HealthCheck

```bash
grpcurl -plaintext \
  -d '{"detailed": true}' \
  localhost:9090 \
  events.v1.EventGateway/HealthCheck
```

**Expected response:**
```json
{
  "status": "SERVICE_STATUS_HEALTHY",
  "version": "1.0.0",
  "components": {
    "kafka": {
      "status": "HEALTH_STATUS_UP",
      "message": "Kafka producer is healthy",
      "lastCheck": "2024-01-15T10:30:00Z"
    }
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### Test IngestEvent

```bash
grpcurl -plaintext \
  -d '{
    "event": {
      "type": "user.created",
      "source": "auth-service",
      "tenant_id": "tenant-1",
      "data": {
        "user_id": "12345",
        "email": "test@example.com",
        "name": "John Doe"
      },
      "schema_version": "1.0",
      "correlation_id": "corr-123",
      "priority": 5,
      "metadata": {
        "region": "us-east-1"
      }
    },
    "wait_for_ack": true
  }' \
  localhost:9090 \
  events.v1.EventGateway/IngestEvent
```

**Expected response:**
```json
{
  "eventId": "generated-uuid-here",
  "requestId": "generated-request-id",
  "acceptedAt": "2024-01-15T10:30:00Z",
  "partition": 0,
  "offset": 123,
  "status": "INGESTION_STATUS_ACCEPTED"
}
```

#### Test IngestEventBatch

```bash
grpcurl -plaintext \
  -d '{
    "events": [
      {
        "type": "batch.event.1",
        "source": "batch-service",
        "tenant_id": "tenant-1",
        "data": {"index": 0}
      },
      {
        "type": "batch.event.2",
        "source": "batch-service",
        "tenant_id": "tenant-1",
        "data": {"index": 1}
      },
      {
        "type": "batch.event.3",
        "source": "batch-service",
        "tenant_id": "tenant-1",
        "data": {"index": 2}
      }
    ],
    "wait_for_ack": true,
    "fail_fast": false
  }' \
  localhost:9090 \
  events.v1.EventGateway/IngestEventBatch
```

#### Test ValidateEvent

```bash
# Valid event
grpcurl -plaintext \
  -d '{
    "event": {
      "type": "test.event",
      "source": "test-service",
      "data": {"test": "data"}
    }
  }' \
  localhost:9090 \
  events.v1.EventGateway/ValidateEvent

# Invalid event (missing required fields)
grpcurl -plaintext \
  -d '{
    "event": {
      "type": ""
    }
  }' \
  localhost:9090 \
  events.v1.EventGateway/ValidateEvent
```

#### Test with Custom Metadata (Request ID)

```bash
grpcurl -plaintext \
  -H "x-request-id: my-custom-request-123" \
  -d '{
    "event": {
      "type": "user.login",
      "source": "auth-service",
      "data": {"user_id": "12345"}
    }
  }' \
  localhost:9090 \
  events.v1.EventGateway/IngestEvent
```

---

### Method 2: Using Evans (Interactive Client)

**Evans** provides an interactive REPL for testing gRPC services.

#### Start Evans

```bash
evans --host localhost --port 9090 -r repl
```

#### Interactive Commands

```bash
# Show available services
show service

# Select the EventGateway service
service events.v1.EventGateway

# Show available RPCs
show rpc

# Call HealthCheck
call HealthCheck

# Call IngestEvent
call IngestEvent
# Then enter the JSON payload when prompted
```

**Example interactive session:**
```
evans> service events.v1.EventGateway
events.v1.EventGateway@localhost:9090> call HealthCheck
detailed (bool) => true
{
  "status": "SERVICE_STATUS_HEALTHY",
  "version": "1.0.0",
  ...
}

evans.v1.EventGateway@localhost:9090> call IngestEvent
event::type (string) => user.created
event::source (string) => test-service
event::tenant_id (string) => tenant-1
event::data (google.protobuf.Struct) => {"user_id": "123"}
wait_for_ack (bool) => true
{
  "eventId": "...",
  "status": "INGESTION_STATUS_ACCEPTED"
}
```

---

### Method 3: Using the Go Client Example

#### Run the Example Client

```bash
# From the project root
cd services/event-gateway

# Run the example client
go run examples/grpc_client_example.go
```

This will test all RPC methods including bidirectional streaming.

**Expected output:**
```
=== Testing HealthCheck ===
Status: SERVICE_STATUS_HEALTHY
Version: 1.0.0

=== Testing IngestEvent ===
Event ID: abc-123
Partition: 0, Offset: 45

=== Testing IngestEventBatch ===
Success count: 3
Failure count: 0
Processing time: 25 ms

=== Testing ValidateEvent ===
Valid: true

=== Testing StreamEvents ===
  ✓ ACK: Event xyz-789 (partition: 0, offset: 46)
  ✓ ACK: Event xyz-790 (partition: 1, offset: 47)
  ↔ PONG: 2024-01-15 10:30:00
```

---

### Method 4: Using BloomRPC (GUI Client)

**BloomRPC** is a GUI tool similar to Postman for REST APIs.

#### Install BloomRPC

```bash
# macOS
brew install --cask bloomrpc

# Or download from: https://github.com/bloomrpc/bloomrpc/releases
```

#### Setup

1. Open BloomRPC
2. Click **"Import Protos"**
3. Select `shared/proto/events/v1/events.proto`
4. Set server address: `localhost:9090`
5. Uncheck **"Use TLS"** (we're using plaintext for development)
6. Select a method from the left sidebar
7. Fill in the request JSON
8. Click **"Send"**

---

## Verifying Events in Kafka

After sending events, verify they're in Kafka:

```bash
# Using Docker
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic events \
  --from-beginning \
  --max-messages 5

# Or use Kafka UI
# Open: http://localhost:8080
# Navigate to Topics > events > Messages
```

---

## Performance Testing

### Load Testing with ghz

Install ghz (gRPC benchmarking tool):

```bash
# macOS
brew install ghz

# Linux
wget https://github.com/bojand/ghz/releases/download/v0.116.0/ghz-linux-x86_64.tar.gz
tar -xvf ghz-linux-x86_64.tar.gz
sudo mv ghz /usr/local/bin/
```

#### Benchmark IngestEvent

```bash
ghz --insecure \
  --proto shared/proto/events/v1/events.proto \
  --call events.v1.EventGateway.IngestEvent \
  -d '{
    "event": {
      "type": "benchmark.event",
      "source": "load-test",
      "data": {"test": "data"}
    }
  }' \
  --connections 10 \
  --concurrency 100 \
  --total 10000 \
  localhost:9090
```

**Expected output:**
```
Summary:
  Count:        10000
  Total:        5.23 s
  Slowest:      45.12 ms
  Fastest:      0.89 ms
  Average:      5.02 ms
  Requests/sec: 1912.45

Response time histogram:
  0.890 [1]     |
  5.313 [7234]  |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  9.736 [2156]  |∎∎∎∎∎∎∎∎∎∎∎∎
  ...
```

#### Benchmark StreamEvents

```bash
ghz --insecure \
  --proto shared/proto/events/v1/events.proto \
  --call events.v1.EventGateway.StreamEvents \
  --stream-interval 10 \
  -d '{
    "event": {
      "type": "stream.benchmark",
      "source": "load-test",
      "data": {"test": "data"}
    }
  }' \
  --connections 5 \
  --concurrency 50 \
  --total 50000 \
  localhost:9090
```

---

## Monitoring gRPC Metrics

### View Logs

```bash
# If running in terminal, logs appear in stdout
# Look for lines like:
# INFO gRPC request {"method": "/events.v1.EventGateway/IngestEvent", "duration": "5ms", "code": "OK"}
```

### Prometheus Metrics

gRPC metrics are exposed at: http://localhost:8090/metrics

**Key metrics:**
- `grpc_server_handled_total` - Total RPC completions
- `grpc_server_handling_seconds` - Response time histogram
- `grpc_server_started_total` - Total RPC starts

Query in Grafana or Prometheus UI:
```promql
# Request rate
rate(grpc_server_handled_total[5m])

# 95th percentile latency
histogram_quantile(0.95, rate(grpc_server_handling_seconds_bucket[5m]))
```

---

## Common Issues and Solutions

### 1. "Connection refused"

**Problem:** Can't connect to gRPC server

**Solutions:**
- Check if service is running: `ps aux | grep event-gateway`
- Check port: `lsof -i :9090`
- Check config: gRPC enabled in config file
- Check logs for startup errors

### 2. "Method not implemented"

**Problem:** RPC returns "Unimplemented"

**Solutions:**
- Regenerate proto files: `cd shared/proto && ./generate.sh`
- Rebuild service: `go build ./cmd/gateway`
- Ensure EventHandler embeds `UnimplementedEventGatewayServer`

### 3. "Invalid argument" errors

**Problem:** Event validation failing

**Solutions:**
- Check required fields: `type`, `source`, `data` must be set
- Ensure `data` is a valid JSON object (not null)
- Check field types match proto definition

### 4. Kafka connection errors

**Problem:** Events not being produced

**Solutions:**
- Ensure Kafka is running: `docker ps | grep kafka`
- Test Kafka connectivity: `make test-kafka`
- Check Kafka broker config in `config.yaml`

---

## Next Steps

1. **Add authentication** - Implement JWT/TLS for production
2. **Add metrics interceptor** - Track custom business metrics
3. **Add tracing** - Integrate OpenTelemetry for distributed tracing
4. **Add rate limiting** - Per-tenant rate limiting in interceptor
5. **Add integration tests** - Test with real Kafka using testcontainers

---

## Resources

- [gRPC Documentation](https://grpc.io/docs/)
- [Protocol Buffers Guide](https://protobuf.dev/programming-guides/)
- [grpcurl Documentation](https://github.com/fullstorydev/grpcurl)
- [Evans Documentation](https://github.com/ktr0731/evans)
- [ghz Benchmarking Tool](https://ghz.sh/)
