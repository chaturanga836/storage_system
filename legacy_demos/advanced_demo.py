"""
Advanced Features Demo: Auto-scaling, Compaction, and Query Optimization
"""
import asyncio
import logging
import time
import pandas as pd
from pathlib import Path

from tenant_node.config import TenantConfig, SourceConfig
from tenant_node.tenant_node import TenantNode

# Setup logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


async def demo_advanced_features():
    """Demonstrate auto-scaling, compaction, and query optimization"""
    
    # Create tenant configuration with advanced features enabled
    config = TenantConfig(
        tenant_id="advanced_demo",
        tenant_name="Advanced Features Demo",
        base_data_path="./demo_data_advanced",
        
        # Enable all advanced features
        auto_scaling_enabled=True,
        auto_compaction_enabled=True,
        query_optimization_enabled=True,
        
        # Auto-scaling settings
        min_workers=2,
        max_workers=20,
        scale_up_threshold_cpu=60.0,
        scale_down_threshold_cpu=20.0,
        
        # Compaction settings
        compaction_interval_minutes=1,  # Aggressive for demo
        min_file_size_mb=1.0,  # Small for demo
        target_file_size_mb=10.0,
        
        # Query optimization
        statistics_refresh_interval_minutes=1,  # Frequent for demo
        cost_model_learning_enabled=True
    )
    
    # Initialize tenant node
    tenant = TenantNode(config)
    await tenant.initialize()
    
    # Start background tasks
    await tenant.start_background_tasks()
    
    try:
        # Demo 1: Auto-scaling
        print("\n=== Auto-Scaling Demo ===")
        await demo_auto_scaling(tenant)
        
        # Demo 2: Query Optimization
        print("\n=== Query Optimization Demo ===")
        await demo_query_optimization(tenant)
        
        # Demo 3: File Compaction
        print("\n=== File Compaction Demo ===")
        await demo_file_compaction(tenant)
        
        # Show comprehensive status
        print("\n=== System Status ===")
        await show_system_status(tenant)
        
    finally:
        await tenant.shutdown()


async def demo_auto_scaling(tenant: TenantNode):
    """Demonstrate auto-scaling capabilities"""
    print("Setting up data source for auto-scaling demo...")
    
    # Add a data source
    source_config = SourceConfig(
        source_id="scaling_demo",
        name="Auto-scaling Demo Source",
        connection_string="",
        data_path=str(Path(tenant.config.base_data_path) / "sources" / "scaling_demo"),
        schema_definition={
            "id": "int64",
            "timestamp": "datetime64[ns]",
            "value": "float64",
            "category": "string"
        },
        partition_columns=["category"],
        index_columns=["id", "timestamp"]
    )
    
    await tenant.source_manager.add_source(source_config)
    source = await tenant.source_manager.get_source("scaling_demo")
    
    print("Generating load to trigger auto-scaling...")
    
    # Generate multiple concurrent queries to trigger scaling
    query_tasks = []
    for i in range(10):
        # Create some data first
        df = pd.DataFrame({
            'id': range(i * 1000, (i + 1) * 1000),
            'timestamp': pd.date_range('2024-01-01', periods=1000, freq='1H'),
            'value': pd.np.random.randn(1000),
            'category': [f'cat_{j % 5}' for j in range(1000)]
        })
        
        await source.write_data(df)
        
        # Create query task
        async def query_task(query_id):
            filters = {"category": f"cat_{query_id % 5}"}
            results = []
            async for chunk in source.search_data(filters, limit=100):
                results.append(chunk)
            return len(results)
        
        task = asyncio.create_task(query_task(i))
        query_tasks.append(task)
    
    # Wait for queries to complete
    results = await asyncio.gather(*query_tasks)
    
    print(f"Completed {len(results)} concurrent queries")
    
    # Show auto-scaler status
    if tenant.auto_scaler:
        status = tenant.auto_scaler.get_scaling_status()
        print(f"Current workers: {status['current_workers']}")
        print(f"Active queries: {status['active_queries']}")
        print(f"Recent CPU usage: {status['recent_metrics'].get('cpu_usage', 0):.1f}%")


