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

# Protocol Buffers (protoc) Setup and Go Code Generation

## Download and Install protoc
1. Download the latest release of Protocol Buffers from:
   https://github.com/protocolbuffers/protobuf/releases
2. Extract the archive and place the contents in `C:\protoc` (or any preferred location).
3. Add `C:\protoc\bin` to your system PATH for easy access.

## Install Go Plugins
Run these commands in PowerShell:
```
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```
Make sure `$GOPATH/bin` is in your PATH.

## Generate Go Code from Proto Files
Run these commands from your project root:

```
& "C:\protoc\bin\protoc.exe" --go_out=internal/pb --go-grpc_out=internal/pb -I proto -I "C:\protoc\include" proto/storage/common.proto
& "C:\protoc\bin\protoc.exe" --go_out=internal/pb --go-grpc_out=internal/pb -I proto -I "C:\protoc\include" proto/storage/ingestion.proto
```

- This will generate `.pb.go` and `_grpc.pb.go` files in `internal/pb/storage/`.
- Make sure your proto files have the correct `option go_package` set for your desired import path.

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

# Ingestion gRPC API: Method Responsibilities and Request Journey

The ingestion server exposes several gRPC methods for data ingestion, status, and health. Here is a summary of each method's responsibilities and the typical request journey:

## IngestRecord
- **Purpose:** Handle a single record ingestion request.
- **Journey:**
  1. Receives a gRPC request (`IngestRecordRequest`).
  2. Validates the request.
  3. Calls business/service logic to ingest the record.
  4. Maps the result to a protobuf response (`IngestRecordResponse`).
  5. Returns the response to the client.

## IngestBatch
- **Purpose:** Handle batch ingestion of multiple records.
- **Journey:**
  1. Receives a gRPC request (`IngestBatchRequest`) with multiple records.
  2. Validates the request.
  3. Calls business/service logic to ingest all records, possibly transactionally.
  4. Maps the results to a protobuf response (`IngestBatchResponse`).
  5. Returns the response to the client.

## IngestStream
- **Purpose:** Handle high-throughput streaming ingestion.
- **Journey:**
  1. Opens a bidirectional gRPC stream with the client.
  2. Continuously receives messages from the client.
  3. Processes each message using service logic.
  4. Sends acknowledgments, errors, or stats back to the client.
  5. Closes the stream when the client disconnects or an error occurs.

## GetIngestionStatus
- **Purpose:** Provide ingestion status and metrics.
- **Journey:**
  1. Receives a gRPC request (`IngestionStatusRequest`).
  2. Calls service logic to collect current metrics and status.
  3. Maps the metrics/status to a protobuf response (`IngestionStatusResponse`).
  4. Returns the response to the client.

## HealthCheck
- **Purpose:** Report health status of the ingestion service.
- **Journey:**
  1. Receives a gRPC health check request.
  2. Checks service health (dependencies, DB, etc.).
  3. Maps the health status to a protobuf response (`HealthCheckResponse`).
  4. Returns the response to the client.

---

Each method receives a gRPC request, validates and processes it using your business logic, and returns a protobuf response. The journey path is:
Client → gRPC Method → Validation → Service Logic → Response Mapping → Client

---
