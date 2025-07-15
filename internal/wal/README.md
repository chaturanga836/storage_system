# Write-Ahead Log (WAL) Module

## Overview

The Write-Ahead Log (WAL) module provides crash recovery and durability guarantees for the storage system. It ensures that all write operations are logged before being applied to the main data structures, enabling recovery from system failures.

## Architecture

### Core Components

- **WAL Manager**: Coordinates WAL operations and manages segments
- **WAL Segments**: Individual log files containing write operations
- **WAL Writer**: Handles writing operations to the log
- **WAL Reader**: Reads operations from log segments
- **Checkpoint Manager**: Manages recovery checkpoints

### File Structure

```
wal/
├── segments/
│   ├── wal-000001.log
│   ├── wal-000002.log
│   └── wal-000003.log
├── checkpoints/
│   ├── checkpoint-000100.meta
│   └── checkpoint-000200.meta
└── metadata/
    └── wal.meta
```

## Features

### Durability Guarantees

- **Immediate Persistence**: All writes are immediately flushed to disk
- **Ordered Writes**: Operations are logged in strict order
- **Atomic Operations**: Group operations can be committed atomically
- **Crash Recovery**: Complete recovery from any point of failure

### Performance Optimizations

- **Batched Writes**: Multiple operations can be batched for efficiency
- **Asynchronous Flushing**: Optional async flush for performance
- **Compression**: Log compression to reduce storage overhead
- **Segment Rotation**: Automatic rotation of log segments

### Recovery Features

- **Point-in-Time Recovery**: Recover to any specific timestamp
- **Incremental Recovery**: Replay only necessary operations
- **Parallel Recovery**: Concurrent replay of independent operations
- **Checkpointing**: Periodic checkpoints to minimize recovery time

## API Reference

### WAL Manager Interface

```go
type Manager interface {
    // Write operations
    WriteEntry(entry *Entry) error
    WriteBatch(entries []*Entry) error
    
    // Read operations
    ReadSegment(segmentID SegmentID) (*Segment, error)
    ReadRange(start, end SequenceNumber) ([]*Entry, error)
    
    // Recovery operations
    Replay(checkpoint SequenceNumber, handler ReplayHandler) error
    CreateCheckpoint() (SequenceNumber, error)
    
    // Management operations
    Truncate(sequenceNumber SequenceNumber) error
    Compact() error
    Close() error
}
```

### Entry Structure

```go
type Entry struct {
    SequenceNumber SequenceNumber
    Timestamp      time.Time
    TenantID       string
    Operation      Operation
    Data           []byte
    Checksum       uint32
}

type Operation int

const (
    OperationInsert Operation = iota
    OperationUpdate
    OperationDelete
    OperationBeginTx
    OperationCommitTx
    OperationAbortTx
)
```

## Configuration

### WAL Settings

```yaml
wal:
  directory: "/data/wal"
  segment_size: 256MB
  sync_policy: "immediate"  # immediate, batch, async
  batch_size: 100
  compression: "lz4"
  retention_hours: 168  # 7 days
```

### Performance Tuning

- **Segment Size**: Balance between file count and I/O efficiency
- **Sync Policy**: Trade-off between durability and performance
- **Batch Size**: Number of entries per write batch
- **Compression**: Reduce storage overhead at CPU cost

## Operations

### Basic Usage

```go
// Initialize WAL manager
manager, err := wal.NewManager(config)
if err != nil {
    return err
}
defer manager.Close()

// Write single entry
entry := &wal.Entry{
    TenantID:  "tenant-123",
    Operation: wal.OperationInsert,
    Data:      recordData,
}
err = manager.WriteEntry(entry)

// Write batch
entries := []*wal.Entry{entry1, entry2, entry3}
err = manager.WriteBatch(entries)

// Recovery
err = manager.Replay(lastCheckpoint, replayHandler)
```

### Recovery Process

1. **Identify Last Checkpoint**: Find the most recent valid checkpoint
2. **Scan Segments**: Identify WAL segments to replay
3. **Validate Entries**: Check checksums and sequence order
4. **Apply Operations**: Replay entries to restore state
5. **Update Checkpoint**: Create new checkpoint after recovery

## Testing

### Unit Tests

```bash
go test ./internal/wal/...
```

### Integration Tests

```bash
go test ./tests/integration/wal/...
```

### Performance Tests

```bash
go test ./tests/performance/wal/...
```

## Monitoring

### Key Metrics

- **Write Throughput**: Entries per second
- **Segment Count**: Number of active WAL segments
- **Disk Usage**: Total WAL storage usage
- **Sync Latency**: Time to sync writes to disk
- **Recovery Time**: Time to replay WAL during startup

### Health Checks

- Verify WAL directory accessibility
- Check segment file integrity
- Monitor disk space usage
- Validate checkpoint consistency

## Troubleshooting

### Common Issues

1. **Disk Full**: WAL segments can't be created
   - Solution: Clean up old segments or increase disk space

2. **Corruption**: WAL segments are corrupted
   - Solution: Use checksums to detect and handle corruption

3. **Slow Recovery**: WAL replay takes too long
   - Solution: Increase checkpoint frequency or optimize replay logic

4. **High Latency**: Write operations are slow
   - Solution: Tune sync policy and batch size

### Debug Commands

```bash
# Inspect WAL status
storage-admin wal inspect

# Force checkpoint creation
storage-admin wal checkpoint

# Validate WAL integrity
storage-admin wal validate
```

## Implementation Files

- `manager.go`: Main WAL manager implementation
- `segment.go`: WAL segment handling
- `entry.go`: WAL entry structure and serialization
- `recovery.go`: Recovery and replay logic
- `checkpoint.go`: Checkpoint management
- `config.go`: Configuration handling
