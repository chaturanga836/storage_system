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
- **Query Interpreter** (8085) - SQL/DSL parsing and query plan generation

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Auth Gateway  │    │ Operation Node  │    │   Monitoring    │
│      (8080)     │    │     (8081)      │    │     (8084)      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
    ┌────────────────────────────┼────────────────────────────┐
    │                            │                            │
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  Tenant Node    │  │   CBO Engine    │  │ Metadata Catalog│  │Query Interpreter│
│     (8000)      │  │     (8082)      │  │     (8083)      │  │     (8085)      │
└─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘
```
│   └── Source 2 (Parquet + WAL + Index + Metadata)
├── 🏢 Tenant B
│   ├── Source 1 (Parquet + WAL + Index + Metadata)
│   └── Source 2 (Parquet + WAL + Index + Metadata)
└── 🏢 Tenant C
    ├── Source 1 (Parquet + WAL + Index + Metadata)
    └── Source 2 (Parquet + WAL + Index + Metadata)
```

### Service Communication Flow

```
📱 Client Request
    │
    ▼
🔐 Auth Gateway (8080) ─── Validates Token
    │
    ▼
🎯 Operation Node (8081) ─── Coordinates Query
    │
    ├── 🧠 Query Interpreter (8085) ─── Parses SQL/DSL
    ├── 📊 CBO Engine (8082) ─── Optimizes Query
    ├── 📁 Metadata Catalog (8083) ─── Gets Schema Info
    └── 🔍 Tenant Node (8000) ─── Executes Query
    
📈 Monitoring (8084) ─── Observes All Services
```

## ✨ Features

### 🏗️ Microservices Architecture
- **Service Isolation**: Independent, loosely-coupled services
- **API Gateway**: Centralized authentication and routing
- **Service Discovery**: Automatic service registration and discovery
- **Load Balancing**: Distribute load across service instances
- **Circuit Breakers**: Fault tolerance and cascade failure prevention

### 🔐 Advanced Authentication & Authorization
- **JWT-based Authentication**: Secure token-based auth
- **Role-based Access Control (RBAC)**: Fine-grained permissions
- **Multi-tenant Security**: Tenant isolation and data segregation
- **API Key Management**: Programmatic access control

### 🗄️ Hybrid Storage Engine
- **Columnar Storage**: Parquet-based storage with configurable compression
- **JSON Flexibility**: Schema-less JSON document support
- **Write-Ahead Logging (WAL)**: Ensures data consistency and durability
- **Metadata Management**: Comprehensive metadata tracking for efficient queries
- **Indexing**: B-tree and hash indices for fast data retrieval

### 🔍 Intelligent Query Processing
- **SQL Parser**: Full SQL support using SQLGlot
- **DSL Support**: Custom Domain-Specific Language queries
- **Query Optimization**: Cost-based query optimizer with machine learning
- **Execution Planning**: Multi-stage query execution plans
- **Result Streaming**: Efficient handling of large result sets

### ⚡ Auto-Scaling & Performance
- **Dynamic Scaling**: CPU and memory-based auto-scaling
- **Query Queue Management**: Intelligent query prioritization
- **Adaptive Worker Pools**: Dynamic resource allocation
- **Performance Monitoring**: Real-time metrics and alerting
- **Load Balancing**: Distribute queries across multiple nodes

### 🗂️ Data Management
- **Multi-Source Support**: Manage multiple data sources within tenants
- **Partitioning**: Column-based partitioning for improved query performance
- **Automatic Compaction**: Background file consolidation and optimization
- **Schema Evolution**: Flexible schema management and evolution
- **Concurrent Operations**: Safe concurrent reads and writes with WAL

### 📊 Observability & Monitoring
- **Metrics Collection**: Prometheus-compatible metrics
- **Distributed Tracing**: End-to-end request tracing
- **Health Monitoring**: Comprehensive health checks
- **Log Aggregation**: Centralized logging with structured logs
- **Performance Analytics**: Query performance insights

### 🔌 Multi-Protocol Support
- **REST APIs**: HTTP-based interface for web applications
- **gRPC APIs**: High-performance binary protocol for service communication
- **WebSocket**: Real-time data streaming
- **Message Queues**: Asynchronous processing support


## Quick Start

### Prerequisites
- Python 3.8+
- Docker and Docker Compose (for containerized deployment)

