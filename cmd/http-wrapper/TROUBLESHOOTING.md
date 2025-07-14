# HTTP Wrapper Troubleshooting Guide

## Overview
This guide provides solutions for common issues encountered when running the HTTP Wrapper service.

## Quick Diagnostics

### 1. Service Health Check
```bash
# Basic health check
curl -X GET http://localhost:8082/health

# Expected response
{
  "status": "healthy",
  "service": "ingestion-http-wrapper",
  "timestamp": "2025-07-12T14:30:00Z",
  "version": "1.0.0"
}
```

### 2. Service Status Check
```bash
# Detailed status
curl -X GET http://localhost:8082/api/v1/status

# Check if ingestion service is reachable
telnet localhost 8001
```

### 3. Log Analysis
```bash
# Check recent logs
journalctl -u http-wrapper -n 50

# Follow live logs
journalctl -u http-wrapper -f

# Filter for errors
journalctl -u http-wrapper | grep -i error
```

## Common Issues and Solutions

### 1. Service Won't Start

#### Issue: Port Already in Use
```
Error: listen tcp :8082: bind: address already in use
```

**Solutions:**
```bash
# Find process using the port
netstat -tlnp | grep 8082
# or
lsof -i :8082

# Kill the process
kill -9 <PID>

# Or change the port
HTTP_WRAPPER_PORT=8083 go run cmd/http-wrapper/main.go
```

#### Issue: Configuration File Not Found
```
Warning: config file not found, using defaults
```

**Solutions:**
```bash
# Create default config file
echo '{"http_wrapper": {"port": 8082}}' > config.json

# Specify config file path
go run cmd/http-wrapper/main.go --config /path/to/config.json

# Use environment variables instead
HTTP_WRAPPER_PORT=8082 go run cmd/http-wrapper/main.go
```

#### Issue: Permission Denied
```
Error: listen tcp :80: bind: permission denied
```

**Solutions:**
```bash
# Use unprivileged port
HTTP_WRAPPER_PORT=8082 go run cmd/http-wrapper/main.go

# Or run with sudo (not recommended)
sudo go run cmd/http-wrapper/main.go

# Or set capabilities (Linux)
sudo setcap 'cap_net_bind_service=+ep' /path/to/http-wrapper
```

### 2. Cannot Connect to Ingestion Service

#### Issue: Connection Refused
```
Error: connection refused to ingestion service at localhost:8001
```

**Solutions:**
```bash
# 1. Check if ingestion service is running
netstat -tlnp | grep 8001

# 2. Start ingestion service
go run cmd/ingestion-server/main.go

# 3. Verify ingestion service health
grpcurl -plaintext localhost:8001 list

# 4. Check ingestion service logs
journalctl -u ingestion-service -f
```

#### Issue: Service Discovery Failure
```
Error: failed to resolve ingestion service address
```

**Solutions:**
```bash
# 1. Check DNS resolution
nslookup ingestion-service

# 2. Use IP address instead
INGESTION_SERVICE_ADDRESS=127.0.0.1:8001 go run cmd/http-wrapper/main.go

# 3. Check network connectivity
ping ingestion-service
telnet ingestion-service 8001
```

#### Issue: gRPC Timeout
```
Error: context deadline exceeded while calling ingestion service
```

**Solutions:**
```bash
# 1. Increase timeout
INGESTION_SERVICE_TIMEOUT=30s go run cmd/http-wrapper/main.go

# 2. Check ingestion service performance
grpcurl -plaintext -d '{}' localhost:8001 HealthService/Check

# 3. Monitor ingestion service resources
top -p $(pgrep ingestion-server)
```

### 3. Request Validation Issues

#### Issue: Invalid JSON Request
```
Error: invalid character 'x' looking for beginning of value
```

**Solutions:**
```bash
# 1. Validate JSON format
echo '{"test": "data"}' | jq .

# 2. Check Content-Type header
curl -X POST http://localhost:8082/api/v1/ingest/record \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": "test", "record_id": "test", "timestamp": "2025-07-12T14:30:00Z"}'

# 3. Use proper JSON escaping
curl -X POST http://localhost:8082/api/v1/ingest/record \
  -H "Content-Type: application/json" \
  -d @valid_request.json
```

