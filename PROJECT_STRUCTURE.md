# Go Storage Engine - Project Structure

## Overview

This document outlines the complete project structure for the **Go Core Storage Engine** - a high-performance, distributed storage system designed to handle massive data ingestion, processing, and querying operations.

## Architecture Philosophy

- **Go = Core Storage Engine**: High-performance, low-level storage operations
- **Python = Intelligent Orchestration**: ML-driven transformations and orchestration layer
- **Communication**: gRPC for inter-service communication
- **Storage Format**: Parquet for efficient columnar storage
- **Durability**: Write-Ahead Log (WAL) for data consistency

## Project Structure

```
your-project-root/
├── .github/
│   └── workflows/
│       └── ci.yaml
│
├── cmd/
│   ├── ingestion-server/          # The deployable Write Service (initial data reception)
│   │   └── main.go
│   ├── query-server/              # The deployable Read Service
│   │   └── main.go
│   ├── data-processor/            # The deployable Background Jobs / WAL/File/Metadata Manager
│   │   └── main.go
│   └── admin-cli/                 # (Optional) Command-line interface for admin tasks
│       └── main.go
│
├── internal/                      # Private application and library code
│   ├── auth/                      # Authentication and authorization logic
│   │   └── authenticator.go       # Interface and implementation for auth checks
│   │   └── token.go               # Token handling (JWT, API keys)
│   ├── config/                    # Configuration loading for all services
│   │   └── config.go
│   │   └── types.go               # Config structs
│   ├── wal/                       # Write-Ahead Log implementation (Your "WAL Manager" module)
│   │   ├── manager.go             # Core WAL interface, append, replay, checkpoint management
│   │   ├── segment.go             # Handles individual WAL segments
│   │   ├── reader.go              # WAL reader for recovery/replay
│   │   └── types.go               # WAL entry types
│   ├── storage/                   # Core data storage management
│   │   ├── block/                 # Low-level disk/cloud block interactions (underlying for Parquet)
│   │   │   ├── local_fs.go        # Local file system specific operations
│   │   │   ├── s3_fs.go           # S3 client wrapper for remote storage
│   │   │   └── interface.go       # Generic block storage interface (e.g., Reader, Writer, Stat, Delete)
│   │   ├── memtable/              # In-memory data structures before flushing to disk
│   │   │   └── memtable.go
│   │   │   └── skiplist.go        # (Example: if using skiplist for ordered memtable)
│   │   ├── parquet/               # Parquet specific read/write logic
│   │   │   ├── writer.go          # Handles creating and writing to Parquet files (integrates with schema)
│   │   │   ├── reader.go          # Handles reading from Parquet files (integrates with schema)
│   │   │   └── types.go           # Common types for Parquet interaction (e.g., RowGroup metadata)
│   │   ├── index/                 # In-memory and persistent indexing logic
│   │   │   ├── primary_index.go   # Primary key index (TenantID, EntityID -> Logical Version/Location)
│   │   │   ├── secondary_index.go # Example: for queryable secondary indexes
│   │   │   └── serializer.go      # For persisting index data to disk
│   │   ├── compaction/            # Logic for background compaction and merge-sort processes (part of "Background Jobs")
│   │   │   └── compactor.go       # Orchestrates compaction runs
│   │   │   └── strategy.go        # Defines different compaction policies
│   │   ├── mvcc/                  # Multi-Version Concurrency Control (how versions are managed)
│   │   │   └── resolver.go        # Logic to resolve latest version across memtables/files
│   │   │   └── version.go         # Versioning primitives
│   │   └── types.go               # Core data types for storage layer (Record, Location, Version)
│   ├── catalog/                   # Metadata & Catalog (Your "Metadata & Catalog" layer)
│   │   ├── catalog.go             # Interface and core logic for catalog operations
│   │   ├── persistence.go         # Persistence layer for catalog data (e.g., BadgerDB, or gRPC client to external catalog service)
│   │   ├── models.go              # Structs for catalog entries (e.g., FileMetadata, SchemaVersionEntry, ColumnStats)
│   │   └── stats.go               # Logic for managing column statistics
│   ├── schema/                    # Management of tenant-defined data schemas
│   │   ├── registry.go            # Loads and manages schemas from a source
│   │   ├── parser.go              # Parses schema definitions (e.g., JSON/YAML)
│   │   ├── translator.go          # Translates logical schema to physical Parquet schema
│   │   └── types.go               # Internal representation of schema objects
│   ├── services/                  # Business logic orchestrators for deployable services
│   │   ├── ingestion/             # Orchestrates WAL write, memtable, queue for flush
│   │   │   └── service.go
│   │   ├── data_processing/       # Orchestrates WAL replay, memtable flushing, compaction, indexing
│   │   │   └── service.go
│   │   ├── query/                 # Orchestrates index lookup, Parquet read, result aggregation, MVCC resolution
│   │   │   └── service.go
│   │   └── storage_manager.go     # A higher-level manager that composes core storage operations
│   ├── api/                       # gRPC service definitions and internal implementation handlers
│   │   ├── ingestion/
│   │   │   └── handler.go
│   │   ├── query/
│   │   │   └── handler.go
│   │   └── client/                # Internal Go gRPC client if services call each other
│   │       └── client.go
│   ├── messaging/                 # (Optional) Internal message queue integration (e.g., NATS, Kafka, or simple Go channel abstraction)
│   │   └── publisher.go
│   │   └── consumer.go
│   ├── common/                    # General utilities, shared error types, common data types
│   │   └── errors.go
│   │   └── utils.go
│   │   └── identifiers.go         # TenantID, EndpointID, etc.
│
├── pkg/                           # Publicly reusable libraries (if any, e.g., a generic Parquet util)
│
├── proto/                         # Protocol Buffer definitions (shared with Python clients)
│   └── storage/
│       ├── ingestion.proto
│       ├── query.proto
│       └── common.proto
│
├── deployments/
│   ├── docker/
│   ├── kubernetes/
│
├── scripts/
│   ├── generate_proto.sh
│   ├── build.sh
│   └── run_local.sh
│
├── tests/
│   └── integration/
│   └── performance/
│
├── vendor/
├── go.mod
├── go.sum
└── README.md
```

