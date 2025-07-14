# HTTP Wrapper Configuration

## Overview
The HTTP Wrapper uses a layered configuration system that supports both file-based and environment variable configuration.

## Configuration Sources (Priority Order)
1. Environment variables (highest priority)
2. Configuration file (`config.json`)
3. Default values (lowest priority)

## Configuration File Format

### Basic Configuration (`config.json`)
```json
{
  "http_wrapper": {
    "port": 8082,
    "host": "localhost",
    "timeout": "30s",
    "enable_cors": true,
    "log_level": "info",
    "max_request_size": "10MB"
  },
  "ingestion_service": {
    "address": "localhost:8001",
    "timeout": "10s",
    "max_retry_attempts": 3,
    "retry_delay": "1s"
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
  }
}
```

## Environment Variables

### HTTP Wrapper Settings
| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `HTTP_WRAPPER_PORT` | HTTP server port | `8082` | `8082` |
| `HTTP_WRAPPER_HOST` | HTTP server host | `localhost` | `0.0.0.0` |
| `HTTP_WRAPPER_TIMEOUT` | Request timeout | `30s` | `60s` |
| `HTTP_WRAPPER_ENABLE_CORS` | Enable CORS | `true` | `false` |
| `HTTP_WRAPPER_LOG_LEVEL` | Log level | `info` | `debug` |
| `HTTP_WRAPPER_MAX_REQUEST_SIZE` | Max request size | `10MB` | `50MB` |

### Ingestion Service Settings
| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `INGESTION_SERVICE_ADDRESS` | gRPC service address | `localhost:8001` | `ingestion:8001` |
| `INGESTION_SERVICE_TIMEOUT` | Service call timeout | `10s` | `30s` |
| `INGESTION_SERVICE_MAX_RETRY` | Max retry attempts | `3` | `5` |
| `INGESTION_SERVICE_RETRY_DELAY` | Retry delay | `1s` | `2s` |

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
HTTP_WRAPPER_PORT=8082
HTTP_WRAPPER_HOST=localhost
LOG_LEVEL=debug
ENABLE_REQUEST_LOGGING=true
INGESTION_SERVICE_ADDRESS=localhost:8001
```

### Production Environment
```bash
# Environment variables
export HTTP_WRAPPER_PORT=8080
export HTTP_WRAPPER_HOST=0.0.0.0
export LOG_LEVEL=info
export LOG_FORMAT=json
export ENABLE_AUTH=true
export CORS_ORIGINS=["https://app.example.com","https://dashboard.example.com"]
export RATE_LIMIT_ENABLED=true
export RATE_LIMIT_RPS=1000
export INGESTION_SERVICE_ADDRESS=ingestion-service:8001
```

### Docker Configuration
```yaml
# docker-compose.yml
version: '3.8'
services:
  http-wrapper:
    build: .
    ports:
      - "8080:8080"
    environment:
      - HTTP_WRAPPER_PORT=8080
      - HTTP_WRAPPER_HOST=0.0.0.0
      - INGESTION_SERVICE_ADDRESS=ingestion-service:8001
      - LOG_LEVEL=info
      - ENABLE_CORS=true
    depends_on:
      - ingestion-service
```

## Configuration Validation

The HTTP Wrapper validates configuration on startup:

### Required Settings
- `HTTP_WRAPPER_PORT`: Must be valid port number (1-65535)
- `INGESTION_SERVICE_ADDRESS`: Must be valid host:port format

### Optional Settings with Validation
- `HTTP_WRAPPER_TIMEOUT`: Must be valid duration string (e.g., "30s", "1m")
- `LOG_LEVEL`: Must be one of: debug, info, warn, error
- `CORS_ORIGINS`: Must be valid URLs or "*"

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
go run cmd/http-wrapper/main.go --config-check

# Test with specific config file
go run cmd/http-wrapper/main.go --config config/production.json --config-check
```

### Environment Variable Testing
```bash
# Test with environment variables
HTTP_WRAPPER_PORT=9999 go run cmd/http-wrapper/main.go --config-check
```

## Troubleshooting

### Common Configuration Issues

1. **Port Already in Use**
   ```
   Error: listen tcp :8082: bind: address already in use
   ```
   Solution: Change `HTTP_WRAPPER_PORT` to an available port

2. **Cannot Connect to Ingestion Service**
   ```
   Error: connection refused to ingestion service
   ```
   Solution: Verify `INGESTION_SERVICE_ADDRESS` and ensure ingestion service is running

3. **Invalid Timeout Format**
   ```
   Error: invalid timeout format
   ```
   Solution: Use valid Go duration format (e.g., "30s", "1m", "1h")

4. **Configuration File Not Found**
   ```
   Warning: config file not found, using defaults
   ```
   Solution: Create `config.json` file or specify path with `--config` flag

### Debug Configuration Loading
Enable debug logging to see configuration loading process:
```bash
LOG_LEVEL=debug go run cmd/http-wrapper/main.go
```

This will show:
- Configuration sources loaded
- Environment variables detected
- Final merged configuration
- Validation results
