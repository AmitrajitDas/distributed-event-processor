# Event Gateway - End-to-End Testing Guide

## 📋 Completion Status

### ✅ Completed Features

The Event Gateway service is **FULLY IMPLEMENTED** with the following features:

- ✅ **Multi-protocol support**: HTTP REST API and gRPC
- ✅ **Event ingestion endpoints**:
  - Single event ingestion (`POST /api/v1/events`)
  - Batch event ingestion (`POST /api/v1/events/batch`)
  - Event validation/dry-run (`POST /api/v1/events/validate`)
- ✅ **Health check endpoints**:
  - Basic health (`GET /health`)
  - Detailed health with dependencies (`GET /health/detailed`)
  - Kubernetes readiness probe (`GET /health/ready`)
  - Kubernetes liveness probe (`GET /health/live`)
- ✅ **Kafka integration**: Producer with batching and retry logic
- ✅ **Request validation**: JSON schema validation with detailed error messages
- ✅ **Rate limiting**: Token bucket algorithm
- ✅ **Middleware**: Request ID, CORS, security headers, timeout handling
- ✅ **Metrics**: Prometheus metrics endpoint (`GET /metrics`)
- ✅ **Graceful shutdown**: Proper cleanup and connection draining
- ✅ **Comprehensive test coverage**: Unit and integration tests

---

## 🚀 Quick Start - Local E2E Testing

### Step 1: Start Infrastructure

```bash
# Start Kafka, Redis, PostgreSQL, MongoDB, and monitoring stack
make infra-up

# OR start only infrastructure without monitoring
make infra-only

# Verify infrastructure is running
docker compose -f infrastructure/docker/docker-compose.yml ps
```

**Expected Services:**
- Kafka: localhost:9092
- Kafka UI: http://localhost:8080
- Redis: localhost:6379
- PostgreSQL: localhost:5432
- MongoDB: localhost:27017
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/event123)

### Step 2: Build and Run Event Gateway

```bash
# From project root
cd services/event-gateway

# Build the service
go build -o event-gateway cmd/gateway/main.go

# Run the service
./event-gateway
```

**OR run directly without building:**
```bash
cd services/event-gateway
go run cmd/gateway/main.go
```

The service will start on:
- **HTTP**: http://localhost:8090
- **gRPC**: localhost:9090

---

## 🧪 Testing with Postman

### Import Postman Collection

Create a new Postman collection with the following requests:

### 1️⃣ Health Check

**Basic Health Check**
```http
GET http://localhost:8090/health
```

Expected Response (200 OK):
```json
{
  "status": "healthy",
  "timestamp": "2024-11-27T10:30:00Z",
  "version": "1.0.0",
  "services": {}
}
```

**Detailed Health Check**
```http
GET http://localhost:8090/health/detailed
```

Expected Response (200 OK):
```json
{
  "status": "healthy",
  "timestamp": "2024-11-27T10:30:00Z",
  "version": "1.0.0",
  "services": {
    "kafka": "healthy"
  },
  "system": {
    "uptime_seconds": 120,
    "uptime_human": "2m0s",
    "started_at": "2024-11-27T10:28:00Z"
  },
  "performance": {
    "goroutines": 15,
    "memory_alloc_mb": 12.5,
    "memory_sys_mb": 25.3,
    "memory_heap_mb": 12.5,
    "gc_cycles": 3,
    "gc_pause_total_ms": 0.5
  }
}
```

---

### 2️⃣ Single Event Ingestion

**Request:**
```http
POST http://localhost:8090/api/v1/events
Content-Type: application/json

{
  "type": "user.created",
  "source": "user-service",
  "subject": "user-123",
  "data": {
    "user_id": "123",
    "email": "john.doe@example.com",
    "name": "John Doe",
    "created_at": "2024-11-27T10:30:00Z"
  },
  "metadata": {
    "region": "us-east-1",
    "environment": "production"
  }
}
```

