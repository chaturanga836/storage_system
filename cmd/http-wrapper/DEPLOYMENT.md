# HTTP Wrapper Deployment Guide

## Overview
This guide covers deployment options for the HTTP Wrapper service in various environments.

## Prerequisites
- Go 1.21+ installed
- Access to ingestion service (gRPC on port 8001)
- Network access to required ports

## Local Development Deployment

### 1. Quick Start
```bash
# Terminal 1: Start ingestion service
go run cmd/ingestion-server/main.go

# Terminal 2: Start HTTP wrapper
go run cmd/http-wrapper/main.go

# Terminal 3: Test health endpoint
curl http://localhost:8082/health
```

### 2. Custom Configuration
```bash
# Using custom config file
go run cmd/http-wrapper/main.go --config config/development.json

# Using environment variables
HTTP_WRAPPER_PORT=9090 go run cmd/http-wrapper/main.go
```

## Production Deployment

### 1. Binary Deployment

#### Build Binary
```bash
# Build for current platform
go build -o http-wrapper cmd/http-wrapper/main.go

# Build for Linux (if on Windows)
GOOS=linux GOARCH=amd64 go build -o http-wrapper-linux cmd/http-wrapper/main.go

# Build with optimization
go build -ldflags="-s -w" -o http-wrapper cmd/http-wrapper/main.go
```

#### Deploy Binary
```bash
# Copy binary to server
scp http-wrapper user@server:/opt/storage-system/

# Create systemd service
sudo systemctl enable http-wrapper
sudo systemctl start http-wrapper
```

#### Systemd Service File (`/etc/systemd/system/http-wrapper.service`)
```ini
[Unit]
Description=Storage System HTTP Wrapper
After=network.target
Requires=ingestion-service.service

[Service]
Type=simple
User=storage-user
Group=storage-group
WorkingDirectory=/opt/storage-system
ExecStart=/opt/storage-system/http-wrapper
Restart=always
RestartSec=5
Environment=HTTP_WRAPPER_PORT=8080
Environment=HTTP_WRAPPER_HOST=0.0.0.0
Environment=INGESTION_SERVICE_ADDRESS=localhost:8001
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
RUN go build -ldflags="-s -w" -o http-wrapper cmd/http-wrapper/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/http-wrapper .
COPY --from=builder /app/config.json .

EXPOSE 8080

CMD ["./http-wrapper"]
```

#### Build and Run Docker Image
```bash
# Build image
docker build -t storage-http-wrapper .

# Run container
docker run -d \
  --name http-wrapper \
  -p 8080:8080 \
  -e HTTP_WRAPPER_PORT=8080 \
  -e HTTP_WRAPPER_HOST=0.0.0.0 \
  -e INGESTION_SERVICE_ADDRESS=ingestion-service:8001 \
  storage-http-wrapper
```

#### Docker Compose
```yaml
version: '3.8'

services:
  ingestion-service:
    build:
      context: .
      dockerfile: deployments/docker/Dockerfile.ingestion-server
    ports:
      - "8001:8001"
    environment:
      - LOG_LEVEL=info

  http-wrapper:
    build:
      context: .
      dockerfile: deployments/docker/Dockerfile.http-wrapper
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
      - http-wrapper
    restart: unless-stopped
```

### 3. Kubernetes Deployment

#### Deployment YAML
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: http-wrapper
  labels:
    app: http-wrapper
spec:
  replicas: 3
  selector:
    matchLabels:
      app: http-wrapper
  template:
    metadata:
      labels:
        app: http-wrapper
    spec:
      containers:
      - name: http-wrapper
        image: storage-http-wrapper:latest
        ports:
        - containerPort: 8080
        env:
        - name: HTTP_WRAPPER_PORT
          value: "8080"
        - name: HTTP_WRAPPER_HOST
          value: "0.0.0.0"
        - name: INGESTION_SERVICE_ADDRESS
          value: "ingestion-service:8001"
        - name: LOG_LEVEL
          value: "info"
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "500m"
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
  name: http-wrapper-service
spec:
  selector:
    app: http-wrapper
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

