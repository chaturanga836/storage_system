# Ingestion Server Reference

## Quick Reference

### Key Files
- `main.go` - Server startup and initialization
- `server.go` - gRPC server implementation (likely in internal/services/ingestion/)
- `handler.go` - Request handlers (likely in internal/services/ingestion/)
- `config.yaml` - Configuration file

### gRPC Service Definition
```protobuf
service IngestionService {
  rpc IngestRecord(IngestRequest) returns (IngestResponse);
  rpc IngestBatch(IngestBatchRequest) returns (IngestBatchResponse);
  rpc ValidateSchema(ValidateSchemaRequest) returns (ValidateSchemaResponse);
  rpc GetIngestionStats(StatsRequest) returns (StatsResponse);
}
```

### Configuration
```yaml
server:
  port: 8080
  max_connections: 1000
  timeout: 30s
  
ingestion:
  batch_size: 1000
  flush_interval: 5s
  validation_enabled: true
  
wal:
  directory: "/data/wal"
  sync_policy: "immediate"
  
memtable:
  max_size: "64MB"
  flush_threshold: 0.8
```

### Common Operations
```bash
# Start server
go run cmd/ingestion-server/main.go

# With custom config
go run cmd/ingestion-server/main.go -config=/path/to/config.yaml

# Docker
docker run -p 8080:8080 storage-engine:latest ingestion-server

# Health check
curl http://localhost:8080/health
```

### API Examples
```go
// Single record ingestion
request := &pb.IngestRequest{
    TenantId: "tenant-123",
    TableName: "users",
    Record: map[string]*pb.Value{
        "id": {Value: &pb.Value_StringValue{StringValue: "user-456"}},
        "email": {Value: &pb.Value_StringValue{StringValue: "user@example.com"}},
    },
}

// Batch ingestion
batchRequest := &pb.IngestBatchRequest{
    TenantId: "tenant-123",
    TableName: "users",
    Records: []*pb.Record{record1, record2, record3},
}
```

### Error Handling
- `InvalidSchema` - Record doesn't match table schema
- `ValidationFailed` - Data validation failed
- `TenantNotFound` - Tenant doesn't exist
- `TableNotFound` - Table doesn't exist
- `InternalError` - Server internal error

### Testing
```bash
# Unit tests
go test ./cmd/ingestion-server/...

# Integration tests
go test ./tests/integration/ingestion/...

# Load tests
go test ./tests/performance/ingestion/...
```

### Monitoring
- **Port**: 8080
- **Health**: `/health`
- **Metrics**: `/metrics`
- **Ready**: `/ready`

### Key Metrics
- `ingestion_requests_total` - Total ingestion requests
- `ingestion_records_total` - Total records ingested
- `ingestion_errors_total` - Total ingestion errors
- `ingestion_duration_seconds` - Request processing time
- `ingestion_batch_size` - Current batch sizes
- `memtable_size_bytes` - Current memtable size

### Admin Commands
```bash
# Check server status
storage-admin status --service ingestion-server

# View ingestion stats
storage-admin metrics --service ingestion-server

# Force memtable flush
storage-admin flush --service ingestion-server
```

### Dependencies
- `internal/wal` - Write-ahead logging
- `internal/storage` - Storage management
- `internal/catalog` - Schema validation
- `internal/auth` - Authentication
- `internal/config` - Configuration management