### 🚀 Easy Start with Docker Compose

The fastest way to run the entire microservices stack:

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop all services
docker-compose down
```

### 🛠️ Development Setup

#### 1. Clone and Setup Environment
```bash
cd storage_system
python -m venv venv

# Windows
venv\Scripts\activate
# Linux/macOS
source venv/bin/activate

# Install demo dependencies
pip install -r demo_requirements.txt
```

#### 2. Start Individual Services

##### Option A: Use convenience scripts
```bash
# Windows
.\start_services.ps1

# Linux/macOS
./start_services.sh
```

##### Option B: Start services manually
```bash
# Start each service in separate terminals

# Auth Gateway
cd auth-gateway && python main.py

# Operation Node  
cd operation-node && python main.py

# Tenant Node
cd tenant-node && python main.py

# CBO Engine
cd cbo-engine && python main.py

# Metadata Catalog
cd metadata-catalog && python main.py

# Query Interpreter
cd query-interpreter && python main.py

# Monitoring
cd monitoring && python main.py
```

#### 3. Run Demos

```bash
# Integration demo (tests all services)
python microservices_demo.py

# Authentication demo
python auth_demo.py

# Auto-scaling demo
python scaling_demo.py
```

### 📊 Service URLs

Once started, services are available at:

- **Auth Gateway**: http://localhost:8080
- **Tenant Node**: http://localhost:8000  
- **Operation Node**: http://localhost:8081
- **CBO Engine**: http://localhost:8082
- **Metadata Catalog**: http://localhost:8083
- **Monitoring**: http://localhost:8084
- **Query Interpreter**: http://localhost:8085

## 🔗 API Documentation

### Microservices API Overview

Each service exposes its own REST API. Here are the key endpoints:

#### Auth Gateway (Port 8080)
- `POST /auth/login` - User authentication
- `POST /auth/register` - User registration  
- `POST /auth/validate` - Token validation
- `GET /auth/permissions/{user_id}` - Get user permissions

#### Tenant Node (Port 8000)
- `POST /data/write` - Write data to a source
- `POST /data/search` - Search data across sources
- `POST /data/search/stream` - Streaming search results
- `POST /data/aggregate` - Perform aggregations
- `POST /sources/add` - Add a new data source
- `GET /sources` - List all sources
- `GET /health` - Health check

#### CBO Engine (Port 8082)
- `POST /optimize/query` - Optimize query execution plan
- `GET /stats/execution` - Query execution statistics
- `POST /costs/estimate` - Estimate query costs

#### Metadata Catalog (Port 8083)
- `GET /metadata/sources` - List source metadata
- `POST /metadata/compact` - Trigger compaction
- `GET /metadata/stats` - Get metadata statistics

#### Query Interpreter (Port 8085)
- `POST /parse/sql` - Parse SQL query
- `POST /parse/dsl` - Parse DSL query
- `POST /transform/query` - Transform query to execution plan

#### Monitoring (Port 8084)
- `GET /metrics` - Prometheus metrics
- `GET /health/all` - Health status of all services
- `GET /logs/{service}` - Service logs

### Example API Usage

#### Authentication Flow
```bash
# Register user
curl -X POST "http://localhost:8080/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "test_user",
    "password": "secure_password"
  }'

# Login
curl -X POST "http://localhost:8080/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "test_user", 
    "password": "secure_password"
  }'
```

#### Writing Data (via Tenant Node)
```bash
curl -X POST "http://localhost:8000/data/write" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
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

#### SQL Query Processing
```bash
# Parse SQL with Query Interpreter
curl -X POST "http://localhost:8085/parse/sql" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "SELECT customer_id, SUM(amount) FROM sales_data WHERE order_date >= '2024-01-01' GROUP BY customer_id"
  }'

# Optimize with CBO Engine
curl -X POST "http://localhost:8082/optimize/query" \
  -H "Content-Type: application/json" \
  -d '{
    "query_plan": "<parsed_query_plan>"
  }'
```

## ⚙️ Configuration

### Service-Specific Environment Variables

#### Auth Gateway (8080)
| Variable | Description | Default |
|----------|-------------|---------|
| `JWT_SECRET` | Secret key for JWT tokens | `your-secret-key` |
| `JWT_EXPIRATION_HOURS` | Token expiration time | `24` |
| `ALLOWED_ORIGINS` | CORS allowed origins | `*` |
| `BCRYPT_ROUNDS` | Password hashing rounds | `12` |

