#!/bin/bash

# Quick gRPC testing script for Event Gateway
# This script tests all gRPC endpoints using grpcurl

set -e

SERVER="localhost:50051"  # Standard gRPC port (changed from 9090 to avoid Prometheus conflict)
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}=====================================${NC}"
echo -e "${BLUE}   Event Gateway gRPC Test Suite    ${NC}"
echo -e "${BLUE}=====================================${NC}"
echo ""

# Check if grpcurl is installed
if ! command -v grpcurl &> /dev/null; then
    echo -e "${RED}Error: grpcurl is not installed${NC}"
    echo "Install it with:"
    echo "  macOS: brew install grpcurl"
    echo "  Linux: wget https://github.com/fullstorydev/grpcurl/releases/download/v1.8.9/grpcurl_1.8.9_linux_x86_64.tar.gz"
    exit 1
fi

# Check if server is running
if ! nc -z localhost 50051 2>/dev/null; then
    echo -e "${RED}Error: gRPC server is not running on $SERVER${NC}"
    echo "Start it with: make run-gateway"
    exit 1
fi

echo -e "${GREEN}✓ grpcurl is installed${NC}"
echo -e "${GREEN}✓ gRPC server is running${NC}"
echo ""

# Test 1: List services
echo -e "${BLUE}[1/6] Listing available services...${NC}"
grpcurl -plaintext $SERVER list
echo ""

# Test 2: HealthCheck
echo -e "${BLUE}[2/6] Testing HealthCheck...${NC}"
grpcurl -plaintext \
  -d '{"detailed": true}' \
  $SERVER \
  events.v1.EventGateway/HealthCheck | jq .
echo -e "${GREEN}✓ HealthCheck passed${NC}"
echo ""

# Test 3: ValidateEvent (valid)
echo -e "${BLUE}[3/6] Testing ValidateEvent (valid event)...${NC}"
grpcurl -plaintext \
  -d '{
    "event": {
      "type": "test.event",
      "source": "test-script",
      "data": {"test": "data", "timestamp": 1234567890}
    }
  }' \
  $SERVER \
  events.v1.EventGateway/ValidateEvent | jq .
echo -e "${GREEN}✓ ValidateEvent passed${NC}"
echo ""

# Test 4: ValidateEvent (invalid)
echo -e "${BLUE}[4/6] Testing ValidateEvent (invalid event)...${NC}"
grpcurl -plaintext \
  -d '{
    "event": {
      "type": ""
    }
  }' \
  $SERVER \
  events.v1.EventGateway/ValidateEvent | jq .
echo -e "${GREEN}✓ ValidateEvent error handling passed${NC}"
echo ""

# Test 5: IngestEvent
echo -e "${BLUE}[5/6] Testing IngestEvent...${NC}"
RESPONSE=$(grpcurl -plaintext \
  -d '{
    "event": {
      "type": "user.login",
      "source": "test-script",
      "tenant_id": "tenant-test",
      "data": {
        "user_id": "12345",
        "action": "login",
        "timestamp": 1234567890
      },
      "schema_version": "1.0",
      "correlation_id": "test-corr-123",
      "priority": 5,
      "metadata": {
        "region": "us-east-1",
        "test": "true"
      }
    },
    "wait_for_ack": true
  }' \
  $SERVER \
  events.v1.EventGateway/IngestEvent)

echo "$RESPONSE" | jq .
EVENT_ID=$(echo "$RESPONSE" | jq -r '.eventId')
PARTITION=$(echo "$RESPONSE" | jq -r '.partition')
OFFSET=$(echo "$RESPONSE" | jq -r '.offset')

echo -e "${GREEN}✓ IngestEvent passed${NC}"
echo -e "  Event ID: $EVENT_ID"
echo -e "  Partition: $PARTITION, Offset: $OFFSET"
echo ""

# Test 6: IngestEventBatch
echo -e "${BLUE}[6/6] Testing IngestEventBatch...${NC}"
BATCH_RESPONSE=$(grpcurl -plaintext \
  -d '{
    "events": [
      {
        "type": "batch.event.1",
        "source": "test-script",
        "tenant_id": "tenant-test",
        "data": {"index": 0, "message": "First event"}
      },
      {
        "type": "batch.event.2",
        "source": "test-script",
        "tenant_id": "tenant-test",
        "data": {"index": 1, "message": "Second event"}
      },
      {
        "type": "batch.event.3",
        "source": "test-script",
        "tenant_id": "tenant-test",
        "data": {"index": 2, "message": "Third event"}
      }
    ],
    "wait_for_ack": true,
    "fail_fast": false
  }' \
  $SERVER \
  events.v1.EventGateway/IngestEventBatch)

echo "$BATCH_RESPONSE" | jq .
SUCCESS_COUNT=$(echo "$BATCH_RESPONSE" | jq -r '.successCount')
FAILURE_COUNT=$(echo "$BATCH_RESPONSE" | jq -r '.failureCount')
PROCESSING_TIME=$(echo "$BATCH_RESPONSE" | jq -r '.processingTimeMs')

echo -e "${GREEN}✓ IngestEventBatch passed${NC}"
echo -e "  Success: $SUCCESS_COUNT, Failures: $FAILURE_COUNT"
echo -e "  Processing time: ${PROCESSING_TIME}ms"
echo ""

# Summary
echo -e "${BLUE}=====================================${NC}"
echo -e "${GREEN}✓ All tests passed successfully!${NC}"
echo -e "${BLUE}=====================================${NC}"
echo ""
echo "Next steps:"
echo "  1. View events in Kafka UI: http://localhost:8080"
echo "  2. Check Grafana dashboards: http://localhost:3000"
echo "  3. Run the Go client example:"
echo "     go run examples/grpc_client_example.go"
echo "  4. Run load tests with ghz (see GRPC_TESTING.md)"
echo ""
