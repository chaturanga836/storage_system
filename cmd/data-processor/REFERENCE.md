# Data Processor Reference

## Quick Reference

### Key Files
- `main.go` - Service startup and initialization
- `processor.go` - Main processing logic (likely in internal/services/processor/)
- `scheduler.go` - Background task scheduling (likely in internal/services/processor/)
- `config.yaml` - Configuration file

### Core Components
- **WAL Replay Engine** - Processes write-ahead log entries
- **Memtable Flush Manager** - Handles memtable-to-disk persistence
- **Compaction Coordinator** - Orchestrates data compaction
- **Background Task Scheduler** - Manages periodic maintenance

### Configuration
```yaml
processor:
  worker_count: 4
  queue_size: 10000
  
wal_replay:
  batch_size: 1000
  flush_interval: 5s
  max_parallel_replays: 4
  
memtable_flush:
  max_memtable_size: "64MB"
  flush_threshold: 0.8
  max_concurrent_flushes: 2
  
compaction:
  trigger_threshold: 4
  max_concurrent_compactions: 1
  target_file_size: "256MB"
  strategy: "level_based"  # level_based, size_tiered
```

### Common Operations
```bash
# Start processor
go run cmd/data-processor/main.go

# With custom config
go run cmd/data-processor/main.go -config=/path/to/config.yaml

# Docker
docker run storage-engine:latest data-processor

# Check processor status
curl http://localhost:8082/health
```

### Processing Workflows

#### WAL Replay
```go
// Trigger manual WAL replay
replayManager.ReplayFromCheckpoint(lastCheckpoint)

// Batch replay
entries := walManager.ReadBatch(batchSize)
for _, entry := range entries {
    err := replayManager.ApplyEntry(entry)
}
```

#### Memtable Flush
```go
// Check if flush needed
if memtable.Size() > flushThreshold {
    flushManager.TriggerFlush(memtable)
}

// Manual flush
err := flushManager.FlushMemtable(memtableID)
```

#### Compaction
```go
// Check compaction triggers
if fileCount > compactionThreshold {
    compactionManager.TriggerCompaction(level)
}

// Manual compaction
err := compactionManager.CompactLevel(level, files)
```

### Task Types
- **WAL Replay** - Recovery from write-ahead log
- **Memtable Flush** - Persist in-memory data to disk
- **Compaction** - Merge and optimize data files
- **Cleanup** - Remove obsolete files and data
- **Statistics Update** - Refresh table and column statistics

### Error Handling
- `ReplayError` - Error during WAL replay
- `FlushError` - Error during memtable flush
- `CompactionError` - Error during compaction
- `DiskSpaceError` - Insufficient disk space
- `LockError` - Resource lock conflict

### Testing
```bash
# Unit tests
go test ./cmd/data-processor/...

# Integration tests
go test ./tests/integration/processor/...

# Performance tests
go test ./tests/performance/processor/...
```

### Monitoring
- **Port**: 8082 (if has HTTP interface)
- **Health**: `/health`
- **Metrics**: `/metrics`

### Key Metrics
- `processor_tasks_total` - Total tasks processed
- `processor_queue_depth` - Current queue size
- `wal_replay_rate` - Entries replayed per second
- `memtable_flush_rate` - Flushes per hour
- `compaction_rate` - Files compacted per hour
- `processor_memory_usage` - Memory consumption
- `processor_cpu_usage` - CPU utilization

### Admin Commands
```bash
# Check processor status
storage-admin status --service data-processor

# View processing metrics
storage-admin metrics --service data-processor

# Trigger manual compaction
storage-admin compact --force

# Force WAL replay
storage-admin wal replay

# View queue status
storage-admin processor queue-status
```

### Performance Tuning
1. **Worker Count**: Match to available CPU cores
2. **Queue Size**: Balance memory usage vs throughput
3. **Batch Size**: Optimize for disk I/O patterns
4. **Concurrency**: Limit based on disk bandwidth
5. **Scheduling**: Avoid peak traffic times

### Dependencies
- `internal/wal` - WAL reading and management
- `internal/storage` - Storage operations
- `internal/catalog` - Metadata updates
- `internal/config` - Configuration management
- `internal/storage/compaction` - Compaction logic
