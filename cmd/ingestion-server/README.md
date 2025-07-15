# Ingestion Server

## Overview
The Ingestion Server is responsible for receiving and processing incoming data into the storage system. It provides HTTP endpoints for data ingestion and handles real-time data streaming.

## Architecture
- **HTTP Server**: Handles REST API requests for data ingestion
- **Stream Processing**: Processes real-time data streams
- **Validation**: Validates incoming data against schemas
- **Batching**: Groups records for efficient storage
- **WAL Integration**: Writes to Write-Ahead Log for durability

## Key Components

### HTTP Handlers
- `POST /ingest` - Single record ingestion
- `POST /ingest/batch` - Batch record ingestion
- `POST /ingest/stream` - Stream data ingestion
- `GET /health` - Health check endpoint

### Data Processing Pipeline
1. **Input Validation** - Schema validation and data sanitization
2. **Transformation** - Data format conversion and enrichment
3. **Batching** - Efficient grouping of records
4. **Storage** - Persistence to storage layer via WAL

## Configuration
```yaml
server:
  port: 8081
  read_timeout: 30s
  write_timeout: 30s

ingestion:
  batch_size: 1000
  batch_timeout: 5s
  max_concurrent_batches: 10

storage:
  wal_enabled: true
  compression: snappy
```

## API Reference

### Ingest Single Record
```http
POST /ingest
Content-Type: application/json

{
  "table": "users",
  "record": {
    "id": "user123",
    "name": "John Doe",
    "email": "john@example.com"
  }
}
```

### Ingest Batch Records
```http
POST /ingest/batch
Content-Type: application/json

{
  "table": "users",
  "records": [
    {"id": "user1", "name": "Alice"},
    {"id": "user2", "name": "Bob"}
  ]
}
```

## Monitoring
- **Metrics**: Ingestion rate, latency, error rate
- **Logging**: Structured logging with correlation IDs
- **Health Checks**: Database connectivity, WAL status

## Performance
- **Throughput**: Up to 100K records/second
- **Latency**: <10ms P99 for single records
- **Batch Size**: Optimal at 1000 records per batch

**Gatekeeper Function**: This is the first point where security is enforced.

Uses the `internal/auth` module to:
- **Authenticate the incoming client** (e.g., validate API keys, JWTs)
- **Authorize the client's action** (e.g., ensuring they have permission to write data for the specified TenantID and EndpointID)
- **Extract the TenantID** from the authenticated client's credentials, ensuring requests are correctly attributed and isolated

### 3. Write-Ahead Log (WAL) Appending

**Core Durability**: This is the most critical step for crash-safety.

For every incoming DataRecord (or an atomic batch of records), the server interacts with the `internal/wal/manager.go` to:

1. **Create a WAL entry** representing the "write record" operation. This entry includes:
   - TenantID
   - EndpointID
   - Unique EventID
   - Full data payload
   - Any versioning information

2. **Append the entry to the WAL file**

3. **Crucially, fsync the WAL file to disk**. This ensures the data is physically written to non-volatile storage before the server acknowledges success to the Python client. This is your guarantee against data loss in case of a crash.

### 4. In-Memory Memtable/Buffer Management

**Hot Data Storage**: After successfully writing to the WAL, the DataRecord is also added to an in-memory data structure (a "memtable" or buffer, managed by `internal/storage/memtable`).

**Batching for Efficiency**: Memtables accumulate records for a specific (TenantID, EndpointID) until they reach a certain size or a time limit expires.

**Immutable Flush Trigger**: When a memtable is "full" or "stale," it's marked as immutable and queued for flushing to persistent Parquet storage by the Data Processor.

### 5. Acknowledge Success to Client

Only after the data has been successfully written to the WAL and added to the memtable does the Ingestion Server send a success response back to the Python client via gRPC. This provides the client with a strong guarantee that their data has been durably received.

### 6. Internal Event Emission (Optional, for decoupled processing)

For advanced scaling and decoupling, the Ingestion Server could emit internal "new data available" events (e.g., to a Kafka topic or an internal Go channel) after WAL write and memtable addition. The Data Processor would then consume these events to know which memtables/WAL segments to process. This makes the ingestion path truly non-blocking from the processing path.

## Why These Responsibilities for Ingestion Server?

### Speed
By primarily focusing on WAL writes and in-memory additions, the Ingestion Server can process incoming data streams at very high velocity, as fsync on sequential appends is relatively fast.

### Durability Guarantee
The WAL is the backbone of crash recovery. The Ingestion Server is the guardian of this guarantee.

### Decoupling
It decouples the fast ingestion path from the slower, more resource-intensive Parquet writing and indexing processes, which are handled asynchronously by the Data Processor.

### Statelessness (Mostly)
The Ingestion Server itself doesn't need to maintain a complex persistent state beyond the WAL itself. Its memtables are transient and are eventually handled by the Data Processor. This makes it easier to scale horizontally.

## Performance Characteristics

The Ingestion Server is designed to be the highly available, high-throughput funnel that captures all incoming data with strict durability guarantees.

### Target Performance
- **Throughput**: 1M+ records/second
- **Latency**: <1ms for WAL write + memtable insert
- **Durability**: Zero data loss on crash (via WAL)
- **Scalability**: Horizontal scaling support

### Key Optimizations
- Batch processing for efficiency
- Asynchronous memtable flushing
- Optimized WAL sequential writes
- Connection pooling and reuse
- Memory-efficient data structures

## Technology Stack

- **gRPC Server**: High-performance streaming support
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