## Component Details

### 🚀 Main Applications (`cmd/`)

#### Ingestion Server (`cmd/ingestion-server/main.go`)

**Description**: This service is the primary entry point for processed, structured data coming from your Python orchestration layer. It's responsible for receiving incoming data batches via gRPC, ensuring their immediate durability, and making them available for eventual persistent storage.

**Key Responsibilities**:
- Receive gRPC requests with processed DataRecords
- Authenticate and authorize incoming requests (via `internal/auth`)
- Write records to the Write-Ahead Log (WAL) (`internal/wal`)
- Add records to in-memory memtables (`internal/storage/memtable`)
- Acknowledge successful receipt to the Python client
- Queue memtables for asynchronous flushing to Parquet

**Technology Stack**:
- gRPC server with streaming support
- High-throughput concurrent processing
- Memory-efficient batching

**Performance Features**:
- WAL writing for durability
- Memtable management with configurable flush policies
- Batch processing optimization
- Concurrent request handling

#### Data Processor (`cmd/data-processor/main.go`)

**Description**: This service is a background worker that performs the heavy lifting of processing data into its final, optimized columnar format and maintaining the storage. It embodies the LSM-tree principles.

**Key Responsibilities**:
- Read from the WAL (for crash recovery replay on startup) and process queued memtables
- Write memtables to the Processed Data Store (Parquet files) (`internal/storage/parquet/writer`)
- Update the primary and secondary indexes (`internal/storage/index`) as data is written
- Run continuous compaction processes (`internal/storage/compaction`) in the background to merge smaller files, consolidate data versions (MVCC), and reclaim space
- Handle schema evolution and re-processing if needed (if raw data is kept in a separate layer and schemas change, this service would re-read and re-process it)

**Technology Stack**:
- Event-driven processing pipeline
- Background task scheduler
- LSM-tree implementation

**Performance Features**:
- Schema application and validation
- Parquet file compaction with configurable strategies
- Index maintenance and optimization
- MVCC (Multi-Version Concurrency Control) support

#### Query Server (`cmd/query-server/main.go`)

**Description**: This service handles all client requests for reading and querying the stored data. It provides an efficient and consistent view of the data.

**Key Responsibilities**:
- Receive gRPC query requests from the Python layer (your query interpreter)
- Authenticate and authorize queries (`internal/auth`)
- Consult the indexes (`internal/storage/index`) to locate relevant data in Parquet files
- Read data from Parquet files (`internal/storage/parquet/reader`), leveraging columnar optimizations (column pruning, predicate pushdown)
- Apply Multi-Version Concurrency Control (MVCC) to ensure only the latest versions of records are returned
- Filter, sort, and aggregate data as requested by the query
- Return structured results via gRPC to the Python client