async def demo_query_optimization(tenant: TenantNode):
    """Demonstrate query optimization capabilities"""
    print("Setting up data for query optimization demo...")
    
    source = await tenant.source_manager.get_source("scaling_demo")
    if not source:
        print("Source not found, skipping query optimization demo")
        return
    
    # Generate queries with different patterns to test optimization
    test_queries = [
        # Simple equality filter
        {"category": "cat_0"},
        
        # Range filter
        {"value": {"$gt": 0.5}},
        
        # Multiple conditions
        {"category": "cat_1", "value": {"$lt": -0.5}},
        
        # IN condition
        {"category": {"$in": ["cat_0", "cat_1", "cat_2"]}}
    ]
    
    aggregations = [
        {"type": "count", "column": "*", "alias": "total_count"},
        {"type": "avg", "column": "value", "alias": "avg_value"},
        {"type": "max", "column": "value", "alias": "max_value"}
    ]
    
    if tenant.query_optimizer:
        print("Testing query optimization...")
        
        for i, query_filter in enumerate(test_queries):
            print(f"\nQuery {i + 1}: {query_filter}")
            
            # Generate optimized execution plan
            try:
                plan = await tenant.query_optimizer.optimize_query(
                    query_filter, aggregations, ["scaling_demo"], 1000
                )
                
                print(f"  Selected plan: {plan.plan_id}")
                print(f"  Estimated cost: {plan.estimated_cost.total_cost:.2f}")
                print(f"  Estimated time: {plan.estimated_time_ms:.1f}ms")
                print(f"  Parallelism: {plan.parallelism_factor}")
                
                if plan.index_usage:
                    print(f"  Index usage: {plan.index_usage}")
                
                if plan.file_pruning:
                    print(f"  File pruning: {plan.file_pruning}")
                
            except Exception as e:
                print(f"  Optimization failed: {e}")
    
    # Show optimizer statistics
    if tenant.query_optimizer:
        stats = tenant.query_optimizer.get_optimizer_stats()
        print(f"\nOptimizer statistics:")
        print(f"  Tables analyzed: {stats['tables_analyzed']}")
        print(f"  Execution history: {stats['execution_history']}")


async def demo_file_compaction(tenant: TenantNode):
    """Demonstrate file compaction capabilities"""
    print("Creating small files to trigger compaction...")
    
    source = await tenant.source_manager.get_source("scaling_demo")
    if not source:
        print("Source not found, skipping compaction demo")
        return
    
    # Create many small files
    for i in range(15):  # Create enough files to trigger compaction
        small_df = pd.DataFrame({
            'id': range(i * 10, (i + 1) * 10),
            'timestamp': pd.date_range('2024-01-01', periods=10, freq='1H'),
            'value': pd.np.random.randn(10),
            'category': [f'compact_cat_{i % 3}' for _ in range(10)]
        })
        
        await source.write_data(small_df)
    
    print(f"Created 15 small files")
    
    # Show compaction status
    if tenant.compaction_manager:
        status = tenant.compaction_manager.get_compaction_status()
        print(f"Compaction status:")
        print(f"  Active jobs: {status['active_jobs']}")
        print(f"  Completed jobs: {status['completed_jobs']}")
        print(f"  Total compactions: {status['total_compactions']}")
        print(f"  Size reduced: {status['total_size_reduced_mb']:.1f} MB")
        print(f"  Files reduced: {status['total_files_reduced']}")
        
        # Trigger manual compaction
        print("\nTriggering manual compaction...")
        success = await tenant.compaction_manager.trigger_manual_compaction("scaling_demo")
        print(f"Manual compaction {'succeeded' if success else 'failed'}")
        
        # Wait a moment for compaction to start
        await asyncio.sleep(2)
        
        # Show updated status
        status = tenant.compaction_manager.get_compaction_status()
        print(f"Updated compaction status:")
        print(f"  Active jobs: {status['active_jobs']}")
        print(f"  Completed jobs: {status['completed_jobs']}")


