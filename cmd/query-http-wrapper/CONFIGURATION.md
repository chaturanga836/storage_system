# Query HTTP Wrapper Configuration

## Overview
The Query HTTP Wrapper uses a layered configuration system that supports both file-based and environment variable configuration.

## Configuration Sources (Priority Order)
1. Environment variables (highest priority)
2. Configuration file (`config.json`)
3. Default values (lowest priority)

## Configuration File Format

### Basic Configuration (`config.json`)
```json
{
  "query_http_wrapper": {
    "port": 8083,
    "host": "localhost",
    "timeout": "30s",
    "enable_cors": true,
    "log_level": "info",
    "max_request_size": "10MB"
  },
  "query_service": {
    "address": "localhost:8002",
    "timeout": "30s",
    "max_retry_attempts": 3,
    "retry_delay": "1s"
  },
  "query": {
    "port": 8002,
    "max_connections": 500,
    "query_timeout": "30s",
    "cache_size": 1000000,
    "parallel_queries": 10
  },
  "logging": {
    "level": "info",
    "format": "json",
    "output": "stdout",
    "enable_request_logging": true
  },
  "security": {
    "enable_auth": false,
    "cors_origins": ["*"],
    "rate_limit_enabled": false,
    "rate_limit_requests_per_second": 100
  },
  "performance": {
    "query_cache_enabled": true,
    "query_cache_ttl": "300s",
    "max_result_size": "100MB",
    "connection_pool_size": 10
  }
}
```

## Environment Variables

### Query HTTP Wrapper Settings
| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `QUERY_HTTP_WRAPPER_PORT` | HTTP server port | `8083` | `8083` |
| `QUERY_HTTP_WRAPPER_HOST` | HTTP server host | `localhost` | `0.0.0.0` |
| `QUERY_HTTP_WRAPPER_TIMEOUT` | Request timeout | `30s` | `60s` |
| `QUERY_HTTP_WRAPPER_ENABLE_CORS` | Enable CORS | `true` | `false` |
| `QUERY_HTTP_WRAPPER_LOG_LEVEL` | Log level | `info` | `debug` |
| `QUERY_HTTP_WRAPPER_MAX_REQUEST_SIZE` | Max request size | `10MB` | `50MB` |

### Query Service Settings
| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `QUERY_SERVICE_ADDRESS` | gRPC service address | `localhost:8002` | `query-service:8002` |
| `QUERY_SERVICE_TIMEOUT` | Service call timeout | `30s` | `60s` |
| `QUERY_SERVICE_MAX_RETRY` | Max retry attempts | `3` | `5` |
| `QUERY_SERVICE_RETRY_DELAY` | Retry delay | `1s` | `2s` |

### Query Engine Settings
| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `QUERY_PORT` | Query service gRPC port | `8002` | `8002` |
| `QUERY_MAX_CONNECTIONS` | Max concurrent connections | `500` | `1000` |
| `QUERY_TIMEOUT` | Query execution timeout | `30s` | `60s` |
| `QUERY_CACHE_SIZE` | Query result cache size | `1000000` | `5000000` |
| `QUERY_PARALLEL_QUERIES` | Max parallel queries | `10` | `20` |

### Performance Settings
| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `QUERY_CACHE_ENABLED` | Enable query result caching | `true` | `false` |
| `QUERY_CACHE_TTL` | Cache time-to-live | `300s` | `600s` |
| `MAX_RESULT_SIZE` | Maximum result set size | `100MB` | `500MB` |
| `CONNECTION_POOL_SIZE` | Connection pool size | `10` | `20` |

### Logging Settings
| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `LOG_LEVEL` | Global log level | `info` | `debug` |
| `LOG_FORMAT` | Log output format | `json` | `text` |
| `LOG_OUTPUT` | Log output destination | `stdout` | `file` |
| `ENABLE_REQUEST_LOGGING` | Enable request logging | `true` | `false` |

### Security Settings
| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `ENABLE_AUTH` | Enable authentication | `false` | `true` |
| `CORS_ORIGINS` | Allowed CORS origins | `["*"]` | `["http://localhost:3000"]` |
| `RATE_LIMIT_ENABLED` | Enable rate limiting | `false` | `true` |
| `RATE_LIMIT_RPS` | Requests per second limit | `100` | `1000` |

## Configuration Examples

### Development Environment
```bash
# .env file
QUERY_HTTP_WRAPPER_PORT=8083
QUERY_HTTP_WRAPPER_HOST=localhost
LOG_LEVEL=debug
ENABLE_REQUEST_LOGGING=true
QUERY_SERVICE_ADDRESS=localhost:8002
QUERY_CACHE_ENABLED=true
```