#### Issue: Missing Required Fields
```
Error: Validation failed: tenant_id is required
```

**Solutions:**
```bash
# 1. Include all required fields
{
  "tenant_id": "tenant-123",
  "record_id": "record-456", 
  "timestamp": "2025-07-12T14:30:00Z",
  "data": {
    "tenant_id": "tenant-123",
    "id": "record-456",
    "timestamp": "2025-07-12T14:30:00Z",
    "data": {}
  }
}

# 2. Check API specification
curl http://localhost:8082/api/v1/help
```

#### Issue: Invalid Timestamp Format
```
Error: Validation failed: invalid timestamp format
```

**Solutions:**
```bash
# 1. Use RFC3339 format
"timestamp": "2025-07-12T14:30:00Z"
"timestamp": "2025-07-12T14:30:00.123Z"
"timestamp": "2025-07-12T14:30:00+00:00"

# 2. Generate valid timestamp
date -u +"%Y-%m-%dT%H:%M:%SZ"

# 3. Validate timestamp format
echo "2025-07-12T14:30:00Z" | grep -E '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'
```

### 4. Performance Issues

#### Issue: Slow Response Times
```
Response time: 30+ seconds for simple requests
```

**Solutions:**
```bash
# 1. Check ingestion service performance
time grpcurl -plaintext -d '{}' localhost:8001 IngestionService/IngestRecord

# 2. Monitor system resources
htop
iostat -x 1

# 3. Check for memory leaks
curl http://localhost:8082/debug/pprof/heap

# 4. Analyze with profiling
go tool pprof http://localhost:8082/debug/pprof/profile
```

#### Issue: Memory Usage High
```
Memory usage: 2GB+ for HTTP wrapper
```

**Solutions:**
```bash
# 1. Set garbage collection target
GOGC=50 go run cmd/http-wrapper/main.go

# 2. Profile memory usage
go tool pprof http://localhost:8082/debug/pprof/heap

# 3. Check for goroutine leaks
curl http://localhost:8082/debug/pprof/goroutine

# 4. Set memory limits (Docker)
docker run --memory=512m storage-http-wrapper
```

#### Issue: Connection Pool Exhaustion
```
Error: no available connections to ingestion service
```

**Solutions:**
```bash
# 1. Increase connection pool size
INGESTION_SERVICE_MAX_CONNECTIONS=100 go run cmd/http-wrapper/main.go

# 2. Reduce connection timeout
INGESTION_SERVICE_TIMEOUT=5s go run cmd/http-wrapper/main.go

# 3. Monitor active connections
netstat -an | grep :8001 | wc -l
```

### 5. CORS Issues

#### Issue: CORS Preflight Failure
```
Error: Access to fetch at 'http://localhost:8082' from origin 'http://localhost:3000' 
has been blocked by CORS policy
```

**Solutions:**
```bash
# 1. Enable CORS
ENABLE_CORS=true go run cmd/http-wrapper/main.go

# 2. Set specific origins
CORS_ORIGINS=["http://localhost:3000"] go run cmd/http-wrapper/main.go

# 3. Test with curl (bypass CORS)
curl -X POST http://localhost:8082/api/v1/ingest/record \
  -H "Origin: http://localhost:3000" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": "test"}'
```

### 6. Docker Issues

#### Issue: Container Won't Start
```
Error: container exits immediately
```

**Solutions:**
```bash
# 1. Check container logs
docker logs http-wrapper

# 2. Run interactively
docker run -it storage-http-wrapper /bin/sh

# 3. Check environment variables
docker run storage-http-wrapper env

# 4. Verify image build
docker build -t storage-http-wrapper . --no-cache
```

#### Issue: Service Discovery in Docker
```
Error: cannot resolve ingestion-service hostname
```

**Solutions:**
```bash
# 1. Use Docker Compose networking
version: '3.8'
services:
  ingestion-service:
    # ...
  http-wrapper:
    depends_on:
      - ingestion-service

# 2. Use custom network
docker network create storage-net
docker run --network storage-net --name ingestion-service ...
docker run --network storage-net --name http-wrapper ...

# 3. Use host networking (development only)
docker run --network host storage-http-wrapper
```

