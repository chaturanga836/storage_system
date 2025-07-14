# Query Server - Detailed Documentation

## Overview

**Location**: `cmd/query-server/main.go` (and its core logic in `internal/services/query/service.go`, `internal/api/query/handler.go`)

**Role Type**: External-Facing API for Data Retrieval, focused on efficient query execution, data consistency, and high availability. It directly implements your "Read Service" functional layer.

**Primary Goal**: Provide a fast, consistent, and flexible interface for Python (and potentially other) clients to retrieve and analyze the stored data by executing various types of queries.

## Detailed Responsibilities of the Query Server

### 1. Receive gRPC Query Requests

- Listens for incoming gRPC requests from Python clients (your query interpreter)
- Expects query definitions (e.g., filter predicates, column projections, aggregations, sort orders) as defined in `proto/storage/query.proto`

### 2. Authentication and Authorization

**Access Control**: Uses the `internal/auth` module to:

- **Authenticate the querying client**
- **Authorize the client's permissions** (e.g., ensuring they can query data for the specified TenantID or EndpointID, or if they have permission to access certain sensitive columns)
- **Enforce multi-tenant data isolation** by automatically filtering queries based on the authenticated TenantID, ensuring a client from Tenant A can only see Tenant A's data

### 3. Query Parsing and Validation

- Parses the incoming query request (which might be a high-level logical query)
- Validates the query syntax and ensures the requested columns and filters are valid against the known schemas for the targeted (TenantID, EndpointID)

### 4. Metadata & Catalog Lookup

Interacts with the Metadata & Catalog (`internal/catalog` module) to:

- **Identify all relevant Parquet files** and their locations (paths, S3 URLs) that might contain data matching the query's TenantID, EndpointID, and time range (if applicable)
- **Retrieve schema information and column statistics** (e.g., min/max values for columns within files or row groups), which are crucial for query optimization

### 5. Efficient Parquet File Reading

Uses the `internal/storage/parquet/reader.go` to access the actual Parquet files.

Leverages **columnar query optimizations**:

- **Column Pruning**: Reads only the columns explicitly requested in the query (SELECT column1, column2), skipping unnecessary I/O for other columns
- **Predicate Pushdown**: Uses column statistics from the Metadata & Catalog to skip entire Parquet files or row groups that cannot possibly contain matching data (e.g., if a file's timestamp range is outside the query's filter). This drastically reduces data read from disk

### 6. Multi-Version Concurrency Control (MVCC) Resolution

A **critical responsibility** to ensure read consistency.

The Query Server must return the logically latest version of a record. It does this by interacting with `internal/storage/mvcc/resolver.go`.

This might involve:

- **Checking the active in-memory memtables** (from the Ingestion Server's state, if shared or communicated)
- **Reading records from recently written Parquet files** that haven't been compacted yet
- **Reading records from fully compacted Parquet files**

When multiple versions of the same logical record (EventID/PrimaryKey) are found across these sources, the MVCC resolver determines and returns only the newest one (based on timestamp or version number).

### 7. In-Memory Query Processing (Filtering, Aggregation, Sorting)

After reading the relevant data, the Query Server performs any remaining query operations that couldn't be "pushed down" to the Parquet read level.

This includes:
- Applying more complex filters
- Performing aggregations (SUM, AVG, COUNT, etc.)
- Sorting the final results before returning them

### 8. Streamed Results via gRPC

Returns the query results to the Python client. For large result sets, this will typically use **gRPC server-side streaming** to send results in chunks, avoiding large memory footprints on both server and client and enabling faster initial data display.

## Why These Responsibilities for Query Server?

### 🚀 **Read Performance**
By focusing on columnar reads, predicate pushdown, and MVCC resolution, it's designed for analytical queries over large datasets.

### 🔒 **Consistency**
MVCC ensures that readers see a consistent view of the data, even while writes and compaction are ongoing.

### 🛡️ **Isolation**
Authorization and multi-tenancy enforcement happen at the query boundary, preventing unauthorized access and cross-tenant data leakage.

### 📈 **Scalability**
Designed to be horizontally scalable, allowing you to add more Query Server instances as read load increases.

### 🎯 **Abstracts Storage Complexity**
Hides the underlying Parquet file management, indexing details, and MVCC versioning from the client, providing a simple, high-level query interface.

## Architecture Flow

```
Python Client → gRPC Query Request → Query Server
                                          ↓
                              Authentication & Authorization
                                          ↓
                              Query Parsing & Validation
                                          ↓
                              Metadata & Catalog Lookup
                                          ↓
                              Parquet File Reading (with optimizations)
                                          ↓
                              MVCC Resolution
                                          ↓
                              In-Memory Processing
                                          ↓
                              Streamed gRPC Response → Python Client
```

## Key Components Integration

- **`internal/auth`**: Authentication and authorization
- **`internal/catalog`**: Metadata and file location lookup
- **`internal/storage/parquet/reader`**: Efficient columnar data reading
- **`internal/storage/mvcc/resolver`**: Version consistency management
- **`internal/services/query/service`**: Core business logic orchestration
- **`internal/api/query/handler`**: gRPC request/response handling
- **`proto/storage/query.proto`**: API contract definition

## Performance Optimizations

1. **Column Pruning**: Only read necessary columns
2. **Predicate Pushdown**: Skip irrelevant files/row groups
3. **Index Utilization**: Fast data location lookup
4. **Streaming Results**: Memory-efficient large result handling
5. **Parallel Processing**: Concurrent query execution
6. **Query Caching**: Result caching for repeated queries
