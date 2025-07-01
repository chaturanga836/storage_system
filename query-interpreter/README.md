# Query Interpreter Service

The Query Interpreter Service transforms SQL queries and custom query expressions into optimized internal execution plans. It serves as the parsing and transformation layer in our distributed query processing pipeline.

## 🎯 Overview

The Query Interpreter is a critical component that bridges the gap between user-facing SQL/DSL queries and the internal distributed execution engine. It provides standardized query representations that enable efficient distributed query processing across multiple tenant nodes.

## ✨ Features

- **SQL Parsing & Validation** using SQLGlot with multi-dialect support
- **Multi-dialect SQL Support** (PostgreSQL, MySQL, SQLite, BigQuery, etc.)
- **Custom Query DSL** for domain-specific operations
- **Query Transformation** to internal execution plans
- **Logical Plan Generation** for distributed execution
- **Integration with CBO Engine** for cost-based optimization
- **Query Validation** with comprehensive error reporting
- **Partition-aware Planning** for distributed data access

## 🏗️ System Integration

The Query Interpreter plays a central role in our microservices read flow:

```
@startuml
participant User
participant "Operation Node" as OpNode
participant "Query Interpreter" as Interpreter
participant "Metadata Store" as Metadata
participant "Tenant Node A" as TenantA
participant "Tenant Node B" as TenantB

User -> OpNode : SQL query
OpNode -> Interpreter : parse query
Interpreter --> OpNode : logical plan

OpNode -> Metadata : fetch file stats
Metadata --> OpNode : file metadata

par Parallel query dispatch
    OpNode -> TenantA : execute subquery (partition A)
    TenantA --> OpNode : partial result A

    OpNode -> TenantB : execute subquery (partition B)
    TenantB --> OpNode : partial result B
end

OpNode -> User : final aggregated result
@enduml
```

### Query Processing Flow

1. **User submits SQL query** to Operation Node
2. **Operation Node forwards query** to Query Interpreter
3. **Query Interpreter parses and validates** the SQL query
4. **Logical plan generation** with partition awareness
5. **Operation Node uses logical plan** to coordinate distributed execution
6. **Parallel execution** across multiple Tenant Nodes
7. **Result aggregation** and return to user

## 📖 Complete Read Operation Flow

### Step-by-Step Read Process

Let's walk through exactly how a read operation flows through our microservices architecture:

#### 1. **User Query Initiation**
```sql
-- User submits a SQL query
SELECT customer_id, SUM(amount) as total_spent, COUNT(*) as order_count
FROM orders 
WHERE order_date >= '2024-01-01' 
GROUP BY customer_id 
ORDER BY total_spent DESC 
LIMIT 100
```

#### 2. **Operation Node - Query Reception** 
```python
# Operation Node (Port 8081) receives the query
@app.post("/query/execute")
async def execute_query(request: QueryRequest):
    sql_query = request.query
    user_id = request.user_id
    
    # Log query reception
    logger.info(f"Received query from user {user_id}: {sql_query}")
    
    # Forward to Query Interpreter for parsing
    return await process_distributed_query(sql_query, user_id)
```

#### 3. **Query Interpreter - SQL Parsing**
```python
# Query Interpreter (Port 8085) parses the SQL
@app.post("/parse/sql")
async def parse_sql_query(request: SQLParseRequest):
    sql_query = request.query
    
    # Parse SQL into Abstract Syntax Tree (AST)
    try:
        ast = sqlglot.parse_one(sql_query, dialect='mysql')
        
        # Transform AST to logical plan
        logical_plan = {
            "operation": "aggregate",
            "source_table": "orders",
            "filters": [
                {"column": "order_date", "operator": ">=", "value": "2024-01-01"}
            ],
            "group_by": ["customer_id"],
            "aggregations": [
                {"function": "sum", "column": "amount", "alias": "total_spent"},
                {"function": "count", "column": "*", "alias": "order_count"}
            ],
            "sort": [{"column": "total_spent", "direction": "desc"}],
            "limit": 100,
            
            # Critical: Partition information for distributed execution
            "partition_info": {
                "partition_column": "order_date",
                "partition_strategy": "date_based",
                "affected_partitions": ["2024-01", "2024-02", "2024-03"],
                "parallel_execution_possible": True
            }
        }
        
        return {"success": True, "logical_plan": logical_plan}
        
    except Exception as e:
        return {"success": False, "error": str(e)}
```

