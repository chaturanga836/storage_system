#!/bin/bash

# Multi-Tenant Storage System - Development Startup Script

echo "🚀 Starting Multi-Tenant Storage System (Microservices)"
echo "========================================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to start a service
start_service() {
    local service_name=$1
    local service_dir=$2
    local port=$3
    
    echo -e "${BLUE}Starting $service_name on port $port...${NC}"
    
    cd "$service_dir" || {
        echo -e "${RED}Failed to enter directory: $service_dir${NC}"
        return 1
    }
    
    # Install dependencies if requirements.txt exists
    if [ -f "requirements.txt" ]; then
        echo -e "${YELLOW}Installing dependencies for $service_name...${NC}"
        pip install -r requirements.txt > /dev/null 2>&1
    fi
    
    # Start the service in background
    python main.py &
    local pid=$!
    echo -e "${GREEN}$service_name started with PID: $pid${NC}"
    
    # Store PID for cleanup
    echo $pid >> .pids
    
    cd - > /dev/null
    sleep 2
}

# Create PID file for cleanup
rm -f .pids
touch .pids

echo -e "${YELLOW}Starting services in dependency order...${NC}"
echo

# 1. Start Auth Gateway first (other services depend on it)
start_service "Auth Gateway" "auth-gateway" "8080"

# 2. Start supporting services
start_service "Monitoring Service" "monitoring" "8084"
start_service "CBO Engine" "cbo-engine" "8082"
start_service "Metadata Catalog" "metadata-catalog" "8083"
start_service "Operation Node" "operation-node" "8081"

# 3. Start Tenant Node last (depends on other services)
start_service "Tenant Node" "tenant-node" "8000"

echo
echo -e "${GREEN}All services started successfully!${NC}"
echo
echo "🌐 Service URLs:"
echo "  Auth Gateway:      http://localhost:8080"
echo "  Tenant Node:       http://localhost:8000"
echo "  Operation Node:    http://localhost:8081"
echo "  CBO Engine:        http://localhost:8082"
echo "  Metadata Catalog:  http://localhost:8083"
echo "  Monitoring:        http://localhost:8084"
echo
echo "📊 Health Checks:"
echo "  curl http://localhost:8080/health"
echo "  curl http://localhost:8000/health"
echo "  curl http://localhost:8084/status"
echo
echo "🔑 Authentication:"
echo "  curl -X POST http://localhost:8080/auth/login \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"username\":\"admin\",\"password\":\"admin123\"}'"
echo
echo -e "${YELLOW}To stop all services, run: ./stop_services.sh${NC}"
echo
echo "Press Ctrl+C to stop monitoring..."

# Monitor services
while true; do
    sleep 5
    # Check if all services are still running
    failed_services=""
    for pid in $(cat .pids); do
        if ! kill -0 $pid 2>/dev/null; then
            failed_services="$failed_services $pid"
        fi
    done
    
    if [ -n "$failed_services" ]; then
        echo -e "${RED}Some services have stopped: $failed_services${NC}"
    fi
done
