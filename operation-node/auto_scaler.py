"""
Auto-scaling manager for dynamic resource allocation
"""
import asyncio
import time
import logging
import psutil
from typing import Dict, Any, List, Optional
from dataclasses import dataclass
from datetime import datetime, timedelta
from collections import deque

logger = logging.getLogger(__name__)


@dataclass
class ScalingMetrics:
    """Metrics for scaling decisions"""
    timestamp: datetime
    cpu_usage: float
    memory_usage: float
    active_queries: int
    query_queue_length: int
    avg_query_time: float
    io_wait: float


@dataclass
class ScalingPolicy:
    """Configuration for auto-scaling behavior"""
    # CPU thresholds
    cpu_scale_up_threshold: float = 70.0
    cpu_scale_down_threshold: float = 30.0
    
    # Memory thresholds  
    memory_scale_up_threshold: float = 80.0
    memory_scale_down_threshold: float = 40.0
    
    # Query-based thresholds
    max_concurrent_queries: int = 50
    query_queue_threshold: int = 10
    avg_query_time_threshold: float = 30.0
    
    # Scaling parameters
    min_workers: int = 2
    max_workers: int = 100
    scale_up_factor: float = 1.5
    scale_down_factor: float = 0.7
    
    # Timing parameters
    scale_up_cooldown: int = 300  # 5 minutes
    scale_down_cooldown: int = 600  # 10 minutes
    metrics_window: int = 300  # 5 minutes of metrics


