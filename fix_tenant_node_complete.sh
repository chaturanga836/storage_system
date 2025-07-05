#!/bin/bash

# Complete fix script for tenant-node issues
# Run this on your EC2 instance to fix all tenant-node problems

echo "=== Complete Tenant-Node Fix ==="

# Stop the tenant-node service first
echo "Stopping tenant-node service..."
docker-compose stop tenant-node
docker-compose rm -f tenant-node

# Fix the files
echo "Fixing tenant-node files..."
cd tenant-node

# 1. Fix import statements
echo "  - Fixing imports..."
sed -i 's/from \.config import/from config import/g' grpc_service.py
sed -i 's/from \.data_source import/from data_source import/g' grpc_service.py
sed -i 's/from \.generated import/from generated import/g' grpc_service.py
sed -i 's/from \.config import/from config import/g' rest_api.py
sed -i 's/from \.data_source import/from data_source import/g' rest_api.py

# 2. Fix main.py entry point
echo "  - Fixing main.py..."
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

# 3. Add missing stop method to rest_api.py
echo "  - Adding stop() method to TenantNodeAPI..."
# Find the line where start method ends and add stop method after it
sed -i '/await self\.server\.serve()/a\\n    async def stop(self):\n        """Stop the REST API server"""\n        if hasattr(self, '\''server'\'') and self.server:\n            logger.info("Stopping Tenant Node REST API server")\n            self.server.should_exit = True\n            # Give it a moment to shutdown gracefully\n            await asyncio.sleep(0.1)' rest_api.py

# Also modify the start method to store server reference
sed -i 's/server = uvicorn\.Server(config)/self.server = uvicorn.Server(config)/' rest_api.py
sed -i 's/await server\.serve()/await self.server.serve()/' rest_api.py

cd ..

# 4. Remove old image and rebuild
echo "Rebuilding tenant-node service..."
docker rmi storage_system-tenant-node 2>/dev/null || true
docker-compose build --no-cache tenant-node

# 5. Start the service
echo "Starting tenant-node service..."
docker-compose up -d tenant-node

# 6. Wait and check status
echo "Waiting for tenant-node to start..."
sleep 20

echo "=== Status Check ==="
docker-compose ps tenant-node

echo "=== Recent Logs ==="
docker-compose logs --tail=15 tenant-node

# 7. Final verification
if docker-compose ps tenant-node | grep -q "Up"; then
    echo ""
    echo "🎉 SUCCESS! Tenant-node is now running!"
    echo "Service available at: http://$(curl -s http://checkip.amazonaws.com/ 2>/dev/null || echo 'localhost'):8001"
    echo ""
    echo "All services status:"
    docker-compose ps
else
    echo ""
    echo "❌ Tenant-node still not running. Manual investigation needed."
    echo "Check logs with: docker-compose logs tenant-node"
fi

echo "=== Fix Complete ==="
