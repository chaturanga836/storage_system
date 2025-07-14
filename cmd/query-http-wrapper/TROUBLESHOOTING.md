# Query HTTP Wrapper Troubleshooting Guide

## Overview
This guide provides solutions for common issues encountered when running the Query HTTP Wrapper service.

## Quick Diagnostics

### 1. Service Health Check
```bash
# Basic health check
curl -X GET http://localhost:8083/health

# Expected response
{
  "status": "healthy",
  "service": "query-http-wrapper",
  "timestamp": "2025-07-13T14:30:00Z",
  "version": "1.0.0"
}
```

### 2. Service Status Check
```bash
# Detailed status
curl -X GET http://localhost:8083/api/v1/status

# Check if query service is reachable
telnet localhost 8002
```

### 3. Log Analysis
```bash
# Check recent logs
journalctl -u query-http-wrapper -n 50

# Follow live logs
journalctl -u query-http-wrapper -f

# Filter for errors
journalctl -u query-http-wrapper | grep -i error
```

## Common Issues and Solutions

### 1. Service Won't Start

#### Issue: Port Already in Use
```
Error: listen tcp :8083: bind: address already in use
```

**Solutions:**
```bash
# Find process using the port
netstat -tlnp | grep 8083
# or
lsof -i :8083

# Kill the process
kill -9 <PID>

# Or change the port
QUERY_HTTP_WRAPPER_PORT=8084 go run cmd/query-http-wrapper/main.go
```

#### Issue: Configuration File Not Found
```
Warning: config file not found, using defaults
```

**Solutions:**
```bash
# Create default config file
echo '{"query_http_wrapper": {"port": 8083}}' > config.json

# Specify config file path
go run cmd/query-http-wrapper/main.go --config /path/to/config.json

# Use environment variables instead
QUERY_HTTP_WRAPPER_PORT=8083 go run cmd/query-http-wrapper/main.go
```

#### Issue: Permission Denied
```
Error: listen tcp :80: bind: permission denied
```

**Solutions:**
```bash
# Use unprivileged port
QUERY_HTTP_WRAPPER_PORT=8083 go run cmd/query-http-wrapper/main.go

# Or run with sudo (not recommended)
sudo go run cmd/query-http-wrapper/main.go

# Or set capabilities (Linux)
sudo setcap 'cap_net_bind_service=+ep' /path/to/query-http-wrapper
```

### 2. Cannot Connect to Query Service

#### Issue: Connection Refused
```
Error: connection refused to query service at localhost:8002
```

**Solutions:**
```bash
# 1. Check if query service is running
netstat -tlnp | grep 8002

# 2. Start query service
go run cmd/query-server/main.go

# 3. Verify query service health
grpcurl -plaintext localhost:8002 list

# 4. Check query service logs
journalctl -u query-service -f
```

#### Issue: Service Discovery Failure
```
Error: failed to resolve query service address
```

**Solutions:**
```bash
# 1. Check DNS resolution
nslookup query-service

# 2. Use IP address instead
QUERY_SERVICE_ADDRESS=127.0.0.1:8002 go run cmd/query-http-wrapper/main.go

# 3. Check network connectivity
ping query-service
telnet query-service 8002
```

#### Issue: gRPC Timeout
```
Error: context deadline exceeded while calling query service
```

**Solutions:**
```bash
# 1. Increase timeout
QUERY_SERVICE_TIMEOUT=60s go run cmd/query-http-wrapper/main.go

# 2. Check query service performance
grpcurl -plaintext -d '{}' localhost:8002 QueryService/HealthCheck

# 3. Monitor query service resources
top -p $(pgrep query-server)
```

### 3. Query Execution Issues

#### Issue: Invalid Query Parameters
```
Error: Invalid request format: unknown filter operator
```

**Solutions:**
```bash
# 1. Check valid operators
# Valid: eq, ne, gt, lt, gte, lte, in, contains

# 2. Example valid query
curl -X POST http://localhost:8083/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "test",
    "filters": [
      {"field": "status", "operator": "eq", "value": "active"}
    ]
  }'

# 3. Check API specification
curl http://localhost:8083/api/v1/help
```

#### Issue: Query Timeout
```
Error: Query execution failed: context deadline exceeded
```

**Solutions:**
```bash
# 1. Increase query timeout
QUERY_TIMEOUT=60s go run cmd/query-http-wrapper/main.go

# 2. Optimize query with filters
{
  "tenant_id": "test",
  "filters": [
    {"field": "indexed_field", "operator": "eq", "value": "value"}
  ],
  "limit": 100
}

# 3. Use pagination for large results
{
  "tenant_id": "test",
  "limit": 50,
  "offset": 0
}
```

#### Issue: Large Result Set Memory Issues
```
Error: result set too large, out of memory
```