async def show_system_status(tenant: TenantNode):
    """Show comprehensive system status"""
    status = tenant.get_status()
    
    print("Comprehensive system status:")
    print(f"  Tenant: {status['tenant_id']} ({status['tenant_name']})")
    print(f"  Features enabled:")
    print(f"    Auto-scaling: {status['auto_scaling_enabled']}")
    print(f"    Auto-compaction: {status['auto_compaction_enabled']}")
    print(f"    Query optimization: {status['query_optimization_enabled']}")
    
    if 'auto_scaler' in status:
        auto_scaler = status['auto_scaler']
        print(f"  Auto-scaler:")
        print(f"    Current workers: {auto_scaler['current_workers']}")
        print(f"    Active queries: {auto_scaler['active_queries']}")
        print(f"    Scaling in progress: {auto_scaler['scaling_in_progress']}")
    
    if 'compaction' in status:
        compaction = status['compaction']
        print(f"  Compaction:")
        print(f"    Active jobs: {compaction['active_jobs']}")
        print(f"    Total compactions: {compaction['total_compactions']}")
        print(f"    Size reduced: {compaction['total_size_reduced_mb']:.1f} MB")
    
    if 'query_optimizer' in status:
        optimizer = status['query_optimizer']
        print(f"  Query optimizer:")
        print(f"    Tables analyzed: {optimizer['tables_analyzed']}")
        print(f"    Recent plans: {optimizer['recent_plans']}")


async def performance_comparison():
    """Compare performance with and without optimizations"""
    print("\n=== Performance Comparison ===")
    
    # Test with optimizations disabled
    config_basic = TenantConfig(
        tenant_id="basic_demo",
        tenant_name="Basic Demo",
        base_data_path="./demo_data_basic",
        auto_scaling_enabled=False,
        auto_compaction_enabled=False,
        query_optimization_enabled=False
    )
    
    # Test with optimizations enabled
    config_optimized = TenantConfig(
        tenant_id="optimized_demo",
        tenant_name="Optimized Demo", 
        base_data_path="./demo_data_optimized",
        auto_scaling_enabled=True,
        auto_compaction_enabled=True,
        query_optimization_enabled=True
    )
    
    async def run_benchmark(config, label):
        tenant = TenantNode(config)
        await tenant.initialize()
        
        try:
            # Add source and data
            source_config = SourceConfig(
                source_id="benchmark",
                name="Benchmark Source",
                connection_string="",
                data_path=str(Path(config.base_data_path) / "sources" / "benchmark"),
                schema_definition={"id": "int64", "value": "float64"},
                index_columns=["id"]
            )
            
            await tenant.source_manager.add_source(source_config)
            source = await tenant.source_manager.get_source("benchmark")
            
            # Generate test data
            df = pd.DataFrame({
                'id': range(10000),
                'value': pd.np.random.randn(10000)
            })
            await source.write_data(df)
            
            # Run benchmark queries
            start_time = time.time()
            
            for _ in range(10):
                filters = {"value": {"$gt": 0}}
                results = []
                async for chunk in source.search_data(filters, limit=1000):
                    results.append(chunk)
            
            end_time = time.time()
            
            print(f"{label}: {(end_time - start_time) * 1000:.1f}ms for 10 queries")
            
        finally:
            await tenant.shutdown()
    
    await run_benchmark(config_basic, "Basic (no optimizations)")
    await run_benchmark(config_optimized, "Optimized")


if __name__ == "__main__":
    async def main():
        await demo_advanced_features()
        await performance_comparison()
    
    asyncio.run(main())
