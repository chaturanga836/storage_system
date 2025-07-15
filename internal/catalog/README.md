# Catalog Module

## Overview

The Catalog module manages metadata for the storage system, including table schemas, file locations, statistics, and index information. It serves as the central metadata repository that enables query planning, data discovery, and system administration.

## Architecture

### Core Components

- **Schema Registry**: Manages table and column schemas
- **File Catalog**: Tracks file locations and metadata
- **Statistics Manager**: Maintains data statistics for query optimization
- **Index Registry**: Manages index definitions and metadata
- **Partition Manager**: Handles table partitioning information

### Storage Structure

```
catalog/
├── schemas/
│   ├── schema-v1.json
│   ├── schema-v2.json
│   └── latest.json
├── files/
│   ├── tenant-123/
│   │   ├── table-users/
│   │   └── table-orders/
│   └── tenant-456/
├── statistics/
│   ├── table-stats.json
│   └── column-stats.json
└── indexes/
    ├── primary/
    └── secondary/
```

## Features

### Schema Management

- **Schema Evolution**: Support for backward-compatible schema changes
- **Version Control**: Track schema versions and migrations
- **Validation**: Ensure data consistency with schema definitions
- **Multi-tenant**: Isolate schemas by tenant

### File Management

- **File Tracking**: Track all data files and their metadata
- **Location Management**: Manage file locations across storage tiers
- **Size Monitoring**: Track file sizes and growth patterns
- **Compaction Tracking**: Monitor compaction status and history

### Statistics

- **Table Statistics**: Row counts, data sizes, modification times
- **Column Statistics**: Min/max values, null counts, distinct counts
- **Histogram Data**: Value distribution for query optimization
- **Real-time Updates**: Incremental statistics updates

### Index Management

- **Index Definitions**: Store index schemas and configurations
- **Index Statistics**: Track index usage and performance
- **Index Health**: Monitor index consistency and health
- **Auto-indexing**: Suggest indexes based on query patterns

## API Reference

### Catalog Manager Interface

```go
type CatalogManager interface {
    // Schema operations
    RegisterSchema(schema *TableSchema) error
    GetSchema(tableName string, version int) (*TableSchema, error)
    ListSchemas(tenantID string) ([]*TableSchema, error)
    
    // File operations
    RegisterFile(file *FileMetadata) error
    GetFiles(tableName string) ([]*FileMetadata, error)
    UpdateFileStats(fileID string, stats *FileStats) error
    
    // Statistics operations
    UpdateTableStats(tableName string, stats *TableStats) error
    GetTableStats(tableName string) (*TableStats, error)
    UpdateColumnStats(tableName string, columnName string, stats *ColumnStats) error
    
    // Index operations
    RegisterIndex(index *IndexMetadata) error
    GetIndexes(tableName string) ([]*IndexMetadata, error)
    DropIndex(indexName string) error
}
```

### Schema Structure

```go
type TableSchema struct {
    Name         string
    TenantID     string
    Version      int
    Columns      []ColumnSchema
    Partitions   []PartitionSchema
    Indexes      []IndexSchema
    CreatedAt    time.Time
    ModifiedAt   time.Time
}

type ColumnSchema struct {
    Name         string
    Type         DataType
    Nullable     bool
    DefaultValue interface{}
    Constraints  []Constraint
}
```

### File Metadata

```go
type FileMetadata struct {
    ID           string
    Path         string
    TableName    string
    PartitionKey string
    Size         int64
    RowCount     int64
    MinTimestamp time.Time
    MaxTimestamp time.Time
    Status       FileStatus
    CreatedAt    time.Time
}

type FileStatus int

const (
    FileStatusActive FileStatus = iota
    FileStatusCompacting
    FileStatusObsolete
    FileStatusDeleted
)
```

## Configuration

### Catalog Settings

```yaml
catalog:
  storage_path: "/data/catalog"
  backup_interval: 1h
  retention_days: 30
  cache_size: 1GB
  sync_interval: 5m
```

### Performance Tuning

- **Cache Size**: Memory allocated for metadata caching
- **Sync Interval**: How often to persist changes
- **Backup Frequency**: Metadata backup schedule
- **Retention**: How long to keep historical metadata

## Operations

### Basic Usage

```go
// Initialize catalog
catalog, err := NewCatalogManager(config)
if err != nil {
    return err
}

// Register a new table schema
schema := &TableSchema{
    Name:     "users",
    TenantID: "tenant-123",
    Columns: []ColumnSchema{
        {Name: "id", Type: TypeString, Nullable: false},
        {Name: "email", Type: TypeString, Nullable: false},
        {Name: "created_at", Type: TypeTimestamp, Nullable: false},
    },
}
err = catalog.RegisterSchema(schema)

// Register a file
fileMetadata := &FileMetadata{
    ID:        "file-001",
    Path:      "/data/users/2024/01/file-001.parquet",
    TableName: "users",
    Size:      1024000,
    RowCount:  10000,
}
err = catalog.RegisterFile(fileMetadata)

// Query metadata
files, err := catalog.GetFiles("users")
stats, err := catalog.GetTableStats("users")
```

### Schema Evolution

1. **Version Check**: Verify compatibility with existing data
2. **Migration Plan**: Create plan for schema migration
3. **Gradual Rollout**: Apply changes incrementally
4. **Validation**: Verify data consistency after migration

## Testing

### Unit Tests

```bash
go test ./internal/catalog/...
```

### Integration Tests

```bash
go test ./tests/integration/catalog/...
```

### Performance Tests

```bash
go test ./tests/performance/catalog/...
```

## Monitoring

### Key Metrics

- **Schema Count**: Number of registered schemas
- **File Count**: Total tracked files per tenant/table
- **Catalog Size**: Total metadata storage size
- **Query Rate**: Metadata queries per second
- **Cache Hit Rate**: Percentage of cache hits

### Health Checks

- Validate metadata consistency
- Check catalog storage health
- Monitor cache performance
- Verify backup integrity

## Troubleshooting

### Common Issues

1. **Schema Conflicts**: Incompatible schema changes
   - Solution: Use proper schema evolution practices

2. **Missing Files**: Files referenced but not found
   - Solution: Run consistency checks and repair

3. **Stale Statistics**: Outdated statistics affecting performance
   - Solution: Refresh statistics more frequently

4. **Cache Misses**: Poor cache performance
   - Solution: Tune cache size and eviction policies

### Debug Commands

```bash
# Show catalog statistics
storage-admin catalog stats

# Validate catalog consistency
storage-admin catalog validate

# Refresh table statistics
storage-admin catalog refresh-stats --table users
```

## Implementation Files

- `catalog.go`: Main catalog manager implementation
- `models.go`: Data structures and types
- `persistence.go`: Metadata persistence logic
- `stats.go`: Statistics management
- `schema.go`: Schema management and validation
- `cache.go`: Metadata caching layer
