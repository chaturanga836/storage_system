"""
Main Tenant Node Application
"""
import asyncio
import logging
import signal
import sys
from pathlib import Path
from typing import Optional

from .config import TenantConfig, SourceConfig
from .data_source import SourceManager
from .grpc_service import TenantNodeServer
from .rest_api import TenantNodeAPI
from .auto_scaler import AutoScaler, ScalingPolicy
from .compaction_manager import FileCompactionManager, CompactionPolicy
from .query_optimizer import QueryOptimizer

logger = logging.getLogger(__name__)


class TenantNode:
    """Main tenant node application"""
    
    def __init__(self, config: TenantConfig):
        self.config = config
        self.source_manager = None
        self.grpc_server = None
        self.rest_api = None
        
        # New components
        self.auto_scaler = None
        self.compaction_manager = None
        self.query_optimizer = None
        
        self._shutdown_event = asyncio.Event()
        self._background_tasks = []
        
    async def initialize(self):
        """Initialize the tenant node"""
        logger.info("Initializing Tenant Node", tenant_id=self.config.tenant_id)
        
        # Initialize source manager
        self.source_manager = SourceManager(self.config)
        await self.source_manager.initialize()
        
        # Initialize auto-scaler
        if self.config.auto_scaling_enabled:
            scaling_policy = ScalingPolicy(
                cpu_scale_up_threshold=self.config.scale_up_threshold_cpu,
                cpu_scale_down_threshold=self.config.scale_down_threshold_cpu,
                memory_scale_up_threshold=self.config.scale_up_threshold_memory,
                memory_scale_down_threshold=self.config.scale_down_threshold_memory,
                min_workers=self.config.min_workers,
                max_workers=self.config.max_workers
            )
            self.auto_scaler = AutoScaler(self.config, scaling_policy)
            logger.info("Auto-scaler initialized")
        
        # Initialize compaction manager
        if self.config.auto_compaction_enabled:
            compaction_policy = CompactionPolicy(
                min_file_size_mb=self.config.min_file_size_mb,
                max_file_size_mb=self.config.max_file_size_mb,
                target_file_size_mb=self.config.target_file_size_mb,
                compaction_interval_minutes=self.config.compaction_interval_minutes
            )
            self.compaction_manager = FileCompactionManager(self.config, compaction_policy)
            logger.info("Compaction manager initialized")
        
        # Initialize query optimizer
        if self.config.query_optimization_enabled:
            self.query_optimizer = QueryOptimizer(self.config)
            logger.info("Query optimizer initialized")
        
        # Initialize gRPC server
        self.grpc_server = TenantNodeServer(self.config, self.source_manager)
        
        # Initialize REST API
        self.rest_api = TenantNodeAPI(self.config, self.source_manager)
        
        # Integrate new components with existing ones
        await self._integrate_components()
        
        logger.info("Tenant Node initialized successfully", tenant_id=self.config.tenant_id)
    
    async def _integrate_components(self):
        """Integrate new components with existing systems"""
        # Integrate auto-scaler with source manager
        if self.auto_scaler and self.source_manager:
            self.source_manager.auto_scaler = self.auto_scaler
        
        # Integrate compaction manager with source manager
        if self.compaction_manager and self.source_manager:
            self.source_manager.compaction_manager = self.compaction_manager
        
        # Integrate query optimizer with source manager
        if self.query_optimizer and self.source_manager:
            self.source_manager.query_optimizer = self.query_optimizer
    
    async def start_background_tasks(self):
        """Start background tasks for auto-scaling, compaction, etc."""
        if self.auto_scaler:
            task = asyncio.create_task(self.auto_scaler.start_monitoring())
            self._background_tasks.append(task)
            logger.info("Auto-scaler monitoring started")
        
        if self.compaction_manager:
            task = asyncio.create_task(self.compaction_manager.start_compaction_scheduler())
            self._background_tasks.append(task)
            logger.info("Compaction scheduler started")
        
        # Start statistics refresh for query optimizer
        if self.query_optimizer:
            task = asyncio.create_task(self._refresh_statistics_periodically())
            self._background_tasks.append(task)
            logger.info("Statistics refresh started")
    
    async def _refresh_statistics_periodically(self):
        """Periodically refresh statistics for query optimization"""
        while not self._shutdown_event.is_set():
            try:
                # Get all source IDs
                source_ids = await self.source_manager.list_sources()
                
                # Refresh statistics for each source
                if source_ids:
                    await self.query_optimizer._collect_statistics(source_ids)
                    logger.debug(f"Refreshed statistics for {len(source_ids)} sources")
                
                # Wait for next refresh
                await asyncio.sleep(self.config.statistics_refresh_interval_minutes * 60)
                
            except Exception as e:
                logger.error(f"Error refreshing statistics: {e}")
                await asyncio.sleep(300)  # 5 minute back-off on error
    
    async def start_grpc_server(self):
        """Start the gRPC server"""
        try:
            logger.info("Starting gRPC server", port=self.config.grpc_port)
            await self.grpc_server.start()
        except Exception as e:
            logger.error("Failed to start gRPC server", error=str(e))
            raise
    
    async def start_rest_api(self):
        """Start the REST API server"""
        try:
            logger.info("Starting REST API server", 
                       host=self.config.rest_host, 
                       port=self.config.rest_port)
            await self.rest_api.start()
        except Exception as e:
            logger.error("Failed to start REST API server", error=str(e))
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
            elif mode == "rest":
                await self.start_rest_api()
            elif mode == "both":
                # Run both servers concurrently
                await asyncio.gather(
                    self.start_grpc_server(),
                    self.start_rest_api()
                )
            else:
                raise ValueError(f"Unknown mode: {mode}. Use 'rest', 'grpc', or 'both'")
                
        except KeyboardInterrupt:
            logger.info("Received keyboard interrupt, shutting down...")
        except Exception as e:
            logger.error("Error running tenant node", error=str(e))
            raise
        finally:
            await self.shutdown()
    
    def _setup_signal_handlers(self):
        """Setup signal handlers for graceful shutdown"""
        def signal_handler(signum, frame):
            logger.info("Received signal, initiating shutdown", signal=signum)
            self._shutdown_event.set()
        
        signal.signal(signal.SIGINT, signal_handler)
        signal.signal(signal.SIGTERM, signal_handler)
    
    async def shutdown(self):
        """Graceful shutdown of the tenant node"""
        logger.info("Shutting down Tenant Node", tenant_id=self.config.tenant_id)
        
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
        status = {
            "tenant_id": self.config.tenant_id,
            "tenant_name": self.config.tenant_name,
            "grpc_port": self.config.grpc_port,
            "rest_port": self.config.rest_port,
            "auto_scaling_enabled": self.config.auto_scaling_enabled,
            "auto_compaction_enabled": self.config.auto_compaction_enabled,
            "query_optimization_enabled": self.config.query_optimization_enabled
        }
        
        # Add auto-scaler status
        if self.auto_scaler:
            status["auto_scaler"] = self.auto_scaler.get_scaling_status()
        
        # Add compaction status
        if self.compaction_manager:
            status["compaction"] = self.compaction_manager.get_compaction_status()
        
        # Add optimizer status
        if self.query_optimizer:
            status["query_optimizer"] = self.query_optimizer.get_optimizer_stats()
        
        return status
    
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
                logger.info("Added sample source", source_id=source_config.source_id)
            except Exception as e:
                logger.error("Failed to add sample source", 
                           source_id=source_config.source_id, 
                           error=str(e))


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
