#!/bin/bash

echo "🚀 Testing Event Gateway"
echo "========================"

GATEWAY_URL="http://localhost:8090"

# Test 1: Health Check
echo "1. Testing health endpoint..."
curl -s "$GATEWAY_URL/health" | jq .
echo ""

# Test 2: API Documentation
echo "2. Testing API docs..."
curl -s "$GATEWAY_URL/api/docs" | jq .service
echo ""

# Test 3: Single Event Ingestion
echo "3. Testing single event ingestion..."
curl -X POST "$GATEWAY_URL/api/v1/events" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "user.created",
    "source": "test-service",
    "subject": "user-123",
    "data": {
      "user_id": "123",
      "email": "test@example.com",
      "name": "Test User"
    },
    "metadata": {
      "region": "us-west-2"
    }
  }' | jq .
echo ""

# Test 4: Event Validation
echo "4. Testing event validation..."
curl -X POST "$GATEWAY_URL/api/v1/events/validate" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "order.completed",
    "source": "order-service", 
    "data": {
      "order_id": "order-456",
      "amount": 99.99,
      "currency": "USD"
    }
  }' | jq .
echo ""

# Test 5: Batch Event Ingestion
echo "5. Testing batch event ingestion..."
curl -X POST "$GATEWAY_URL/api/v1/events/batch" \
  -H "Content-Type: application/json" \
  -d '{
    "events": [
      {
        "type": "page.viewed",
        "source": "web-app",
        "data": {"page": "/home", "user_id": "user-1"}
      },
      {
        "type": "page.viewed", 
        "source": "web-app",
        "data": {"page": "/products", "user_id": "user-1"}
      },
      {
        "type": "button.clicked",
        "source": "web-app",
        "data": {"button": "signup", "user_id": "user-1"}
      }
    ]
  }' | jq .
echo ""

# Test 6: Invalid Event (should fail validation)
echo "6. Testing invalid event (missing required fields)..."
curl -X POST "$GATEWAY_URL/api/v1/events" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "invalid.event"
  }' | jq .
echo ""

# Test 7: Metrics
echo "7. Testing metrics endpoint..."
curl -s "$GATEWAY_URL/metrics" | grep "http_requests_total" | head -3
echo ""

# Test 8: Detailed Health Check
echo "8. Testing detailed health check..."
curl -s "$GATEWAY_URL/health/detailed" | jq .
echo ""

echo "✅ Event Gateway testing completed!"
echo ""
echo "Next steps:"
echo "- Check Kafka for ingested events: docker exec kafka kafka-console-consumer --topic events --from-beginning --bootstrap-server localhost:9092"
echo "- View metrics: curl http://localhost:8090/metrics"
echo "- Monitor logs in the gateway terminal"
