# Admin CLI Reference

## Quick Reference

### Key Files
- `main.go` - CLI entry point and command setup
- `cmd/` - Individual command implementations (likely structure)
- `config.yaml` - CLI configuration file

### Command Structure
```
storage-admin [global-flags] <command> [command-flags] [args]
```

### Global Flags
- `--config` - Configuration file path
- `--format` - Output format (table, json, yaml)
- `--debug` - Enable debug output
- `--timeout` - Operation timeout
- `--endpoints` - Service endpoints

### System Commands

#### Status and Health
```bash
# Overall system status
storage-admin status

# Detailed system info
storage-admin status --detailed

# Service-specific status
storage-admin status --service ingestion-server

# Health check all services
storage-admin health
```

#### Configuration
```bash
# Show configuration
storage-admin config show

# Update config value
storage-admin config update --key query.max_connections --value 2000

# Reload configuration
storage-admin config reload --service query-server

# Validate config
storage-admin config validate --file new-config.yaml
```

### WAL Commands
```bash
# WAL overview
storage-admin wal status

# Inspect WAL contents
storage-admin wal inspect --segment segment-001

# Manual replay
storage-admin wal replay --from-checkpoint 12345

# Truncate WAL (dangerous)
storage-admin wal truncate --to-checkpoint
```

### Compaction Commands
```bash
# Trigger compaction
storage-admin compact

# Force compaction
storage-admin compact --force --level 0,1

# Tenant-specific compaction
storage-admin compact --tenant tenant-123

# Dry run
storage-admin compact --dry-run
```

### Schema Commands
```bash
# List schemas
storage-admin schema list --tenant tenant-123

# Show schema details
storage-admin schema show --schema-id schema-456

# Validate schemas
storage-admin schema validate

# Check compatibility
storage-admin schema check-compatibility --schema schema.json
```

### Index Commands
```bash
# Index status
storage-admin index status

# Rebuild indexes
storage-admin index rebuild --table users

# Optimize indexes
storage-admin index optimize --type secondary
```

### Data Management
```bash
# Export data
storage-admin data export --table users --output backup.parquet

# Import data
storage-admin data import --input backup.parquet --validate-schema

# Cleanup expired data
storage-admin data cleanup --tenant tenant-123 --dry-run
```

### Monitoring Commands
```bash
# System metrics
storage-admin metrics

# Service-specific metrics
storage-admin metrics --service query-server

# Export metrics
storage-admin metrics --export metrics.json

# View logs
storage-admin logs --service ingestion-server --follow

# Search logs
storage-admin logs --grep "error" --level error
```

### Tenant Commands
```bash
# List tenants
storage-admin tenant list

# Create tenant
storage-admin tenant create --id tenant-789 --name "New Tenant"

# Tenant statistics
storage-admin tenant stats --tenant tenant-123
```

### Configuration File
```yaml
# ~/.storage-admin/config.yaml
admin:
  endpoints:
    ingestion: "localhost:8080"
    query: "localhost:8081"
    processor: "localhost:8082"
  timeout: 30s
  retries: 3
  format: table
  log_level: info
```

### Environment Variables
- `STORAGE_ADMIN_CONFIG` - Config file path
- `STORAGE_ENDPOINTS` - Service endpoints
- `STORAGE_TIMEOUT` - Default timeout
- `STORAGE_FORMAT` - Output format
- `STORAGE_DEBUG` - Enable debug mode

### Output Formats

#### Table (Default)
```
+----------+--------+--------+
| Service  | Status | Port   |
+----------+--------+--------+
| ingestion| UP     | 8080   |
| query    | UP     | 8081   |
+----------+--------+--------+
```

#### JSON
```json
{
  "services": [
    {"name": "ingestion", "status": "UP", "port": 8080},
    {"name": "query", "status": "UP", "port": 8081}
  ]
}
```

#### YAML
```yaml
services:
  - name: ingestion
    status: UP
    port: 8080
  - name: query
    status: UP
    port: 8081
```

### Exit Codes
- `0` - Success
- `1` - General error
- `2` - Configuration error
- `3` - Service unavailable
- `4` - Authentication error
- `5` - Permission denied

### Scripting Examples
```bash
#!/bin/bash
# Health check script
if storage-admin status --format json | jq -e '.all_healthy'; then
    echo "System healthy"
    exit 0
else
    echo "System unhealthy"
    exit 1
fi

# Maintenance script
storage-admin wal truncate --to-checkpoint
storage-admin compact --force
storage-admin index optimize
storage-admin data cleanup
```

### Common Issues
1. **Connection errors** - Check service endpoints and network
2. **Permission errors** - Verify credentials and access rights
3. **Timeout errors** - Increase timeout or check service health
4. **Format errors** - Ensure proper command syntax

### Dependencies
- Service endpoints (ingestion, query, processor)
- Authentication system
- Network connectivity