class AutoScaler:
    """Dynamic resource scaling based on workload metrics"""
    
    def __init__(self, tenant_config, scaling_policy: Optional[ScalingPolicy] = None):
        self.tenant_config = tenant_config
        self.policy = scaling_policy or ScalingPolicy()
        
        # Metrics tracking
        self.metrics_history = deque(maxlen=100)  # Keep last 100 metrics
        self.current_workers = tenant_config.grpc_max_workers
        
        # Query tracking
        self.active_queries = {}
        self.query_start_times = {}
        self.query_queue = deque()
        
        # Scaling state
        self.last_scale_up = None
        self.last_scale_down = None
        self.scaling_in_progress = False
        
        # Memory management
        self.memory_allocations = {}
        self.index_cache_sizes = {}
        
        logger.info(f"AutoScaler initialized with {self.current_workers} workers")
    
    async def start_monitoring(self):
        """Start continuous monitoring and scaling"""
        logger.info("Starting auto-scaling monitoring")
        
        while True:
            try:
                await self._collect_metrics()
                await self._evaluate_scaling()
                await self._manage_memory()
                await asyncio.sleep(30)  # Check every 30 seconds
                
            except Exception as e:
                logger.error(f"Error in auto-scaling monitoring: {e}")
                await asyncio.sleep(60)  # Back off on error
    
    async def _collect_metrics(self):
        """Collect current system and query metrics"""
        try:
            # System metrics
            cpu_usage = psutil.cpu_percent(interval=1)
            memory = psutil.virtual_memory()
            memory_usage = memory.percent
            
            # I/O metrics
            io_counters = psutil.disk_io_counters()
            io_wait = getattr(io_counters, 'busy_time', 0) if io_counters else 0
            
            # Query metrics
            active_queries = len(self.active_queries)
            queue_length = len(self.query_queue)
            
            # Calculate average query time
            current_time = time.time()
            completed_queries = [
                current_time - start_time 
                for query_id, start_time in self.query_start_times.items()
                if query_id not in self.active_queries
            ]
            avg_query_time = sum(completed_queries) / len(completed_queries) if completed_queries else 0
            
            # Create metrics object
            metrics = ScalingMetrics(
                timestamp=datetime.now(),
                cpu_usage=cpu_usage,
                memory_usage=memory_usage,
                active_queries=active_queries,
                query_queue_length=queue_length,
                avg_query_time=avg_query_time,
                io_wait=io_wait
            )
            
            self.metrics_history.append(metrics)
            
            logger.debug(f"Metrics collected: CPU={cpu_usage}%, Memory={memory_usage}%, "
                        f"Queries={active_queries}, Queue={queue_length}")
            
        except Exception as e:
            logger.error(f"Error collecting metrics: {e}")
    
    async def _evaluate_scaling(self):
        """Evaluate if scaling action is needed"""
        if self.scaling_in_progress or len(self.metrics_history) < 3:
            return
        
        # Get recent metrics (last 3 measurements)
        recent_metrics = list(self.metrics_history)[-3:]
        
        # Calculate averages
        avg_cpu = sum(m.cpu_usage for m in recent_metrics) / len(recent_metrics)
        avg_memory = sum(m.memory_usage for m in recent_metrics) / len(recent_metrics)
        avg_queries = sum(m.active_queries for m in recent_metrics) / len(recent_metrics)
        avg_queue = sum(m.query_queue_length for m in recent_metrics) / len(recent_metrics)
        avg_query_time = sum(m.avg_query_time for m in recent_metrics) / len(recent_metrics)
        
        current_time = datetime.now()
        
        # Check for scale-up conditions
        should_scale_up = (
            (avg_cpu > self.policy.cpu_scale_up_threshold) or
            (avg_memory > self.policy.memory_scale_up_threshold) or
            (avg_queries > self.policy.max_concurrent_queries * 0.8) or
            (avg_queue > self.policy.query_queue_threshold) or
            (avg_query_time > self.policy.avg_query_time_threshold)
        )
        
        # Check for scale-down conditions
        should_scale_down = (
            (avg_cpu < self.policy.cpu_scale_down_threshold) and
            (avg_memory < self.policy.memory_scale_down_threshold) and
            (avg_queries < self.policy.max_concurrent_queries * 0.3) and
            (avg_queue == 0) and
            (avg_query_time < self.policy.avg_query_time_threshold * 0.5)
        )
        
        # Apply cooldown periods
        if should_scale_up:
            if (self.last_scale_up is None or 
                (current_time - self.last_scale_up).seconds > self.policy.scale_up_cooldown):
                await self._scale_up()
        
        elif should_scale_down:
            if (self.last_scale_down is None or 
                (current_time - self.last_scale_down).seconds > self.policy.scale_down_cooldown):
                await self._scale_down()
    
    async def _scale_up(self):
        """Scale up resources"""
        if self.current_workers >= self.policy.max_workers:
            logger.warning("Already at maximum worker capacity")
            return
        
        self.scaling_in_progress = True
        old_workers = self.current_workers
        
        try:
            # Calculate new worker count
            new_workers = min(
                int(self.current_workers * self.policy.scale_up_factor),
                self.policy.max_workers
            )
            
            logger.info(f"Scaling up from {old_workers} to {new_workers} workers")
            
            # Update configuration
            self.current_workers = new_workers
            self.tenant_config.grpc_max_workers = new_workers
            self.tenant_config.max_concurrent_searches = new_workers
            
            # Record scaling event
            self.last_scale_up = datetime.now()
            
            logger.info(f"Successfully scaled up to {new_workers} workers")
            
        except Exception as e:
            logger.error(f"Error during scale-up: {e}")
            self.current_workers = old_workers
        
        finally:
            self.scaling_in_progress = False
    
    async def _scale_down(self):
        """Scale down resources"""
        if self.current_workers <= self.policy.min_workers:
            logger.debug("Already at minimum worker capacity")
            return
        
        self.scaling_in_progress = True
        old_workers = self.current_workers
        
        try:
            # Calculate new worker count
            new_workers = max(
                int(self.current_workers * self.policy.scale_down_factor),
                self.policy.min_workers
            )
            
            logger.info(f"Scaling down from {old_workers} to {new_workers} workers")
            
            # Update configuration
            self.current_workers = new_workers
            self.tenant_config.grpc_max_workers = new_workers
            self.tenant_config.max_concurrent_searches = new_workers
            
            # Record scaling event
            self.last_scale_down = datetime.now()
            
            logger.info(f"Successfully scaled down to {new_workers} workers")
            
        except Exception as e:
            logger.error(f"Error during scale-down: {e}")
            self.current_workers = old_workers
        
        finally:
            self.scaling_in_progress = False
    
    async def _manage_memory(self):
        """Adaptive memory management"""
        try:
            memory = psutil.virtual_memory()
            
            # If memory usage is high, trigger cleanup
            if memory.percent > 85:
                await self._cleanup_memory_caches()
            
            # Adjust index cache sizes based on available memory
            await self._adjust_index_caches(memory.available)
            
        except Exception as e:
            logger.error(f"Error in memory management: {e}")
    
    async def _cleanup_memory_caches(self):
        """Clean up memory caches when under pressure"""
        logger.info("Cleaning up memory caches due to high memory usage")
        
        # Clear least recently used index caches
        # This would integrate with the index manager
        for cache_name, size in list(self.index_cache_sizes.items()):
            if size > 100 * 1024 * 1024:  # > 100MB
                logger.debug(f"Clearing cache: {cache_name}")
                # Signal cache cleanup
                # await self.index_manager.clear_cache(cache_name)
    
    async def _adjust_index_caches(self, available_memory: int):
        """Adjust index cache sizes based on available memory"""
        # Allocate 30% of available memory to index caches
        target_cache_memory = available_memory * 0.3
        
        # This would integrate with adaptive indexing
        logger.debug(f"Target cache memory: {target_cache_memory / (1024*1024):.1f} MB")
    
    # Query lifecycle tracking methods
    
    def register_query_start(self, query_id: str):
        """Register when a query starts"""
        self.query_start_times[query_id] = time.time()
        self.active_queries[query_id] = True
    
    def register_query_complete(self, query_id: str):
        """Register when a query completes"""
        if query_id in self.active_queries:
            del self.active_queries[query_id]
    
    def add_to_queue(self, query_id: str):
        """Add query to processing queue"""
        self.query_queue.append({
            "query_id": query_id,
            "timestamp": time.time()
        })
    
    def remove_from_queue(self, query_id: str):
        """Remove query from processing queue"""
        self.query_queue = deque(
            item for item in self.query_queue 
            if item["query_id"] != query_id
        )
    
    def get_scaling_status(self) -> Dict[str, Any]:
        """Get current scaling status"""
        recent_metrics = list(self.metrics_history)[-1] if self.metrics_history else None
        
        return {
            "current_workers": self.current_workers,
            "scaling_in_progress": self.scaling_in_progress,
            "active_queries": len(self.active_queries),
            "queued_queries": len(self.query_queue),
            "recent_metrics": {
                "cpu_usage": recent_metrics.cpu_usage if recent_metrics else 0,
                "memory_usage": recent_metrics.memory_usage if recent_metrics else 0,
                "avg_query_time": recent_metrics.avg_query_time if recent_metrics else 0
            } if recent_metrics else {},
            "last_scale_up": self.last_scale_up.isoformat() if self.last_scale_up else None,
            "last_scale_down": self.last_scale_down.isoformat() if self.last_scale_down else None
        }