**Solutions:**
```bash
# 1. Set result size limits
MAX_RESULT_SIZE=50MB go run cmd/query-http-wrapper/main.go

# 2. Use pagination
{
  "tenant_id": "test",
  "limit": 100,
  "offset": 0
}

# 3. Use projections to reduce data
{
  "tenant_id": "test",
  "projection": ["id", "name"],
  "limit": 1000
}
```

### 4. Performance Issues

#### Issue: Slow Query Response Times
```
Response time: 30+ seconds for simple queries
```

**Solutions:**
```bash
# 1. Check query service performance
time grpcurl -plaintext -d '{}' localhost:8002 QueryService/Query

# 2. Monitor system resources
htop
iostat -x 1

# 3. Enable query caching
QUERY_CACHE_ENABLED=true QUERY_CACHE_TTL=300s go run cmd/query-http-wrapper/main.go

# 4. Check query execution plan
curl -X POST http://localhost:8083/api/v1/query/explain \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": "test", "filters": [...]}'
```

#### Issue: High Memory Usage
```
Memory usage: 2GB+ for query wrapper
```

**Solutions:**
```bash
# 1. Set garbage collection target
GOGC=50 go run cmd/query-http-wrapper/main.go

# 2. Profile memory usage
go tool pprof http://localhost:8083/debug/pprof/heap

# 3. Check for memory leaks
curl http://localhost:8083/debug/pprof/goroutine

# 4. Set memory limits (Docker)
docker run --memory=512m storage-query-http-wrapper

# 5. Reduce cache size
QUERY_CACHE_SIZE=500000 go run cmd/query-http-wrapper/main.go
```

#### Issue: Connection Pool Exhaustion
```
Error: no available connections to query service
```

**Solutions:**
```bash
# 1. Increase connection pool size
CONNECTION_POOL_SIZE=20 go run cmd/query-http-wrapper/main.go

# 2. Reduce connection timeout
QUERY_SERVICE_TIMEOUT=10s go run cmd/query-http-wrapper/main.go

# 3. Monitor active connections
netstat -an | grep :8002 | wc -l
```

### 5. Caching Issues

#### Issue: Cache Not Working
```
Warning: query cache disabled, all queries hitting service
```

**Solutions:**
```bash
# 1. Enable caching explicitly
QUERY_CACHE_ENABLED=true go run cmd/query-http-wrapper/main.go

# 2. Check cache configuration
curl http://localhost:8083/api/v1/status | jq '.metrics.cache_status'

# 3. Verify cache size
QUERY_CACHE_SIZE=1000000 go run cmd/query-http-wrapper/main.go

# 4. Check memory availability
free -h
```

#### Issue: Stale Cache Results
```
Error: receiving outdated data from cache
```

**Solutions:**
```bash
# 1. Reduce cache TTL
QUERY_CACHE_TTL=60s go run cmd/query-http-wrapper/main.go

# 2. Clear cache (restart service for now)
systemctl restart query-http-wrapper

# 3. Disable caching for critical queries
{
  "tenant_id": "test",
  "options": {
    "disable_cache": "true"
  }
}
```

### 6. CORS Issues

#### Issue: CORS Preflight Failure
```
Error: Access to fetch at 'http://localhost:8083' from origin 'http://localhost:3000' 
has been blocked by CORS policy
```

**Solutions:**
```bash
# 1. Enable CORS
ENABLE_CORS=true go run cmd/query-http-wrapper/main.go

# 2. Set specific origins
CORS_ORIGINS=["http://localhost:3000"] go run cmd/query-http-wrapper/main.go

# 3. Test with curl (bypass CORS)
curl -X POST http://localhost:8083/api/v1/query \
  -H "Origin: http://localhost:3000" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": "test"}'
```

### 7. Docker Issues

#### Issue: Container Won't Start
```
Error: container exits immediately
```

**Solutions:**
```bash
# 1. Check container logs
docker logs query-http-wrapper

# 2. Run interactively
docker run -it storage-query-http-wrapper /bin/sh

# 3. Check environment variables
docker run storage-query-http-wrapper env

# 4. Verify image build
docker build -t storage-query-http-wrapper . --no-cache
```

#### Issue: Service Discovery in Docker
```
Error: cannot resolve query-service hostname
```

**Solutions:**
```bash
# 1. Use Docker Compose networking
version: '3.8'
services:
  query-service:
    # ...
  query-http-wrapper:
    depends_on:
      - query-service

# 2. Use custom network
docker network create storage-net
docker run --network storage-net --name query-service ...
docker run --network storage-net --name query-http-wrapper ...

# 3. Use host networking (development only)
docker run --network host storage-query-http-wrapper
```

## Debug Mode

### Enable Debug Logging
```bash
# Method 1: Environment variable
LOG_LEVEL=debug go run cmd/query-http-wrapper/main.go

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
journalctl -u query-http-wrapper | grep "DEBUG"

# Monitor request flow
tail -f /var/log/query-http-wrapper.log | grep -E "(REQUEST|RESPONSE|DEBUG)"

# Query execution tracing
grep "🔍\|📄\|📊" /var/log/query-http-wrapper.log
```

