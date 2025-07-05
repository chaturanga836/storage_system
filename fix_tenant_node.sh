#!/bin/bash

# Quick fix script for tenant-node import issues
# Run this on your EC2 instance to fix the import errors

echo "=== Quick Fix: Tenant Node Import Issues ==="

# Fix grpc_service.py relative imports
echo "Fixing grpc_service.py imports..."
sed -i 's/from \.config import/from config import/g' tenant-node/grpc_service.py
sed -i 's/from \.data_source import/from data_source import/g' tenant-node/grpc_service.py

# Fix rest_api.py relative imports
echo "Fixing rest_api.py imports..."
sed -i 's/from \.config import/from config import/g' tenant-node/rest_api.py
sed -i 's/from \.data_source import/from data_source import/g' tenant-node/rest_api.py

# Remove obsolete version from docker-compose.yml
echo "Fixing docker-compose.yml version warning..."
sed -i '/^version:/d' docker-compose.yml

# Rebuild and restart tenant-node service
echo "Rebuilding tenant-node service..."
docker-compose build tenant-node

echo "Restarting tenant-node service..."
docker-compose restart tenant-node

# Wait a moment and check status
sleep 10
echo "Checking tenant-node status..."
docker-compose ps tenant-node

# Show logs if still failing
if ! docker-compose ps tenant-node | grep -q "Up"; then
    echo "Tenant-node still not running. Checking logs:"
    docker-compose logs --tail=20 tenant-node
else
    echo "✓ Tenant-node is now running successfully!"
    echo "Service available at: http://$(curl -s http://checkip.amazonaws.com/):8001"
fi

echo "=== Fix Complete ==="
