"""
Monitoring Service - Comprehensive observability and metrics
"""
import asyncio
import logging
import time
from typing import Dict, List
from datetime import datetime
from fastapi import FastAPI
import uvicorn
from prometheus_client import Counter, Histogram, Gauge, generate_latest
from prometheus_client.exposition import make_asgi_app

logger = logging.getLogger(__name__)

# Prometheus metrics
REQUEST_COUNT = Counter('http_requests_total', 'Total HTTP requests', ['method', 'endpoint', 'status'])
REQUEST_DURATION = Histogram('http_request_duration_seconds', 'HTTP request duration')
ACTIVE_CONNECTIONS = Gauge('active_connections', 'Active connections')
SYSTEM_HEALTH = Gauge('system_health_score', 'Overall system health score')

app = FastAPI(title="Storage System Monitoring", version="1.0.0")

# Add Prometheus metrics endpoint
metrics_app = make_asgi_app()
app.mount("/metrics", metrics_app)


class MetricsCollector:
    """Metrics collection service"""
    
    def __init__(self):
        self.metrics_store = {}
        self.health_scores = {}
    
    async def collect_system_metrics(self):
        """Collect system-wide metrics"""
        # Simulate metrics collection
        timestamp = datetime.utcnow()
        
        # Collect from all services
        services = ["tenant-node", "cbo-engine", "metadata-catalog", "auth-gateway"]
        
        for service in services:
            # Simulate health check
            health_score = 0.95  # Simulated health score
            self.health_scores[service] = health_score
            SYSTEM_HEALTH.labels(service=service).set(health_score)
        
        logger.info(f"Collected metrics at {timestamp}")
    
    async def start_collection(self):
        """Start periodic metrics collection"""
        while True:
            try:
                await self.collect_system_metrics()
                await asyncio.sleep(30)  # Collect every 30 seconds
            except Exception as e:
                logger.error(f"Error collecting metrics: {e}")
                await asyncio.sleep(10)


class HealthMonitor:
    """Health monitoring and alerting"""
    
    def __init__(self):
        self.service_status = {}
        self.alerts = []
    
    async def check_service_health(self, service_url: str) -> Dict:
        """Check health of a specific service"""
        # Simulate health check
        return {
            "status": "healthy",
            "response_time": 0.05,
            "timestamp": datetime.utcnow().isoformat()
        }
    
    async def monitor_services(self):
        """Monitor all services continuously"""
        services = {
            "tenant-node": "http://localhost:8000/health",
            "cbo-engine": "http://localhost:8082/health",
            "metadata-catalog": "http://localhost:8083/health",
            "auth-gateway": "http://localhost:8080/health"
        }
        
        while True:
            try:
                for service_name, service_url in services.items():
                    health = await self.check_service_health(service_url)
                    self.service_status[service_name] = health
                
                await asyncio.sleep(15)  # Check every 15 seconds
            except Exception as e:
                logger.error(f"Error monitoring services: {e}")
                await asyncio.sleep(5)


@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "healthy", "service": "monitoring"}


@app.get("/status")
async def get_system_status():
    """Get overall system status"""
    return {
        "timestamp": datetime.utcnow().isoformat(),
        "status": "operational",
        "services": monitoring_service.health_monitor.service_status,
        "metrics": monitoring_service.metrics_collector.health_scores
    }


@app.get("/alerts")
async def get_alerts():
    """Get current alerts"""
    return {
        "alerts": monitoring_service.health_monitor.alerts,
        "count": len(monitoring_service.health_monitor.alerts)
    }


class MonitoringService:
    """Main monitoring service"""
    
    def __init__(self, host: str = "0.0.0.0", port: int = 8084):
        self.host = host
        self.port = port
        self.metrics_collector = MetricsCollector()
        self.health_monitor = HealthMonitor()
    
    async def start(self):
        """Start the monitoring service"""
        logger.info(f"Starting Monitoring Service on {self.host}:{self.port}")
        
        # Start background tasks
        asyncio.create_task(self.metrics_collector.start_collection())
        asyncio.create_task(self.health_monitor.monitor_services())
        
        # Start web server
        config = uvicorn.Config(
            app=app,
            host=self.host,
            port=self.port,
            log_level="info"
        )
        
        server = uvicorn.Server(config)
        await server.serve()


# Global instance for FastAPI endpoints
monitoring_service = MonitoringService()


if __name__ == "__main__":
    import sys
    
    # Configure logging
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    
    try:
        asyncio.run(monitoring_service.start())
    except KeyboardInterrupt:
        logger.info("Monitoring service stopped")
        sys.exit(0)
