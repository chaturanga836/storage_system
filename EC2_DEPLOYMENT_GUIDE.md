# EC2 Deployment Guide - Optimized Docker Storage System

## Prerequisites
- EC2 instance with Docker and Docker Compose installed
- Git access to your repository
- Sufficient disk space (at least 10GB free)

## Step 1: Update Your Repository

First, commit and push all the changes to your repository:

```bash
# On your local machine
cd d:\python\storage_system
git add .
git commit -m "Add optimized Docker setup with shared base image"
git push origin main
```

## Step 2: Pull Changes on EC2

Connect to your EC2 instance and pull the latest changes:

```bash
# SSH to your EC2 instance
ssh -i your-key.pem ubuntu@your-ec2-ip

# Navigate to your project directory
cd /path/to/your/storage_system

# Pull the latest changes
git pull origin main

# Verify the new files are present
ls -la Dockerfile.base
ls -la */Dockerfile.optimized
```

## Step 3: Build and Deploy

### Option A: Use the Automated Script

```bash
# Make the script executable
chmod +x build_optimized.sh

# Run the automated build and deployment
./build_optimized.sh
```

### Option B: Manual Step-by-Step

```bash
# 1. Stop any running services
docker-compose down --volumes --remove-orphans

# 2. Clean up old images (optional but recommended)
docker system prune -a --volumes -f

# 3. Build the shared base image
docker build -f Dockerfile.base -t storage-python-base:latest .

# 4. Build all services with optimized Dockerfiles
docker-compose build --no-cache

# 5. Start all services
docker-compose up -d

# 6. Check status
docker-compose ps
```

## Step 4: Verify Deployment

Check that all services are running:

```bash
# Check container status
docker-compose ps

# Check logs for any issues
docker-compose logs

# Test endpoints
curl http://localhost:8080/health  # auth-gateway
curl http://localhost:8001/health  # tenant-node
curl http://localhost:8087/health  # metadata-catalog
```

## Troubleshooting

### Common Issues and Quick Fixes:

#### 1. Tenant-Node Import Error (Most Common)
If you see `ImportError: attempted relative import with no known parent package` for tenant-node:

```bash
# Quick fix with the provided script
chmod +x fix_tenant_node.sh
./fix_tenant_node.sh
```

**OR** manually fix:
```bash
# Fix the import statements
sed -i 's/from \.config import/from config import/g' tenant-node/grpc_service.py
sed -i 's/from \.data_source import/from data_source import/g' tenant-node/grpc_service.py
sed -i 's/from \.config import/from config import/g' tenant-node/rest_api.py
sed -i 's/from \.data_source import/from data_source import/g' tenant-node/rest_api.py

# Rebuild and restart
docker-compose build tenant-node
docker-compose restart tenant-node
```

#### 2. Docker Compose Version Warning
To remove the version warning:
```bash
sed -i '/^version:/d' docker-compose.yml
```

### If build fails:
```bash
# Check Docker daemon is running
sudo systemctl status docker

# Check disk space
df -h

# Check Docker permissions
docker ps
```

### If services don't start:
```bash
# Check individual service logs
docker-compose logs auth-gateway
docker-compose logs tenant-node
docker-compose logs metadata-catalog

# Restart specific service
docker-compose restart service-name
```

### Performance Monitoring:
```bash
# Monitor resource usage
docker stats

# Check build times
time docker-compose build service-name
```

## Expected Build Time Improvements

- **First build** (with base image): ~8-12 minutes total
- **Subsequent builds**: ~3-5 minutes total
- **Code-only changes**: ~1-2 minutes total

## Services and Ports

After successful deployment, these services will be available:

- **auth-gateway**: http://your-ec2-ip:8080
- **operation-node**: http://your-ec2-ip:8086
- **cbo-engine**: http://your-ec2-ip:8088
- **metadata-catalog**: http://your-ec2-ip:8087
- **monitoring**: http://your-ec2-ip:8089
- **tenant-node**: http://your-ec2-ip:8001
- **grafana**: http://your-ec2-ip:3000
- **prometheus**: http://your-ec2-ip:9090

## Security Note

Make sure to configure your EC2 security group to allow inbound traffic on the required ports, or use a reverse proxy/load balancer for production deployments.
