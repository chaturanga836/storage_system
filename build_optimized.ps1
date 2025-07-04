# Build optimized Docker images script for PowerShell
# This script builds the shared base image first, then all service images

Write-Host "=== Building Optimized Storage System Docker Images ===" -ForegroundColor Green

# Step 1: Build the shared base image
Write-Host "Step 1: Building shared base image..." -ForegroundColor Yellow
docker build -f Dockerfile.base -t storage-python-base:latest .
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Base image built successfully" -ForegroundColor Green
} else {
    Write-Host "✗ Failed to build base image" -ForegroundColor Red
    exit 1
}

# Step 2: Clean up any existing containers/images
Write-Host "Step 2: Cleaning up existing containers..." -ForegroundColor Yellow
docker-compose down --volumes --remove-orphans 2>$null
Write-Host "✓ Cleanup completed" -ForegroundColor Green

# Step 3: Build all services with the new optimized Dockerfiles
Write-Host "Step 3: Building all services with optimized Dockerfiles..." -ForegroundColor Yellow
docker-compose build --no-cache
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ All services built successfully" -ForegroundColor Green
} else {
    Write-Host "✗ Failed to build services" -ForegroundColor Red
    exit 1
}

# Step 4: Start all services
Write-Host "Step 4: Starting all services..." -ForegroundColor Yellow
docker-compose up -d
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ All services started" -ForegroundColor Green
} else {
    Write-Host "✗ Failed to start services" -ForegroundColor Red
    exit 1
}

# Step 5: Show status
Write-Host "Step 5: Checking service status..." -ForegroundColor Yellow
Start-Sleep -Seconds 10
docker-compose ps

Write-Host ""
Write-Host "=== Build Complete ===" -ForegroundColor Green
Write-Host "All services should now be running with optimized Docker images." -ForegroundColor Cyan
Write-Host "Check the status above. If any service is not 'Up', check logs with:" -ForegroundColor Cyan
Write-Host "  docker-compose logs [service-name]" -ForegroundColor White
Write-Host ""
Write-Host "Available services:" -ForegroundColor Cyan
Write-Host "  - auth-gateway:        http://localhost:8080" -ForegroundColor White
Write-Host "  - operation-node:      http://localhost:8086" -ForegroundColor White  
Write-Host "  - cbo-engine:          http://localhost:8088" -ForegroundColor White
Write-Host "  - metadata-catalog:    http://localhost:8087" -ForegroundColor White
Write-Host "  - monitoring:          http://localhost:8089" -ForegroundColor White
Write-Host "  - tenant-node:         http://localhost:8001" -ForegroundColor White
Write-Host "  - postgres:            localhost:5432" -ForegroundColor White
Write-Host "  - redis:               localhost:6379" -ForegroundColor White
Write-Host "  - prometheus:          http://localhost:9090" -ForegroundColor White
Write-Host "  - grafana:             http://localhost:3000" -ForegroundColor White
