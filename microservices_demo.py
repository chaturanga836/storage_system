"""
Microservices Demo - Full System Integration Test

This demo shows how all microservices work together:
- Authentication via Auth Gateway
- Data operations via Tenant Node
- Query optimization via CBO Engine
- Metadata management via Metadata Catalog
- Monitoring and health checks
"""
import asyncio
import aiohttp
import json
import time
from datetime import datetime, timedelta
import pandas as pd
import logging

# Setup logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Service URLs
SERVICES = {
    "auth": "http://localhost:8085",
    "tenant": "http://localhost:8000", 
    "cbo": "http://localhost:8082",
    "metadata": "http://localhost:8083",
    "operation": "http://localhost:8081",
    "monitoring": "http://localhost:8084"
}

class MicroservicesDemo:
    """Demo class for testing microservices integration"""
    
    def __init__(self):
        self.session = None
        self.auth_token = None
        
    async def __aenter__(self):
        self.session = aiohttp.ClientSession()
        return self
        
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        if self.session:
            await self.session.close()
    
    async def check_services_health(self):
        """Check health of all services"""
        logger.info("🏥 Checking health of all microservices...")
        
        health_results = {}
        for service_name, base_url in SERVICES.items():
            try:
                async with self.session.get(f"{base_url}/health", timeout=5) as response:
                    if response.status == 200:
                        health_data = await response.json()
                        health_results[service_name] = {"status": "healthy", "data": health_data}
                        logger.info(f"✅ {service_name.upper()}: Healthy")
                    else:
                        health_results[service_name] = {"status": "unhealthy", "code": response.status}
                        logger.warning(f"⚠️ {service_name.upper()}: Unhealthy ({response.status})")
            except Exception as e:
                health_results[service_name] = {"status": "error", "error": str(e)}
                logger.error(f"❌ {service_name.upper()}: Error - {e}")
        
        return health_results
    
    async def authenticate(self, username="admin", password="admin123"):
        """Authenticate with Auth Gateway"""
        logger.info("🔐 Authenticating with Auth Gateway...")
        
        auth_data = {"username": username, "password": password}
        
        try:
            async with self.session.post(
                f"{SERVICES['auth']}/auth/login",
                json=auth_data,
                headers={"Content-Type": "application/json"}
            ) as response:
                if response.status == 200:
                    auth_result = await response.json()
                    self.auth_token = auth_result["access_token"]
                    logger.info(f"✅ Authentication successful! Token: {self.auth_token[:20]}...")
                    return auth_result
                else:
                    error_text = await response.text()
                    logger.error(f"❌ Authentication failed: {response.status} - {error_text}")
                    return None
        except Exception as e:
            logger.error(f"❌ Authentication error: {e}")
            return None
    
    def get_auth_headers(self):
        """Get headers with authentication token"""
        if not self.auth_token:
            raise ValueError("Not authenticated. Call authenticate() first.")
        return {"Authorization": f"Bearer {self.auth_token}"}
    
    async def create_sample_data(self):
        """Create sample data for testing"""
        logger.info("📊 Creating sample sales data...")
        
        # Generate sample sales data
        data = []
        regions = ["North", "South", "East", "West"]
        products = ["Laptop", "Mouse", "Keyboard", "Monitor", "Phone"]
        
        for i in range(1000):
            record = {
                "order_id": f"ORD-{i+1:06d}",
                "customer_id": f"CUST-{(i % 100) + 1:04d}",
                "product_name": products[i % len(products)],
                "quantity": (i % 5) + 1,
                "unit_price": round(50 + (i % 500), 2),
                "region": regions[i % len(regions)],
                "order_date": (datetime.now() - timedelta(days=i % 90)).isoformat()
            }
            record["total_amount"] = round(record["quantity"] * record["unit_price"], 2)
            data.append(record)
        
        return data
    
    async def add_data_source(self):
        """Add a data source to the tenant node"""
        logger.info("📂 Adding data source to Tenant Node...")
        
        source_config = {
            "config": {
                "source_id": "demo_sales",
                "name": "Demo Sales Data",
                "connection_string": "file://demo_sales",
                "data_path": "./data/demo_sales",
                "schema_definition": {
                    "order_id": "string",
                    "customer_id": "string", 
                    "product_name": "string",
                    "quantity": "int64",
                    "unit_price": "float64",
                    "total_amount": "float64",
                    "order_date": "datetime64[ns]",
                    "region": "string"
                },
                "partition_columns": ["region"],
                "index_columns": ["customer_id", "product_name"],
                "compression": "snappy",
                "max_file_size_mb": 256,
                "wal_enabled": True
            }
        }
        
        try:
            async with self.session.post(
                f"{SERVICES['tenant']}/sources",
                json=source_config,
                headers={**self.get_auth_headers(), "Content-Type": "application/json"}
            ) as response:
                if response.status == 200:
                    result = await response.json()
                    logger.info("✅ Data source added successfully!")
                    return result
                else:
                    error_text = await response.text()
                    logger.error(f"❌ Failed to add data source: {response.status} - {error_text}")
                    return None
        except Exception as e:
            logger.error(f"❌ Error adding data source: {e}")
            return None
    
    async def write_data(self, data):
        """Write data to the tenant node"""
        logger.info(f"✏️ Writing {len(data)} records to Tenant Node...")
        
        # Convert to the expected format
        write_request = {
            "source_id": "demo_sales",
            "records": [
                {
                    "fields": {k: {"string_value": str(v)} for k, v in record.items()}
                }
                for record in data[:100]  # Send first 100 records
            ],
            "partition_columns": ["region"]
        }
        
        try:
            async with self.session.post(
                f"{SERVICES['tenant']}/data",
                json=write_request,
                headers={**self.get_auth_headers(), "Content-Type": "application/json"}
            ) as response:
                if response.status == 200:
                    result = await response.json()
                    logger.info(f"✅ Successfully wrote {result.get('rows_written', 0)} records!")
                    return result
                else:
                    error_text = await response.text()
                    logger.error(f"❌ Failed to write data: {response.status} - {error_text}")
                    return None
        except Exception as e:
            logger.error(f"❌ Error writing data: {e}")
            return None
    
    async def search_data(self):
        """Search data from the tenant node"""
        logger.info("🔍 Searching data from Tenant Node...")
        
        search_request = {
            "source_ids": ["demo_sales"],
            "filters": {
                "region": {"string_value": "North"}
            },
            "limit": 10,
            "offset": 0
        }
        
        try:
            async with self.session.post(
                f"{SERVICES['tenant']}/search",
                json=search_request,
                headers={**self.get_auth_headers(), "Content-Type": "application/json"}
            ) as response:
                if response.status == 200:
                    result = await response.json()
                    logger.info(f"✅ Search completed! Found {len(result.get('records', []))} records")
                    return result
                else:
                    error_text = await response.text()
                    logger.error(f"❌ Search failed: {response.status} - {error_text}")
                    return None
        except Exception as e:
            logger.error(f"❌ Error searching data: {e}")
            return None
    
    async def get_system_status(self):
        """Get comprehensive system status"""
        logger.info("📈 Getting system status from Monitoring service...")
        
        try:
            async with self.session.get(f"{SERVICES['monitoring']}/status") as response:
                if response.status == 200:
                    status = await response.json()
                    logger.info("✅ System status retrieved successfully!")
                    return status
                else:
                    error_text = await response.text()
                    logger.error(f"❌ Failed to get system status: {response.status} - {error_text}")
                    return None
        except Exception as e:
            logger.error(f"❌ Error getting system status: {e}")
            return None


