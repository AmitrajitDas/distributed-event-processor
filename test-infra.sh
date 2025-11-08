#!/bin/bash
# Unified Infrastructure Test Script

cd /Users/amitrajitdas31/Developer/Coding/Dev/Projects/distributed-event-processor

echo "🚀 Starting Unified Infrastructure Testing..."
echo "================================================"

# 1. Start all services with unified Docker Compose
echo "📦 Starting all infrastructure and monitoring services..."
make infra-up

echo "⏳ Waiting for services to start..."
sleep 45

# 2. Check infrastructure service health
echo "🔍 Checking infrastructure services..."

# Check Kafka
echo "  ✓ Checking Kafka..."
docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list 2>/dev/null && echo "    ✅ Kafka is running" || echo "    ❌ Kafka failed"

# Check Redis
echo "  ✓ Checking Redis..."
docker exec redis redis-cli ping 2>/dev/null && echo "    ✅ Redis is running" || echo "    ❌ Redis failed"

# Check PostgreSQL
echo "  ✓ Checking PostgreSQL..."
docker exec postgres pg_isready -U eventuser -d event_processor 2>/dev/null && echo "    ✅ PostgreSQL is running" || echo "    ❌ PostgreSQL failed"

# Check MongoDB
echo "  ✓ Checking MongoDB..."
docker exec mongodb mongosh --eval "db.runCommand('ismaster')" --quiet 2>/dev/null && echo "    ✅ MongoDB is running" || echo "    ❌ MongoDB failed"

# Check MinIO
echo "  ✓ Checking MinIO..."
curl -f http://localhost:9000/minio/health/live 2>/dev/null && echo "    ✅ MinIO is running" || echo "    ❌ MinIO failed"

# Check Schema Registry
echo "  ✓ Checking Schema Registry..."
curl -f http://localhost:8081/subjects 2>/dev/null && echo "    ✅ Schema Registry is running" || echo "    ❌ Schema Registry failed"

# 3. Check monitoring services
echo "🔍 Checking monitoring services..."

# Check Prometheus
echo "  ✓ Checking Prometheus..."
curl -f http://localhost:9090/-/healthy 2>/dev/null && echo "    ✅ Prometheus is running" || echo "    ❌ Prometheus failed"

# Check Grafana
echo "  ✓ Checking Grafana..."
curl -f http://localhost:3000/api/health 2>/dev/null && echo "    ✅ Grafana is running" || echo "    ❌ Grafana failed"

# Check Loki
echo "  ✓ Checking Loki..."
curl -f http://localhost:3100/ready 2>/dev/null && echo "    ✅ Loki is running" || echo "    ❌ Loki failed"

# Check Jaeger
echo "  ✓ Checking Jaeger..."
curl -f http://localhost:16686/ 2>/dev/null && echo "    ✅ Jaeger is running" || echo "    ❌ Jaeger failed"

# 4. Test Kafka functionality
echo "🧪 Testing Kafka functionality..."
make test-kafka

# 5. Test database connections
echo "🗄️ Testing database connections..."
make test-databases

echo "================================================"
echo "🎯 Infrastructure Test Summary"
echo "================================================"

# Show all service URLs
make show-urls

echo ""
echo "📋 Next Steps:"
echo "  1. Run 'make health-check' for detailed health status"
echo "  2. Open Kafka UI to verify cluster and create topics"
echo "  3. Open Grafana to set up dashboards"
echo "  4. Check Prometheus targets are all UP"
echo "  5. Ready to implement Event Gateway service!"

echo ""
echo "🔧 Quick Test Commands:"
echo "  make test-kafka      # Test Kafka functionality"
echo "  make test-databases  # Test database connections"
echo "  make health-check    # Check service health"
echo "  make infra-logs      # View all service logs"