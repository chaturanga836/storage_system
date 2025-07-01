# Multi-Tenant Hybrid Columnar & JSON Storage System

A high-performance, multi-tenant storage system built with **microservices architecture** that supports both columnar (Parquet) and JSON data formats with advanced features like partitioning, compression, auto-scaling, and cost-based optimization.

## 🏗️ Microservices Architecture

The system is designed as a collection of independent microservices:

- **Auth Gateway** (8080) - Authentication, authorization, and API gateway
- **Operation Node** (8081) - Tenant coordination and auto-scaling  
- **Tenant Node** (8000) - Core data processing and storage
- **CBO Engine** (8082) - Cost-based query optimization
- **Metadata Catalog** (8083) - Metadata management and compaction
- **Monitoring** (8084) - Observability and health monitoring

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Auth Gateway  │    │ Operation Node  │    │   Monitoring    │
│      (8080)     │    │     (8081)      │    │     (8084)      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
            ┌────────────────────┼────────────────────┐
            │                    │                    │
   ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
   │  Tenant Node    │  │   CBO Engine    │  │ Metadata Catalog│
   │     (8000)      │  │     (8082)      │  │     (8083)      │
   └─────────────────┘  └─────────────────┘  └─────────────────┘
```
│   └── Source 2 (Parquet + WAL + Index + Metadata)
├── EC2: Tenant B (Python)
│   ├── Source 1 (Parquet + WAL + Index + Metadata)
│   └── Source 2 (Parquet + WAL + Index + Metadata)
└── EC2: Tenant C (Python)
    ├── Source 1 (Parquet + WAL + Index + Metadata)
    └── Source 2 (Parquet + WAL + Index + Metadata)
```

## Features

### Core Storage Engine
- **Columnar Storage**: Parquet-based storage with configurable compression
- **Write-Ahead Logging (WAL)**: Ensures data consistency and durability
- **Metadata Management**: Comprehensive metadata tracking for efficient queries
- **Indexing**: B-tree and hash indices for fast data retrieval

### Advanced Capabilities
- **Auto-Scaling**: Dynamic resource allocation based on workload
  - CPU and memory-based scaling
  - Query queue management
  - Adaptive worker pool sizing
- **Automatic File Compaction**: Optimizes storage efficiency
  - Small file consolidation
  - Background compaction scheduling
  - Configurable compaction policies
- **Cost-Based Query Optimizer (CBO)**: Intelligent query execution planning
  - Statistics-based cost estimation
  - Multiple execution plan generation
  - Adaptive learning from execution history
  - Index and partition pruning optimization

### Multi-API Support
- **REST API**: HTTP-based interface for web applications
- **gRPC API**: High-performance binary protocol for service-to-service communication
- **Streaming Support**: Efficient handling of large result sets

### Data Management
- **Multi-Source Support**: Manage multiple data sources within a single tenant
- **Partitioning**: Column-based partitioning for improved query performance
- **Schema Evolution**: Flexible schema management and evolution
- **Concurrent Operations**: Safe concurrent reads and writes with WAL

## Quick Start

### 1. Installation

```bash
# Clone and setup
cd storage_system
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
pip install -r requirements.txt
```

### 2. Configuration

Copy the example configuration:
```bash
cp config.env.example config.env
```

Edit `config.env` with your tenant-specific settings:
```env
TENANT_ID=your_tenant_id
TENANT_NAME=Your Tenant Name
DATA_PATH=./data
GRPC_PORT=50051
REST_PORT=8000

# Auto-scaling configuration
AUTO_SCALING_ENABLED=true
MIN_WORKERS=2
MAX_WORKERS=50
SCALE_UP_THRESHOLD_CPU=70.0
SCALE_DOWN_THRESHOLD_CPU=30.0
SCALE_UP_THRESHOLD_MEMORY=80.0
SCALE_DOWN_THRESHOLD_MEMORY=40.0

# Compaction configuration
AUTO_COMPACTION_ENABLED=true
COMPACTION_INTERVAL_MINUTES=60
MIN_FILE_SIZE_MB=50.0
MAX_FILE_SIZE_MB=512.0
TARGET_FILE_SIZE_MB=256.0

# Query optimization
QUERY_OPTIMIZATION_ENABLED=true
STATISTICS_REFRESH_INTERVAL_MINUTES=30
COST_MODEL_LEARNING_ENABLED=true
```

### 3. Running the Tenant Node

#### REST API Mode (Default)
```bash
python run.py
# or
python run.py rest
```

#### gRPC Mode
```bash
python run.py grpc
```

#### Both APIs
```bash
python run.py both
```

### 4. Test with Demo

Run the interactive demo:
```bash
python demo.py
```

Choose option 1 for standalone data operations demo, or option 2 to test the REST API.

## API Documentation

### REST API Endpoints

#### Data Operations
- `POST /data/write` - Write data to a source
- `POST /data/search` - Search data across sources
- `POST /data/search/stream` - Streaming search results
- `POST /data/aggregate` - Perform aggregations

#### Source Management
- `POST /sources/add` - Add a new data source
- `DELETE /sources/{source_id}` - Remove a data source
- `GET /sources` - List all sources
- `GET /sources/{source_id}/stats` - Get source statistics

#### Tenant Operations
- `GET /tenant/stats` - Get tenant statistics
- `GET /health` - Health check

### Example API Usage

#### Writing Data
```bash
curl -X POST "http://localhost:8000/data/write" \
  -H "Content-Type: application/json" \
  -d '{
    "source_id": "sales_data",
    "records": [
      {
        "fields": {
          "order_id": "ORD_001",
          "customer_id": "CUST_001",
          "amount": 100.50,
          "order_date": "2024-01-15"
        }
      }
    ]
  }'
```