#### 4. **Operation Node - Plan Analysis**
```python
async def process_distributed_query(sql_query: str, user_id: str):
    # Step 1: Parse query with Query Interpreter
    parse_response = await call_query_interpreter(sql_query)
    logical_plan = parse_response["logical_plan"]
    
    # Step 2: Determine which tenant nodes have relevant data
    partition_info = logical_plan["partition_info"]
    affected_partitions = partition_info["affected_partitions"]
    
    # Step 3: Query Metadata Catalog for partition locations
    metadata_response = await call_metadata_catalog(affected_partitions)
    
    return await execute_distributed_plan(logical_plan, metadata_response)
```

#### 5. **Metadata Catalog - Partition Location Discovery**
```python
# Metadata Catalog (Port 8083) provides partition metadata
@app.post("/metadata/partitions")
async def get_partition_metadata(request: PartitionRequest):
    partitions = request.partition_list
    
    # Query internal metadata store
    partition_metadata = {}
    for partition in partitions:
        partition_metadata[partition] = {
            "location": determine_partition_location(partition),
            "tenant_nodes": get_tenant_nodes_for_partition(partition),
            "file_stats": {
                "row_count": get_partition_row_count(partition),
                "file_size": get_partition_file_size(partition),
                "last_modified": get_partition_last_modified(partition)
            },
            "index_info": get_partition_indexes(partition)
        }
    
    # Example response
    return {
        "partition_metadata": {
            "2024-01": {
                "tenant_nodes": ["tenant-node-a"],
                "file_stats": {"row_count": 1000000, "file_size": "500MB"},
                "files": ["/data/orders/2024/01/part-001.parquet", "/data/orders/2024/01/part-002.parquet"]
            },
            "2024-02": {
                "tenant_nodes": ["tenant-node-b"],
                "file_stats": {"row_count": 1200000, "file_size": "600MB"},
                "files": ["/data/orders/2024/02/part-001.parquet"]
            },
            "2024-03": {
                "tenant_nodes": ["tenant-node-a", "tenant-node-b"],
                "file_stats": {"row_count": 800000, "file_size": "400MB"},
                "files": ["/data/orders/2024/03/part-001.parquet", "/data/orders/2024/03/part-002.parquet"]
            }
        }
    }
```

#### 6. **Operation Node - Subquery Generation**
```python
async def execute_distributed_plan(logical_plan: dict, metadata: dict):
    partition_metadata = metadata["partition_metadata"]
    
    # Generate subqueries for each tenant node
    subqueries = []
    
    for partition, meta in partition_metadata.items():
        for tenant_node in meta["tenant_nodes"]:
            # Create partition-specific subquery
            subquery = generate_subquery_for_partition(logical_plan, partition, meta)
            subqueries.append({
                "tenant_node": tenant_node,
                "partition": partition,
                "query": subquery,
                "files": meta["files"]
            })
    
    # Example generated subqueries
    subqueries = [
        {
            "tenant_node": "tenant-node-a",
            "partition": "2024-01",
            "query": """
                SELECT customer_id, 
                       SUM(amount) as partial_sum, 
                       COUNT(*) as partial_count
                FROM orders 
                WHERE order_date >= '2024-01-01' 
                  AND order_date < '2024-02-01'
                  AND file_path IN ('/data/orders/2024/01/part-001.parquet', '/data/orders/2024/01/part-002.parquet')
                GROUP BY customer_id
            """,
            "files": ["/data/orders/2024/01/part-001.parquet", "/data/orders/2024/01/part-002.parquet"]
        },
        {
            "tenant_node": "tenant-node-b", 
            "partition": "2024-02",
            "query": """
                SELECT customer_id, 
                       SUM(amount) as partial_sum, 
                       COUNT(*) as partial_count
                FROM orders 
                WHERE order_date >= '2024-02-01' 
                  AND order_date < '2024-03-01'
                  AND file_path = '/data/orders/2024/02/part-001.parquet'
                GROUP BY customer_id
            """,
            "files": ["/data/orders/2024/02/part-001.parquet"]
        }
        # ... more subqueries for other partitions
    ]
    
    # Execute subqueries in parallel
    return await execute_parallel_subqueries(subqueries)
```