**Technology Stack**:
- gRPC server with query optimization
- Columnar storage optimizations
- Advanced query planning

**Performance Features**:
- Index-based query planning
- Parquet scanning with predicate pushdown
- Result aggregation and streaming
- Query result caching
- Parallel query execution

#### Admin CLI (`cmd/admin-cli/main.go`)

**Description**: A command-line tool for administrative tasks, monitoring, and debugging. While not a continuously running "server," it's a vital part of managing your system.

**Key Responsibilities**:
- Display system status and metrics
- Manually trigger (or inspect progress of) compaction jobs
- Inspect WAL contents for debugging
- Perform data repair or consistency checks
- Manage schema definitions (e.g., upload new schemas to `internal/schema/registry`)

**Technology Stack**:
- Command-line interface with rich formatting
- Direct access to internal components
- Comprehensive monitoring and debugging tools

**Administrative Features**:
- Database inspection and diagnostics
- Manual compaction triggers
- Performance monitoring and profiling
- Schema management operations
- System health checks

### 🔧 Core Libraries (`internal/`)

#### Authentication (`auth/`)
- Multi-tenant authentication
- JWT token validation
- API key management
- Role-based access control

#### Write-Ahead Log (`wal/`)
- Crash recovery support
- Atomic write operations
- Segment-based file management
- Replay mechanism for recovery

#### Storage Engine (`storage/`)
- **Disk Management**: Low-level file system operations
- **Memtable**: In-memory data structures (Skip List, B-Trees)
- **Parquet Integration**: Efficient columnar storage
- **Indexing**: Primary and secondary index management
- **Compaction**: Background merge and optimization

#### Schema Management (`schema/`)
- Dynamic schema registry
- Schema evolution support
- Data type validation
- Transform rule application

#### API Layer (`api/`)
- gRPC service implementations
- Request/response handling
- Error management
- Connection pooling

#### Business Services (`services/`)
- High-level business logic
- Component orchestration
- Transaction management
- Performance optimization

### 📡 Protocol Buffers (`proto/`)
- **Ingestion API**: Streaming data ingestion
- **Query API**: Complex query operations
- **Common Types**: Shared data structures

### 🚢 Deployment (`deployments/`)
- **Docker**: Multi-stage builds for each service
- **Kubernetes**: Production-ready deployments
- **Helm**: Package management for K8s
- **Terraform**: Infrastructure as Code

### 🧪 Testing (`tests/`)
- **Integration Tests**: End-to-end workflows
- **Performance Tests**: Benchmarking and profiling
- **Unit Tests**: Component-level testing

## Key Design Principles

### 1. **High Performance**
- Zero-copy operations where possible
- Efficient memory management
- Optimized data structures (Skip Lists, B-Trees)
- Parallel processing capabilities

### 2. **Scalability**
- Horizontal scaling support
- Sharding and partitioning
- Load balancing
- Resource isolation

### 3. **Reliability**
- Write-Ahead Log for durability
- Atomic operations
- Graceful degradation
- Health monitoring

### 4. **Flexibility**
- Dynamic schema support
- Pluggable storage backends
- Configurable indexing strategies
- Multi-tenant architecture

## Data Flow

```
Python Services → gRPC → Ingestion Server → WAL → Memtable → Parquet Files
                                                    ↓
Query Server ← gRPC ← Python Services ← Index ← Background Processor
```

## Development Workflow

1. **Proto Generation**: `./scripts/generate_proto.sh`
2. **Build**: `./scripts/build.sh`
3. **Local Testing**: `./scripts/run_local.sh`
4. **Integration Tests**: `go test ./tests/integration/...`
5. **Performance Testing**: `go test ./tests/performance/...`

## Performance Targets

- **Ingestion**: 1M+ records/second
- **Query Latency**: <10ms for indexed queries
- **Storage Efficiency**: 80%+ compression ratio
- **Availability**: 99.9% uptime

## Security Features

- **Authentication**: Multi-tenant JWT validation
- **Authorization**: Role-based access control
- **Encryption**: Data at rest and in transit
- **Audit Logging**: Complete operation tracking

## Key Changes and Rationale for this Go Structure

### `cmd/` for Distinct Services

Explicitly defines `ingestion-server`, `query-server`, and `data-processor` as separate executables. This naturally supports a microservices approach, allowing them to scale independently.

