"""
Demo script for testing Tenant Node functionality
"""
import asyncio
import pandas as pd
import requests
import json
from datetime import datetime, timedelta
import random
import uuid

from tenant_node.config import TenantConfig, SourceConfig
from tenant_node.data_source import SourceManager


async def demo_data_operations():
    """Demonstrate basic data operations"""
    print("=== Tenant Node Demo ===")
    
    # Create configuration
    config = TenantConfig(
        tenant_id="demo_tenant",
        tenant_name="Demo Tenant",
        base_data_path="./demo_data",
        grpc_port=50051,
        rest_port=8000
    )
    
    # Create source manager
    source_manager = SourceManager(config)
    await source_manager.initialize()
    
    try:
        # Add a demo source
        demo_source_config = SourceConfig(
            source_id="demo_sales",
            name="Demo Sales Data",
            connection_string="file://demo_sales",
            data_path=str(config.get_source_data_path("demo_sales")),
            schema_definition={
                "order_id": "string",
                "customer_id": "string",
                "product_name": "string",
                "quantity": "int64",
                "unit_price": "float64",
                "total_amount": "float64",
                "order_date": "datetime64[ns]",
                "region": "string"
            },
            partition_columns=["region", "order_date"],
            index_columns=["customer_id", "product_name", "order_date"],
            compression="snappy",
            max_file_size_mb=64,
            wal_enabled=True
        )
        
        print(f"Adding source: {demo_source_config.source_id}")
        await source_manager.add_source(demo_source_config)
        
        # Generate sample data
        print("Generating sample data...")
        sample_data = generate_sample_sales_data(1000)
        
        # Get the source and write data
        source = await source_manager.get_source("demo_sales")
        if source:
            print("Writing sample data...")
            write_id = await source.write_data(sample_data)
            print(f"Data written with ID: {write_id}")
            
            # Test search functionality
            print("\nTesting search functionality...")
            
            # Search for specific customer
            customer_filter = {"customer_id": "CUST_001"}
            print(f"Searching for customer: {customer_filter}")
            
            async for chunk in source.search_data(customer_filter, limit=10):
                print(f"Found {len(chunk)} records for customer_id=CUST_001")
                print(chunk.head().to_string())
                break
            
            # Search by date range
            date_filter = {
                "order_date": {
                    "$gte": "2024-01-01",
                    "$lt": "2024-02-01"
                }
            }
            print(f"\nSearching by date range: {date_filter}")
            
            total_records = 0
            async for chunk in source.search_data(date_filter, limit=50):
                total_records += len(chunk)
            print(f"Found {total_records} records in January 2024")
            
            # Test aggregations
            print("\nTesting aggregation functionality...")
            
            aggregations = [
                {"type": "count", "alias": "total_orders"},
                {"type": "sum", "column": "total_amount", "alias": "total_revenue"},
                {"type": "avg", "column": "total_amount", "alias": "avg_order_value"},
                {"type": "max", "column": "total_amount", "alias": "max_order_value"}
            ]
            
            agg_results = await source.aggregate_data(aggregations)
            print("Aggregation results:")
            for key, value in agg_results.items():
                print(f"  {key}: {value}")
            
            # Test tenant-wide aggregations
            print("\nTesting tenant-wide aggregations...")
            tenant_agg_results = await source_manager.aggregate_all_sources(aggregations)
            print("Tenant aggregation results:")
            print(json.dumps(tenant_agg_results, indent=2, default=str))
            
            # Get statistics
            print("\nGetting source statistics...")
            stats = await source.get_statistics()
            print("Source statistics:")
            for key, value in stats.items():
                print(f"  {key}: {value}")
            
            print("\nGetting tenant statistics...")
            tenant_stats = await source_manager.get_tenant_statistics()
            print("Tenant statistics:")
            for key, value in tenant_stats.items():
                if key != "sources":  # Skip detailed source info for brevity
                    print(f"  {key}: {value}")
        
        print("\n=== Demo completed successfully! ===")
        
    except Exception as e:
        print(f"Demo error: {e}")
        import traceback
        traceback.print_exc()
    
    finally:
        # Cleanup
        await source_manager.cleanup()


def generate_sample_sales_data(num_records: int) -> pd.DataFrame:
    """Generate sample sales data for testing"""
    
    # Sample data pools
    customers = [f"CUST_{i:03d}" for i in range(1, 51)]  # 50 customers
    products = [
        "Laptop", "Mouse", "Keyboard", "Monitor", "Headphones",
        "Tablet", "Phone", "Charger", "Speaker", "Camera",
        "Printer", "Scanner", "Router", "Switch", "Cable"
    ]
    regions = ["North", "South", "East", "West", "Central"]
    
    # Generate records
    records = []
    base_date = datetime(2024, 1, 1)
    
    for i in range(num_records):
        order_date = base_date + timedelta(days=random.randint(0, 365))
        customer_id = random.choice(customers)
        product_name = random.choice(products)
        quantity = random.randint(1, 10)
        unit_price = round(random.uniform(10.0, 1000.0), 2)
        total_amount = round(quantity * unit_price, 2)
        region = random.choice(regions)
        
        record = {
            "order_id": f"ORD_{i+1:06d}",
            "customer_id": customer_id,
            "product_name": product_name,
            "quantity": quantity,
            "unit_price": unit_price,
            "total_amount": total_amount,
            "order_date": order_date,
            "region": region
        }
        records.append(record)
    
    return pd.DataFrame(records)


def test_rest_api():
    """Test the REST API functionality"""
    print("\n=== Testing REST API ===")
    
    base_url = "http://localhost:8000"
    
    try:
        # Test health check
        response = requests.get(f"{base_url}/health")
        if response.status_code == 200:
            print("✓ Health check passed")
            print(f"  Status: {response.json()}")
        else:
            print(f"✗ Health check failed: {response.status_code}")
            return
        
        # Test list sources
        response = requests.get(f"{base_url}/sources")
        if response.status_code == 200:
            sources = response.json()
            print(f"✓ Found {len(sources['source_ids'])} sources")
            for source_id in sources['source_ids']:
                print(f"  - {source_id}")
        else:
            print(f"✗ List sources failed: {response.status_code}")
        
        # Test tenant stats
        response = requests.get(f"{base_url}/tenant/stats")
        if response.status_code == 200:
            stats = response.json()
            print("✓ Tenant statistics:")
            print(f"  Tenant ID: {stats['tenant_id']}")
            print(f"  Total Sources: {stats['total_sources']}")
            print(f"  Total Files: {stats['total_files']}")
            print(f"  Total Rows: {stats['total_rows']}")
        else:
            print(f"✗ Tenant stats failed: {response.status_code}")
    
    except requests.exceptions.ConnectionError:
        print("✗ Could not connect to REST API. Make sure the server is running.")
    except Exception as e:
        print(f"✗ REST API test error: {e}")


if __name__ == "__main__":
    print("Choose demo mode:")
    print("1. Data operations demo (standalone)")
    print("2. REST API test (requires running server)")
    
    choice = input("Enter choice (1 or 2): ").strip()
    
    if choice == "1":
        asyncio.run(demo_data_operations())
    elif choice == "2":
        test_rest_api()
    else:
        print("Invalid choice. Please run again and select 1 or 2.")
