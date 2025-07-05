#!/bin/bash

# Quick fix for the 'TenantNodeAPI' object has no attribute 'self' error

echo "=== Quick Fix for TenantNodeAPI 'self' Error ==="

# Stop the tenant-node service
echo "Stopping tenant-node service..."
docker-compose stop tenant-node
docker-compose rm -f tenant-node

cd tenant-node

# The issue is likely in the start method. Let's rewrite it cleanly
echo "Fixing TenantNodeAPI start method..."

# Create a clean version of the start and stop methods
cat > temp_fix.py << 'EOF'
    async def start(self):
        """Start the REST API server"""
        config = uvicorn.Config(
            self.app,
            host=self.tenant_config.rest_host,
            port=self.tenant_config.rest_port,
            log_level="info"
        )
        
        self.server = uvicorn.Server(config)
        
        logger.info(f"Tenant Node REST API starting on {self.tenant_config.rest_host}:{self.tenant_config.rest_port}")
        
        await self.server.serve()
    
    async def stop(self):
        """Stop the REST API server"""
        if hasattr(self, 'server') and self.server:
            logger.info("Stopping Tenant Node REST API server")
            self.server.should_exit = True
            await asyncio.sleep(0.1)
    
    def get_app(self):
        """Get the FastAPI application"""
        return self.app
EOF

# Replace the methods in rest_api.py
# First, remove the existing start/stop/get_app methods
sed -i '/async def start(self):/,/def get_app(self):/c\' rest_api.py

# Add the clean methods at the end of the class
sed -i '/^class TenantNodeAPI:/,/^[[:space:]]*def get_app/ { /^[[:space:]]*def get_app/r temp_fix.py
}' rest_api.py

# Remove the temp file
rm temp_fix.py

cd ..

# Rebuild and restart
echo "Rebuilding tenant-node..."
docker-compose build --no-cache tenant-node

echo "Starting tenant-node..."
docker-compose up -d tenant-node

# Wait and check
sleep 20
echo "=== Status Check ==="
docker-compose ps tenant-node

echo "=== Recent Logs ==="
docker-compose logs --tail=15 tenant-node

if docker-compose ps tenant-node | grep -q "Up"; then
    echo ""
    echo "🎉 SUCCESS! Tenant-node is finally running!"
    PUBLIC_IP=$(curl -s http://checkip.amazonaws.com/ 2>/dev/null || echo 'localhost')
    echo "Service available at: http://$PUBLIC_IP:8001"
else
    echo ""
    echo "❌ Still having issues. Let's try a simpler approach..."
    echo "Checking the actual error in rest_api.py..."
    grep -n "self" tenant-node/rest_api.py | tail -10
fi

echo "=== Fix Complete ==="