**Expected Response (202 Accepted):**
```json
{
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "accepted",
  "timestamp": "2024-11-27T10:30:00Z",
  "message": "Event ingested successfully"
}
```

**Response Headers:**
```
X-Event-ID: 550e8400-e29b-41d4-a716-446655440000
X-Request-ID: req_abc123xyz
```

---

### 3️⃣ Batch Event Ingestion

**Request:**
```http
POST http://localhost:8090/api/v1/events/batch
Content-Type: application/json

{
  "events": [
    {
      "type": "user.created",
      "source": "user-service",
      "subject": "user-101",
      "data": {
        "user_id": "101",
        "email": "alice@example.com",
        "name": "Alice Smith"
      }
    },
    {
      "type": "user.updated",
      "source": "user-service",
      "subject": "user-102",
      "data": {
        "user_id": "102",
        "email": "bob@example.com",
        "status": "active"
      }
    },
    {
      "type": "order.placed",
      "source": "order-service",
      "subject": "order-5001",
      "data": {
        "order_id": "5001",
        "user_id": "101",
        "total_amount": 299.99,
        "items": ["item-1", "item-2"]
      }
    }
  ]
}
```

**Expected Response (202 Accepted):**
```json
{
  "processed_count": 3,
  "failed_count": 0,
  "results": [
    {
      "event_id": "event-uuid-1",
      "status": "accepted"
    },
    {
      "event_id": "event-uuid-2",
      "status": "accepted"
    },
    {
      "event_id": "event-uuid-3",
      "status": "accepted"
    }
  ],
  "errors": []
}
```

---

### 4️⃣ Event Validation (Dry Run)

**Request:**
```http
POST http://localhost:8090/api/v1/events/validate
Content-Type: application/json

{
  "type": "payment.processed",
  "source": "payment-service",
  "data": {
    "payment_id": "pay_12345",
    "amount": 150.00,
    "currency": "USD"
  }
}
```

**Expected Response (200 OK):**
```json
{
  "valid": true,
  "message": "Event is valid",
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2024-11-27T10:30:00Z",
  "request_id": "req_xyz789"
}
```

---

### 5️⃣ Error Cases Testing

**Missing Required Fields**
```http
POST http://localhost:8090/api/v1/events
Content-Type: application/json

{
  "type": "user.created",
  "data": {
    "user_id": "123"
  }
}
```

**Expected Response (400 Bad Request):**
```json
{
  "error": "validation_failed",
  "message": "Event validation failed",
  "details": "Field 'Source' failed validation: required",
  "request_id": "req_abc123"
}
```

**Invalid JSON**
```http
POST http://localhost:8090/api/v1/events
Content-Type: application/json

{
  "type": "user.created",
  "source": "user-service"
  // Missing closing brace
```

**Expected Response (400 Bad Request):**
```json
{
  "error": "invalid_json",
  "message": "Invalid JSON format",
  "details": "unexpected EOF",
  "request_id": "req_def456"
}
```

---

## 🔍 Verify Events in Kafka

After sending events, verify they're in Kafka:

### Option 1: Kafka UI (Recommended)

1. Open http://localhost:8080
2. Navigate to Topics → `events`
3. Click "Messages" tab
4. You should see your events with their data

### Option 2: Command Line

```bash
# List topics
docker exec kafka kafka-topics --list --bootstrap-server localhost:9092

# Consume messages from events topic
docker exec kafka kafka-console-consumer \
  --topic events \
  --from-beginning \
  --bootstrap-server localhost:9092 \
  --property print.key=true \
  --property print.timestamp=true
```

Expected output:
```
CreateTime:1701086400000	user-123	{"id":"550e8400-...","type":"user.created",...}
CreateTime:1701086401000	user-101	{"id":"660f9511-...","type":"user.created",...}
```

---

## 📊 Monitor Metrics

### Prometheus Metrics