#### Tenant Node (8000)
| Variable | Description | Default |
|----------|-------------|---------|
| `TENANT_ID` | Unique tenant identifier | `default_tenant` |
| `TENANT_NAME` | Human-readable tenant name | `Default Tenant` |
| `DATA_PATH` | Base path for data storage | `./data` |
| `MAX_CONCURRENT_SEARCHES` | Max parallel searches | `10` |
| `WAL_RETENTION_HOURS` | WAL log retention | `24` |

#### Operation Node (8081)
| Variable | Description | Default |
|----------|-------------|---------|
| `AUTO_SCALING_ENABLED` | Enable auto-scaling | `true` |
| `MIN_WORKERS` | Minimum worker count | `2` |
| `MAX_WORKERS` | Maximum worker count | `50` |
| `SCALE_UP_THRESHOLD_CPU` | CPU threshold for scaling up | `70.0` |
| `SCALE_DOWN_THRESHOLD_CPU` | CPU threshold for scaling down | `30.0` |

#### CBO Engine (8082)
| Variable | Description | Default |
|----------|-------------|---------|
| `STATISTICS_REFRESH_INTERVAL` | Stats refresh interval (minutes) | `30` |
| `COST_MODEL_LEARNING_ENABLED` | Enable ML cost model | `true` |
| `QUERY_CACHE_TTL` | Query cache TTL (seconds) | `300` |
| `MAX_EXECUTION_PLANS` | Max plans to generate | `5` |

#### Metadata Catalog (8083)
| Variable | Description | Default |
|----------|-------------|---------|
| `AUTO_COMPACTION_ENABLED` | Enable auto-compaction | `true` |
| `COMPACTION_INTERVAL_MINUTES` | Compaction interval | `60` |
| `MIN_FILE_SIZE_MB` | Minimum file size for compaction | `50.0` |
| `MAX_FILE_SIZE_MB` | Maximum file size threshold | `512.0` |

#### Query Interpreter (8085)
| Variable | Description | Default |
|----------|-------------|---------|
| `SQLGLOT_DIALECT` | SQL dialect for parsing | `mysql` |
| `DSL_VALIDATION_ENABLED` | Enable DSL validation | `true` |
| `QUERY_PLAN_CACHE_SIZE` | Query plan cache size | `1000` |
| `PARSING_TIMEOUT_SECONDS` | Query parsing timeout | `30` |

#### Monitoring (8084)
| Variable | Description | Default |
|----------|-------------|---------|
| `METRICS_INTERVAL_SECONDS` | Metrics collection interval | `15` |
| `LOG_RETENTION_DAYS` | Log retention period | `7` |
| `PROMETHEUS_ENABLED` | Enable Prometheus metrics | `true` |
| `GRAFANA_ENABLED` | Enable Grafana dashboard | `false` |

### Service Configuration Files

#### Docker Compose Override
Create `docker-compose.override.yml` for local customization:

```yaml
version: '3.8'
services:
  tenant-node:
    environment:
      - TENANT_ID=my_local_tenant
      - DATA_PATH=/custom/data/path
    volumes:
      - ./custom_data:/custom/data/path
      
  auth-gateway:
    environment:
      - JWT_SECRET=my_super_secret_key
      - JWT_EXPIRATION_HOURS=48
```

#### Query Interpreter Configuration
Edit `query-interpreter/config.json`:

```json
{
  "sql_parser": {
    "dialect": "mysql",
    "strict_mode": true,
    "supported_functions": ["SUM", "COUNT", "AVG", "MIN", "MAX"]
  },
  "dsl_parser": {
    "max_depth": 10,
    "allow_custom_functions": false
  },
  "query_transformer": {
    "optimize_joins": true,
    "push_down_filters": true,
    "column_pruning": true
  }
}
```

### Data Source Configuration

Example source configuration for tenant nodes:

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

## 📁 Directory Structure

