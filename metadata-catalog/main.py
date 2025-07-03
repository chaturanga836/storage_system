"""
Metadata Catalog - Centralized metadata and compaction management
"""
import asyncio
import logging
from metadata import MetadataManager
from compaction_manager import CompactionManager

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
        
        # Initialize managers
        self.metadata_manager = MetadataManager()
        self.compaction_manager = CompactionManager()
        
        # Start gRPC server
        # TODO: Implement gRPC server for metadata operations
        
        logger.info("Metadata Catalog service started")
        
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
