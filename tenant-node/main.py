"""
Tenant Node main entry point
"""
import asyncio
import logging
import os
from pathlib import Path

# Update imports to work with new structure
from config import TenantConfig
from data_source import SourceManager
from tenant_node import TenantNode
from grpc_service import TenantNodeServer
from rest_api import create_rest_api

logger = logging.getLogger(__name__)


async def main():
    """Main entry point for tenant node service"""
    # Configure logging
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    
    logger.info("Starting Tenant Node service...")
    
    # Load configuration
    config_path = os.getenv("TENANT_CONFIG", "config.env")
    tenant_config = TenantConfig.from_env_file(config_path)
    
    # Initialize source manager
    source_manager = SourceManager(tenant_config)
    await source_manager.initialize()
    
    # Create tenant node
    tenant_node = TenantNode(tenant_config, source_manager)
    
    # Start services
    tasks = []
    
    # Start gRPC server if enabled
    if tenant_config.grpc_enabled:
        grpc_server = TenantNodeServer(tenant_config, source_manager)
        tasks.append(asyncio.create_task(grpc_server.start()))
    
    # Start REST API if enabled
    if tenant_config.rest_enabled:
        rest_app = create_rest_api(tenant_config, source_manager)
        # REST API would be started with uvicorn in production
        logger.info(f"REST API ready on port {tenant_config.rest_port}")
    
    # Wait for all services
    if tasks:
        await asyncio.gather(*tasks)
    else:
        # Keep running if no tasks
        await asyncio.Event().wait()


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("Tenant Node service stopped")
