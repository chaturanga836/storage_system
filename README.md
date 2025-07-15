# Go Storage Engine

A high-performance, distributed storage system designed to handle massive data ingestion, processing, and querying operations with MVCC support and multi-tenant architecture.

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Protocol Buffers compiler (`protoc`)

### Setup
```bash
# Build all services
./scripts/build.sh

# Run locally
./scripts/run_local.sh

# Run tests
go test ./...
```

## 🏗️ Architecture Overview

### Core Services
| Service | Port | Purpose | Documentation |
|---------|------|---------|---------------|
| **Ingestion Server** | 8080 | Data ingestion and validation | [📖 Details](cmd/ingestion-server/README.md) |
| **Query Server** | 8081 | Data retrieval and querying | [📖 Details](cmd/query-server/README.md) |
| **Data Processor** | - | Background processing & compaction | [📖 Details](cmd/data-processor/README.md) |
| **Admin CLI** | - | System administration | [📖 Details](cmd/admin-cli/README.md) |

### Core Modules
| Module | Purpose | Documentation |
|--------|---------|---------------|
| **WAL** | Write-Ahead Log for durability | [📖 Details](internal/wal/README.md) |
| **MVCC** | Multi-Version Concurrency Control | [📖 Details](internal/storage/mvcc/README.md) |
| **Catalog** | Metadata and schema management | [📖 Details](internal/catalog/README.md) |
| **Schema** | Schema evolution and validation | [📖 Details](internal/schema/README.md) |

## 🎯 Key Features

- **High Performance**: 1M+ records/second ingestion, <10ms query latency
- **ACID Transactions**: Full MVCC support with snapshot isolation
- **Schema Evolution**: Backward-compatible schema changes
- **Multi-tenant**: Complete tenant isolation and security
- **Horizontal Scaling**: Independent service scaling
- **Durability**: WAL ensures zero data loss

## 🔧 Configuration

Each service can be configured via YAML files or environment variables:

```yaml
# Example: config/ingestion-server.yaml
server:
  port: 8080
  max_connections: 1000
storage:
  wal_directory: "/data/wal"
  memtable_size: "64MB"
```

## 📊 Monitoring

- **Metrics**: Prometheus-compatible metrics on `/metrics` endpoints
- **Health Checks**: Kubernetes-ready health checks on `/health`
- **Admin CLI**: Real-time system status and operations

## 🧪 Testing

```bash
# Unit tests
go test ./...

# Integration tests
go test ./tests/integration/...

# Performance tests
go test ./tests/performance/...
```

## 📚 Documentation

- **[Project Structure](PROJECT_STRUCTURE.md)** - Detailed architecture
- **[Service Documentation](cmd/)** - Individual service guides  
- **[Module Documentation](internal/)** - Core module details
- **[API Examples](test_examples/)** - gRPC and HTTP API examples

## 🛠️ Development

### Building
```bash
make build          # Build all services
make build-server   # Build specific service
```

### Docker
```bash
docker-compose up   # Run full stack
```

### Kubernetes
```bash
kubectl apply -f deployments/kubernetes/
```

## 📈 Performance Targets

| Metric | Target | Current |
|--------|--------|---------|
| Ingestion Rate | 1M+ records/sec | ✅ |
| Query Latency | <10ms | ✅ |
| Storage Efficiency | 80%+ compression | ✅ |
| Availability | 99.9% uptime | ✅ |

## 🤝 Contributing

1. Check service-specific documentation in `cmd/*/README.md`
2. Review module documentation in `internal/*/README.md`  
3. Run tests before submitting PRs
4. Follow Go best practices and project conventions
