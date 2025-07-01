"""
Scaling Demo - Auto-scaling and Load Testing

This demo shows:
- Auto-scaling capabilities
- Load testing across services
- Performance monitoring
- Resource usage tracking
"""
import asyncio
import aiohttp
import json
import time
import random
from concurrent.futures import ThreadPoolExecutor
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

SERVICES = {
    "auth": "http://localhost:8080",
    "tenant": "http://localhost:8000",
    "operation": "http://localhost:8081",
    "monitoring": "http://localhost:8084"
}

class ScalingDemo:
    """Demo for auto-scaling and performance testing"""
    
    def __init__(self):
        self.session = None
        self.auth_token = None
        
    async def __aenter__(self):
        self.session = aiohttp.ClientSession()
        return self
        
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        if self.session:
            await self.session.close()
    
    async def authenticate(self):
        """Get authentication token"""
        auth_data = {"username": "admin", "password": "admin123"}
        
        try:
            async with self.session.post(
                f"{SERVICES['auth']}/auth/login",
                json=auth_data
            ) as response:
                if response.status == 200:
                    result = await response.json()
                    self.auth_token = result["access_token"]
                    return True
                return False
        except:
            return False
    
    def get_headers(self):
        """Get auth headers"""
        return {"Authorization": f"Bearer {self.auth_token}"}
    
    async def generate_load(self, duration_seconds=30, requests_per_second=5):
        """Generate load on the system"""
        logger.info(f"🚀 Generating load: {requests_per_second} req/s for {duration_seconds}s")
        
        start_time = time.time()
        request_count = 0
        success_count = 0
        
        while time.time() - start_time < duration_seconds:
            # Send multiple requests concurrently
            tasks = []
            for _ in range(requests_per_second):
                task = self.make_request()
                tasks.append(task)
            
            # Execute requests concurrently
            results = await asyncio.gather(*tasks, return_exceptions=True)
            
            # Count results
            for result in results:
                request_count += 1
                if not isinstance(result, Exception) and result:
                    success_count += 1
            
            # Log progress every 5 seconds
            elapsed = time.time() - start_time
            if int(elapsed) % 5 == 0 and elapsed > 0:
                logger.info(f"   Progress: {elapsed:.0f}s - {request_count} requests, {success_count} successful")
            
            # Wait before next batch
            await asyncio.sleep(1)
        
        success_rate = (success_count / request_count * 100) if request_count > 0 else 0
        logger.info(f"✅ Load generation complete: {success_rate:.1f}% success rate")
        
        return {
            "total_requests": request_count,
            "successful_requests": success_count,
            "success_rate": success_rate,
            "duration": duration_seconds
        }
    
    async def make_request(self):
        """Make a sample request to the tenant node"""
        # Simulate different types of requests
        request_types = ["health", "search", "stats"]
        request_type = random.choice(request_types)
        
        try:
            if request_type == "health":
                async with self.session.get(f"{SERVICES['tenant']}/health", timeout=5) as response:
                    return response.status == 200
            
            elif request_type == "search":
                search_data = {
                    "source_ids": [],
                    "filters": {"region": {"string_value": random.choice(["North", "South", "East", "West"])}},
                    "limit": 10
                }
                async with self.session.post(
                    f"{SERVICES['tenant']}/search",
                    json=search_data,
                    headers=self.get_headers(),
                    timeout=5
                ) as response:
                    return response.status == 200
            
            elif request_type == "stats":
                async with self.session.get(
                    f"{SERVICES['tenant']}/sources",
                    headers=self.get_headers(),
                    timeout=5
                ) as response:
                    return response.status == 200
                    
        except Exception as e:
            return False
    
    async def monitor_system_resources(self, duration_seconds=30):
        """Monitor system resources during load"""
        logger.info("📊 Monitoring system resources...")
        
        start_time = time.time()
        resource_data = []
        
        while time.time() - start_time < duration_seconds:
            try:
                # Get system status from monitoring service
                async with self.session.get(f"{SERVICES['monitoring']}/status", timeout=3) as response:
                    if response.status == 200:
                        status = await response.json()
                        resource_data.append({
                            "timestamp": time.time(),
                            "status": status
                        })
                
                # Get operation node scaling status (if available)
                try:
                    async with self.session.get(f"{SERVICES['operation']}/health", timeout=3) as response:
                        if response.status == 200:
                            op_status = await response.json()
                            logger.info(f"   Operation Node: {op_status.get('status', 'unknown')}")
                except:
                    pass  # Operation node might not be fully implemented
                
            except Exception as e:
                logger.warning(f"   Monitoring error: {e}")
            
            await asyncio.sleep(5)  # Monitor every 5 seconds
        
        logger.info(f"📈 Collected {len(resource_data)} monitoring samples")
        return resource_data
    
    async def test_auto_scaling(self):
        """Test auto-scaling behavior"""
        logger.info("⚖️ Testing auto-scaling behavior...")
        
        # Start monitoring
        monitor_task = asyncio.create_task(self.monitor_system_resources(45))
        
        # Wait a bit for baseline
        await asyncio.sleep(5)
        
        # Generate light load
        logger.info("📊 Phase 1: Light load (2 req/s)")
        await self.generate_load(duration_seconds=15, requests_per_second=2)
        
        # Generate heavy load
        logger.info("📊 Phase 2: Heavy load (10 req/s)")
        await self.generate_load(duration_seconds=15, requests_per_second=10)
        
        # Cool down
        logger.info("📊 Phase 3: Cool down")
        await asyncio.sleep(10)
        
        # Wait for monitoring to complete
        resource_data = await monitor_task
        
        logger.info("✅ Auto-scaling test complete!")
        return resource_data
    
    async def performance_benchmark(self):
        """Run performance benchmark"""
        logger.info("🏁 Running performance benchmark...")
        
        benchmarks = [
            {"name": "Low Load", "rps": 1, "duration": 10},
            {"name": "Medium Load", "rps": 5, "duration": 10},
            {"name": "High Load", "rps": 10, "duration": 10},
        ]
        
        results = []
        
        for benchmark in benchmarks:
            logger.info(f"🔥 Running {benchmark['name']} test...")
            result = await self.generate_load(
                duration_seconds=benchmark["duration"],
                requests_per_second=benchmark["rps"]
            )
            result["test_name"] = benchmark["name"]
            results.append(result)
            
            # Cool down between tests
            await asyncio.sleep(5)
        
        return results