### Production Environment
```bash
# Environment variables
export QUERY_HTTP_WRAPPER_PORT=8080
export QUERY_HTTP_WRAPPER_HOST=0.0.0.0
export LOG_LEVEL=info
export LOG_FORMAT=json
export ENABLE_AUTH=true
export CORS_ORIGINS=["https://app.example.com","https://dashboard.example.com"]
export RATE_LIMIT_ENABLED=true
export RATE_LIMIT_RPS=1000
export QUERY_SERVICE_ADDRESS=query-service:8002
export QUERY_CACHE_ENABLED=true
export QUERY_CACHE_TTL=600s
```

### Docker Configuration
```yaml
# docker-compose.yml
version: '3.8'
services:
  query-service:
    build:
      context: .
      dockerfile: deployments/docker/Dockerfile.query-server
    ports:
      - "8002:8002"
    environment:
      - LOG_LEVEL=info

  query-http-wrapper:
    build:
      context: .
      dockerfile: deployments/docker/Dockerfile.query-http-wrapper
    ports:
      - "8080:8080"
    environment:
      - QUERY_HTTP_WRAPPER_PORT=8080
      - QUERY_HTTP_WRAPPER_HOST=0.0.0.0
      - QUERY_SERVICE_ADDRESS=query-service:8002
      - LOG_LEVEL=info
      - ENABLE_CORS=true
      - QUERY_CACHE_ENABLED=true
    depends_on:
      - query-service
    restart: unless-stopped
```

## Configuration Validation

The Query HTTP Wrapper validates configuration on startup:

### Required Settings
- `QUERY_HTTP_WRAPPER_PORT`: Must be valid port number (1-65535)
- `QUERY_SERVICE_ADDRESS`: Must be valid host:port format

### Optional Settings with Validation
- `QUERY_HTTP_WRAPPER_TIMEOUT`: Must be valid duration string (e.g., "30s", "1m")
- `LOG_LEVEL`: Must be one of: debug, info, warn, error
- `CORS_ORIGINS`: Must be valid URLs or "*"
- `QUERY_CACHE_TTL`: Must be valid duration string

### Validation Errors
If configuration validation fails, the service will:
1. Log the specific validation error
2. Exit with non-zero status code
3. Not start the HTTP server

## Dynamic Configuration

Currently, configuration changes require a service restart. Future versions may support:
- Hot reloading of non-critical settings
- Configuration management via API
- Runtime configuration updates

## Configuration Testing

### Verify Configuration
```bash
# Check current configuration
go run cmd/query-http-wrapper/main.go --config-check

# Test with specific config file
go run cmd/query-http-wrapper/main.go --config config/production.json --config-check
```

### Environment Variable Testing
```bash
# Test with environment variables
QUERY_HTTP_WRAPPER_PORT=9999 go run cmd/query-http-wrapper/main.go --config-check
```

## Troubleshooting

### Common Configuration Issues

1. **Port Already in Use**
   ```
   Error: listen tcp :8083: bind: address already in use
   ```
   Solution: Change `QUERY_HTTP_WRAPPER_PORT` to an available port

2. **Cannot Connect to Query Service**
   ```
   Error: connection refused to query service
   ```
   Solution: Verify `QUERY_SERVICE_ADDRESS` and ensure query service is running

3. **Invalid Timeout Format**
   ```
   Error: invalid timeout format
   ```
   Solution: Use valid Go duration format (e.g., "30s", "1m", "1h")

4. **Query Cache Issues**
   ```
   Warning: query cache disabled due to memory constraints
   ```
   Solution: Adjust `QUERY_CACHE_SIZE` or disable caching with `QUERY_CACHE_ENABLED=false`

5. **Configuration File Not Found**
   ```
   Warning: config file not found, using defaults
   ```
   Solution: Create `config.json` file or specify path with `--config` flag

### Debug Configuration Loading
Enable debug logging to see configuration loading process:
```bash
LOG_LEVEL=debug go run cmd/query-http-wrapper/main.go
```

This will show:
- Configuration sources loaded
- Environment variables detected
- Final merged configuration
- Validation results

## Performance Tuning

### Query Performance
- **Cache Settings**: Tune `QUERY_CACHE_SIZE` and `QUERY_CACHE_TTL`
- **Connection Pool**: Adjust `CONNECTION_POOL_SIZE` based on load
- **Parallel Queries**: Set `QUERY_PARALLEL_QUERIES` for concurrent execution
- **Timeouts**: Balance `QUERY_TIMEOUT` for responsiveness vs. completion

### Memory Management
- **Result Size Limits**: Set `MAX_RESULT_SIZE` to prevent memory issues
- **Cache Size**: Balance `QUERY_CACHE_SIZE` with available memory
- **Connection Limits**: Control `QUERY_MAX_CONNECTIONS`

### Example Performance Configuration
```bash
# High-performance setup
export QUERY_CACHE_SIZE=10000000
export QUERY_CACHE_TTL=600s
export CONNECTION_POOL_SIZE=20
export QUERY_PARALLEL_QUERIES=20
export MAX_RESULT_SIZE=500MB
export QUERY_TIMEOUT=60s
```
