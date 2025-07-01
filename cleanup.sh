#!/bin/bash

# Cleanup script for development environment

echo "🧹 Cleaning up development environment..."

# Remove Python cache files
echo "Removing Python cache files..."
find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null
find . -type d -name ".pytest_cache" -exec rm -rf {} + 2>/dev/null
find . -name "*.pyc" -delete 2>/dev/null
find . -name "*.pyo" -delete 2>/dev/null

# Remove log files
echo "Removing log files..."
find . -name "*.log" -delete 2>/dev/null

# Remove temporary data directories
echo "Removing temporary data..."
rm -rf demo_data 2>/dev/null
rm -rf demo_data_advanced 2>/dev/null
rm -rf ./data 2>/dev/null

# Remove service process files
echo "Removing process files..."
rm -f .pids 2>/dev/null

# Remove Docker volumes (optional - uncomment if needed)
# echo "Removing Docker volumes..."
# docker-compose down -v 2>/dev/null

echo "✅ Cleanup complete!"
