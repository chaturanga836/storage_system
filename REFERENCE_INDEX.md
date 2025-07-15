# Storage System Reference Index

## Quick Access to All References

This file provides quick links to all reference documentation for rapid development and troubleshooting.

### 🏗️ Core Modules

| Module | Reference | Key Purpose |
|--------|-----------|-------------|
| **WAL** | [internal/wal/REFERENCE.md](internal/wal/REFERENCE.md) | Write-ahead logging for durability |
| **MVCC** | [internal/storage/mvcc/REFERENCE.md](internal/storage/mvcc/REFERENCE.md) | Multi-version concurrency control |
| **Catalog** | [internal/catalog/REFERENCE.md](internal/catalog/REFERENCE.md) | Metadata and schema management |
| **Schema** | [internal/schema/REFERENCE.md](internal/schema/REFERENCE.md) | Schema evolution and validation |

### 🚀 Services

| Service | Reference | Default Port |
|---------|-----------|--------------|
| **Ingestion Server** | [cmd/ingestion-server/REFERENCE.md](cmd/ingestion-server/REFERENCE.md) | 8080 |
| **Query Server** | [cmd/query-server/REFERENCE.md](cmd/query-server/REFERENCE.md) | 8081 |
| **Data Processor** | [cmd/data-processor/REFERENCE.md](cmd/data-processor/REFERENCE.md) | - |
| **Admin CLI** | [cmd/admin-cli/REFERENCE.md](cmd/admin-cli/REFERENCE.md) | - |

### 🧪 Testing

| Type | Reference | Purpose |
|------|-----------|---------|
| **Tests Overview** | [tests/REFERENCE.md](tests/REFERENCE.md) | Testing framework and utilities |

## 🔧 Quick Development Commands

### Build & Run
```bash
# Build all services
./scripts/build.sh

# Run locally
./scripts/run_local.sh

# Build specific service
go build ./cmd/ingestion-server
```

### Testing
```bash
# All tests
go test ./...

# Integration tests
go test ./tests/integration/...

# Performance tests
go test ./tests/performance/...

# With race detection
go test -race ./...
```

### Admin Operations
```bash
# System status
storage-admin status

# Force compaction
storage-admin compact --force

# View metrics
storage-admin metrics

# Check WAL
storage-admin wal status
```

## 🔍 Common Troubleshooting

### Service Issues
1. **Port conflicts**: Check if ports 8080, 8081 are available
2. **Configuration**: Verify config files and environment variables
3. **Dependencies**: Ensure all required services are running

### Performance Issues  
1. **High latency**: Check index usage and query optimization
2. **Memory growth**: Monitor memtable sizes and garbage collection
3. **Disk I/O**: Check WAL sync policy and compaction settings

### Data Issues
1. **Schema conflicts**: Use schema validation and compatibility checking
2. **MVCC conflicts**: Monitor transaction patterns and isolation levels
3. **WAL corruption**: Check disk health and backup procedures

## 📋 Configuration Quick Reference

### Environment Variables
- `STORAGE_CONFIG_PATH` - Configuration file path
- `LOG_LEVEL` - Logging level (debug, info, warn, error)
- `METRICS_PORT` - Metrics server port
- `STORAGE_DATA_DIR` - Data storage directory

### Key Config Sections
```yaml
server:
  port: 8080
  max_connections: 1000

storage:
  wal_directory: "/data/wal"
  memtable_size: "64MB"

mvcc:
  max_versions_per_key: 100
  gc_interval: 30m

catalog:
  storage_path: "/data/catalog"
  cache_size: 1GB
```

## 🎯 Performance Targets

| Metric | Target | Notes |
|--------|--------|-------|
| Ingestion Rate | 1M+ records/sec | With appropriate hardware |
| Query Latency | <10ms | For indexed queries |
| Storage Efficiency | 80%+ compression | Using Parquet format |
| Availability | 99.9% uptime | With proper deployment |

## 📚 Additional Documentation

- **[README.md](README.md)** - Project overview and quick start
- **[PROJECT_STRUCTURE.md](PROJECT_STRUCTURE.md)** - Detailed architecture
- **[DOCUMENTATION_RESTRUCTURE.md](DOCUMENTATION_RESTRUCTURE.md)** - Documentation organization

## 🛠️ Development Workflow

1. **Check references** for the module/service you're working on
2. **Run relevant tests** to understand current behavior  
3. **Use admin CLI** for debugging and monitoring
4. **Follow configuration examples** for setup
5. **Refer to troubleshooting sections** for common issues

This reference index is designed to minimize context switching and provide immediate access to the information you need for productive development.