#### 7. **Parallel Execution Across Tenant Nodes**
```python
async def execute_parallel_subqueries(subqueries: list):
    tasks = []
    
    # Create async tasks for each tenant node
    for subquery in subqueries:
        task = execute_on_tenant_node(
            subquery["tenant_node"], 
            subquery["query"], 
            subquery["files"]
        )
        tasks.append(task)
    
    # Execute all subqueries in parallel
    partial_results = await asyncio.gather(*tasks)
    
    return partial_results
```

#### 8. **Tenant Node A - Data Processing**
```python
# Tenant Node A (Port 8000) executes its portion
@app.post("/data/execute")
async def execute_subquery(request: SubQueryRequest):
    query = request.query
    files = request.files
    
    # Load data from Parquet files
    dataframes = []
    for file_path in files:
        df = pd.read_parquet(file_path)
        dataframes.append(df)
    
    # Combine dataframes
    combined_df = pd.concat(dataframes, ignore_index=True)
    
    # Apply filters
    filtered_df = combined_df[
        combined_df['order_date'] >= '2024-01-01'
    ]
    
    # Perform aggregation
    result_df = filtered_df.groupby('customer_id').agg({
        'amount': 'sum',
        'order_id': 'count'
    }).rename(columns={
        'amount': 'partial_sum',
        'order_id': 'partial_count'
    }).reset_index()
    
    # Return partial results
    return {
        "success": True,
        "partition": "2024-01", 
        "results": result_df.to_dict('records'),
        "row_count": len(result_df),
        "execution_time_ms": 150
    }
```

#### 9. **Tenant Node B - Data Processing**
```python
# Tenant Node B (Port 8000) executes its portion
# Similar to Tenant Node A but for different partitions
@app.post("/data/execute") 
async def execute_subquery(request: SubQueryRequest):
    # ... similar processing for 2024-02 partition ...
    
    return {
        "success": True,
        "partition": "2024-02",
        "results": result_df.to_dict('records'),
        "row_count": len(result_df), 
        "execution_time_ms": 200
    }
```

#### 10. **Operation Node - Result Aggregation**
```python
async def aggregate_final_results(partial_results: list, logical_plan: dict):
    # Combine partial results from all tenant nodes
    all_results = []
    for partial_result in partial_results:
        if partial_result["success"]:
            all_results.extend(partial_result["results"])
    
    # Convert to DataFrame for final aggregation
    combined_df = pd.DataFrame(all_results)
    
    # Perform final aggregation (sum the partial sums, sum the partial counts)
    final_df = combined_df.groupby('customer_id').agg({
        'partial_sum': 'sum',      # Sum of partial sums = total sum
        'partial_count': 'sum'     # Sum of partial counts = total count
    }).rename(columns={
        'partial_sum': 'total_spent',
        'partial_count': 'order_count'
    }).reset_index()
    
    # Apply final sorting and limit
    final_df = final_df.sort_values('total_spent', ascending=False).head(100)
    
    return {
        "success": True,
        "results": final_df.to_dict('records'),
        "total_rows": len(final_df),
        "partitions_processed": [r["partition"] for r in partial_results],
        "total_execution_time_ms": sum(r["execution_time_ms"] for r in partial_results)
    }
```

#### 11. **Final Response to User**
```python
# Operation Node returns final results to user
@app.post("/query/execute")
async def execute_query(request: QueryRequest):
    # ... all the above processing ...
    
    final_results = await aggregate_final_results(partial_results, logical_plan)
    
    return {
        "query_id": generate_query_id(),
        "success": True,
        "results": final_results["results"],
        "metadata": {
            "total_rows_returned": final_results["total_rows"],
            "partitions_processed": final_results["partitions_processed"],
            "execution_time_ms": final_results["total_execution_time_ms"],
            "nodes_involved": ["tenant-node-a", "tenant-node-b"]
        }
    }
```

## 🏔️ Comparison with Snowflake-like Systems

### System Architecture Comparison

