# Multi-Tenant Storage System - Development Startup Script

param(
    [switch]$Docker,
    [switch]$Individual
)

# Colors for PowerShell output
function Write-ColorOutput($ForegroundColor) {
    $fc = $host.UI.RawUI.ForegroundColor
    $host.UI.RawUI.ForegroundColor = $ForegroundColor
    if ($args) {
        Write-Output $args
    } else {
        $input | Write-Output
    }
    $host.UI.RawUI.ForegroundColor = $fc
}

Write-ColorOutput Green "🚀 Starting Multi-Tenant Storage System (Microservices)"
Write-ColorOutput Green "========================================================"

if ($Docker) {
    Write-ColorOutput Yellow "Starting with Docker Compose..."
    docker-compose up --build
} elseif ($Individual) {
    Write-ColorOutput Yellow "Starting individual services..."
    
    # Array to store job objects
    $jobs = @()
    
    # Function to start a service
    function Start-Service($ServiceName, $ServiceDir, $Port) {
        Write-ColorOutput Blue "Starting $ServiceName on port $Port..."
        
        $job = Start-Job -ScriptBlock {
            param($dir)
            Set-Location $dir
            if (Test-Path "requirements.txt") {
                pip install -r requirements.txt
            }
            python main.py
        } -ArgumentList (Join-Path $PWD $ServiceDir)
        
        $jobs += $job
        Write-ColorOutput Green "$ServiceName started (Job ID: $($job.Id))"
        Start-Sleep 2
        return $job
    }
    
    Write-ColorOutput Yellow "Starting services in dependency order..."
    
    # Start services
    $authJob = Start-Service "Auth Gateway" "auth-gateway" "8080"
    $monitorJob = Start-Service "Monitoring Service" "monitoring" "8084"
    $cboJob = Start-Service "CBO Engine" "cbo-engine" "8082"
    $metadataJob = Start-Service "Metadata Catalog" "metadata-catalog" "8083"
    $operationJob = Start-Service "Operation Node" "operation-node" "8081"
    $tenantJob = Start-Service "Tenant Node" "tenant-node" "8000"
    
    Write-ColorOutput Green "All services started successfully!"
    Write-ColorOutput Yellow ""
    Write-ColorOutput Green "🌐 Service URLs:"
    Write-Output "  Auth Gateway:      http://localhost:8080"
    Write-Output "  Tenant Node:       http://localhost:8000"
    Write-Output "  Operation Node:    http://localhost:8081"
    Write-Output "  CBO Engine:        http://localhost:8082"
    Write-Output "  Metadata Catalog:  http://localhost:8083"
    Write-Output "  Monitoring:        http://localhost:8084"
    Write-ColorOutput Yellow ""
    Write-ColorOutput Green "📊 Health Checks:"
    Write-Output "  Invoke-RestMethod http://localhost:8080/health"
    Write-Output "  Invoke-RestMethod http://localhost:8000/health"
    Write-Output "  Invoke-RestMethod http://localhost:8084/status"
    Write-ColorOutput Yellow ""
    Write-ColorOutput Green "🔑 Authentication:"
    Write-Output "  `$body = @{username='admin'; password='admin123'} | ConvertTo-Json"
    Write-Output "  Invoke-RestMethod -Uri http://localhost:8080/auth/login -Method POST -Body `$body -ContentType 'application/json'"
    Write-ColorOutput Yellow ""
    Write-ColorOutput Red "To stop all services, run: ./stop_services.ps1"
    Write-ColorOutput Yellow ""
    Write-ColorOutput Yellow "Press Ctrl+C to stop monitoring and all services..."
    
    # Monitor services
    try {
        while ($true) {
            Start-Sleep 5
            
            # Check job status
            $failedJobs = $jobs | Where-Object { $_.State -eq "Failed" -or $_.State -eq "Completed" }
            if ($failedJobs) {
                Write-ColorOutput Red "Some services have stopped:"
                $failedJobs | ForEach-Object { Write-ColorOutput Red "  Job $($_.Id): $($_.State)" }
            }
        }
    } catch {
        Write-ColorOutput Yellow "Stopping all services..."
        $jobs | Stop-Job
        $jobs | Remove-Job
        Write-ColorOutput Green "All services stopped."
    }
} else {
    Write-ColorOutput Yellow "Usage:"
    Write-Output "  ./start_services.ps1 -Docker        # Start with Docker Compose"
    Write-Output "  ./start_services.ps1 -Individual    # Start individual Python services"
    Write-Output "  ./start_services.ps1                # Show this help"
}
