# Query Server Reference

## Quick Reference

### Key Files
- `main.go` - Server startup and initialization
- `server.go` - gRPC server implementation (likely in internal/services/query/)
- `handler.go` - Query handlers (likely in internal/services/query/)
- `config.yaml` - Configuration file

### gRPC Service Definition
```protobuf
service QueryService {
  rpc GetRecord(GetRecordRequest) returns (GetRecordResponse);
  rpc QueryRecords(QueryRequest) returns (QueryResponse);
  rpc StreamQuery(QueryRequest) returns (stream QueryStreamResponse);
  rpc GetSchema(GetSchemaRequest) returns (GetSchemaResponse);
  rpc GetTableStats(TableStatsRequest) returns (TableStatsResponse);
}
```

### Configuration
```yaml
server:
  port: 8081
  max_connections: 1000
  timeout: 30s
  
query:
  max_query_complexity: 1000
  cache_size: 1GB
  result_limit: 10000
  streaming_batch_size: 1000
  
indexes:
  cache_size: 512MB
  rebuild_threshold: 0.7
```

### Common Operations
```bash
# Start server
go run cmd/query-server/main.go

# With custom config
go run cmd/query-server/main.go -config=/path/to/config.yaml

# Docker
docker run -p 8081:8081 storage-engine:latest query-server

# Health check
curl http://localhost:8081/health
```

### API Examples
```go
// Get single record
request := &pb.GetRecordRequest{
    TenantId: "tenant-123",
    RecordId: "user-456",
    Projection: []string{"id", "email", "name"},
}

// Query with filters
queryRequest := &pb.QueryRequest{
    TenantId: "tenant-123",
    TableName: "users",
    Filters: []*pb.Filter{
        {
            Field: "status",
            Operator: pb.FilterOperator_EQUALS,
            Value: &pb.Value{Value: &pb.Value_StringValue{StringValue: "active"}},
        },
    },
    OrderBy: []*pb.OrderBy{
        {Field: "created_at", Direction: pb.SortDirection_DESC},
    },
    Limit: 100,
    Offset: 0,
}

// Streaming query
stream, err := client.StreamQuery(ctx, queryRequest)
for {
    response, err := stream.Recv()
    if err == io.EOF {
        break
    }
    // Process response
}
```

### Query Operators
- `EQUALS`, `NOT_EQUALS`
- `GREATER_THAN`, `LESS_THAN`
- `GREATER_THAN_OR_EQUAL`, `LESS_THAN_OR_EQUAL`
- `IN`, `NOT_IN`
- `LIKE`, `NOT_LIKE`
- `IS_NULL`, `IS_NOT_NULL`

### Error Handling
- `RecordNotFound` - Record doesn't exist
- `TableNotFound` - Table doesn't exist
- `InvalidQuery` - Query syntax or logic error
- `QueryTimeout` - Query exceeded timeout
- `InsufficientPermissions` - Access denied

### Testing
```bash
# Unit tests
go test ./cmd/query-server/...

# Integration tests
go test ./tests/integration/query/...

# Performance tests
go test ./tests/performance/query/...
```

### Monitoring
- **Port**: 8081
- **Health**: `/health`
- **Metrics**: `/metrics`
- **Ready**: `/ready`

### Key Metrics
- `query_requests_total` - Total query requests
- `query_duration_seconds` - Query execution time
- `query_cache_hit_ratio` - Cache hit percentage
- `query_result_size_bytes` - Size of query results
- `query_concurrent_count` - Concurrent queries
- `index_usage_total` - Index utilization

### Performance Tips
1. Use appropriate indexes for filter conditions
2. Limit projection to required fields only
3. Use pagination for large result sets
4. Consider time-range filters for historical data
5. Use streaming for large datasets

### Admin Commands
```bash
# Check query server status
storage-admin status --service query-server

# View query performance
storage-admin metrics --service query-server

# Check index status
storage-admin index status
```

### Dependencies
- `internal/storage` - Data storage access
- `internal/catalog` - Schema and metadata
- `internal/auth` - Authentication
- `internal/config` - Configuration management
- `internal/storage/index` - Index management