- `main.go` in each `cmd` subdirectory will be minimal, primarily setting up and starting the respective server/process, delegating logic to `internal/`.
- Each service can be deployed, scaled, and maintained independently
- Clear separation of concerns between different system responsibilities

### `internal/` for Core Logic

#### `internal/storage/`
This is the heart of the system. It encapsulates everything about how data is stored, read, indexed, and maintained on disk. This includes:
- `wal/`: Write-Ahead Log for durability
- `parquet/`: Columnar storage implementation
- `index/`: Primary and secondary indexing
- `compaction/`: Background optimization processes
- `memtable/`: In-memory data structures

#### `internal/schema/`
Handles the critical aspect of understanding and applying tenant-defined schemas. This is where:
- Mapping from structured data (received from Python) to specific Parquet columnar types happens
- Schema evolution and validation occurs
- Query planning against dynamic schemas is implemented

#### `internal/api/`
The gRPC handlers live here. These handlers:
- Receive gRPC requests from Python services
- Delegate actual business logic to the `internal/services/` layer
- Cleanly separate RPC concerns from core business logic
- Handle request/response marshaling and error management

#### `internal/services/`
This layer contains the orchestrators of Go business logic. For example:
- `internal/services/ingestion/service.go` coordinates calling:
  - `internal/wal/wal.go` for durability
  - `internal/storage/memtable/memtable.go` for in-memory staging
  - `internal/storage/parquet/writer.go` for persistent storage
- Adheres to a "layered" architecture pattern
- Provides high-level business operations that compose lower-level components

#### `internal/auth/`
Centralized authentication and authorization:
- Multi-tenant security model
- JWT token validation
- Role-based access control
- API key management

#### `internal/common/`
For truly shared, generic utilities or types:
- Common error types and handling
- Shared data structures
- Utility functions used across components

### `proto/` as the Single Source of Truth

All `.proto` files are in `proto/`. This is crucial because:
- Python will also generate client code from these same definitions
- Ensures consistency between Go server and Python client
- Provides a contract-first approach to API design
- `generate_proto.sh` script automates code generation for both languages

### `deployments/` for Production Readiness

Essential for defining how distinct Go services are packaged and deployed:
- **Docker**: Multi-stage builds optimized for each service
- **Kubernetes**: Production-ready deployments with proper resource limits
- **Helm**: Package management for complex deployments
- **Terraform**: Infrastructure as Code for cloud resources

## Architectural Benefits

### 1. **Separation of Concerns**
- Each package has a single, well-defined responsibility
- Clear boundaries between components
- Easier to test and maintain individual components

### 2. **Modularity**
- Components can be developed and tested independently
- Easy to swap implementations (e.g., different storage backends)
- Supports incremental development and deployment

### 3. **Scalability**
- Services can be scaled independently based on load
- Clear interfaces between components
- Supports horizontal scaling patterns

### 4. **Maintainability**
- Clear code organization makes navigation intuitive
- Well-defined interfaces reduce coupling
- Easy to onboard new developers

### 5. **Testability**
- Each layer can be unit tested independently
- Clear dependency injection points
- Integration tests can focus on specific workflows

## Implementation Strategy

### Phase 1: Core Foundation
1. Set up `go.mod` and basic project structure
2. Implement `internal/common/` types and utilities
3. Create basic `proto/` definitions
4. Implement `internal/config/` for configuration management

### Phase 2: Storage Layer
1. Implement `internal/storage/disk/` for file system operations
2. Build `internal/wal/` for write-ahead logging
3. Create `internal/storage/memtable/` for in-memory operations
4. Implement `internal/storage/parquet/` for persistent storage

### Phase 3: Service Layer
1. Build `internal/services/ingestion/` for data ingestion
2. Implement `internal/services/query/` for data retrieval
3. Create `internal/services/processing/` for background tasks
4. Add `internal/auth/` for security

### Phase 4: API Layer
1. Implement `internal/api/` gRPC handlers
2. Create `cmd/` executables for each service
3. Add comprehensive error handling and logging
4. Implement health checks and monitoring

### Phase 5: Production Readiness
1. Add comprehensive testing suite
2. Create deployment configurations
3. Add monitoring and observability
4. Performance optimization and tuning

---

*This structure provides a robust foundation for a high-performance storage engine that can scale to handle enterprise-level data processing requirements. The clear separation of concerns and modular design ensures maintainability and extensibility as the system grows.*
