# MVCC Module Reference

## Quick Reference

### Key Files
- `version.go` - Version management and data structures
- `transaction.go` - Transaction lifecycle management (if exists)
- `resolver.go` - Conflict detection and resolution
- `snapshot.go` - Snapshot management (if exists)
- `gc.go` - Garbage collection (if exists)

### Core Types
```go
type VersionManager interface {
    CreateVersion(key string, value []byte, txID TransactionID) (*Version, error)
    GetVersion(key string, snapshot Snapshot) (*Version, error)
    BeginTransaction(isolation IsolationLevel) (*Transaction, error)
    CommitTransaction(txID TransactionID) error
    AbortTransaction(txID TransactionID) error
    CreateSnapshot() (Snapshot, error)
    GarbageCollect() error
}

type Version struct {
    Key            string
    Value          []byte
    TransactionID  TransactionID
    CommitTime     time.Time
    Status         VersionStatus
    PreviousVersion *Version
}

type Transaction struct {
    ID            TransactionID
    StartTime     time.Time
    IsolationLevel IsolationLevel
    Snapshot      Snapshot
    WriteSet      map[string]*Version
    Status        TransactionStatus
}
```

### Isolation Levels
```go
const (
    ReadCommitted IsolationLevel = iota
    SnapshotIsolation  // Default
    Serializable
)
```

### Configuration
```yaml
mvcc:
  max_versions_per_key: 100
  gc_interval: 30m
  version_retention: 24h
  conflict_resolution: "abort_on_conflict"  # abort_on_conflict, last_write_wins
  snapshot_cache_size: 1000
```

### Common Operations
```go
// Begin transaction
tx, err := mvcc.BeginTransaction(SnapshotIsolation)

// Read data (sees snapshot at transaction start)
version, err := mvcc.GetVersion("user:123", tx.Snapshot)

// Write data
newVersion, err := mvcc.CreateVersion("user:123", newData, tx.ID)

// Commit transaction
err = mvcc.CommitTransaction(tx.ID)
if err != nil {
    // Handle conflict or retry
}
```

### Error Types
- `ErrTransactionConflict` - Write conflict detected
- `ErrTransactionAborted` - Transaction was aborted
- `ErrVersionNotFound` - Requested version doesn't exist
- `ErrInvalidSnapshot` - Snapshot is invalid or expired

### Testing
```bash
# Unit tests
go test ./internal/storage/mvcc/...

# Concurrency tests
go test -race ./internal/storage/mvcc/...

# Integration tests
go test ./tests/integration/mvcc/...
```

### Monitoring Metrics
- `mvcc_transactions_total` - Total transactions processed
- `mvcc_conflicts_total` - Total conflicts detected
- `mvcc_versions_count` - Current number of versions
- `mvcc_gc_duration_seconds` - Time spent in garbage collection
- `mvcc_memory_usage_bytes` - Memory used by version chains

### Performance Tips
1. Keep transactions short to reduce conflicts
2. Use appropriate isolation level for use case
3. Monitor version chain lengths
4. Tune garbage collection frequency
5. Partition data to reduce contention

### Dependencies
- `internal/common` - Error types and utilities
- `internal/config` - Configuration management
- Potentially `internal/wal` for transaction logging