Visit http://localhost:8090/metrics to see raw metrics:

```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="POST",endpoint="/api/v1/events",status="202"} 45

# HELP http_request_duration_seconds HTTP request duration
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{method="POST",endpoint="/api/v1/events",le="0.001"} 42
http_request_duration_seconds_bucket{method="POST",endpoint="/api/v1/events",le="0.01"} 45

# HELP events_ingested_total Total events ingested
# TYPE events_ingested_total counter
events_ingested_total{type="user.created",source="user-service"} 25
```

### Grafana Dashboards

1. Open http://localhost:3000
2. Login: `admin` / `event123`
3. Navigate to Dashboards
4. Look for Event Gateway dashboard (if pre-configured)

---

## 🧰 Testing with cURL

### Single Event
```bash
curl -X POST http://localhost:8090/api/v1/events \
  -H "Content-Type: application/json" \
  -d '{
    "type": "user.created",
    "source": "user-service",
    "subject": "user-123",
    "data": {
      "user_id": "123",
      "email": "test@example.com"
    }
  }' | jq
```

### Batch Events
```bash
curl -X POST http://localhost:8090/api/v1/events/batch \
  -H "Content-Type: application/json" \
  -d '{
    "events": [
      {
        "type": "user.created",
        "source": "user-service",
        "data": {"user_id": "101"}
      },
      {
        "type": "user.updated",
        "source": "user-service",
        "data": {"user_id": "102"}
      }
    ]
  }' | jq
```

### Health Check
```bash
curl http://localhost:8090/health | jq
curl http://localhost:8090/health/detailed | jq
```

---

## 🎯 Load Testing (Optional)

### Using Apache Bench (ab)

```bash
# Install ab (if not already installed)
# macOS: brew install httpd
# Ubuntu: sudo apt-get install apache2-utils

# Send 1000 requests with 10 concurrent connections
ab -n 1000 -c 10 -T "application/json" \
  -p event-payload.json \
  http://localhost:8090/api/v1/events
```

Create `event-payload.json`:
```json
{
  "type": "load.test",
  "source": "load-test",
  "data": {
    "test_id": "load-001",
    "timestamp": "2024-11-27T10:30:00Z"
  }
}
```

### Using k6 (Recommended)

```bash
# Install k6: brew install k6

# Create load-test.js
cat > load-test.js <<'EOF'
import http from 'k6/http';
import { check } from 'k6';

export let options = {
  stages: [
    { duration: '30s', target: 50 },  // Ramp up to 50 users
    { duration: '1m', target: 100 },  // Stay at 100 users
    { duration: '30s', target: 0 },   // Ramp down
  ],
};

export default function () {
  const payload = JSON.stringify({
    type: 'load.test',
    source: 'k6-load-test',
    data: {
      test_id: `test-${__VU}-${__ITER}`,
      timestamp: new Date().toISOString(),
    },
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  const res = http.post('http://localhost:8090/api/v1/events', payload, params);

  check(res, {
    'status is 202': (r) => r.status === 202,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
}
EOF

# Run the load test
k6 run load-test.js
```

---

## 🐛 Troubleshooting

### Service won't start

**Check if ports are available:**
```bash
lsof -i :8090  # HTTP port
lsof -i :9090  # gRPC port
```

**Check configuration:**
```bash
cd services/event-gateway
cat config.yaml
```

### Kafka connection errors

**Verify Kafka is running:**
```bash
docker compose -f infrastructure/docker/docker-compose.yml ps kafka
```

**Test Kafka connectivity:**
```bash
make test-kafka
```

**Check Kafka logs:**
```bash
docker logs kafka
```

### Events not appearing in Kafka

**Check Event Gateway logs:**
```bash
# The service logs are printed to stdout
# Look for errors like:
# "Failed to send event to Kafka"
```

**Verify topic exists:**
```bash
docker exec kafka kafka-topics --describe --topic events --bootstrap-server localhost:9092
```