## Performance Profiling

### CPU Profiling
```bash
# Start profiling
go tool pprof http://localhost:8083/debug/pprof/profile?seconds=30

# Commands in pprof
(pprof) top10
(pprof) list main.executeQuery
(pprof) web
```

### Memory Profiling
```bash
# Memory profile
go tool pprof http://localhost:8083/debug/pprof/heap

# Commands in pprof
(pprof) top10 -cum
(pprof) list main.
(pprof) web
```

### Goroutine Analysis
```bash
# Check for goroutine leaks
curl http://localhost:8083/debug/pprof/goroutine

# Analyze goroutines
go tool pprof http://localhost:8083/debug/pprof/goroutine
```

## Load Testing

### Simple Load Test
```bash
# Apache Bench
ab -n 1000 -c 10 http://localhost:8083/health

# Wrk
wrk -t4 -c100 -d30s http://localhost:8083/health
```

### Query Load Test
```bash
# Create test query file
echo '{
  "tenant_id": "test",
  "filters": [
    {"field": "status", "operator": "eq", "value": "active"}
  ],
  "limit": 10
}' > test_query.json

# Load test query endpoint
wrk -t4 -c10 -d10s -s post.lua http://localhost:8083/api/v1/query

# post.lua script
wrk.method = "POST"
wrk.body = open("test_query.json"):read("*all")
wrk.headers["Content-Type"] = "application/json"
```

### Aggregation Load Test
```bash
# Create aggregation test
echo '{
  "tenant_id": "test",
  "aggregation": "count",
  "filters": [
    {"field": "category", "operator": "eq", "value": "A"}
  ]
}' > test_aggregate.json

# Load test aggregation endpoint
wrk -t2 -c5 -d5s -s post.lua http://localhost:8083/api/v1/query/aggregate
```

## Monitoring Setup

### Basic Monitoring Script
```bash
#!/bin/bash
# monitor.sh

while true; do
  # Health check
  if ! curl -sf http://localhost:8083/health > /dev/null; then
    echo "$(date): Health check failed" >> /var/log/query-http-wrapper-monitor.log
  fi
  
  # Memory usage
  MEMORY=$(ps -o pid,vsz,rss,comm -p $(pgrep query-http-wrapper) | tail -1)
  echo "$(date): Memory usage: $MEMORY" >> /var/log/query-http-wrapper-monitor.log
  
  # Query performance
  PERF=$(curl -s http://localhost:8083/api/v1/status | jq -r '.metrics.avg_latency')
  echo "$(date): Avg latency: $PERF" >> /var/log/query-http-wrapper-monitor.log
  
  sleep 60
done
```

### Query Performance Monitoring
```bash
# Monitor slow queries
tail -f /var/log/query-http-wrapper.log | grep -E "execution_time.*[5-9][0-9][0-9]ms|[0-9]+s"

# Cache hit rate
curl -s http://localhost:8083/api/v1/status | jq '.metrics.cache_hit_rate'

# Connection pool status
curl -s http://localhost:8083/api/v1/status | jq '.metrics.connection_pool'
```

### Prometheus Metrics (Future)
```bash
# Metrics endpoint (when implemented)
curl http://localhost:8083/metrics

# Sample metrics
http_requests_total{method="POST",endpoint="/api/v1/query"} 150
http_request_duration_seconds{method="POST",endpoint="/api/v1/query"} 0.245
query_cache_hits_total 89
query_cache_misses_total 61
```

## Getting Help

### Enable Verbose Logging
```bash
# Maximum verbosity
LOG_LEVEL=debug ENABLE_REQUEST_LOGGING=true go run cmd/query-http-wrapper/main.go
```

### Collect Debug Information
```bash
#!/bin/bash
# debug_info.sh

echo "=== System Information ==="
uname -a
go version

echo "=== Service Status ==="
systemctl status query-http-wrapper

echo "=== Network Status ==="
netstat -tlnp | grep -E "(8083|8002)"

echo "=== Recent Logs ==="
journalctl -u query-http-wrapper -n 50

echo "=== Configuration ==="
cat config.json

echo "=== Environment Variables ==="
env | grep -E "(QUERY_HTTP_WRAPPER|QUERY|LOG)"

echo "=== Process Information ==="
ps aux | grep -E "(query-http-wrapper|query-server)"

echo "=== Memory Usage ==="
free -h
ps -o pid,vsz,rss,comm -p $(pgrep query-http-wrapper)

echo "=== Query Service Status ==="
curl -s http://localhost:8083/api/v1/status | jq .
```

### Community Support
- Check GitHub issues for similar problems
- Search documentation for error messages
- Create detailed bug reports with debug information
- Include reproduction steps and system information
- Provide query examples that cause issues