## Debug Mode

### Enable Debug Logging
```bash
# Method 1: Environment variable
LOG_LEVEL=debug go run cmd/http-wrapper/main.go

# Method 2: Configuration file
{
  "logging": {
    "level": "debug"
  }
}
```

### Debug Output Analysis
```bash
# Filter debug logs
journalctl -u http-wrapper | grep "DEBUG"

# Monitor request flow
tail -f /var/log/http-wrapper.log | grep -E "(REQUEST|RESPONSE|DEBUG)"
```

## Performance Profiling

### CPU Profiling
```bash
# Start profiling
go tool pprof http://localhost:8082/debug/pprof/profile?seconds=30

# Commands in pprof
(pprof) top10
(pprof) list main.handleIngestRecord
(pprof) web
```

### Memory Profiling
```bash
# Memory profile
go tool pprof http://localhost:8082/debug/pprof/heap

# Commands in pprof
(pprof) top10 -cum
(pprof) list main.
(pprof) web
```

### Goroutine Analysis
```bash
# Check for goroutine leaks
curl http://localhost:8082/debug/pprof/goroutine

# Analyze goroutines
go tool pprof http://localhost:8082/debug/pprof/goroutine
```

## Load Testing

### Simple Load Test
```bash
# Apache Bench
ab -n 1000 -c 10 http://localhost:8082/health

# Wrk
wrk -t4 -c100 -d30s http://localhost:8082/health
```

### API Load Test
```bash
# Create test data file
echo '{"tenant_id":"test","record_id":"test","timestamp":"2025-07-12T14:30:00Z","data":{"tenant_id":"test","id":"test","timestamp":"2025-07-12T14:30:00Z","data":{}}}' > test_record.json

# Load test ingestion endpoint
wrk -t4 -c10 -d10s -s post.lua http://localhost:8082/api/v1/ingest/record

# post.lua script
wrk.method = "POST"
wrk.body = open("test_record.json"):read("*all")
wrk.headers["Content-Type"] = "application/json"
```

## Monitoring Setup

### Basic Monitoring Script
```bash
#!/bin/bash
# monitor.sh

while true; do
  # Health check
  if ! curl -sf http://localhost:8082/health > /dev/null; then
    echo "$(date): Health check failed" >> /var/log/http-wrapper-monitor.log
  fi
  
  # Memory usage
  MEMORY=$(ps -o pid,vsz,rss,comm -p $(pgrep http-wrapper) | tail -1)
  echo "$(date): Memory usage: $MEMORY" >> /var/log/http-wrapper-monitor.log
  
  sleep 60
done
```

### Prometheus Metrics (Future)
```bash
# Metrics endpoint (when implemented)
curl http://localhost:8082/metrics

# Sample metrics
http_requests_total{method="POST",endpoint="/api/v1/ingest/record"} 150
http_request_duration_seconds{method="POST",endpoint="/api/v1/ingest/record"} 0.045
```

## Getting Help

### Enable Verbose Logging
```bash
# Maximum verbosity
LOG_LEVEL=debug ENABLE_REQUEST_LOGGING=true go run cmd/http-wrapper/main.go
```

### Collect Debug Information
```bash
#!/bin/bash
# debug_info.sh

echo "=== System Information ==="
uname -a
go version

echo "=== Service Status ==="
systemctl status http-wrapper

echo "=== Network Status ==="
netstat -tlnp | grep -E "(8082|8001)"

echo "=== Recent Logs ==="
journalctl -u http-wrapper -n 50

echo "=== Configuration ==="
cat config.json

echo "=== Environment Variables ==="
env | grep -E "(HTTP_WRAPPER|INGESTION|LOG)"

echo "=== Process Information ==="
ps aux | grep -E "(http-wrapper|ingestion)"
```

### Community Support
- Check GitHub issues for similar problems
- Search documentation for error messages
- Create detailed bug reports with debug information
- Include reproduction steps and system information
