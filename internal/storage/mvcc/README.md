# Multi-Version Concurrency Control (MVCC) Module

## Overview

The MVCC (Multi-Version Concurrency Control) module provides transaction isolation and consistency guarantees for the storage system. It manages multiple versions of data to enable concurrent reads and writes without blocking, implementing snapshot isolation semantics.

## Architecture

### Core Components

- **Version Manager**: Manages data versions and version chains
- **Transaction Manager**: Handles transaction lifecycle and isolation
- **Snapshot Manager**: Creates and manages read snapshots
- **Conflict Resolver**: Detects and resolves write conflicts
- **Garbage Collector**: Cleans up obsolete versions

### Version Chain Structure

```
Record Key: "user:123"
Version Chain: [v3] -> [v2] -> [v1] -> [v0]
               latest    ↑       ↑      oldest
                      visible  committed
```

## Features

### Isolation Levels

- **Snapshot Isolation**: Default isolation level providing consistent snapshots
- **Read Committed**: Reads see committed data as of statement start
- **Serializable**: Strongest isolation with conflict detection

### Concurrency Features

- **Non-blocking Reads**: Readers never block writers or other readers
- **Optimistic Concurrency**: Writers detect conflicts at commit time
- **Version Visibility**: Each transaction sees a consistent data snapshot
- **Conflict Detection**: Automatic detection and resolution of write conflicts

### Performance Optimizations

- **Version Pruning**: Automatic cleanup of obsolete versions
- **Hot Spotting**: Optimization for frequently accessed data
- **Batch Operations**: Efficient handling of bulk operations
- **Index Versioning**: Version-aware index operations

## API Reference

### Version Manager Interface

```go
type VersionManager interface {
    // Version operations
    CreateVersion(key string, value []byte, txID TransactionID) (*Version, error)
    GetVersion(key string, snapshot Snapshot) (*Version, error)
    GetVersions(key string, maxVersions int) ([]*Version, error)
    
    // Transaction operations
    BeginTransaction(isolation IsolationLevel) (*Transaction, error)
    CommitTransaction(txID TransactionID) error
    AbortTransaction(txID TransactionID) error
    
    // Snapshot operations
    CreateSnapshot() (Snapshot, error)
    ReleaseSnapshot(snapshot Snapshot) error
    
    // Cleanup operations
    GarbageCollect() error
    PruneVersions(beforeTimestamp time.Time) error
}
```

### Version Structure

```go
type Version struct {
    Key            string
    Value          []byte
    TransactionID  TransactionID
    CommitTime     time.Time
    CreatedTime    time.Time
    Status         VersionStatus
    PreviousVersion *Version
    NextVersion    *Version
}

type VersionStatus int

const (
    VersionStatusActive VersionStatus = iota
    VersionStatusCommitted
    VersionStatusAborted
    VersionStatusDeleted
)
```

### Transaction Management

```go
type Transaction struct {
    ID            TransactionID
    StartTime     time.Time
    IsolationLevel IsolationLevel
    Snapshot      Snapshot
    WriteSet      map[string]*Version
    ReadSet       map[string]time.Time
    Status        TransactionStatus
}

type IsolationLevel int

const (
    ReadCommitted IsolationLevel = iota
    SnapshotIsolation
    Serializable
)
```

## Configuration

### MVCC Settings

```yaml
mvcc:
  max_versions_per_key: 100
  gc_interval: 30m
  version_retention: 24h
  conflict_resolution: "last_write_wins"  # abort_on_conflict, last_write_wins
  snapshot_cache_size: 1000
```

### Performance Tuning

- **Version Limit**: Maximum versions to keep per key
- **GC Interval**: How often to run garbage collection
- **Retention**: How long to keep old versions
- **Cache Size**: Number of snapshots to cache

## Operations

### Basic Usage

```go
// Initialize MVCC manager
mvcc, err := NewVersionManager(config)
if err != nil {
    return err
}

// Begin transaction
tx, err := mvcc.BeginTransaction(SnapshotIsolation)
if err != nil {
    return err
}

// Read data (sees snapshot at transaction start time)
version, err := mvcc.GetVersion("user:123", tx.Snapshot)
if err != nil {
    mvcc.AbortTransaction(tx.ID)
    return err
}

// Write data
newVersion, err := mvcc.CreateVersion("user:123", newData, tx.ID)
if err != nil {
    mvcc.AbortTransaction(tx.ID)
    return err
}

// Commit transaction
err = mvcc.CommitTransaction(tx.ID)
if err != nil {
    // Handle conflict or other error
    return err
}
```

### Conflict Resolution

1. **Detection**: Check for concurrent modifications during commit
2. **Resolution**: Apply configured conflict resolution strategy
3. **Retry**: Allow application to retry conflicted transactions
4. **Abort**: Roll back transaction on unresolvable conflicts

## Testing

### Unit Tests

```bash
go test ./internal/storage/mvcc/...
```

### Integration Tests

```bash
go test ./tests/integration/mvcc/...
```

### Concurrency Tests

```bash
go test ./tests/performance/mvcc/...
```

## Monitoring

### Key Metrics

- **Transaction Rate**: Transactions per second
- **Conflict Rate**: Percentage of conflicted transactions
- **Version Count**: Average versions per key
- **GC Performance**: Time spent in garbage collection
- **Memory Usage**: Memory consumed by version chains

### Health Checks

- Monitor transaction queue depth
- Check version chain lengths
- Validate GC effectiveness
- Track conflict resolution success

## Troubleshooting

### Common Issues

1. **High Conflict Rate**: Too many concurrent writes to same keys
   - Solution: Partition data or use different conflict resolution

2. **Memory Growth**: Version chains growing too long
   - Solution: Increase GC frequency or reduce retention

3. **Read Performance**: Slow reads due to long version chains
   - Solution: Optimize version traversal or increase pruning

4. **Transaction Timeouts**: Long-running transactions
   - Solution: Break into smaller transactions or increase timeout

### Debug Commands

```bash
# Show MVCC statistics
storage-admin mvcc stats

# Force garbage collection
storage-admin mvcc gc

# Inspect version chains
storage-admin mvcc inspect --key "user:123"
```

## Implementation Files

- `version.go`: Version structure and management
- `transaction.go`: Transaction lifecycle management
- `snapshot.go`: Snapshot creation and management
- `resolver.go`: Conflict detection and resolution
- `gc.go`: Garbage collection logic
- `cache.go`: Version and snapshot caching