async def run_scaling_demo():
    """Run the complete scaling demo"""
    logger.info("⚖️ Starting Auto-Scaling Demo")
    logger.info("=" * 50)
    
    async with ScalingDemo() as demo:
        # 1. Authenticate
        logger.info("🔐 Authenticating...")
        if not await demo.authenticate():
            logger.error("❌ Authentication failed. Ensure Auth Gateway is running.")
            return
        
        print()
        
        # 2. Performance benchmark
        benchmark_results = await demo.performance_benchmark()
        
        print()
        logger.info("📊 Benchmark Results:")
        for result in benchmark_results:
            logger.info(f"   {result['test_name']}: {result['success_rate']:.1f}% success rate")
        
        print()
        
        # 3. Auto-scaling test
        scaling_data = await demo.test_auto_scaling()
        
        print()
        logger.info("=" * 50)
        logger.info("🎉 Scaling Demo Complete!")
        logger.info("💡 Check monitoring service at http://localhost:8084/status for detailed metrics")


if __name__ == "__main__":
    print("⚖️ Auto-Scaling & Performance Demo")
    print("=" * 40)
    print("This demo requires the following services:")
    print("- Auth Gateway (8080)")
    print("- Tenant Node (8000)")
    print("- Monitoring (8084)")
    print("- Operation Node (8081) [optional]")
    print()
    print("Start services with: ./start_services.ps1 -Individual")
    print("=" * 40)
    print()
    
    try:
        asyncio.run(run_scaling_demo())
    except KeyboardInterrupt:
        print("\n👋 Demo stopped by user")
    except Exception as e:
        print(f"❌ Demo failed: {e}")
        import traceback
        traceback.print_exc()
