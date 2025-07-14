# Query HTTP Wrapper Deployment Guide

## Overview
This guide covers deployment options for the Query HTTP Wrapper service in various environments.

## Prerequisites
- Go 1.21+ installed
- Access to query service (gRPC on port 8002)
- Network access to required ports

## Local Development Deployment

### 1. Quick Start
```bash
# Terminal 1: Start query service
go run cmd/query-server/main.go

# Terminal 2: Start query HTTP wrapper
go run cmd/query-http-wrapper/main.go

# Terminal 3: Test health endpoint
curl http://localhost:8083/health
```

### 2. Custom Configuration
```bash
# Using custom config file
go run cmd/query-http-wrapper/main.go --config config/development.json

# Using environment variables
QUERY_HTTP_WRAPPER_PORT=9090 go run cmd/query-http-wrapper/main.go
```

## Production Deployment

### 1. Binary Deployment

#### Build Binary
```bash
# Build for current platform
go build -o query-http-wrapper cmd/query-http-wrapper/main.go

# Build for Linux (if on Windows)
GOOS=linux GOARCH=amd64 go build -o query-http-wrapper-linux cmd/query-http-wrapper/main.go

# Build with optimization
go build -ldflags="-s -w" -o query-http-wrapper cmd/query-http-wrapper/main.go
```

#### Deploy Binary
```bash
# Copy binary to server
scp query-http-wrapper user@server:/opt/storage-system/

# Create systemd service
sudo systemctl enable query-http-wrapper
sudo systemctl start query-http-wrapper
```

#### Systemd Service File (`/etc/systemd/system/query-http-wrapper.service`)
```ini
[Unit]
Description=Storage System Query HTTP Wrapper
After=network.target
Requires=query-service.service

[Service]
Type=simple
User=storage-user
Group=storage-group
WorkingDirectory=/opt/storage-system
ExecStart=/opt/storage-system/query-http-wrapper
Restart=always
RestartSec=5
Environment=QUERY_HTTP_WRAPPER_PORT=8080
Environment=QUERY_HTTP_WRAPPER_HOST=0.0.0.0
Environment=QUERY_SERVICE_ADDRESS=localhost:8002
Environment=LOG_LEVEL=info

[Install]
WantedBy=multi-user.target
```

### 2. Docker Deployment

#### Dockerfile
```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags="-s -w" -o query-http-wrapper cmd/query-http-wrapper/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/query-http-wrapper .
COPY --from=builder /app/config.json .

EXPOSE 8080

CMD ["./query-http-wrapper"]
```

#### Build and Run Docker Image
```bash
# Build image
docker build -t storage-query-http-wrapper .

# Run container
docker run -d \
  --name query-http-wrapper \
  -p 8080:8080 \
  -e QUERY_HTTP_WRAPPER_PORT=8080 \
  -e QUERY_HTTP_WRAPPER_HOST=0.0.0.0 \
  -e QUERY_SERVICE_ADDRESS=query-service:8002 \
  storage-query-http-wrapper
```

#### Docker Compose
```yaml
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

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl
    depends_on:
      - query-http-wrapper
    restart: unless-stopped
```

### 3. Kubernetes Deployment

#### Deployment YAML
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: query-http-wrapper
  labels:
    app: query-http-wrapper
spec:
  replicas: 3
  selector:
    matchLabels:
      app: query-http-wrapper
  template:
    metadata:
      labels:
        app: query-http-wrapper
    spec:
      containers:
      - name: query-http-wrapper
        image: storage-query-http-wrapper:latest
        ports:
        - containerPort: 8080
        env:
        - name: QUERY_HTTP_WRAPPER_PORT
          value: "8080"
        - name: QUERY_HTTP_WRAPPER_HOST
          value: "0.0.0.0"
        - name: QUERY_SERVICE_ADDRESS
          value: "query-service:8002"
        - name: LOG_LEVEL
          value: "info"
        - name: QUERY_CACHE_ENABLED
          value: "true"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: query-http-wrapper-service
spec:
  selector:
    app: query-http-wrapper
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

#### Deploy to Kubernetes
```bash
# Apply deployment
kubectl apply -f k8s/query-http-wrapper-deployment.yaml

# Check status
kubectl get pods -l app=query-http-wrapper
kubectl get services

# View logs
kubectl logs -l app=query-http-wrapper -f
```

## Load Balancer Configuration

### Nginx Configuration
```nginx
upstream query_http_wrapper {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
    server 127.0.0.1:8082;
}

server {
    listen 80;
    server_name api.query.example.com;

    location /health {
        proxy_pass http://query_http_wrapper;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location /api/ {
        proxy_pass http://query_http_wrapper;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_timeout 120s;
        proxy_read_timeout 120s;
        proxy_send_timeout 120s;
        
        # Increase buffer sizes for large result sets
        proxy_buffer_size 64k;
        proxy_buffers 8 64k;
        proxy_busy_buffers_size 128k;
    }
}
```

### HAProxy Configuration
```
backend query_http_wrapper
    balance roundrobin
    option httpchk GET /health
    server wrapper1 127.0.0.1:8080 check
    server wrapper2 127.0.0.1:8081 check
    server wrapper3 127.0.0.1:8082 check

frontend query_http_wrapper_frontend
    bind *:80
    timeout client 120s
    default_backend query_http_wrapper
```

## Environment-Specific Configurations