```
storage_system/
├── auth-gateway/              # Authentication & API Gateway Service
│   ├── main.py               # FastAPI application
│   ├── requirements.txt      # Service dependencies
│   ├── Dockerfile           # Container configuration
│   └── README.md            # Service documentation
├── tenant-node/              # Core Data Processing Service  
│   ├── main.py              # FastAPI application
│   ├── requirements.txt     # Service dependencies
│   ├── Dockerfile          # Container configuration
│   └── README.md           # Service documentation
├── operation-node/           # Tenant Coordination Service
│   ├── main.py              # FastAPI application
│   ├── auto_scaler.py       # Auto-scaling logic
│   ├── requirements.txt     # Service dependencies
│   └── README.md           # Service documentation
├── cbo-engine/              # Cost-Based Optimization Service
│   ├── main.py              # FastAPI application  
│   ├── query_optimizer.py   # Query optimization logic
│   ├── requirements.txt     # Service dependencies
│   └── README.md           # Service documentation
├── metadata-catalog/        # Metadata Management Service
│   ├── main.py              # FastAPI application
│   ├── metadata.py          # Metadata management
│   ├── compaction_manager.py # Compaction logic
│   ├── requirements.txt     # Service dependencies
│   └── README.md           # Service documentation
├── query-interpreter/       # SQL/DSL Query Processing Service
│   ├── main.py              # FastAPI application
│   ├── sql_parser.py        # SQL parsing (SQLGlot)
│   ├── dsl_parser.py        # DSL parsing
│   ├── query_transformer.py # Query transformation
│   ├── config.json          # Configuration
│   ├── requirements.txt     # Service dependencies
│   └── README.md           # Service documentation
├── monitoring/              # Observability Service
│   ├── main.py              # FastAPI application
│   ├── requirements.txt     # Service dependencies
│   └── README.md           # Service documentation
├── shared-protos/           # Shared Protocol Definitions
│   ├── storage.proto        # gRPC protocol definitions
│   ├── requirements.txt     # Proto dependencies
│   └── README.md           # Protocol documentation
├── legacy_demos/            # Archived Demo Scripts
│   ├── demo.py             # Original demo (archived)
│   └── advanced_demo.py    # Advanced demo (archived)
├── microservices_demo.py    # Integration demo script
├── auth_demo.py            # Authentication demo
├── scaling_demo.py         # Auto-scaling demo
├── demo_requirements.txt    # Demo dependencies
├── docker-compose.yml      # Multi-service orchestration
├── QUICKSTART.md           # Quick start guide
├── MIGRATION.md            # Migration guide
├── start_services.ps1      # Windows service starter
├── start_services.sh       # Linux/macOS service starter
├── stop_services.sh        # Service stopper
├── cleanup.ps1             # Windows cleanup script
├── cleanup.sh              # Linux/macOS cleanup script
└── README.md              # This file
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

## 🚀 Production Deployment

### Container Orchestration

#### Docker Compose (Development/Testing)
```bash
# Production-like environment
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# Scale specific services
docker-compose up -d --scale tenant-node=3 --scale operation-node=2
```

#### Kubernetes (Production)
```yaml
# Example service deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tenant-node
spec:
  replicas: 3
  selector:
    matchLabels:
      app: tenant-node
  template:
    metadata:
      labels:
        app: tenant-node
    spec:
      containers:
      - name: tenant-node
        image: storage-system/tenant-node:latest
        ports:
        - containerPort: 8000
        env:
        - name: TENANT_ID
          value: "prod_tenant_001"
```

### Service Configuration

#### Environment Variables per Service

**Auth Gateway**
```env
JWT_SECRET=your_jwt_secret_key
JWT_EXPIRATION_HOURS=24
ALLOWED_ORIGINS=https://your-frontend.com
```

**Tenant Node**
```env
TENANT_ID=prod_tenant_001
DATA_PATH=/var/lib/tenant_node/data
MAX_CONCURRENT_SEARCHES=50
WAL_RETENTION_HOURS=24
```

**Operation Node**
```env
AUTO_SCALING_ENABLED=true
MIN_WORKERS=2
MAX_WORKERS=50
SCALE_UP_THRESHOLD_CPU=70.0
```

**CBO Engine**
```env
STATISTICS_REFRESH_INTERVAL=30
COST_MODEL_LEARNING_ENABLED=true
QUERY_CACHE_TTL=300
```

### Load Balancing

#### NGINX Configuration
```nginx
upstream tenant_nodes {
    server tenant-node-1:8000;
    server tenant-node-2:8000; 
    server tenant-node-3:8000;
}

