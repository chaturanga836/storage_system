#!/bin/bash

# Build optimized Docker images script
# This script builds the shared base image first, then all service images

set -e  # Exit on any error

echo "=== Building Optimized Storage System Docker Images ==="

# Step 1: Build the shared base image
echo "Step 1: Building shared base image..."
docker build -f Dockerfile.base -t storage-python-base:latest .
echo "✓ Base image built successfully"

# Step 2: Clean up any existing containers/images
echo "Step 2: Cleaning up existing containers..."
docker-compose down --volumes --remove-orphans || true

# Step 3: Build all services with the new optimized Dockerfiles
echo "Step 3: Building all services with optimized Dockerfiles..."
docker-compose build --no-cache

# Step 4: Start all services
echo "Step 4: Starting all services..."
docker-compose up -d

# Step 5: Show status
echo "Step 5: Checking service status..."
sleep 10
docker-compose ps

echo ""
echo "=== Build Complete ==="
echo "All services should now be running with optimized Docker images."
echo "Check the status above. If any service is not 'Up', check logs with:"
echo "  docker-compose logs [service-name]"
echo ""
echo "Available services:"
echo "  - auth-gateway:        http://localhost:8080"
echo "  - operation-node:      http://localhost:8086"
echo "  - cbo-engine:          http://localhost:8088"
echo "  - metadata-catalog:    http://localhost:8087"
echo "  - monitoring:          http://localhost:8089"
echo "  - tenant-node:         http://localhost:8001"
echo "  - postgres:            localhost:5432"
echo "  - redis:               localhost:6379"
echo "  - prometheus:          http://localhost:9090"
echo "  - grafana:             http://localhost:3000"
