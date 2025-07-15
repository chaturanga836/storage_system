# Admin CLI

## Overview

The Admin CLI is a command-line interface for managing and monitoring the distributed storage system. It provides administrators with tools for system maintenance, monitoring, troubleshooting, and operational tasks.

## Architecture

### Core Components

- **Command Framework**: Built on Cobra for structured CLI commands
- **Service Integration**: Connects to storage services for operations
- **Configuration Management**: Handles system configuration updates
- **Monitoring Interface**: Provides real-time system status and metrics

### Installation

```bash
# Build from source
go build ./cmd/admin-cli

# Install globally
go install ./cmd/admin-cli

# Run directly
go run cmd/admin-cli/main.go [command]
```

## Available Commands

### System Status

#### `status`
Display overall system health and status.

```bash
storage-admin status
```

**Output:**
- Service status (running/stopped/error)
- WAL health and replay progress
- Active memtables count
- Parquet file statistics
- Index status and health

#### `status --detailed`
Show detailed system information including performance metrics.

### Compaction Management

#### `compact`
Trigger manual compaction operations.

```bash
# Trigger automatic compaction
storage-admin compact

# Force compaction of specific levels
storage-admin compact --level 0,1

# Compact specific tenant data
storage-admin compact --tenant tenant-123
```

**Options:**
- `--level`: Target specific compaction levels
- `--tenant`: Compact data for specific tenant
- `--force`: Force compaction regardless of thresholds
- `--dry-run`: Show what would be compacted without executing

### WAL Operations

#### `wal inspect`
Inspect WAL contents and status.

```bash
# Show WAL overview
storage-admin wal inspect

# Inspect specific WAL segment
storage-admin wal inspect --segment segment-001

# Show WAL entries for specific tenant
storage-admin wal inspect --tenant tenant-123
```

#### `wal replay`
Manual WAL replay operations.

```bash
# Replay uncommitted WAL entries
storage-admin wal replay

# Replay specific segment
storage-admin wal replay --segment segment-001

# Replay from specific checkpoint
storage-admin wal replay --from-checkpoint 12345
```

#### `wal truncate`
Truncate WAL to specific point (use with caution).

```bash
# Truncate to latest checkpoint
storage-admin wal truncate --to-checkpoint

# Truncate to specific sequence number
storage-admin wal truncate --to-sequence 12345
```

### Schema Management

#### `schema list`
List all registered schemas.

```bash
# List all schemas
storage-admin schema list

# List schemas for specific tenant
storage-admin schema list --tenant tenant-123
```

#### `schema show`
Display schema details.

```bash
# Show specific schema
storage-admin schema show --schema-id schema-456

# Show schema versions
storage-admin schema show --schema-id schema-456 --all-versions
```

#### `schema validate`
Validate schema definitions and compatibility.

```bash
# Validate all schemas
storage-admin schema validate

# Validate specific schema
storage-admin schema validate --schema-id schema-456
```

### Index Management

#### `index status`
Show index status and statistics.

```bash
# Show all indexes
storage-admin index status

# Show indexes for specific tenant
storage-admin index status --tenant tenant-123
```

#### `index rebuild`
Rebuild indexes from stored data.

```bash
# Rebuild all indexes
storage-admin index rebuild

# Rebuild specific index
storage-admin index rebuild --index-name user_email_idx

# Rebuild tenant indexes
storage-admin index rebuild --tenant tenant-123
```

#### `index optimize`
Optimize index performance and storage.

```bash
# Optimize all indexes
storage-admin index optimize

# Optimize specific index type
storage-admin index optimize --type secondary
```

### Data Management

#### `data export`
Export data for backup or migration.

```bash
# Export all data
storage-admin data export --output /backup/data.parquet

# Export tenant data
storage-admin data export --tenant tenant-123 --output /backup/tenant-123.parquet

# Export time range
storage-admin data export --from 2024-01-01 --to 2024-02-01 --output /backup/january.parquet
```

#### `data import`
Import data from backup or external source.

```bash
# Import from Parquet file
storage-admin data import --input /backup/data.parquet

# Import with schema validation
storage-admin data import --input /backup/data.parquet --validate-schema

# Import to specific tenant
storage-admin data import --input /backup/data.parquet --tenant tenant-123
```

#### `data cleanup`
Clean up expired or deleted data.