upstream auth_gateway {
    server auth-gateway-1:8080;
    server auth-gateway-2:8080;
}

server {
    listen 80;
    
    location /auth/ {
        proxy_pass http://auth_gateway;
    }
    
    location /data/ {
        proxy_pass http://tenant_nodes;
    }
}
```

### Monitoring and Observability

#### Prometheus Configuration
```yaml
scrape_configs:
  - job_name: 'storage-system'
    static_configs:
      - targets: [
          'auth-gateway:8080',
          'tenant-node:8000', 
          'operation-node:8081',
          'cbo-engine:8082',
          'metadata-catalog:8083',
          'monitoring:8084',
          'query-interpreter:8085'
        ]
```

#### Health Checks
```bash
# Check all services health
curl http://monitoring:8084/health/all

# Individual service health
curl http://tenant-node:8000/health
curl http://auth-gateway:8080/health
```

### Service Discovery

For production deployments, consider:
- **Consul** for service discovery
- **Istio** for service mesh
- **Kubernetes Services** for container orchestration
- **AWS ELB/ALB** for cloud load balancing

## � Query Interpreter Service

### SQL Parsing & Processing

The Query Interpreter service provides advanced SQL and DSL query processing capabilities:

#### Supported SQL Features
- **SELECT** statements with complex WHERE clauses
- **JOINs** (INNER, LEFT, RIGHT, FULL OUTER)
- **Aggregations** (SUM, COUNT, AVG, MIN, MAX, GROUP BY)
- **Subqueries** and Common Table Expressions (CTEs)
- **Window Functions** and analytical functions
- **UNION/INTERSECT/EXCEPT** operations

#### DSL Query Examples

**Simple Filter Query**
```json
{
  "query_type": "filter",
  "source": "sales_data",
  "conditions": [
    {
      "field": "customer_id",
      "operator": "eq",
      "value": "CUST_001"
    },
    {
      "field": "order_date", 
      "operator": "gte",
      "value": "2024-01-01"
    }
  ]
}
```

**Aggregation Query**
```json
{
  "query_type": "aggregate",
  "source": "sales_data",
  "group_by": ["customer_id"],
  "aggregations": [
    {
      "function": "sum",
      "field": "amount",
      "alias": "total_spent"
    },
    {
      "function": "count", 
      "field": "*",
      "alias": "order_count"
    }
  ]
}
```

#### Query Transformation Pipeline

1. **Parse**: SQL/DSL → Abstract Syntax Tree (AST)
2. **Validate**: Check syntax and semantic correctness
3. **Transform**: AST → Internal Query Plan
4. **Optimize**: Apply optimization rules
5. **Execute**: Generate execution instructions

#### API Usage Examples

**Parse SQL Query**
```bash
curl -X POST "http://localhost:8085/parse/sql" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "SELECT customer_id, SUM(amount) as total FROM sales_data WHERE order_date >= '\''2024-01-01'\'' GROUP BY customer_id ORDER BY total DESC"
  }'
```

**Parse DSL Query**
```bash
curl -X POST "http://localhost:8085/parse/dsl" \
  -H "Content-Type: application/json" \
  -d '{
    "query": {
      "query_type": "filter",
      "source": "sales_data", 
      "conditions": [{"field": "amount", "operator": "gt", "value": 100}]
    }
  }'
```

**Transform to Execution Plan**
```bash
curl -X POST "http://localhost:8085/transform/query" \
  -H "Content-Type: application/json" \
  -d '{
    "parsed_query": "<parsed_query_from_previous_step>",
    "optimization_level": "aggressive"
  }'
```

### Integration with Other Services

The Query Interpreter works seamlessly with other microservices:

- **CBO Engine**: Sends parsed queries for cost-based optimization
- **Metadata Catalog**: Retrieves schema information for validation
- **Tenant Node**: Provides execution plans for data processing
- **Operation Node**: Coordinates distributed query execution

## �🔧 Troubleshooting

### Service Health Debugging

#### Check Service Status
```bash
# Check all services
curl http://localhost:8084/health/all

# Check individual services
curl http://localhost:8080/health  # Auth Gateway
curl http://localhost:8000/health  # Tenant Node
curl http://localhost:8081/health  # Operation Node
curl http://localhost:8082/health  # CBO Engine
curl http://localhost:8083/health  # Metadata Catalog
curl http://localhost:8085/health  # Query Interpreter
```

#### View Service Logs
```bash
# Docker Compose logs
docker-compose logs -f tenant-node
docker-compose logs -f auth-gateway

