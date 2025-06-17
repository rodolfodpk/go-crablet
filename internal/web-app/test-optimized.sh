#!/bin/bash

echo "🧪 Testing Optimized Web-App Performance"
echo "========================================"

# Stop any existing containers and clean up
echo "📦 Cleaning up existing containers..."
docker-compose -f ../../docker-compose.yaml down -v 2>/dev/null || true

# Build and start the optimized web-app and postgres
echo "🔨 Building optimized web-app and postgres..."
docker-compose -f ../../docker-compose.yaml up -d --build

# Wait for the services to be ready
echo "⏳ Waiting for services to be ready..."
sleep 15

# Check if the services are responding
echo "🔍 Checking service health..."
for i in {1..30}; do
    if curl -s http://localhost:8080/health > /dev/null; then
        echo "✅ Web-app is ready!"
        break
    fi
    echo "⏳ Waiting for web-app to start... (attempt $i/30)"
    sleep 2
done

# Run a quick performance test
echo "🚀 Running quick performance test..."
k6 run --out json=k6-quick-test.json k6-test.js

# Show resource usage for both containers
echo "📊 Resource usage:"
echo "Web-app:"
docker stats --no-stream dcb_webapp 2>/dev/null || echo "Web-app container not found"
echo "Postgres:"
docker stats --no-stream postgres_db 2>/dev/null || echo "Postgres container not found"

echo "✅ Optimized web-app test completed!"
echo "📈 Check k6-quick-test.json for detailed results" 