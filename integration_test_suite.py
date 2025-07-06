#!/usr/bin/env python3
"""
Python Microservices Integration Test Suite
"""

import asyncio
import aiohttp
import json
import time
import pytest
from typing import Dict, List, Any
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class MicroservicesTestSuite:
    def __init__(self, ec2_ip="localhost"):
        # Updated ports to match deployed services
        self.base_urls = {
            'auth_gateway': f'http://{ec2_ip}:8080',
            'tenant_node': f'http://{ec2_ip}:8001', 
            'operation_node': f'http://{ec2_ip}:8086',
            'cbo_engine': f'http://{ec2_ip}:8088',
            'metadata_catalog': f'http://{ec2_ip}:8087',
            'monitoring': f'http://{ec2_ip}:8089'
        }
        self.auth_token = None
        
    async def setup_test_environment(self):
        """Setup test data and authenticate"""
        # 1. Health check all services
        await self.health_check_all_services()
        
        # 2. Authenticate and get token
        self.auth_token = await self.authenticate_user()
        
        # 3. Setup test data
        await self.setup_test_data()
        
    async def health_check_all_services(self):
        """Verify all microservices are running"""
        logger.info("🔍 Checking health of all microservices...")
        
        async with aiohttp.ClientSession() as session:
            for service, url in self.base_urls.items():
                try:
                    async with session.get(f"{url}/health") as response:
                        if response.status == 200:
                            logger.info(f"✅ {service} is healthy")
                        else:
                            raise Exception(f"❌ {service} health check failed: {response.status}")
                except Exception as e:
                    logger.error(f"❌ {service} is not responding: {e}")
                    raise
                    
    async def authenticate_user(self) -> str:
        """Authenticate and return JWT token"""
        logger.info("🔐 Authenticating test user...")
        
        login_data = {
            "username": "test_user",
            "password": "test_password"
        }
        
        async with aiohttp.ClientSession() as session:
            async with session.post(
                f"{self.base_urls['auth_gateway']}/auth/login",
                json=login_data
            ) as response:
                if response.status == 200:
                    data = await response.json()
                    token = data.get('token')
                    logger.info("✅ Authentication successful")
                    return token
                else:
                    raise Exception(f"❌ Authentication failed: {response.status}")
                    
    async def setup_test_data(self):
        """Create test datasets"""
        logger.info("📊 Setting up test data...")
        
        # Sample order data for testing
        test_orders = [
            {"customer_id": "cust_001", "amount": 150.00, "order_date": "2024-01-15"},
            {"customer_id": "cust_002", "amount": 250.00, "order_date": "2024-01-16"},
            {"customer_id": "cust_001", "amount": 75.00, "order_date": "2024-02-01"},
            {"customer_id": "cust_003", "amount": 500.00, "order_date": "2024-02-15"}
        ]
        
        headers = {"Authorization": f"Bearer {self.auth_token}"}
        
        async with aiohttp.ClientSession() as session:
            for order in test_orders:
                async with session.post(
                    f"{self.base_urls['tenant_node']}/data/store",
                    json={"table": "orders", "data": order},
                    headers=headers
                ) as response:
                    if response.status == 200:
                        logger.info(f"✅ Test order created: {order['customer_id']}")
                    else:
                        logger.error(f"❌ Failed to create test order: {response.status}")
                        
    # Test Cases
    
    async def test_complete_query_flow(self):
        """Test end-to-end query processing"""
        logger.info("🔄 Testing complete query flow...")
        
        query = """
        SELECT customer_id, SUM(amount) as total_spent, COUNT(*) as order_count
        FROM orders 
        WHERE order_date >= '2024-01-01' 
        GROUP BY customer_id 
        ORDER BY total_spent DESC
        """
        
        headers = {"Authorization": f"Bearer {self.auth_token}"}
        
        # Step 1: Parse query with Query Interpreter
        start_time = time.time()
        
        async with aiohttp.ClientSession() as session:
            async with session.post(
                f"{self.base_urls['query_interpreter']}/parse/sql",
                json={"query": query, "dialect": "mysql"},
                headers=headers
            ) as response:
                parse_result = await response.json()
                logger.info(f"✅ Query parsed successfully: {parse_result.get('success')}")
                
            # Step 2: Execute distributed query via Operation Node
            async with session.post(
                f"{self.base_urls['operation_node']}/query/execute", 
                json={"query": query, "optimization_level": "high"},
                headers=headers
            ) as response:
                query_result = await response.json()
                execution_time = time.time() - start_time
                
                logger.info(f"✅ Query executed in {execution_time:.3f}s")
                logger.info(f"✅ Results: {len(query_result.get('results', []))} rows")
                
                # Validate results
                assert query_result.get('success') == True
                assert execution_time < 1.0  # Should complete in under 1 second
                
    async def test_tenant_isolation(self):
        """Test multi-tenant data isolation"""
        logger.info("🏢 Testing tenant isolation...")
        
        # Create data for different tenants
        tenant_a_data = {"table": "orders", "data": {"customer_id": "tenant_a_customer", "amount": 100.00}}
        tenant_b_data = {"table": "orders", "data": {"customer_id": "tenant_b_customer", "amount": 200.00}}
        
        headers_a = {"Authorization": f"Bearer {self.auth_token}", "X-Tenant-ID": "tenant_a"}
        headers_b = {"Authorization": f"Bearer {self.auth_token}", "X-Tenant-ID": "tenant_b"}
        
        async with aiohttp.ClientSession() as session:
            # Store data for tenant A
            await session.post(
                f"{self.base_urls['tenant_node']}/data/store",
                json=tenant_a_data,
                headers=headers_a
            )
            
            # Store data for tenant B  
            await session.post(
                f"{self.base_urls['tenant_node']}/data/store",
                json=tenant_b_data,
                headers=headers_b
            )
            
            # Query as tenant A - should only see tenant A data
            async with session.post(
                f"{self.base_urls['tenant_node']}/data/execute",
                json={"query": "SELECT * FROM orders WHERE customer_id LIKE '%tenant%'"},
                headers=headers_a
            ) as response:
                tenant_a_results = await response.json()
                
            # Query as tenant B - should only see tenant B data
            async with session.post(
                f"{self.base_urls['tenant_node']}/data/execute", 
                json={"query": "SELECT * FROM orders WHERE customer_id LIKE '%tenant%'"},
                headers=headers_b
            ) as response:
                tenant_b_results = await response.json()
                
        # Validate isolation
        a_customers = [row['customer_id'] for row in tenant_a_results.get('results', [])]
        b_customers = [row['customer_id'] for row in tenant_b_results.get('results', [])]
        
        assert 'tenant_a_customer' in a_customers
        assert 'tenant_a_customer' not in b_customers
        assert 'tenant_b_customer' in b_customers
        assert 'tenant_b_customer' not in a_customers
        
        logger.info("✅ Tenant isolation verified")
        
    async def test_service_failure_resilience(self):
        """Test system behavior during service failures"""
        logger.info("💥 Testing service failure resilience...")
        
        # TODO: Implement service failure simulation
        # This would require Docker or process management to kill/restart services
        
        logger.info("⚠️ Service failure test requires implementation")
        
    async def test_performance_benchmarks(self):
        """Test performance under load"""
        logger.info("⚡ Testing performance benchmarks...")
        
        queries = [
            "SELECT COUNT(*) FROM orders",
            "SELECT customer_id, SUM(amount) FROM orders GROUP BY customer_id", 
            "SELECT * FROM orders WHERE order_date >= '2024-01-01' ORDER BY amount DESC LIMIT 10"
        ]
        
        headers = {"Authorization": f"Bearer {self.auth_token}"}
        
        # Test concurrent queries
        async def execute_query(session, query):
            start_time = time.time()
            async with session.post(
                f"{self.base_urls['operation_node']}/query/execute",
                json={"query": query},
                headers=headers
            ) as response:
                result = await response.json()
                execution_time = time.time() - start_time
                return execution_time, result.get('success', False)
                
        async with aiohttp.ClientSession() as session:
            # Execute queries concurrently
            tasks = []
            for _ in range(10):  # 10 concurrent queries
                for query in queries:
                    tasks.append(execute_query(session, query))
                    
            results = await asyncio.gather(*tasks)
            
            # Analyze performance
            execution_times = [r[0] for r in results]
            success_rate = sum(1 for r in results if r[1]) / len(results)
            
            avg_time = sum(execution_times) / len(execution_times)
            max_time = max(execution_times)
            
            logger.info(f"✅ Average query time: {avg_time:.3f}s")
            logger.info(f"✅ Max query time: {max_time:.3f}s") 
            logger.info(f"✅ Success rate: {success_rate:.1%}")
            
            # Validate performance targets
            assert avg_time < 0.5  # Average under 500ms
            assert max_time < 2.0  # Max under 2 seconds
            assert success_rate > 0.95  # 95% success rate
            
    async def run_all_tests(self):
        """Execute complete test suite"""
        logger.info("🚀 Starting Python Microservices Integration Tests")
        
        try:
            await self.setup_test_environment()
            
            # Run all test cases
            await self.test_complete_query_flow()
            await self.test_tenant_isolation()
            await self.test_service_failure_resilience()
            await self.test_performance_benchmarks()
            
            logger.info("🎉 All tests completed successfully!")
            
        except Exception as e:
            logger.error(f"❌ Test failed: {e}")
            raise

# Entry point for running tests
async def main():
    import sys
    
    # Allow specifying EC2 IP as command line argument
    ec2_ip = sys.argv[1] if len(sys.argv) > 1 else "localhost"
    
    logger.info(f"🚀 Running tests against: {ec2_ip}")
    test_suite = MicroservicesTestSuite(ec2_ip=ec2_ip)
    await test_suite.run_all_tests()

if __name__ == "__main__":
    asyncio.run(main())