Our system represents a **hybrid approach** between traditional distributed databases and modern cloud data warehouses like Snowflake, BigQuery, and Redshift.

#### **Our Microservices Architecture**
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Auth Gateway  │    │ Query Interpreter│    │   Monitoring    │
│      (8080)     │    │     (8085)       │    │     (8084)      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
    ┌────┴────┐             ┌────┴────┐             ┌────┴────┐
┌───▼───┐ ┌───▼───┐     ┌───▼───┐ ┌───▼───┐     ┌───▼───┐ ┌───▼───┐
│TenantA│ │TenantB│     │ CBO   │ │Metadata│     │OpNode │ │ More  │
│(8000) │ │(8001) │     │(8082) │ │(8083)  │     │(8081) │ │Tenants│
└───────┘ └───────┘     └───────┘ └───────┘     └───────┘ └───────┘
```

#### **Snowflake Architecture**
```
┌─────────────────────────────────────────────────────────────────┐
│                    Snowflake Cloud Services                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Query     │  │  Metadata   │  │    Transaction &        │  │
│  │ Optimization│  │  Management │  │    Security Services    │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────────┐
│                     Compute Layer (Virtual Warehouses)         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │Warehouse XS │  │Warehouse S  │  │    Warehouse L          │  │
│  │(Auto-scale) │  │(Auto-scale) │  │    (Auto-scale)         │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────────┐
│                        Storage Layer (S3/Blob)                 │
│           Automatically managed, columnar, compressed          │
└─────────────────────────────────────────────────────────────────┘
```

### 📊 Detailed Comparison

| Aspect | **Our System** | **Snowflake-like Systems** |
|--------|----------------|----------------------------|
| **🏗️ Architecture** | Microservices, self-managed | Fully managed cloud service |
| **💰 Cost Model** | Infrastructure + operational | Pay-per-query + storage |
| **🔧 Control** | Full control over all components | Limited customization |
| **📍 Deployment** | On-premise, cloud, hybrid | Cloud-only |
| **⚡ Scaling** | Manual + auto-scaling logic | Automatic, serverless |
| **🔒 Security** | Custom implementation | Enterprise-grade built-in |
| **🛠️ Maintenance** | Self-managed | Fully managed |
| **🔌 Integration** | Custom APIs, full control | Standard SQL, limited APIs |

### ✅ **Advantages of Our System**

#### **1. Cost Efficiency for Predictable Workloads**
```python
# Our system cost calculation
monthly_cost = (
    infrastructure_cost +     # Fixed servers/cloud instances
    development_time +        # One-time setup
    operational_overhead      # Monitoring, maintenance
)

# Snowflake cost calculation  
monthly_cost = (
    compute_credits * credit_rate +  # Variable based on usage
    storage_cost +                   # Data volume
    data_transfer_cost              # Network egress
)

# For high-volume, predictable workloads, our system can be 60-80% cheaper
```

#### **2. Complete Data Sovereignty & Control**
```yaml
Our System:
  - Full control over data location
  - Custom encryption at rest/transit
  - Compliance with any regulation
  - No vendor lock-in
  - Custom data retention policies

Snowflake:
  - Limited region choice
  - Standard encryption (good, but fixed)
  - Vendor-controlled compliance
  - Platform lock-in risk
  - Fixed retention options
```

#### **3. Ultra-Low Latency for Real-time Applications**
```python
# Our system latency profile
local_query_latency = "10-50ms"     # Direct database access
network_latency = "minimal"         # Local/private network
cold_start_time = "0ms"             # Always warm

# Snowflake latency profile  
warehouse_spin_up = "0-10 seconds"  # Cold start penalty
network_latency = "50-200ms"        # Internet round-trip
query_execution = "fast"            # Once running
```

#### **4. Hybrid Storage Flexibility**
```python
# Our system supports multiple storage patterns
storage_options = {
    "hot_data": "SSD with indexes",           # Sub-second queries
    "warm_data": "Parquet on standard disk", # Second-range queries  
    "cold_data": "S3/blob with compression", # Minute-range queries
    "real_time": "In-memory + WAL",          # Microsecond queries
}