#### Deploy to Kubernetes
```bash
# Apply deployment
kubectl apply -f k8s/http-wrapper-deployment.yaml

# Check status
kubectl get pods -l app=http-wrapper
kubectl get services

# View logs
kubectl logs -l app=http-wrapper -f
```

## Load Balancer Configuration

### Nginx Configuration
```nginx
upstream http_wrapper {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
    server 127.0.0.1:8082;
}

server {
    listen 80;
    server_name api.storage.example.com;

    location /health {
        proxy_pass http://http_wrapper;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location /api/ {
        proxy_pass http://http_wrapper;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_timeout 60s;
        proxy_read_timeout 60s;
        proxy_send_timeout 60s;
    }
}
```

### HAProxy Configuration
```
backend http_wrapper
    balance roundrobin
    option httpchk GET /health
    server wrapper1 127.0.0.1:8080 check
    server wrapper2 127.0.0.1:8081 check
    server wrapper3 127.0.0.1:8082 check

frontend http_wrapper_frontend
    bind *:80
    default_backend http_wrapper
```

## Environment-Specific Configurations

### Development
```bash
# .env.development
HTTP_WRAPPER_PORT=8082
LOG_LEVEL=debug
ENABLE_REQUEST_LOGGING=true
CORS_ORIGINS=["http://localhost:3000"]
```

### Staging
```bash
# .env.staging
HTTP_WRAPPER_PORT=8080
HTTP_WRAPPER_HOST=0.0.0.0
LOG_LEVEL=info
ENABLE_AUTH=false
CORS_ORIGINS=["https://staging.example.com"]
RATE_LIMIT_ENABLED=true
```

### Production
```bash
# .env.production
HTTP_WRAPPER_PORT=8080
HTTP_WRAPPER_HOST=0.0.0.0
LOG_LEVEL=warn
LOG_FORMAT=json
ENABLE_AUTH=true
CORS_ORIGINS=["https://app.example.com"]
RATE_LIMIT_ENABLED=true
RATE_LIMIT_RPS=1000
```

## Monitoring and Health Checks

### Health Check Endpoints
- `GET /health` - Basic health status
- `GET /api/v1/status` - Detailed service status

### Monitoring Setup
```bash
# Prometheus metrics (future feature)
curl http://localhost:8082/metrics

# Health check for monitoring
curl -f http://localhost:8082/health || exit 1
```

### Log Aggregation
```yaml
# Filebeat configuration for ELK stack
filebeat.inputs:
- type: log
  paths:
    - /var/log/http-wrapper/*.log
  fields:
    service: http-wrapper
```

## Security Considerations

### 1. Network Security
- Use HTTPS in production (terminate SSL at load balancer)
- Restrict ingestion service access to HTTP wrapper only
- Use private networks between services

### 2. Authentication (Future)
- API key authentication
- JWT token validation
- Rate limiting per client

### 3. Input Validation
- Request size limits (configured via `max_request_size`)
- JSON schema validation
- Sanitize log output

## Performance Tuning

### 1. Go Runtime Tuning
```bash
# Set garbage collection target
export GOGC=100

# Set max OS threads
export GOMAXPROCS=4
```

### 2. Connection Pooling
- Configure gRPC connection pool to ingestion service
- Set appropriate timeouts
- Monitor connection metrics

### 3. Resource Limits
```yaml
# Kubernetes resource limits
resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "1000m"
```

## Troubleshooting

### Common Issues

1. **Service Won't Start**
   ```bash
   # Check port availability
   netstat -tlnp | grep 8082
   
   # Check configuration
   go run cmd/http-wrapper/main.go --config-check
   ```

2. **Cannot Connect to Ingestion Service**
   ```bash
   # Test connectivity
   telnet localhost 8001
   
   # Check ingestion service logs
   journalctl -u ingestion-service -f
   ```

3. **High Memory Usage**
   ```bash
   # Check Go memory stats
   curl http://localhost:8082/debug/pprof/heap
   
   # Analyze with go tool pprof
   go tool pprof http://localhost:8082/debug/pprof/heap
   ```

### Log Analysis
```bash
# Filter error logs
journalctl -u http-wrapper | grep "ERROR"

# Monitor request patterns
tail -f /var/log/http-wrapper/access.log | grep "POST /api/v1/ingest"
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
