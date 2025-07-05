# EC2 Deployment Script for Optimized Storage System (PowerShell version)
# Run this script on your EC2 instance (if using PowerShell) or WSL

param(
    [switch]$SkipCleanup,
    [switch]$Verbose
)

Write-Host "=== EC2 Deployment: Optimized Storage System ===" -ForegroundColor Green
Write-Host "Starting deployment at $(Get-Date)" -ForegroundColor Gray

# Function to check if command exists
function Test-Command {
    param($Command)
    try {
        Get-Command $Command -ErrorAction Stop | Out-Null
        return $true
    } catch {
        return $false
    }
}

# Function to check system resources
function Test-SystemResources {
    Write-Host "Checking system resources..." -ForegroundColor Yellow
    
    # Check disk space (this works on Linux/WSL)
    if (Test-Path "/") {
        $diskInfo = df / | Select-Object -Skip 1
        $availableGB = [int]($diskInfo -split '\s+')[3] / 1048576
        
        if ($availableGB -lt 8) {
            Write-Warning "Low disk space. Available: $([math]::Round($availableGB, 1))GB, Recommended: 8GB"
            $continue = Read-Host "Continue anyway? (y/n)"
            if ($continue -notmatch '^[Yy]$') {
                exit 1
            }
        }
    }
    
    Write-Host "✓ Resource check complete" -ForegroundColor Green
}

# Function to verify prerequisites
function Test-Prerequisites {
    Write-Host "Checking prerequisites..." -ForegroundColor Yellow
    
    if (-not (Test-Command "docker")) {
        Write-Error "Docker is not installed. Please install Docker first."
        exit 1
    }
    
    if (-not (Test-Command "docker-compose")) {
        Write-Error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    }
    
    # Test Docker access
    try {
        docker ps | Out-Null
    } catch {
        Write-Error "Cannot access Docker. Check permissions or try with sudo."
        exit 1
    }
    
    Write-Host "✓ Prerequisites check passed" -ForegroundColor Green
}

# Function to backup current state
function Backup-CurrentState {
    Write-Host "Creating backup of current state..." -ForegroundColor Yellow
    
    $backupDir = "backup_$(Get-Date -Format 'yyyyMMdd_HHmmss')"
    New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
    
    # Export current container data if any
    $runningContainers = docker-compose ps -q
    if ($runningContainers) {
        Write-Host "Backing up current container state..."
        docker-compose ps | Out-File "$backupDir/container_state.txt"
        docker-compose logs | Out-File "$backupDir/container_logs.txt" -ErrorAction SilentlyContinue
    }
    
    Write-Host "✓ Backup created in $backupDir" -ForegroundColor Green
}

# Function to clean up old resources
function Remove-OldResources {
    if ($SkipCleanup) {
        Write-Host "Skipping cleanup (SkipCleanup flag set)" -ForegroundColor Yellow
        return
    }
    
    Write-Host "Cleaning up old resources..." -ForegroundColor Yellow
    
    # Stop any running services
    Write-Host "Stopping existing services..."
    docker-compose down --volumes --remove-orphans 2>$null
    
    # Show current Docker disk usage
    Write-Host "Current Docker disk usage:"
    docker system df
    Write-Host ""
    
    $cleanup = Read-Host "Clean up old Docker images to save space? (y/n)"
    if ($cleanup -match '^[Yy]$') {
        Write-Host "Cleaning up old Docker resources..."
        docker system prune -a --volumes -f
        Write-Host "✓ Cleanup complete" -ForegroundColor Green
    } else {
        Write-Host "Skipping cleanup"
    }
}

# Function to build optimized images
function Build-OptimizedImages {
    Write-Host "Building optimized Docker images..." -ForegroundColor Yellow
    
    # Build the shared base image
    Write-Host "Step 1/3: Building shared base image..."
    $startTime = Get-Date
    docker build -f Dockerfile.base -t storage-python-base:latest .
    $baseBuildTime = [int]((Get-Date) - $startTime).TotalSeconds
    Write-Host "✓ Base image built in ${baseBuildTime}s" -ForegroundColor Green
    
    # Build all services
    Write-Host "Step 2/3: Building all services..."
    $startTime = Get-Date
    docker-compose build --parallel
    $servicesBuildTime = [int]((Get-Date) - $startTime).TotalSeconds
    Write-Host "✓ All services built in ${servicesBuildTime}s" -ForegroundColor Green
    
    # Show build time improvement
    $totalBuildTime = $baseBuildTime + $servicesBuildTime
    Write-Host "📊 Total build time: ${totalBuildTime}s (${baseBuildTime}s base + ${servicesBuildTime}s services)" -ForegroundColor Cyan
}

