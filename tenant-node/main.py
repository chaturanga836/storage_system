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