#### Searching Data
```bash
curl -X POST "http://localhost:8000/data/search" \
  -H "Content-Type: application/json" \
  -d '{
    "filters": {
      "customer_id": {"eq": "CUST_001"}
    },
    "limit": 10
  }'
```

#### Aggregations
```bash
curl -X POST "http://localhost:8000/data/aggregate" \
  -H "Content-Type: application/json" \
  -d '{
    "aggregations": [
      {"type": "count", "alias": "total_orders"},
      {"type": "sum", "column": "amount", "alias": "total_revenue"}
    ]
  }'
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TENANT_ID` | Unique tenant identifier | `default_tenant` |
| `TENANT_NAME` | Human-readable tenant name | `Default Tenant` |
| `DATA_PATH` | Base path for data storage | `./data` |
| `GRPC_PORT` | gRPC server port | `50051` |
| `REST_PORT` | REST API server port | `8000` |
| `REST_HOST` | REST API server host | `0.0.0.0` |
| `MAX_CONCURRENT_SEARCHES` | Max parallel searches | `10` |
| `WAL_RETENTION_HOURS` | WAL log retention | `24` |

### Data Source Configuration

Each data source requires:
- **source_id**: Unique identifier
- **name**: Human-readable name
- **data_path**: Storage location
- **schema_definition**: Column types
- **partition_columns**: Partitioning strategy
- **index_columns**: Columns to index

Example:
```python
SourceConfig(
    source_id="sales_data",
    name="Sales Data Source",
    connection_string="file://sales",
    data_path="./data/sources/sales_data",
    schema_definition={
        "order_id": "string",
        "customer_id": "string",
        "amount": "float64",
        "order_date": "datetime64[ns]"
    },
    partition_columns=["order_date"],
    index_columns=["customer_id", "order_date"],
    compression="snappy",
    max_file_size_mb=256,
    wal_enabled=True
)
```

## Directory Structure

```
storage_system/
├── tenant_node/           # Main tenant node package
│   ├── __init__.py
│   ├── config.py         # Configuration management
│   ├── data_source.py    # Data source and source manager
│   ├── wal.py           # Write-Ahead Logging
│   ├── index.py         # Index management
│   ├── metadata.py      # Metadata tracking
│   ├── grpc_service.py  # gRPC service implementation
│   ├── rest_api.py      # REST API implementation
│   └── tenant_node.py   # Main application
├── proto/                # Protocol buffer definitions
│   └── storage.proto
├── data/                 # Data storage (created at runtime)
├── main.py              # Entry point
├── run.py               # Run script
├── demo.py              # Demo and testing
├── requirements.txt     # Python dependencies
└── README.md           # This file
```

## Performance Features

### Indexing
- Automatic min/max indexing for range queries
- Unique value sets for small cardinality columns
- File-level statistics for query optimization

### Partitioning
- Date-based partitioning for time-series data
- Custom partitioning strategies
- Automatic partition pruning

### Parallel Processing
- Concurrent search across multiple sources
- Parallel aggregation processing
- Non-blocking I/O operations

### Compression
- Snappy compression for fast read/write
- Configurable compression algorithms
- Optimized for analytical workloads

## Monitoring and Operations

### Health Checks
```bash
curl http://localhost:8000/health
```

### Statistics
```bash
# Tenant-wide statistics
curl http://localhost:8000/tenant/stats

# Source-specific statistics
curl http://localhost:8000/sources/sales_data/stats
```

### Logs
The system uses structured JSON logging for easy parsing and monitoring.

## Development

### Running Tests
```bash
python demo.py
```

### Code Structure
- **Async/await**: Full async support for non-blocking operations
- **Type hints**: Complete type annotations
- **Error handling**: Comprehensive error handling and logging
- **Modular design**: Clean separation of concerns

### Adding New Features
1. **Data Sources**: Extend `DataSource` class
2. **APIs**: Add endpoints to `rest_api.py` or `grpc_service.py`
3. **Storage**: Modify WAL, index, or metadata managers
4. **Configuration**: Add options to `config.py`

## Integration with Operation Node

This tenant node is designed to work with a Go-based Operation Node that:
1. **Coordinates** queries across multiple tenant nodes
2. **Aggregates** results from parallel tenant processing
3. **Manages** tenant metadata in ClickHouse
4. **Handles** authentication and authorization

## Production Deployment

### EC2 Instance Setup
1. **Instance Type**: Compute-optimized instances (C5/C6i)
2. **Storage**: EBS volumes with appropriate IOPS
3. **Networking**: VPC with proper security groups
4. **Monitoring**: CloudWatch integration

### Environment Variables
Set production-appropriate values:
```env
TENANT_ID=prod_tenant_001
DATA_PATH=/var/lib/tenant_node/data
GRPC_PORT=50051
REST_PORT=8000
MAX_CONCURRENT_SEARCHES=50
```

### Process Management
Use systemd, supervisor, or container orchestration for process management.

## Troubleshooting

### Common Issues
1. **Permission errors**: Check data directory permissions
2. **Port conflicts**: Ensure ports 8000/50051 are available
3. **Memory usage**: Monitor for large result sets
4. **Disk space**: Check available space in data directory

### Debug Mode
Enable debug logging:
```python
import logging
logging.basicConfig(level=logging.DEBUG)
```

## Contributing

1. Follow the existing code style and structure
2. Add appropriate type hints and docstrings
3. Test new features with the demo script
4. Update documentation for new APIs or configuration options

## License

This project is part of a larger hybrid storage system architecture.
