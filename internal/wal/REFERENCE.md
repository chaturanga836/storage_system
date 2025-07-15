# WAL Module Reference

## Quick Reference

### Key Files
- `manager.go` - Main WAL manager implementation
- `segment.go` - WAL segment handling
- `types.go` - Core data structures (if exists)
- `writer.go` - WAL writing operations (if exists)
- `reader.go` - WAL reading operations (if exists)

### Core Types
```go
type Manager interface {
    WriteEntry(entry *Entry) error
    WriteBatch(entries []*Entry) error
    ReadSegment(segmentID SegmentID) (*Segment, error)
    Replay(checkpoint SequenceNumber, handler ReplayHandler) error
    CreateCheckpoint() (SequenceNumber, error)
    Truncate(sequenceNumber SequenceNumber) error
    Close() error
}

type Entry struct {
    SequenceNumber SequenceNumber
    Timestamp      time.Time
    TenantID       string
    Operation      Operation
    Data           []byte
    Checksum       uint32
}

type Segment struct {
    ID         SegmentID
    StartSeq   SequenceNumber
    EndSeq     SequenceNumber
    Size       int64
    FilePath   string
    Status     SegmentStatus
}
```

### Configuration
```yaml
wal:
  directory: "/data/wal"
  segment_size: "64MB"
  sync_policy: "immediate"  # immediate, periodic, async
  retention_hours: 72
  compression: true
```

### Common Operations
```go
// Initialize WAL manager
manager, err := wal.NewManager(config)

// Write single entry
entry := &wal.Entry{
    Operation: wal.OperationInsert,
    Data:      data,
    TenantID:  "tenant-123",
}
err = manager.WriteEntry(entry)

// Write batch
entries := []*wal.Entry{entry1, entry2, entry3}
err = manager.WriteBatch(entries)

// Replay from checkpoint
err = manager.Replay(lastCheckpoint, replayHandler)
```

### Error Handling
- `ErrSegmentNotFound` - Requested segment doesn't exist
- `ErrCorruptedEntry` - Entry checksum mismatch
- `ErrInsufficientSpace` - Not enough disk space
- `ErrInvalidSequence` - Sequence number out of order

### Testing
```bash
# Unit tests
go test ./internal/wal/...

# Integration tests
go test ./tests/integration/wal/...

# Benchmarks
go test -bench=. ./internal/wal/...
```

### Monitoring Metrics
- `wal_entries_written_total` - Total entries written
- `wal_bytes_written_total` - Total bytes written
- `wal_segments_count` - Current number of segments
- `wal_replay_duration_seconds` - Time taken for replay
- `wal_sync_duration_seconds` - Time taken for sync operations

### Troubleshooting
1. **High latency**: Check sync policy and disk I/O
2. **Segment corruption**: Verify checksums and file system
3. **Replay failures**: Check sequence continuity
4. **Disk space**: Monitor segment retention and cleanup

### Dependencies
- `internal/common` - Error types and utilities
- `internal/config` - Configuration management
- File system for segment storage
