#!/bin/bash

# Final complete fix for tenant-node issues
# This includes logging fixes for the structured logging issue

echo "=== Final Tenant-Node Fix (Including Logging) ==="

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

# 3. Fix structured logging issues in tenant_node.py
echo "  - Fixing structured logging..."
sed -i 's/logger\.info("Initializing Tenant Node", tenant_id=self\.config\.tenant_id)/logger.info(f"Initializing Tenant Node for tenant: {self.config.tenant_id}")/' tenant_node.py
sed -i 's/logger\.info("Tenant Node initialized successfully", tenant_id=self\.config\.tenant_id)/logger.info(f"Tenant Node initialized successfully for tenant: {self.config.tenant_id}")/' tenant_node.py
sed -i 's/logger\.info("Starting gRPC server", port=self\.config\.grpc_port)/logger.info(f"Starting gRPC server on port {self.config.grpc_port}")/' tenant_node.py
sed -i 's/logger\.error("Failed to start gRPC server", error=str(e))/logger.error(f"Failed to start gRPC server: {str(e)}")/' tenant_node.py
sed -i 's/logger\.info("Starting REST API server",.*port=self\.config\.rest_port)/logger.info(f"Starting REST API server on {self.config.rest_host}:{self.config.rest_port}")/' tenant_node.py
sed -i 's/logger\.error("Failed to start REST API server", error=str(e))/logger.error(f"Failed to start REST API server: {str(e)}")/' tenant_node.py
sed -i 's/logger\.error("Error running tenant node", error=str(e))/logger.error(f"Error running tenant node: {str(e)}")/' tenant_node.py
sed -i 's/logger\.info("Received signal, initiating shutdown", signal=signum)/logger.info(f"Received signal {signum}, initiating shutdown")/' tenant_node.py
sed -i 's/logger\.info("Shutting down Tenant Node", tenant_id=self\.config\.tenant_id)/logger.info(f"Shutting down Tenant Node for tenant: {self.config.tenant_id}")/' tenant_node.py
sed -i 's/logger\.info("Added sample source", source_id=source_config\.source_id)/logger.info(f"Added sample source: {source_config.source_id}")/' tenant_node.py
sed -i 's/logger\.error("Failed to add sample source",.*error=str(e))/logger.error(f"Failed to add sample source {source_config.source_id}: {str(e)}")/' tenant_node.py

# 4. Add missing stop method to rest_api.py (if not already added)
echo "  - Ensuring stop() method exists in TenantNodeAPI..."
if ! grep -q "async def stop" rest_api.py; then
    # Add the stop method after the start method
    sed -i '/await self\.server\.serve()/a\\n    async def stop(self):\n        """Stop the REST API server"""\n        if hasattr(self, '\''server'\'') and self.server:\n            logger.info("Stopping Tenant Node REST API server")\n            self.server.should_exit = True\n            # Give it a moment to shutdown gracefully\n            await asyncio.sleep(0.1)' rest_api.py
    
    # Also modify the start method to store server reference
    sed -i 's/server = uvicorn\.Server(config)/self.server = uvicorn.Server(config)/' rest_api.py
    sed -i 's/await server\.serve()/await self.server.serve()/' rest_api.py
fi

cd ..

# 5. Remove old image and rebuild
echo "Rebuilding tenant-node service..."
docker rmi storage_system-tenant-node 2>/dev/null || true
docker-compose build --no-cache tenant-node

# 6. Start the service
echo "Starting tenant-node service..."
docker-compose up -d tenant-node

# 7. Wait and check status
echo "Waiting for tenant-node to start..."
sleep 25

echo "=== Status Check ==="
docker-compose ps tenant-node

echo "=== Recent Logs ==="
docker-compose logs --tail=20 tenant-node

# 8. Final verification
if docker-compose ps tenant-node | grep -q "Up"; then
    echo ""
    echo "🎉 SUCCESS! Tenant-node is now running!"
    PUBLIC_IP=$(curl -s http://checkip.amazonaws.com/ 2>/dev/null || echo 'localhost')
    echo "Service available at: http://$PUBLIC_IP:8001"
    echo ""
    echo "🌟 All services status:"
    docker-compose ps
    echo ""
    echo "📋 Available endpoints:"
    echo "  🔐 Auth Gateway:      http://$PUBLIC_IP:8080"
    echo "  🏠 Tenant Node:       http://$PUBLIC_IP:8001"
    echo "  📊 Metadata Catalog:  http://$PUBLIC_IP:8087"
    echo "  🧠 CBO Engine:        http://$PUBLIC_IP:8088"
    echo "  ⚡ Operation Node:    http://$PUBLIC_IP:8086"
    echo "  📈 Monitoring:        http://$PUBLIC_IP:8089"
    echo "  📊 Grafana:           http://$PUBLIC_IP:3000 (admin/admin)"
    echo "  🔍 Prometheus:        http://$PUBLIC_IP:9090"
else
    echo ""
    echo "❌ Tenant-node still not running. Let's debug..."
    echo "=== Full Logs ==="
    docker-compose logs tenant-node
    echo ""
    echo "=== Container inspection ==="
    docker inspect storage_system-tenant-node-1 2>/dev/null | grep -A 5 -B 5 "State" || echo "Container not found"
fi

echo "=== Fix Complete ==="
