"""
Main Tenant Node Application
"""
import asyncio
import logging
import signal
import sys
from pathlib import Path
from typing import Optional

from config import TenantConfig, SourceConfig
from data_source import SourceManager
from grpc_service import TenantNodeServer
from rest_api import TenantNodeAPI

logger = logging.getLogger(__name__)


class TenantNode:
    """Main tenant node application"""
    
    def __init__(self, config: TenantConfig):
        self.config = config
        self.source_manager = None
        self.grpc_server = None
        self.rest_api = None
        
        self._shutdown_event = asyncio.Event()
        self._background_tasks = []
        
    async def initialize(self):
        """Initialize the tenant node"""
        logger.info(f"Initializing Tenant Node for tenant: {self.config.tenant_id}")
        
        # Initialize source manager
        self.source_manager = SourceManager(self.config)
        await self.source_manager.initialize()
        
        # Initialize gRPC server
        self.grpc_server = TenantNodeServer(self.config, self.source_manager)
        
        # Initialize REST API
        self.rest_api = TenantNodeAPI(self.config, self.source_manager)
        
        logger.info(f"Tenant Node initialized successfully for tenant: {self.config.tenant_id}")
    
    async def start_background_tasks(self):
        """Start background tasks (basic implementation for tenant-node)"""
        # No background tasks needed for basic tenant-node functionality
        logger.info("Background tasks started")
    
    async def start_grpc_server(self):
        """Start the gRPC server"""
        try:
            logger.info(f"Starting gRPC server on port {self.config.grpc_port}")
            await self.grpc_server.start()
        except Exception as e:
            logger.error(f"Failed to start gRPC server: {str(e)}")
            raise
    
    async def start_rest_api(self):
        """Start the REST API server"""
        try:
            logger.info(f"Starting REST API server on {self.config.rest_host}:{self.config.rest_port}")
            await self.rest_api.start()
        except Exception as e:
            logger.error(f"Failed to start REST API server: {str(e)}")
            raise
    
    async def run(self, mode: str = "rest"):
        """Run the tenant node in specified mode"""
        await self.initialize()
        
        # Setup signal handlers
        self._setup_signal_handlers()
        
        # Start background tasks
        await self.start_background_tasks()
        
        try:
            if mode == "grpc":
                await self.start_grpc_server()
                # Keep the service running
                await self._shutdown_event.wait()
            elif mode == "rest":
                await self.start_rest_api()
                # Keep the service running
                await self._shutdown_event.wait()
            elif mode == "both":
                # Run both servers concurrently
                await asyncio.gather(
                    self.start_grpc_server(),
                    self.start_rest_api(),
                    self._shutdown_event.wait()
                )
            else:
                raise ValueError(f"Unknown mode: {mode}. Use 'rest', 'grpc', or 'both'")
                
        except KeyboardInterrupt:
            logger.info("Received keyboard interrupt, shutting down...")
        except Exception as e:
            logger.error(f"Error running tenant node: {str(e)}")
            raise
        finally:
            await self.shutdown()
    
    def _setup_signal_handlers(self):
        """Setup signal handlers for graceful shutdown"""
        def signal_handler(signum, frame):
            logger.info(f"Received signal {signum}, initiating shutdown")
            self._shutdown_event.set()
        
        signal.signal(signal.SIGINT, signal_handler)
        signal.signal(signal.SIGTERM, signal_handler)
    
    async def shutdown(self):
        """Graceful shutdown of the tenant node"""
        logger.info(f"Shutting down Tenant Node for tenant: {self.config.tenant_id}")
        
        # Signal shutdown to background tasks
        self._shutdown_event.set()
        
        # Cancel background tasks
        for task in self._background_tasks:
            if not task.done():
                task.cancel()
        
        # Wait for background tasks to complete
        if self._background_tasks:
            await asyncio.gather(*self._background_tasks, return_exceptions=True)
        
        # Shutdown servers
        if self.grpc_server:
            await self.grpc_server.stop()
        
        if self.rest_api:
            await self.rest_api.stop()
        
        # Cleanup source manager
        if self.source_manager:
            await self.source_manager.cleanup()
        
        logger.info("Tenant Node shutdown complete")
    
    def get_status(self) -> dict:
        """Get comprehensive status of the tenant node"""
        return {
            "tenant_id": self.config.tenant_id,
            "tenant_name": self.config.tenant_name,
            "grpc_port": self.config.grpc_port,
            "rest_port": self.config.rest_port,
            "source_count": len(self.source_manager.sources) if self.source_manager else 0
        }
    
    async def add_sample_sources(self):
        """Add some sample data sources for testing"""
        logger.info("Adding sample data sources")
        
        # Sample source configurations
        sample_sources = [
            SourceConfig(
                source_id="sales_data",
                name="Sales Data Source",
                connection_string="file://sales",
                data_path=str(self.config.get_source_data_path("sales_data")),
                schema_definition={
                    "order_id": "string",
                    "customer_id": "string",
                    "product_id": "string",
                    "quantity": "int64",
                    "price": "float64",
                    "order_date": "datetime64[ns]"
                },
                partition_columns=["order_date"],
                index_columns=["customer_id", "product_id", "order_date"],
                compression="snappy",
                max_file_size_mb=128,
                wal_enabled=True
            ),
            SourceConfig(
                source_id="user_events",
                name="User Events Source",
                connection_string="file://events",
                data_path=str(self.config.get_source_data_path("user_events")),
                schema_definition={
                    "event_id": "string",
                    "user_id": "string",
                    "event_type": "string",
                    "event_data": "string",
                    "timestamp": "datetime64[ns]"
                },
                partition_columns=["timestamp"],
                index_columns=["user_id", "event_type", "timestamp"],
                compression="snappy",
                max_file_size_mb=256,
                wal_enabled=True
            )
        ]
        
        for source_config in sample_sources:
            try:
                await self.source_manager.add_source(source_config)
                logger.info(f"Added sample source: {source_config.source_id}")
            except Exception as e:
                logger.error(f"Failed to add sample source {source_config.source_id}: {str(e)}")


def create_default_config() -> TenantConfig:
    """Create a default configuration"""
    config = TenantConfig.from_env()
    
    # Add default sources to config if needed
    if not config.sources:
        # This would normally be loaded from a configuration file
        pass
    
    return config


async def main():
    """Main entry point"""
    # Setup logging
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    
    # Create configuration
    config = create_default_config()
    
    # Create and run tenant node
    tenant_node = TenantNode(config)
    
    # Determine mode from command line arguments
    mode = "rest"  # default
    if len(sys.argv) > 1:
        mode = sys.argv[1].lower()
        if mode not in ["rest", "grpc", "both"]:
            print(f"Invalid mode: {mode}. Use 'rest', 'grpc', or 'both'")
            sys.exit(1)
    
    # Add sample sources if in development mode
    if config.tenant_id == "default_tenant":
        await tenant_node.initialize()
        await tenant_node.add_sample_sources()
        await tenant_node.shutdown()
        
        # Reinitialize for actual run
        tenant_node = TenantNode(config)
    
    # Run the tenant node
    await tenant_node.run(mode)


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\nShutdown complete.")
    except Exception as e:
        print(f"Fatal error: {e}")
        sys.exit(1)