```bash
# Clean up all expired data
storage-admin data cleanup

# Clean up specific tenant data
storage-admin data cleanup --tenant tenant-123

# Dry run to see what would be cleaned
storage-admin data cleanup --dry-run
```

### Configuration Management

#### `config show`
Display current configuration.

```bash
# Show all configuration
storage-admin config show

# Show specific section
storage-admin config show --section ingestion
```

#### `config update`
Update configuration values.

```bash
# Update single value
storage-admin config update --key query.max_connections --value 2000

# Update from file
storage-admin config update --file new-config.yaml

# Validate configuration
storage-admin config update --validate-only --file new-config.yaml
```

#### `config reload`
Reload configuration without restart.

```bash
# Reload all services
storage-admin config reload

# Reload specific service
storage-admin config reload --service query-server
```

### Monitoring and Metrics

#### `metrics`
Display system metrics.

```bash
# Show overview metrics
storage-admin metrics

# Show detailed metrics
storage-admin metrics --detailed

# Show metrics for specific service
storage-admin metrics --service ingestion-server

# Export metrics to file
storage-admin metrics --export /tmp/metrics.json
```

#### `logs`
Access and analyze system logs.

```bash
# Show recent logs
storage-admin logs

# Show logs for specific service
storage-admin logs --service query-server

# Follow logs in real-time
storage-admin logs --follow

# Filter by log level
storage-admin logs --level error

# Search logs
storage-admin logs --grep "timeout"
```

### Tenant Management

#### `tenant list`
List all tenants.

```bash
storage-admin tenant list
```

#### `tenant create`
Create new tenant.

```bash
storage-admin tenant create --id tenant-456 --name "New Tenant"
```

#### `tenant stats`
Show tenant statistics.

```bash
# All tenants
storage-admin tenant stats

# Specific tenant
storage-admin tenant stats --tenant tenant-123
```

## Configuration

### CLI Configuration

```yaml
# ~/.storage-admin/config.yaml
admin:
  endpoints:
    ingestion: "localhost:8080"
    query: "localhost:8081"
    processor: "localhost:8082"
  timeout: 30s
  retries: 3
  format: table  # table, json, yaml
```

### Environment Variables

- `STORAGE_ADMIN_CONFIG`: Configuration file path
- `STORAGE_ENDPOINTS`: Comma-separated service endpoints
- `STORAGE_TIMEOUT`: Default operation timeout
- `STORAGE_FORMAT`: Default output format

## Output Formats

### Table Format (Default)
Human-readable tabular output for terminal use.

### JSON Format
Machine-readable JSON for scripting and automation.

```bash
storage-admin status --format json
```

### YAML Format
Structured YAML output for configuration and documentation.

```bash
storage-admin config show --format yaml
```

## Scripting and Automation

### Exit Codes

- `0`: Success
- `1`: General error
- `2`: Configuration error
- `3`: Service unavailable
- `4`: Authentication error

### Batch Operations

```bash
# Batch script example
#!/bin/bash
set -e

echo "Starting maintenance..."
storage-admin wal truncate --to-checkpoint
storage-admin compact --force
storage-admin index optimize
echo "Maintenance complete."
```

### Integration with CI/CD

```yaml
# GitHub Actions example
- name: System Health Check
  run: |
    storage-admin status --format json > health.json
    if ! storage-admin status | grep -q "✅"; then
      exit 1
    fi
```

## Troubleshooting

### Common Issues

1. **Connection Errors**
   - Verify service endpoints
   - Check network connectivity
   - Validate authentication

2. **Permission Errors**
   - Check file system permissions
   - Verify user credentials
   - Review access control settings

3. **Timeout Errors**
   - Increase timeout values
   - Check service health
   - Review operation complexity

### Debug Mode

Enable verbose output for troubleshooting:

```bash
storage-admin --debug status
storage-admin --verbose compact
```

## Security

### Authentication

- Token-based authentication
- Service account credentials
- Role-based access control

### Authorization

- Operation-level permissions
- Tenant-based isolation
- Administrative privileges

## Development

### Building

```bash
go build ./cmd/admin-cli
```

### Testing

```bash
go test ./cmd/admin-cli/...
```

### Adding Commands

1. Create command struct with Cobra
2. Implement command logic
3. Add to command hierarchy
4. Update documentation
5. Add tests