# Function to start services
function Start-Services {
    Write-Host "Step 3/3: Starting all services..." -ForegroundColor Yellow
    
    $startTime = Get-Date
    docker-compose up -d
    $startupTime = [int]((Get-Date) - $startTime).TotalSeconds
    Write-Host "✓ Services started in ${startupTime}s" -ForegroundColor Green
    
    # Wait for services to be ready
    Write-Host "Waiting for services to initialize..."
    Start-Sleep -Seconds 15
}

# Function to verify deployment
function Test-Deployment {
    Write-Host "Verifying deployment..." -ForegroundColor Yellow
    
    # Check container status
    Write-Host "Container status:"
    docker-compose ps
    Write-Host ""
    
    # Count running containers
    $runningContainers = (docker-compose ps | Select-String "Up").Count
    $totalContainers = (docker-compose ps -a | Select-Object -Skip 2).Count
    
    Write-Host "📊 Service Status: $runningContainers/$totalContainers containers running" -ForegroundColor Cyan
    
    # Check specific services
    $services = @("auth-gateway", "tenant-node", "metadata-catalog", "cbo-engine", "operation-node", "monitoring")
    Write-Host ""
    Write-Host "Service Health Check:"
    
    foreach ($service in $services) {
        $status = docker-compose ps $service
        if ($status -match "Up") {
            Write-Host "✓ $service`: Running" -ForegroundColor Green
        } else {
            Write-Host "❌ $service`: Not running" -ForegroundColor Red
            Write-Host "  Logs for $service`:"
            docker-compose logs --tail=5 $service | ForEach-Object { Write-Host "    $_" }
        }
    }
}

# Function to show final status
function Show-FinalStatus {
    Write-Host ""
    Write-Host "=== Deployment Complete ===" -ForegroundColor Green
    Write-Host "Deployment finished at $(Get-Date)" -ForegroundColor Gray
    Write-Host ""
    
    # Try to get public IP
    $publicIP = "your-ec2-ip"
    try {
        $publicIP = (Invoke-WebRequest -Uri "http://checkip.amazonaws.com/" -UseBasicParsing).Content.Trim()
    } catch {
        $publicIP = "localhost"
    }
    
    Write-Host "🌐 Available Services:" -ForegroundColor Cyan
    Write-Host "  🔐 Auth Gateway:      http://$publicIP`:8080"
    Write-Host "  🏠 Tenant Node:       http://$publicIP`:8001"
    Write-Host "  📊 Metadata Catalog:  http://$publicIP`:8087"
    Write-Host "  🧠 CBO Engine:        http://$publicIP`:8088"
    Write-Host "  ⚡ Operation Node:    http://$publicIP`:8086"
    Write-Host "  📈 Monitoring:        http://$publicIP`:8089"
    Write-Host "  📊 Grafana:           http://$publicIP`:3000 (admin/admin)"
    Write-Host "  🔍 Prometheus:        http://$publicIP`:9090"
    Write-Host ""
    Write-Host "📋 Management Commands:" -ForegroundColor Cyan
    Write-Host "  View logs:           docker-compose logs [service-name]"
    Write-Host "  Restart service:     docker-compose restart [service-name]"
    Write-Host "  Stop all:            docker-compose down"
    Write-Host "  View status:         docker-compose ps"
}

# Main execution
function Main {
    Write-Host "Starting optimized storage system deployment on EC2..." -ForegroundColor White
    Write-Host "Current directory: $(Get-Location)"
    Write-Host "Current user: $env:USERNAME"
    Write-Host ""
    
    Test-Prerequisites
    Test-SystemResources
    Backup-CurrentState
    Remove-OldResources
    Build-OptimizedImages
    Start-Services
    Test-Deployment
    Show-FinalStatus
    
    Write-Host "🎉 Deployment script completed successfully!" -ForegroundColor Green
}

# Run main function
try {
    Main
} catch {
    Write-Error "Deployment failed: $_"
    exit 1
}
