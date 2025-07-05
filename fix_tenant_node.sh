#!/bin/bash

# Quick fix script for tenant-node import issues
# Run this on your EC2 instance to fix the import errors

echo "=== Quick Fix: Tenant Node Import Issues ==="

# Stop the tenant-node service first
echo "Stopping tenant-node service..."
docker-compose stop tenant-node

# Fix grpc_service.py relative imports - more comprehensive fix
echo "Fixing grpc_service.py imports..."
cd tenant-node
# Replace all relative imports in grpc_service.py
sed -i 's/from \.config import/from config import/g' grpc_service.py
sed -i 's/from \.data_source import/from data_source import/g' grpc_service.py
sed -i 's/from \.generated import/from generated import/g' grpc_service.py

# Fix rest_api.py relative imports
echo "Fixing rest_api.py imports..."
sed -i 's/from \.config import/from config import/g' rest_api.py
sed -i 's/from \.data_source import/from data_source import/g' rest_api.py

# Fix main.py to use the correct entry point
echo "Fixing main.py..."
cat > main.py << 'EOF'
"""
Tenant Node main entry point
"""
import asyncio
import logging
import sys

# Import the main tenant node application
from tenant_node import main as tenant_node_main

logger = logging.getLogger(__name__)


if __name__ == "__main__":
    try:
        # Run the tenant node main function
        asyncio.run(tenant_node_main())
    except KeyboardInterrupt:
        logger.info("Tenant Node service stopped")
        print("\nShutdown complete.")
    except Exception as e:
        logger.error(f"Fatal error: {e}")
        print(f"Fatal error: {e}")
        sys.exit(1)
EOF

# Check if there are any other relative imports
echo "Checking for remaining relative imports..."
grep -n "from \." *.py || echo "No relative imports found"

cd ..

# Remove obsolete version from docker-compose.yml
echo "Fixing docker-compose.yml version warning..."
sed -i '/^version:/d' docker-compose.yml

# Clean up the image and rebuild
echo "Removing old tenant-node image..."
docker-compose rm -f tenant-node
docker rmi storage_system-tenant-node 2>/dev/null || true

# Rebuild tenant-node service
echo "Rebuilding tenant-node service..."
docker-compose build --no-cache tenant-node

echo "Starting tenant-node service..."
docker-compose up -d tenant-node

# Wait a moment and check status
sleep 15
echo "Checking tenant-node status..."
docker-compose ps tenant-node

# Show logs
echo "Recent tenant-node logs:"
docker-compose logs --tail=10 tenant-node

# Check if it's running
if docker-compose ps tenant-node | grep -q "Up"; then
    echo "✓ Tenant-node is now running successfully!"
    echo "Service available at: http://$(curl -s http://checkip.amazonaws.com/):8001"
else
    echo "❌ Tenant-node still not running. Let's check the file contents:"
    echo "=== grpc_service.py imports ==="
    head -20 tenant-node/grpc_service.py | grep import
    echo "=== rest_api.py imports ==="
    head -20 tenant-node/rest_api.py | grep import
fi

echo "=== Fix Complete ==="