### Rate limiting triggered

If you see `429 Too Many Requests`, you've hit the rate limit:

```bash
# Increase rate limit in config.yaml
rate_limit:
  requests_per_second: 5000
  burst_size: 10000

# Or via environment variable
export GATEWAY_RATE_LIMIT_REQUESTS_PER_SECOND=5000
export GATEWAY_RATE_LIMIT_BURST_SIZE=10000
```

---

## 🔄 Complete E2E Workflow Test

Run this complete workflow to test the entire system:

```bash
#!/bin/bash

echo "=== Event Gateway E2E Test ==="

# 1. Health check
echo -e "\n1. Testing health endpoint..."
curl -s http://localhost:8090/health | jq

# 2. Single event
echo -e "\n2. Sending single event..."
curl -s -X POST http://localhost:8090/api/v1/events \
  -H "Content-Type: application/json" \
  -d '{
    "type": "user.created",
    "source": "test-script",
    "data": {"user_id": "e2e-test-001"}
  }' | jq

# 3. Batch events
echo -e "\n3. Sending batch events..."
curl -s -X POST http://localhost:8090/api/v1/events/batch \
  -H "Content-Type: application/json" \
  -d '{
    "events": [
      {"type": "order.placed", "source": "test-script", "data": {"order_id": "001"}},
      {"type": "order.placed", "source": "test-script", "data": {"order_id": "002"}},
      {"type": "order.placed", "source": "test-script", "data": {"order_id": "003"}}
    ]
  }' | jq

# 4. Validation
echo -e "\n4. Validating event..."
curl -s -X POST http://localhost:8090/api/v1/events/validate \
  -H "Content-Type: application/json" \
  -d '{
    "type": "payment.processed",
    "source": "test-script",
    "data": {"payment_id": "pay-001", "amount": 100.00}
  }' | jq

# 5. Check metrics
echo -e "\n5. Fetching metrics..."
curl -s http://localhost:8090/metrics | grep events_ingested_total

echo -e "\n=== Test Complete ==="
```

Save as `e2e-test.sh`, make executable, and run:
```bash
chmod +x e2e-test.sh
./e2e-test.sh
```

---

## 📚 Next Steps

After confirming Event Gateway works:

1. **Test gRPC interface** - Use the gRPC testing guide in `services/event-gateway/GRPC_TESTING.md`
2. **Build Stream Processor** - Process events from Kafka
3. **Implement Rule Engine** - Add CEP rules and pattern matching
4. **Setup Event Store** - Persist events to PostgreSQL
5. **Configure Output Manager** - Route processed events to destinations

---

## 💡 Tips for Understanding the System

1. **Watch Kafka messages in real-time:**
   ```bash
   docker exec -it kafka kafka-console-consumer \
     --topic events \
     --from-beginning \
     --bootstrap-server localhost:9092 | jq
   ```

2. **Monitor service logs:**
   ```bash
   # Event Gateway logs are JSON formatted
   # Look for structured logs with request_id, event_id, etc.
   ```

3. **Use Prometheus to query metrics:**
   - Visit http://localhost:9090
   - Try queries like:
     - `rate(http_requests_total[1m])`
     - `histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))`

4. **Check Grafana for visualization:**
   - http://localhost:3000
   - Create custom dashboards for your event types

---

## ✅ Completion Checklist

- [ ] Infrastructure started successfully
- [ ] Event Gateway running on port 8090
- [ ] Health checks return healthy status
- [ ] Single event ingestion works
- [ ] Batch event ingestion works
- [ ] Event validation works
- [ ] Events visible in Kafka (via Kafka UI)
- [ ] Prometheus metrics accessible
- [ ] Error handling tested (missing fields, invalid JSON)
- [ ] Load test completed (optional)

**Congratulations! Your Event Gateway is fully functional and ready for production-grade event processing! 🎉**
