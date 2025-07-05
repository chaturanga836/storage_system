#!/bin/bash

# EC2 Deployment Script for Optimized Storage System
# Run this script on your EC2 instance after pulling the latest changes

set -e  # Exit on any error

echo "=== EC2 Deployment: Optimized Storage System ==="
echo "Starting deployment at $(date)"

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check system resources
check_resources() {
    echo "Checking system resources..."
    
    # Check disk space (need at least 8GB free)
    AVAILABLE_SPACE=$(df / | awk 'NR==2 {print $4}')
    REQUIRED_SPACE=8388608  # 8GB in KB
    
    if [ "$AVAILABLE_SPACE" -lt "$REQUIRED_SPACE" ]; then
        echo "⚠️  Warning: Low disk space. Available: $(($AVAILABLE_SPACE/1048576))GB, Recommended: 8GB"
        echo "Consider cleaning up with: docker system prune -a --volumes"
        read -p "Continue anyway? (y/n): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
    
    # Check memory
    AVAILABLE_MEMORY=$(free -m | awk 'NR==2{print $7}')
    if [ "$AVAILABLE_MEMORY" -lt 2048 ]; then
        echo "⚠️  Warning: Low available memory: ${AVAILABLE_MEMORY}MB"
    fi
    
    echo "✓ Resource check complete"
}

# Function to verify prerequisites
check_prerequisites() {
    echo "Checking prerequisites..."
    
    if ! command_exists docker; then
        echo "❌ Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! command_exists docker-compose; then
        echo "❌ Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    # Check if user is in docker group
    if ! groups $USER | grep -q docker; then
        echo "⚠️  User $USER is not in docker group. You may need to use sudo."
        echo "To fix: sudo usermod -aG docker $USER && newgrp docker"
    fi
    
    # Test Docker access
    if ! docker ps >/dev/null 2>&1; then
        echo "❌ Cannot access Docker. Check permissions or try with sudo."
        exit 1
    fi
    
    echo "✓ Prerequisites check passed"
}

# Function to backup current state
backup_current_state() {
    echo "Creating backup of current state..."
    
    BACKUP_DIR="backup_$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$BACKUP_DIR"
    
    # Export current container data if any
    if docker-compose ps -q | grep -q .; then
        echo "Backing up current container state..."
        docker-compose ps > "$BACKUP_DIR/container_state.txt"
        docker-compose logs > "$BACKUP_DIR/container_logs.txt" 2>/dev/null || true
    fi
    
    echo "✓ Backup created in $BACKUP_DIR"
}

# Function to clean up old resources
cleanup_old_resources() {
    echo "Cleaning up old resources..."
    
    # Stop any running services
    echo "Stopping existing services..."
    docker-compose down --volumes --remove-orphans || true
    
    # Ask user about cleaning old images
    echo "Current Docker disk usage:"
    docker system df
    echo
    read -p "Clean up old Docker images to save space? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Cleaning up old Docker resources..."
        docker system prune -a --volumes -f
        echo "✓ Cleanup complete"
    else
        echo "Skipping cleanup"
    fi
}

# Function to build optimized images
build_optimized_images() {
    echo "Building optimized Docker images..."
    
    # Build the shared base image
    echo "Step 1/3: Building shared base image..."
    START_TIME=$(date +%s)
    docker build -f Dockerfile.base -t storage-python-base:latest .
    BASE_BUILD_TIME=$(($(date +%s) - START_TIME))
    echo "✓ Base image built in ${BASE_BUILD_TIME}s"
    
    # Build all services
    echo "Step 2/3: Building all services..."
    START_TIME=$(date +%s)
    docker-compose build --parallel
    SERVICES_BUILD_TIME=$(($(date +%s) - START_TIME))
    echo "✓ All services built in ${SERVICES_BUILD_TIME}s"
    
    # Show build time improvement
    TOTAL_BUILD_TIME=$((BASE_BUILD_TIME + SERVICES_BUILD_TIME))
    echo "📊 Total build time: ${TOTAL_BUILD_TIME}s (${BASE_BUILD_TIME}s base + ${SERVICES_BUILD_TIME}s services)"
}