async def run_full_demo():
    """Run the complete microservices demo"""
    logger.info("🚀 Starting Microservices Integration Demo")
    logger.info("=" * 60)
    
    async with MicroservicesDemo() as demo:
        # 1. Check all services are healthy
        health_results = await demo.check_services_health()
        healthy_services = sum(1 for result in health_results.values() if result["status"] == "healthy")
        logger.info(f"📊 Health Check: {healthy_services}/{len(SERVICES)} services healthy")
        
        if healthy_services < len(SERVICES):
            logger.warning("⚠️ Some services are not healthy. Demo may not work completely.")
            logger.info("💡 Try starting services with: ./start_services.ps1 -Individual")
        
        # 2. Authenticate
        auth_result = await demo.authenticate()
        if not auth_result:
            logger.error("❌ Cannot proceed without authentication")
            return
        
        # 3. Add data source
        await demo.add_data_source()
        
        # 4. Create and write sample data
        sample_data = await demo.create_sample_data()
        await demo.write_data(sample_data)
        
        # 5. Search data
        await demo.search_data()
        
        # 6. Get system status
        system_status = await demo.get_system_status()
        if system_status:
            logger.info(f"📈 System Status: {system_status.get('status', 'unknown')}")
        
        logger.info("=" * 60)
        logger.info("🎉 Microservices Demo Complete!")
        logger.info("💡 Check the individual service logs for detailed information")


if __name__ == "__main__":
    print("🏗️ Microservices Integration Demo")
    print("=" * 50)
    print("This demo requires all services to be running.")
    print("Start services with: ./start_services.ps1 -Individual")
    print("Or with Docker: docker-compose up")
    print("=" * 50)
    print()
    
    try:
        asyncio.run(run_full_demo())
    except KeyboardInterrupt:
        print("\n👋 Demo stopped by user")
    except Exception as e:
        print(f"❌ Demo failed: {e}")
        import traceback
        traceback.print_exc()
