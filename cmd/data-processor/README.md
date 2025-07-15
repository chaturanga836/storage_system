# Data Processor

## Overview

The Data Processor is a background service responsible for asynchronous data processing tasks in the distributed storage system. It handles WAL replay, memtable flushing, compaction operations, and data lifecycle management.

## Architecture

### Core Components

- **WAL Replay Engine**: Processes write-ahead log entries
- **Memtable Flush Manager**: Manages memtable-to-disk persistence
- **Compaction Coordinator**: Orchestrates data compaction operations
- **Background Task Scheduler**: Manages periodic maintenance tasks

### Configuration

```yaml
data_processor:
  wal_replay:
    batch_size: 1000
    flush_interval: 5s
    max_parallel_replays: 4
  memtable_flush:
    max_memtable_size: 64MB
    flush_threshold: 0.8
    max_concurrent_flushes: 2
  compaction:
    trigger_threshold: 4
    max_concurrent_compactions: 1
    target_file_size: 256MB
```

## Features

### WAL Replay

- **Crash Recovery**: Replays uncommitted WAL entries after system restart
- **Batch Processing**: Processes multiple WAL entries in batches for efficiency
- **Parallel Replay**: Supports parallel replay for independent transactions
- **Progress Tracking**: Maintains replay progress and checkpoints

### Memtable Management

- **Automatic Flushing**: Monitors memtable size and triggers flushes
- **Background Flushing**: Non-blocking flush operations
- **Size Monitoring**: Tracks memory usage and applies backpressure
- **Flush Coordination**: Coordinates with ingestion to prevent data loss

### Compaction Operations

- **Level-Based Compaction**: Implements LSM-tree compaction strategy
- **Size-Tiered Compaction**: Alternative compaction for specific workloads
- **Automatic Triggering**: Monitors file counts and sizes for compaction
- **Background Processing**: Non-blocking compaction operations

### Data Lifecycle

- **TTL Management**: Handles time-to-live data expiration
- **Retention Policies**: Applies data retention rules
- **Cleanup Operations**: Removes expired and deleted data
- **Storage Optimization**: Optimizes storage layout and access patterns

## Processing Workflows

### WAL Replay Workflow

1. **Scan WAL Directory**: Identify uncommitted WAL segments
2. **Order Segments**: Sort by sequence number for proper ordering
3. **Batch Entries**: Group entries for efficient processing
4. **Apply Changes**: Replay entries to memtables and indexes
5. **Update Checkpoints**: Mark successful replay progress
6. **Cleanup**: Remove successfully replayed WAL segments

### Memtable Flush Workflow

1. **Monitor Size**: Continuously monitor memtable memory usage
2. **Trigger Flush**: Initiate flush when threshold is exceeded
3. **Freeze Memtable**: Create immutable snapshot for flushing
4. **Write to Disk**: Serialize memtable to Parquet format
5. **Update Indexes**: Update indexes with new file metadata
6. **Commit Transaction**: Atomically commit the flush operation
7. **Cleanup Memory**: Release memtable memory

### Compaction Workflow

1. **Monitor Metrics**: Track file counts, sizes, and read amplification
2. **Select Files**: Choose files for compaction based on strategy
3. **Merge Data**: Combine multiple files into larger, optimized files
4. **Update Metadata**: Update catalog with new file information
5. **Atomic Swap**: Replace old files with compacted versions
6. **Cleanup**: Remove obsolete files and update indexes

## Configuration Options

### WAL Replay Settings

- **Batch Size**: Number of WAL entries processed per batch
- **Flush Interval**: Maximum time between replay batches
- **Parallel Replays**: Number of concurrent replay workers
- **Memory Limit**: Maximum memory for replay operations

### Memtable Flush Settings

- **Size Threshold**: Memory threshold for triggering flush
- **Flush Interval**: Maximum time between flushes
- **Concurrent Flushes**: Number of simultaneous flush operations
- **Compression**: Compression settings for flushed files

### Compaction Settings

- **Trigger Threshold**: File count threshold for compaction
- **Target File Size**: Desired size for compacted files
- **Concurrent Compactions**: Number of parallel compaction jobs
- **Strategy**: Compaction strategy (level-based, size-tiered)

## Monitoring and Metrics

### Key Metrics

- **WAL Replay Rate**: Entries processed per second
- **Flush Rate**: Memtables flushed per hour
- **Compaction Rate**: Files compacted per hour
- **Memory Usage**: Current memory consumption
- **Queue Depths**: Pending operations in queues

### Performance Metrics

- **Processing Latency**: Time to complete operations
- **Throughput**: Data processed per second
- **Error Rate**: Failed operations percentage
- **Resource Utilization**: CPU, memory, and disk usage

### Health Indicators

- **WAL Health**: WAL replay progress and errors
- **Flush Health**: Memtable flush success rate
- **Compaction Health**: Compaction job success rate
- **Queue Health**: Queue depth and processing rate

## Operations

### Starting the Service

```bash
# Development
go run cmd/data-processor/main.go

# Production
./bin/data-processor

# Docker
docker run storage-engine:latest data-processor
```

### Graceful Shutdown

The service supports graceful shutdown with:
- Completion of active operations
- Checkpoint creation
- Resource cleanup
- Signal handling (SIGTERM, SIGINT)

## Error Handling

### Error Categories

- **WAL Errors**: Corrupted or missing WAL segments
- **Flush Errors**: Disk space or I/O errors during flush
- **Compaction Errors**: File corruption or resource constraints
- **System Errors**: Memory exhaustion or system failures

### Recovery Mechanisms

- **Retry Logic**: Automatic retry for transient failures
- **Fallback Strategies**: Alternative processing paths
- **Error Isolation**: Prevent single failures from affecting entire system
- **Manual Recovery**: Administrative tools for manual intervention

## Troubleshooting

### Common Issues

1. **WAL Replay Lag**
   - Check disk I/O performance
   - Review batch size configuration
   - Monitor memory usage

2. **Flush Failures**
   - Verify disk space availability
   - Check file system permissions
   - Review compression settings

3. **Compaction Delays**
   - Monitor CPU and disk utilization
   - Check compaction strategy effectiveness
   - Review file size distributions

### Debugging

- Enable detailed logging
- Use performance profiling
- Monitor system resources
- Check dependency health

## Integration

### Dependencies

- **Storage Manager**: Coordinates with storage operations
- **WAL Manager**: Reads and processes WAL segments
- **Catalog Service**: Updates metadata and indexes
- **Ingestion Service**: Coordinates with data ingestion

### Coordination

- **Lock Management**: Prevents conflicts between operations
- **Event Notifications**: Publishes operation completion events
- **Health Reporting**: Reports status to service discovery
- **Resource Coordination**: Manages shared resources

## Development

### Building

```bash
go build ./cmd/data-processor
```

### Testing

```bash
go test ./cmd/data-processor/...
go test ./tests/integration/data-processor/...
go test ./tests/performance/data-processor/...
```

### Configuration

Environment variables:
- `STORAGE_CONFIG_PATH`: Configuration file path
- `LOG_LEVEL`: Logging level
- `METRICS_PORT`: Metrics server port
- `PROCESSOR_MODE`: Processing mode (normal, recovery, maintenance)