# Function to start services
start_services() {
    echo "Step 3/3: Starting all services..."
    
    START_TIME=$(date +%s)
    docker-compose up -d
    STARTUP_TIME=$(($(date +%s) - START_TIME))
    echo "✓ Services started in ${STARTUP_TIME}s"
    
    # Wait for services to be ready
    echo "Waiting for services to initialize..."
    sleep 15
}

# Function to verify deployment
verify_deployment() {
    echo "Verifying deployment..."
    
    # Check container status
    echo "Container status:"
    docker-compose ps
    echo
    
    # Count running containers
    RUNNING_CONTAINERS=$(docker-compose ps | grep "Up" | wc -l)
    TOTAL_CONTAINERS=$(docker-compose ps -a | tail -n +3 | wc -l)
    
    echo "📊 Service Status: $RUNNING_CONTAINERS/$TOTAL_CONTAINERS containers running"
    
    # Check specific services
    SERVICES=("auth-gateway" "tenant-node" "metadata-catalog" "cbo-engine" "operation-node" "monitoring")
    echo
    echo "Service Health Check:"
    
    for service in "${SERVICES[@]}"; do
        if docker-compose ps "$service" | grep -q "Up"; then
            echo "✓ $service: Running"
        else
            echo "❌ $service: Not running"
            echo "  Logs for $service:"
            docker-compose logs --tail=5 "$service" | sed 's/^/    /'
        fi
    done
}

# Function to show final status
show_final_status() {
    echo
    echo "=== Deployment Complete ==="
    echo "Deployment finished at $(date)"
    echo
    echo "🌐 Available Services:"
    
    # Get EC2 public IP (try multiple methods)
    PUBLIC_IP=""
    if command_exists curl; then
        PUBLIC_IP=$(curl -s http://checkip.amazonaws.com/ 2>/dev/null || echo "localhost")
    elif command_exists wget; then
        PUBLIC_IP=$(wget -qO- http://checkip.amazonaws.com/ 2>/dev/null || echo "localhost")
    else
        PUBLIC_IP="your-ec2-ip"
    fi
    
    echo "  🔐 Auth Gateway:      http://$PUBLIC_IP:8080"
    echo "  🏠 Tenant Node:       http://$PUBLIC_IP:8001"
    echo "  📊 Metadata Catalog:  http://$PUBLIC_IP:8087"
    echo "  🧠 CBO Engine:        http://$PUBLIC_IP:8088"
    echo "  ⚡ Operation Node:    http://$PUBLIC_IP:8086"
    echo "  📈 Monitoring:        http://$PUBLIC_IP:8089"
    echo "  📊 Grafana:           http://$PUBLIC_IP:3000 (admin/admin)"
    echo "  🔍 Prometheus:        http://$PUBLIC_IP:9090"
    echo
    echo "📋 Management Commands:"
    echo "  View logs:           docker-compose logs [service-name]"
    echo "  Restart service:     docker-compose restart [service-name]"
    echo "  Stop all:            docker-compose down"
    echo "  View status:         docker-compose ps"
    echo
    echo "🔧 Troubleshooting:"
    echo "  If a service fails, check its logs:"
    echo "    docker-compose logs [service-name]"
    echo "  To restart a specific service:"
    echo "    docker-compose restart [service-name]"
}

# Main execution
main() {
    echo "Starting optimized storage system deployment on EC2..."
    echo "Current directory: $(pwd)"
    echo "Current user: $(whoami)"
    echo
    
    check_prerequisites
    check_resources
    backup_current_state
    cleanup_old_resources
    build_optimized_images
    start_services
    verify_deployment
    show_final_status
    
    echo "🎉 Deployment script completed successfully!"
}

# Run main function
main "$@"
