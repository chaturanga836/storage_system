# Query Server

## Overview

The Query Server is a high-performance gRPC service responsible for handling data queries in the distributed storage system. It provides efficient access to stored data with support for complex filtering, projections, and time-range queries.

## Architecture

### Core Components

- **gRPC Server**: Exposes query APIs via Protocol Buffers
- **Query Service**: Core business logic for query processing
- **Query Handler**: gRPC request/response handling
- **Connection Management**: Manages client connections and timeouts

### Configuration

```yaml
query:
  port: 8081
  max_connections: 1000
  query_timeout: 30s
  max_query_complexity: 1000
  cache_size: 1GB
```

## Features

### Query Capabilities

- **Record Retrieval**: Get records by ID or key
- **Complex Filtering**: Support for multiple filter conditions
- **Projections**: Select specific fields to reduce bandwidth
- **Time-Range Queries**: Query data within specific time windows
- **Ordering**: Sort results by multiple fields
- **Pagination**: Limit and offset support for large result sets

### Performance Features

- **Query Optimization**: Automatic query plan optimization
- **Index Utilization**: Leverages primary and secondary indexes
- **Result Caching**: Intelligent caching of frequently accessed data
- **Connection Pooling**: Efficient client connection management

## API Reference

### gRPC Service Definition

```protobuf
service QueryService {
  rpc GetRecord(GetRecordRequest) returns (GetRecordResponse);
  rpc QueryRecords(QueryRequest) returns (QueryResponse);
  rpc StreamQuery(QueryRequest) returns (stream QueryStreamResponse);
  rpc GetSchema(GetSchemaRequest) returns (GetSchemaResponse);
}
```

### Main Endpoints

#### GetRecord
Retrieve a single record by its identifier.

**Request Parameters:**
- `tenant_id`: Tenant identifier
- `record_id`: Unique record identifier
- `projection`: Optional list of fields to return
- `version`: Optional version for MVCC reads

#### QueryRecords
Execute complex queries with filtering and sorting.

**Request Parameters:**
- `tenant_id`: Tenant identifier
- `filters`: Array of filter conditions
- `projection`: Fields to include in results
- `order_by`: Sorting specifications
- `limit`/`offset`: Pagination parameters
- `time_range`: Optional time window filter

#### StreamQuery
Execute queries with streaming response for large result sets.

**Features:**
- Server-side streaming
- Automatic batching
- Back-pressure handling
- Early termination support

## Configuration Options

### Server Settings

- **Port**: gRPC server listening port
- **Max Connections**: Maximum concurrent client connections
- **Query Timeout**: Maximum query execution time
- **TLS**: Optional TLS/SSL configuration

### Performance Tuning

- **Cache Size**: Query result cache size
- **Index Cache**: Index data cache configuration
- **Memory Limits**: Maximum memory usage per query
- **Concurrent Queries**: Maximum parallel query execution

### Monitoring

- **Metrics Endpoint**: Prometheus-compatible metrics
- **Health Checks**: Kubernetes health check endpoints
- **Logging**: Structured logging configuration

## Running the Service

### Local Development

```bash
# Build
go build -o query-server ./cmd/query-server

# Run with default config
./query-server

# Run with custom config
./query-server -config=/path/to/config.yaml
```

### Docker

```bash
# Build image
docker build -f deployments/docker/Dockerfile.query-server -t query-server .

# Run container
docker run -p 8081:8081 query-server
```

### Kubernetes

```bash
# Deploy service
kubectl apply -f deployments/kubernetes/query-server.yaml
```

## Testing

### Unit Tests

```bash
go test ./internal/services/query/...
```

### Integration Tests

```bash
go test ./tests/integration/query/...
```

### Performance Tests

```bash
go test ./tests/performance/query/...
```

## Troubleshooting

### Common Issues

1. **Connection Refused**: Check if the service is running on the correct port
2. **Query Timeout**: Increase query timeout or optimize query performance
3. **Memory Issues**: Tune cache sizes and memory limits
4. **High Latency**: Check index usage and query optimization

### Debug Mode

```bash
./query-server -debug -log-level=debug
```

### Metrics and Monitoring

- **Prometheus metrics**: Available at `/metrics` endpoint
- **Health check**: Available at `/health` endpoint
- **Query stats**: Available via admin API

## Performance Guidelines

### Query Optimization

- Use appropriate indexes for filter conditions
- Limit projection to required fields only
- Use pagination for large result sets
- Consider time-range filters for historical data

### Resource Management

- Monitor memory usage and adjust cache sizes
- Use streaming queries for large datasets
- Implement proper connection pooling
- Set appropriate query timeouts

### Scaling

- Deploy multiple instances behind a load balancer
- Use read replicas for query workloads
- Implement proper caching strategies
- Monitor and scale based on query volume

## Security

### Authentication

- Multi-tenant support with tenant isolation
- JWT token validation
- API key authentication support

### Authorization

- Role-based access control (RBAC)
- Resource-level permissions
- Query-level access control

### Data Protection

- TLS encryption for all communications
- Data encryption at rest
- Audit logging for all operations