# Snowflake has single storage tier
snowflake_storage = "Columnar cloud storage"  # One size fits all
```

#### **5. Custom Query Optimization**
```python
# Our CBO Engine can implement custom optimizations
class CustomOptimizations:
    def domain_specific_rules(self, query):
        """Apply business-specific optimizations"""
        if self.is_time_series_query(query):
            return self.optimize_for_time_series(query)
        elif self.is_geospatial_query(query):
            return self.optimize_for_geospatial(query)
        
    def tenant_aware_optimization(self, query, tenant):
        """Optimize based on tenant-specific patterns"""
        return self.apply_tenant_specific_indexes(query, tenant)

# Snowflake uses general-purpose optimization (very good, but generic)
```

#### **6. Multi-Tenant Architecture Advantages**
```python
# Our system provides true tenant isolation
tenant_benefits = {
    "data_isolation": "Physical separation per tenant",
    "performance_isolation": "Dedicated resources per tenant", 
    "cost_allocation": "Precise per-tenant cost tracking",
    "customization": "Tenant-specific optimizations",
    "scaling": "Independent scaling per tenant"
}

# Snowflake provides logical separation within shared infrastructure
```

### ❌ **Disadvantages of Our System**

#### **1. Operational Complexity**
```yaml
Our System Requires:
  - Infrastructure management
  - Database administration
  - Monitoring and alerting setup
  - Backup and disaster recovery
  - Security patch management
  - Performance tuning expertise

Snowflake Provides:
  - Zero infrastructure management
  - Automatic optimization
  - Built-in monitoring
  - Automatic backups
  - Automatic security updates
  - Self-tuning performance
```

#### **2. Limited Built-in Analytics Features**
```python
# Snowflake includes many advanced features out-of-the-box
snowflake_features = [
    "Time Travel (data versioning)",
    "Zero-copy cloning", 
    "Automatic clustering",
    "Advanced SQL functions",
    "Machine learning functions",
    "Geospatial analytics",
    "Semi-structured data support",
    "Automatic materialized views"
]

# Our system requires custom implementation for advanced features
our_features = [
    "Basic SQL processing ✓",
    "Custom extensions ✓", 
    "Domain-specific optimization ✓",
    "Advanced analytics ❌ (custom development required)",
    "ML integration ❌ (external tools needed)",
    "Time travel ❌ (custom WAL implementation needed)"
]
```

#### **3. Scaling Limitations**
```python
# Our system scaling constraints
scaling_limits = {
    "vertical_scaling": "Limited by single machine resources",
    "horizontal_scaling": "Manual node addition/removal",
    "storage_scaling": "Manual partition management", 
    "compute_scaling": "Custom auto-scaling logic required"
}

# Snowflake automatic scaling
snowflake_scaling = {
    "vertical_scaling": "Automatic warehouse resizing",
    "horizontal_scaling": "Automatic multi-cluster warehouses",
    "storage_scaling": "Unlimited, automatic",
    "compute_scaling": "Serverless, instant"
}
```

#### **4. Development and Maintenance Overhead**
```python
# Total Cost of Ownership (TCO) comparison
our_system_tco = {
    "initial_development": "6-12 months",
    "ongoing_maintenance": "2-3 engineers full-time",
    "feature_development": "Custom timeline per feature",
    "expertise_required": "Database, distributed systems, DevOps"
}

