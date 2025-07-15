# Catalog Module Reference

## Quick Reference

### Key Files
- `catalog.go` - Main catalog manager implementation
- `models.go` - Data structures and types
- `persistence.go` - Metadata persistence logic
- `stats.go` - Statistics management
- `types.go` - Type definitions (if exists)

### Core Types
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
}

type TableSchema struct {
    Name         string
    TenantID     string
    Version      int
    Columns      []ColumnSchema
    Partitions   []PartitionSchema
    CreatedAt    time.Time
    ModifiedAt   time.Time
}

type FileMetadata struct {
    ID           string
    Path         string
    TableName    string
    Size         int64
    RowCount     int64
    MinTimestamp time.Time
    MaxTimestamp time.Time
    Status       FileStatus
}
```

### Configuration
```yaml
catalog:
  storage_path: "/data/catalog"
  backup_interval: 1h
  retention_days: 30
  cache_size: 1GB
  sync_interval: 5m
```

### Common Operations
```go
// Initialize catalog
catalog, err := catalog.NewManager(config)

// Register schema
schema := &catalog.TableSchema{
    Name:     "users",
    TenantID: "tenant-123",
    Columns: []catalog.ColumnSchema{
        {Name: "id", Type: TypeString, Nullable: false},
        {Name: "email", Type: TypeString, Nullable: false},
    },
}
err = catalog.RegisterSchema(schema)

// Register file
fileMetadata := &catalog.FileMetadata{
    ID:        "file-001",
    Path:      "/data/users/file-001.parquet",
    TableName: "users",
    Size:      1024000,
    RowCount:  10000,
}
err = catalog.RegisterFile(fileMetadata)

// Query metadata
files, err := catalog.GetFiles("users")
stats, err := catalog.GetTableStats("users")
```

### Error Types
- `ErrSchemaNotFound` - Schema doesn't exist
- `ErrSchemaConflict` - Schema version conflict
- `ErrFileNotFound` - File metadata not found
- `ErrInvalidSchema` - Schema validation failed

### Testing
```bash
# Unit tests
go test ./internal/catalog/...

# Integration tests
go test ./tests/integration/catalog/...

# Performance tests
go test -bench=. ./internal/catalog/...
```

### Monitoring Metrics
- `catalog_schemas_count` - Number of registered schemas
- `catalog_files_count` - Total tracked files
- `catalog_cache_hit_ratio` - Cache hit percentage
- `catalog_query_duration_seconds` - Metadata query latency
- `catalog_storage_size_bytes` - Catalog storage size

### Admin Commands
```bash
# Show catalog stats
storage-admin catalog stats

# Validate consistency
storage-admin catalog validate

# Refresh statistics
storage-admin catalog refresh-stats --table users

# Backup catalog
storage-admin catalog backup --output /backup/catalog.tar.gz
```

### Dependencies
- `internal/schema` - Schema validation and evolution
- `internal/common` - Error types and utilities
- `internal/config` - Configuration management
- File system for metadata persistence