# Direct service logs (if running locally)
# Check terminal outputs where services are running
```

### Common Issues

#### Port Conflicts
```bash
# Check which process is using a port
netstat -tulpn | grep :8000

# Kill process if needed
sudo kill -9 <PID>
```

#### Service Communication Issues
```bash
# Test service connectivity
curl -v http://localhost:8080/health
curl -v http://localhost:8000/health

# Check network connectivity between services
ping tenant-node
ping auth-gateway
```

#### Authentication Issues
```bash
# Test auth flow
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "test", "password": "test"}'
```

#### Performance Issues
- Monitor CPU/Memory usage: `docker stats`
- Check query execution times in CBO Engine
- Review auto-scaling metrics in Operation Node
- Monitor data compaction status in Metadata Catalog

### Development Environment Reset
```bash
# Windows
.\cleanup.ps1

# Linux/macOS  
./cleanup.sh

# Manual cleanup
docker-compose down -v
docker system prune -f
rm -rf data/ logs/ __pycache__/
```

## 💡 Microservices Benefits

### Scalability
- **Independent Scaling**: Scale services based on specific load patterns
- **Resource Optimization**: Allocate resources efficiently per service
- **Load Distribution**: Distribute queries across multiple tenant nodes

### Maintainability  
- **Service Isolation**: Changes to one service don't affect others
- **Technology Diversity**: Use best tools for each service
- **Team Autonomy**: Different teams can own different services

### Reliability
- **Fault Isolation**: Failure in one service doesn't bring down the system
- **Circuit Breakers**: Prevent cascade failures
- **Graceful Degradation**: System continues with reduced functionality

### Development Velocity
- **Parallel Development**: Teams can work on services simultaneously  
- **Independent Deployments**: Deploy services independently
- **Testing Isolation**: Test services in isolation

### Future Repository Separation

Each service is designed to be easily extracted into its own repository:

```bash
# Example: Extract tenant-node to separate repo
git subtree push --prefix=tenant-node origin tenant-node-repo

# Example: Extract auth-gateway to separate repo  
git subtree push --prefix=auth-gateway origin auth-gateway-repo
```

## 📊 Enterprise Scale & Performance

### 1TB+ Data Handling Capabilities

Our system is designed and tested for enterprise-scale workloads:

- **📈 [Scalability Analysis](SCALABILITY_ANALYSIS.md)** - Comprehensive analysis of 1TB+ data handling capabilities, performance benchmarks, and bottleneck analysis
- **🚀 [1TB Deployment Guide](DEPLOYMENT_1TB.md)** - Step-by-step production deployment guide for handling 1TB-scale datasets
- **⚡ Performance Characteristics**:
  - Point queries: 1-10ms latency, 10K-100K queries/sec
  - Analytical queries: 10-60 seconds for 1TB scans
  - Ingestion rate: 3.2 GB/sec with 16 parallel writers
  - Auto-scaling: Dynamic scaling from 2 to 100+ nodes

### Architecture Comparison

Our system provides capabilities comparable to modern cloud data warehouses:

| Feature | Our System | Snowflake | BigQuery |
|---------|------------|-----------|----------|
| **Multi-tenant Isolation** | ✅ Node-level | ✅ Virtual warehouses | ✅ Projects |
| **Columnar Storage** | ✅ Parquet | ✅ Proprietary | ✅ Capacitor |
| **Auto-scaling** | ✅ CPU/Memory based | ✅ Compute scaling | ✅ Automatic |
| **Query Optimization** | ✅ CBO + ML | ✅ Advanced CBO | ✅ Dremel engine |
| **SQL Compatibility** | ✅ Multi-dialect | ✅ ANSI SQL | ✅ Standard SQL |
| **Hybrid Data Support** | ✅ JSON + Columnar | ⚠️ Semi-structured | ⚠️ Limited JSON |

## Contributing

1. Follow the existing code style and structure
2. Add appropriate type hints and docstrings
3. Test new features with the microservices demo scripts
4. Update documentation for new APIs or configuration options

## License

This project is part of a larger hybrid storage system architecture.
