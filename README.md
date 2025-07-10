# Go Storage Engine

A high-performance, distributed storage system designed to handle massive data ingestion, processing, and querying operations.

## Architecture Overview

- **Go = Core Storage Engine**: High-performance, low-level storage operations
- **Python = Intelligent Orchestration**: ML-driven transformations and orchestration layer
- **Communication**: gRPC for inter-service communication
- **Storage Format**: Parquet for efficient columnar storage
- **Durability**: Write-Ahead Log (WAL) for data consistency

## Services

### 🚀 Main Services

1. **Ingestion Server** (`cmd/ingestion-server/`) - Write Service for initial data reception
2. **Query Server** (`cmd/query-server/`) - Read Service for data retrieval
3. **Data Processor** (`cmd/data-processor/`) - Background Jobs / WAL/File/Metadata Manager
4. **Admin CLI** (`cmd/admin-cli/`) - Command-line interface for admin tasks

### 🏗️ Core Components

- **WAL Manager** (`internal/wal/`) - Write-Ahead Log implementation
- **Storage Engine** (`internal/storage/`) - Block storage, memtables, Parquet, indexing
- **Metadata & Catalog** (`internal/catalog/`) - Schema and file metadata management
- **MVCC** (`internal/storage/mvcc/`) - Multi-Version Concurrency Control
- **Authentication** (`internal/auth/`) - Multi-tenant security

## Quick Start

### Prerequisites

- Go 1.21+
- Protocol Buffers compiler (`protoc`)

### Setup

```bash
# Clone and setup
git clone <repository>
cd storage-system

# Generate protocol buffers
./scripts/generate_proto.sh

# Build all services
./scripts/build.sh

# Run locally
./scripts/run_local.sh
```

### Development

```bash
# Run tests
go test ./...

# Integration tests
go test ./tests/integration/...

# Performance tests
go test ./tests/performance/...
```

## Architecture Principles

### Performance
- Zero-copy operations where possible
- Efficient memory management
- Optimized data structures (Skip Lists, B-Trees)
- Parallel processing capabilities

### Scalability
- Horizontal scaling support
- Independent service scaling
- Sharding and partitioning
- Load balancing

### Reliability
- Write-Ahead Log for durability
- Atomic operations
- Graceful degradation
- Health monitoring

### Flexibility
- Dynamic schema support
- Pluggable storage backends
- Configurable indexing strategies
- Multi-tenant architecture

## Performance Targets

- **Ingestion**: 1M+ records/second
- **Query Latency**: <10ms for indexed queries
- **Storage Efficiency**: 80%+ compression ratio
- **Availability**: 99.9% uptime

## Documentation

See [PROJECT_STRUCTURE.md](PROJECT_STRUCTURE.md) for detailed architecture and implementation guidance.

## License

[Your License Here]
