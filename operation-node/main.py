"""
Operation Node - Central coordinator for multi-tenant operations
"""
import asyncio
import logging
from typing import Dict, List
from auto_scaler import AutoScaler

logger = logging.getLogger(__name__)


class OperationNodeService:
    """Operation Node service for managing tenants and auto-scaling"""
    
    def __init__(self, config_path: str = "config.json"):
        self.config_path = config_path
        self.auto_scaler = None
        self.tenant_registry = {}
        
    async def start(self):
        """Start the operation node service"""
        logger.info("Starting Operation Node service...")
        
        # Create default tenant config
        class TenantConfig:
            def __init__(self):
                self.tenant_id = "default"
                self.tenant_name = "Default Tenant"
                self.base_data_path = "/app/data"
                self.max_memory_mb = 1024
                self.max_cpu_cores = 2
                self.max_concurrent_searches = 4
        
        tenant_config = TenantConfig()
        
        # Initialize auto-scaler
        self.auto_scaler = AutoScaler(tenant_config=tenant_config)
        
        # Start monitoring and coordination services
        # TODO: Implement gRPC server for operation coordination
        
        logger.info("Operation Node service started and running...")
        
        # Keep the service alive
        try:
            while True:
                await asyncio.sleep(60)  # Sleep for 1 minute intervals
        except asyncio.CancelledError:
            logger.info("Operation Node service shutting down...")
            raise
        
    async def stop(self):
        """Stop the operation node service"""
        logger.info("Stopping Operation Node service...")
        # Cleanup resources
        logger.info("Operation Node service stopped")
    
    async def register_tenant(self, tenant_id: str, tenant_config: Dict):
        """Register a new tenant with the operation node"""
        self.tenant_registry[tenant_id] = tenant_config
        logger.info(f"Registered tenant: {tenant_id}")
    
    async def unregister_tenant(self, tenant_id: str):
        """Unregister a tenant from the operation node"""
        if tenant_id in self.tenant_registry:
            del self.tenant_registry[tenant_id]
            logger.info(f"Unregistered tenant: {tenant_id}")


if __name__ == "__main__":
    import sys
    
    # Configure logging
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    
    # Start service
    service = OperationNodeService()
    
    try:
        asyncio.run(service.start())
    except KeyboardInterrupt:
        asyncio.run(service.stop())
        sys.exit(0)
