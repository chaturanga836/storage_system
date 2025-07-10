#!/bin/bash

# Run all services locally for development

set -e

echo "🚀 Starting Storage Engine services locally..."

# Set environment variables for local development
export INGESTION_PORT=8001
export QUERY_PORT=8002
export STORAGE_DATA_PATH="./data"
export WAL_PATH="./wal"
export CATALOG_PATH="./catalog"
export AUTH_ENABLED=false

# Create necessary directories
mkdir -p data wal catalog logs

# Function to cleanup on exit
cleanup() {
    echo "🛑 Stopping all services..."
    kill $(jobs -p) 2>/dev/null || true
    wait
    echo "👋 All services stopped"
}

trap cleanup EXIT

# Start data processor first
echo "⚙️ Starting Data Processor..."
./bin/data-processor > logs/data-processor.log 2>&1 &
DATA_PROCESSOR_PID=$!

# Wait a moment for data processor to initialize
sleep 2

# Start ingestion server
echo "📥 Starting Ingestion Server on port $INGESTION_PORT..."
./bin/ingestion-server > logs/ingestion-server.log 2>&1 &
INGESTION_PID=$!

# Start query server
echo "🔍 Starting Query Server on port $QUERY_PORT..."
./bin/query-server > logs/query-server.log 2>&1 &
QUERY_PID=$!

# Wait for services to start
sleep 3

echo "✅ All services started!"
echo "📊 Service endpoints:"
echo "  - Ingestion Server: localhost:$INGESTION_PORT"
echo "  - Query Server: localhost:$QUERY_PORT"
echo "  - Data Processor: Background service"
echo ""
echo "📋 Admin commands:"
echo "  - Status: ./bin/storage-admin status"
echo "  - WAL inspect: ./bin/storage-admin wal inspect"
echo "  - Schema list: ./bin/storage-admin schema list"
echo ""
echo "📝 Logs available in: logs/"
echo "🛑 Press Ctrl+C to stop all services"

# Wait for user interrupt
wait