snowflake_tco = {
    "initial_setup": "1-2 weeks",
    "ongoing_maintenance": "Minimal operational overhead",
    "feature_development": "Use built-in features",
    "expertise_required": "SQL and basic configuration"
}
```

### 🎯 **When to Choose Our System vs Snowflake**

#### **Choose Our System When:**

1. **🏢 Regulatory/Compliance Requirements**
   ```python
   scenarios = [
       "Data must stay in specific geographic locations",
       "Custom encryption requirements", 
       "Air-gapped environments",
       "Government/defense contracts",
       "HIPAA with custom controls"
   ]
   ```

2. **💰 Cost Optimization for High-Volume Workloads**
   ```python
   # Break-even analysis
   if monthly_query_volume > 10_000_000:
       if workload_predictability > 80%:
           our_system_savings = "60-80% vs Snowflake"
   ```

3. **⚡ Ultra-Low Latency Requirements**
   ```python
   latency_requirements = {
       "real_time_dashboards": "< 100ms end-to-end",
       "operational_analytics": "< 1 second",
       "embedded_analytics": "< 50ms per query"
   }
   ```

4. **🔧 Custom Business Logic Integration**
   ```python
   custom_requirements = [
       "Domain-specific query optimizations",
       "Custom data processing pipelines", 
       "Proprietary algorithms integration",
       "Non-standard data formats",
       "Custom security models"
   ]
   ```

#### **Choose Snowflake When:**

1. **🚀 Rapid Time-to-Market**
   ```python
   business_requirements = {
       "quick_proof_of_concept": "< 1 month",
       "limited_technical_team": "< 5 engineers", 
       "standard_analytics_needs": "BI tools + SQL",
       "variable_workloads": "Unpredictable usage patterns"
   }
   ```

2. **🌍 Global Scale with Minimal Operations**
   ```python
   scale_requirements = {
       "multi_region_deployment": "Automatic",
       "petabyte_scale_storage": "Managed",
       "thousands_of_users": "Built-in concurrency",
       "disaster_recovery": "Automatic"
   }
   ```

3. **📊 Standard Analytics Workloads**
   ```python
   use_cases = [
       "Business intelligence and reporting",
       "Data warehousing with ETL/ELT",
       "Ad-hoc analytics and data science", 
       "Standard SQL-based applications",
       "Integration with popular BI tools"
   ]
   ```

### 🔬 **Performance Comparison**

#### **Query Performance**
```python
# Simple aggregation query: SUM, COUNT, GROUP BY
our_system_performance = {
    "small_dataset": "10-50ms",      # < 1M rows
    "medium_dataset": "100-500ms",   # 1M-100M rows  
    "large_dataset": "1-10s",        # 100M-1B rows
    "cold_start_penalty": "0ms"      # Always warm
}

snowflake_performance = {
    "small_dataset": "100-500ms",    # Network + processing
    "medium_dataset": "200ms-2s",    # Automatic optimization
    "large_dataset": "500ms-5s",     # Massive parallel processing
    "cold_start_penalty": "0-10s"    # Warehouse spin-up
}
```

#### **Concurrent Users**
```python
# Concurrent query handling
our_system_concurrency = {
    "concurrent_queries": "100-1000",     # Per tenant node
    "total_system": "10000+",             # Multiple tenant nodes
    "isolation": "Physical per tenant",
    "resource_contention": "Minimal within tenant"
}

snowflake_concurrency = {
    "concurrent_queries": "Unlimited",    # Auto-scaling
    "total_system": "Unlimited",          # Cloud-scale
    "isolation": "Logical separation",
    "resource_contention": "Managed automatically"
}
```

### 💡 **Hybrid Approach Recommendation**

For many organizations, a **hybrid approach** provides the best of both worlds:

```python
# Recommended architecture for enterprise
hybrid_architecture = {
    "real_time_layer": "Our microservices system",
    "analytical_layer": "Snowflake for complex analytics", 
    "data_flow": "Stream from our system to Snowflake",
    "use_cases": {
        "operational_queries": "Our system (low latency)",
        "complex_analytics": "Snowflake (ease of use)",
        "real_time_dashboards": "Our system (performance)",
        "ad_hoc_analysis": "Snowflake (flexibility)"
    }
}
```

### 🎯 **Summary: System Positioning**

Our microservices-based storage system is positioned as a **high-performance, cost-effective alternative** to cloud data warehouses for organizations that need:

- **🏁 Ultra-low latency** (< 100ms queries)
- **💰 Cost optimization** for predictable, high-volume workloads  
- **🔒 Data sovereignty** and regulatory compliance
- **🎛️ Custom business logic** integration
- **🏢 Multi-tenant isolation** with dedicated resources

While Snowflake excels in **ease of use, automatic scaling, and rich analytics features**, our system provides **maximum control, cost efficiency, and performance optimization** for specialized use cases.

The choice depends on your organization's priorities: **operational simplicity vs. technical control**, **time-to-market vs. cost optimization**, and **standard features vs. custom requirements**.

---

*This comparison helps position our Query Interpreter service within the broader data architecture landscape, highlighting where our microservices approach provides unique value over cloud data warehouse solutions.*