### Development
```bash
# .env.development
QUERY_HTTP_WRAPPER_PORT=8083
LOG_LEVEL=debug
ENABLE_REQUEST_LOGGING=true
CORS_ORIGINS=["http://localhost:3000"]
QUERY_CACHE_ENABLED=true
QUERY_CACHE_TTL=60s
```

### Staging
```bash
# .env.staging
QUERY_HTTP_WRAPPER_PORT=8080
QUERY_HTTP_WRAPPER_HOST=0.0.0.0
LOG_LEVEL=info
ENABLE_AUTH=false
CORS_ORIGINS=["https://staging.example.com"]
RATE_LIMIT_ENABLED=true
QUERY_CACHE_ENABLED=true
QUERY_CACHE_TTL=300s
```

### Production
```bash
# .env.production
QUERY_HTTP_WRAPPER_PORT=8080
QUERY_HTTP_WRAPPER_HOST=0.0.0.0
LOG_LEVEL=warn
LOG_FORMAT=json
ENABLE_AUTH=true
CORS_ORIGINS=["https://app.example.com"]
RATE_LIMIT_ENABLED=true
RATE_LIMIT_RPS=1000
QUERY_CACHE_ENABLED=true
QUERY_CACHE_TTL=600s
MAX_RESULT_SIZE=100MB
```

## Monitoring and Health Checks

### Health Check Endpoints
- `GET /health` - Basic health status
- `GET /api/v1/status` - Detailed service status with query metrics

### Monitoring Setup
```bash
# Prometheus metrics (future feature)
curl http://localhost:8083/metrics

# Health check for monitoring
curl -f http://localhost:8083/health || exit 1

# Query performance monitoring
curl http://localhost:8083/api/v1/status | jq '.metrics.avg_latency'
```

### Log Aggregation
```yaml
# Filebeat configuration for ELK stack
filebeat.inputs:
- type: log
  paths:
    - /var/log/query-http-wrapper/*.log
  fields:
    service: query-http-wrapper
    component: query-api
```

## Security Considerations

### 1. Network Security
- Use HTTPS in production (terminate SSL at load balancer)
- Restrict query service access to HTTP wrapper only
- Use private networks between services

### 2. Query Security
- Implement query complexity limits
- Add tenant isolation checks
- Validate query parameters thoroughly

### 3. Authentication (Future)
- API key authentication
- JWT token validation
- Rate limiting per client
- Query permissions per tenant

### 4. Input Validation
- Request size limits (configured via `max_request_size`)
- Query complexity validation
- Result size limits
- Sanitize log output

## Performance Tuning

### 1. Go Runtime Tuning
```bash
# Set garbage collection target
export GOGC=100

# Set max OS threads
export GOMAXPROCS=4

# Memory limit
export GOMEMLIMIT=1GB
```

### 2. Query Optimization
- **Enable query caching**: `QUERY_CACHE_ENABLED=true`
- **Tune cache TTL**: `QUERY_CACHE_TTL=300s`
- **Connection pooling**: `CONNECTION_POOL_SIZE=20`
- **Parallel queries**: `QUERY_PARALLEL_QUERIES=10`

### 3. Resource Limits
```yaml
# Kubernetes resource limits
resources:
  requests:
    memory: "256Mi"
    cpu: "200m"
  limits:
    memory: "1Gi"
    cpu: "2000m"
```

## Troubleshooting

### Common Issues

1. **Service Won't Start**
   ```bash
   # Check port availability
   netstat -tlnp | grep 8083
   
   # Check configuration
   go run cmd/query-http-wrapper/main.go --config-check
   ```

2. **Cannot Connect to Query Service**
   ```bash
   # Test connectivity
   telnet localhost 8002
   
   # Check query service logs
   journalctl -u query-service -f
   ```

3. **High Memory Usage**
   ```bash
   # Check Go memory stats
   curl http://localhost:8083/debug/pprof/heap
   
   # Analyze with go tool pprof
   go tool pprof http://localhost:8083/debug/pprof/heap
   ```

4. **Slow Query Performance**
   ```bash
   # Check query service status
   curl http://localhost:8083/api/v1/status
   
   # Monitor query latency
   tail -f /var/log/query-http-wrapper/access.log | grep "POST /api/v1/query"
   ```

### Log Analysis
```bash
# Filter error logs
journalctl -u query-http-wrapper | grep "ERROR"

# Monitor query patterns
tail -f /var/log/query-http-wrapper/access.log | grep "POST /api/v1/query"

# Check performance metrics
grep "execution_time" /var/log/query-http-wrapper/query.log
```

## Backup and Recovery

### Configuration Backup
- Store configuration files in version control
- Backup environment variable files
- Document deployment procedures

### Service Recovery
- Automated restart via systemd
- Health check monitoring
- Graceful shutdown handling
- Query cache warm-up procedures

## Cache Management

### Query Result Caching
```bash
# Enable caching
QUERY_CACHE_ENABLED=true
QUERY_CACHE_TTL=300s
QUERY_CACHE_SIZE=1000000

# Cache warming (future feature)
curl -X POST http://localhost:8083/api/v1/cache/warm

# Cache statistics
curl http://localhost:8083/api/v1/status | jq '.metrics.cache_status'
```

### Cache Invalidation
- Automatic TTL-based expiration
- Manual cache clearing (future feature)
- Smart invalidation on data changes
