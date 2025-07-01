# Cleanup script for development environment (PowerShell)

param(
    [switch]$IncludeDocker
)

Write-Host "🧹 Cleaning up development environment..." -ForegroundColor Green

# Remove Python cache files
Write-Host "Removing Python cache files..." -ForegroundColor Yellow
Get-ChildItem -Path . -Recurse -Directory -Name "__pycache__" | ForEach-Object { Remove-Item $_ -Recurse -Force -ErrorAction SilentlyContinue }
Get-ChildItem -Path . -Recurse -Directory -Name ".pytest_cache" | ForEach-Object { Remove-Item $_ -Recurse -Force -ErrorAction SilentlyContinue }
Get-ChildItem -Path . -Recurse -File -Include "*.pyc", "*.pyo" | Remove-Item -Force -ErrorAction SilentlyContinue

# Remove log files
Write-Host "Removing log files..." -ForegroundColor Yellow
Get-ChildItem -Path . -Recurse -File -Include "*.log" | Remove-Item -Force -ErrorAction SilentlyContinue

# Remove temporary data directories
Write-Host "Removing temporary data..." -ForegroundColor Yellow
@("demo_data", "demo_data_advanced", "data") | ForEach-Object {
    if (Test-Path $_) {
        Remove-Item $_ -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# Remove service process files
Write-Host "Removing process files..." -ForegroundColor Yellow
if (Test-Path ".pids") {
    Remove-Item ".pids" -Force -ErrorAction SilentlyContinue
}

# Remove Docker volumes (optional)
if ($IncludeDocker) {
    Write-Host "Removing Docker volumes..." -ForegroundColor Yellow
    docker-compose down -v 2>$null
}

Write-Host "✅ Cleanup complete!" -ForegroundColor Green
