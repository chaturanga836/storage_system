# Ingestion Server

## Overview
The Ingestion Server receives and processes incoming data into the storage system using a high-performance gRPC API. It is responsible for durable, low-latency ingestion, batching, and WAL integration.

## Architecture
- **gRPC Server**: Handles all ingestion requests via protocol buffers
- **Stream Processing**: Supports real-time and batch ingestion
- **Validation**: Validates incoming data against schemas
- **Batching**: Groups records for efficient storage
- **WAL Integration**: Writes to Write-Ahead Log for durability
- **Health Check**: Minimal HTTP `/health` endpoint for readiness/liveness (not for ingestion)

## Key gRPC Services

The API is defined in [`proto/storage/ingestion.proto`](../../proto/storage/ingestion.proto):

- `IngestRecord(IngestRecordRequest)`: Ingest a single record
- `IngestBatch(IngestBatchRequest)`: Ingest multiple records in a batch
- `IngestStream(stream IngestStreamRequest)`: High-throughput streaming ingestion
- `GetIngestionStatus(IngestionStatusRequest)`: Get ingestion status and metrics
- `HealthCheck(HealthCheckRequest)`: Health check for service readiness

### Example gRPC Usage

You can interact with the server using gRPC clients (e.g., Go, Python, grpcurl):

```sh
grpcurl -plaintext localhost:<port> storage.IngestionService/HealthCheck
```

For full message and field definitions, see the proto file.

### Health Endpoint
- `GET /health` (HTTP): Returns `{"status":"ok"}` for readiness/liveness checks only. All ingestion and data operations use gRPC.

## Configuration
Configuration is loaded from a JSON file and matches the Go struct in [`internal/config/config.go`](../../internal/config/config.go):

```json
{
  "ingestion": {
    "port": 8081,
    "max_connections": 100,
    "batch_size": 1000,
    "flush_interval": "5s"
  },
  "storage": {
    "wal_enabled": true,
    "compression_type": "snappy"
  },
  // ... other config sections ...
}
```

## Data Processing Pipeline
1. **Input Validation** - Schema validation and data sanitization
2. **Transformation** - Data format conversion and enrichment
3. **Batching** - Efficient grouping of records
4. **Storage** - Persistence to storage layer via WAL

## Monitoring & Performance
- **Metrics**: Ingestion rate, latency, error rate
- **Logging**: Structured logging with correlation IDs
- **Health Checks**: gRPC and HTTP endpoints
- **Throughput**: 1M+ records/second (target)
- **Latency**: <1ms for WAL write + memtable insert

## Technology Stack
- **gRPC**: Protocol buffer-based API
- **Write-Ahead Log**: Sequential append-only durability
- **In-Memory Memtables**: Fast data staging
- **Authentication**: JWT/API key validation
- **Monitoring**: Health checks and metrics

## Dependencies
- `internal/auth/`: Authentication and authorization
- `internal/wal/`: Write-Ahead Log management
- `internal/storage/memtable/`: In-memory data structures
- `internal/services/ingestion/`: Business logic orchestration
- `internal/api/ingestion/`: gRPC request handling
- `proto/storage/`: Protocol buffer definitions
