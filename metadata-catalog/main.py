"""
Metadata Catalog - Centralized metadata and compaction management
"""
import asyncio
import logging
from pathlib import Path
from metadata import MetadataManager
from compaction_manager import FileCompactionManager

logger = logging.getLogger(__name__)


class MetadataCatalogService:
    """Metadata Catalog service for centralized metadata management"""
    
    def __init__(self, config_path: str = "config.json"):
        self.config_path = config_path
        self.metadata_manager = None
        self.compaction_manager = None
        
    async def start(self):
        """Start the metadata catalog service"""
        logger.info("Starting Metadata Catalog service...")
        
        # Create default tenant config
        class TenantConfig:
            def __init__(self):
                self.tenant_id = "default"
                self.tenant_name = "Default Tenant"
                self.base_data_path = "/app/data"
                self.max_memory_mb = 1024
                self.max_cpu_cores = 2
        
        tenant_config = TenantConfig()
        
        # Initialize managers
        metadata_path = Path("/app/data/metadata")
        self.metadata_manager = MetadataManager(metadata_path)
        await self.metadata_manager.initialize()
        self.compaction_manager = FileCompactionManager(tenant_config=tenant_config)
        
        # Start gRPC server
        # TODO: Implement gRPC server for metadata operations
        
        # For now, keep the service running
        logger.info("Metadata Catalog service started and running...")
        
        # Keep the service alive
        try:
            while True:
                await asyncio.sleep(60)  # Sleep for 1 minute intervals
        except asyncio.CancelledError:
            logger.info("Metadata Catalog service shutting down...")
            raise
        
    async def stop(self):
        """Stop the metadata catalog service"""
        logger.info("Stopping Metadata Catalog service...")
        # Cleanup resources
        logger.info("Metadata Catalog service stopped")


if __name__ == "__main__":
    import sys
    
    # Configure logging
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    
    # Start service
    service = MetadataCatalogService()
    
    try:
        asyncio.run(service.start())
    except KeyboardInterrupt:
        asyncio.run(service.stop())
        sys.exit(0)
