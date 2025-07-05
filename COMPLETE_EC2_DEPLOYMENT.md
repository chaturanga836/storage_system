# Complete EC2 Deployment Process - Optimized Storage System

## Overview

This guide provides a complete process to deploy the optimized Docker storage system to your EC2 instance with the new shared base image architecture.

## What's New

### ✅ Optimizations Added:
- **Shared base image** (`storage-python-base:latest`) with common dependencies
- **Optimized Dockerfiles** for all Python services (6 services)
- **Faster build times** (5-10x improvement for rebuilds)
- **Automated deployment scripts** for EC2
- **Better resource utilization** and layer caching

### 📁 New Files Created:
- `Dockerfile.base` - Shared base image with common dependencies
- `*/Dockerfile.optimized` - Optimized Dockerfiles for each service
- `deploy_ec2.sh` - Automated deployment script (Linux/Bash)
- `deploy_ec2.ps1` - Automated deployment script (PowerShell)
- `EC2_DEPLOYMENT_GUIDE.md` - Comprehensive deployment guide
- `EC2_QUICK_DEPLOY.md` - Quick deployment commands
- `DOCKER_OPTIMIZATION_IMPACT.md` - Detailed impact analysis

## Step-by-Step Deployment

### 1. Local Preparation (Windows Machine)

```powershell
# Navigate to your project directory
cd d:\python\storage_system

# Stage all changes
git add .

# Commit the optimizations
git commit -m "Add optimized Docker setup with shared base image and EC2 deployment"

# Push to your repository
git push origin main
```

### 2. EC2 Connection and Preparation

```bash
# SSH to your EC2 instance
ssh -i your-key.pem ubuntu@your-ec2-ip

# Navigate to your project directory
cd storage_system  # or wherever you cloned the repo

# Pull the latest changes
git pull origin main

# Verify new files are present
ls -la Dockerfile.base
ls -la */Dockerfile.optimized
ls -la deploy_ec2.sh
```

### 3. Automated Deployment (Recommended)

```bash
# Make the deployment script executable
chmod +x deploy_ec2.sh

# Run the automated deployment
./deploy_ec2.sh
```

The script will:
1. ✅ Check prerequisites (Docker, Docker Compose, permissions)
2. ✅ Check system resources (disk space, memory)
3. ✅ Create backup of current state
4. ✅ Clean up old Docker resources (optional)
5. ✅ Build shared base image (~3-5 minutes)
6. ✅ Build all services using optimized Dockerfiles (~3-5 minutes)
7. ✅ Start all services
8. ✅ Verify deployment and show status
9. ✅ Display service URLs and management commands

### 4. Manual Deployment (Alternative)

If you prefer manual control:

```bash
# Stop existing services
docker-compose down --volumes --remove-orphans

# Optional: Clean up old images
docker system prune -a --volumes -f

# Build shared base image
docker build -f Dockerfile.base -t storage-python-base:latest .

# Build all services
docker-compose build --no-cache

# Start all services
docker-compose up -d

# Check status
docker-compose ps
```

## Expected Results

### Build Time Improvements:
- **Before**: 30-45 minutes for clean build
- **After**: 8-12 minutes for clean build
- **Rebuilds**: 3-5 minutes (vs 15-25 minutes before)
- **Code changes**: 1-2 minutes (vs 5-10 minutes before)

### Resource Usage:
- **Memory**: ~2-4GB total (similar to before)
- **Disk**: ~3-5GB Docker images (better utilization)
- **CPU**: Lower during builds due to shared layers

### Services Running:
```
SERVICE              STATUS    PORTS
auth-gateway         Up        0.0.0.0:8080->8080/tcp
operation-node       Up        0.0.0.0:8086->8081/tcp, 0.0.0.0:50054->50054/tcp
cbo-engine           Up        0.0.0.0:8088->8082/tcp, 0.0.0.0:50052->50052/tcp
metadata-catalog     Up        0.0.0.0:8087->8083/tcp, 0.0.0.0:50053->50053/tcp
monitoring           Up        0.0.0.0:8089->8084/tcp
tenant-node          Up        0.0.0.0:8001->8000/tcp, 0.0.0.0:50051->50051/tcp
postgres             Up        5432/tcp
redis                Up        6379/tcp
prometheus           Up        0.0.0.0:9090->9090/tcp
grafana              Up        0.0.0.0:3000->3000/tcp
```

## Verification Steps

### 1. Check Container Status
```bash
docker-compose ps
```
All services should show "Up" status.

### 2. Test Service Endpoints
```bash
# Replace YOUR_EC2_IP with your actual EC2 public IP
curl http://YOUR_EC2_IP:8080/health  # auth-gateway
curl http://YOUR_EC2_IP:8001/health  # tenant-node
curl http://YOUR_EC2_IP:8087/health  # metadata-catalog
```

### 3. Check Logs (if any service fails)
```bash
# Check all logs
docker-compose logs

# Check specific service
docker-compose logs tenant-node
docker-compose logs auth-gateway
```

### 4. Monitor Resources
```bash
# Real-time resource usage
docker stats

# Disk usage
docker system df
```

## Troubleshooting

### Common Issues and Solutions:

#### 1. Permission Denied (Docker)
```bash
# Add user to docker group
sudo usermod -aG docker $USER
newgrp docker

# Or use sudo
sudo docker-compose up -d
```

#### 2. Port Already in Use
```bash
# Check what's using the port
netstat -tulpn | grep :8080

# Kill process or change port in docker-compose.yml
```

#### 3. Service Won't Start
```bash
# Check specific service logs
docker-compose logs service-name

# Restart specific service
docker-compose restart service-name
```

#### 4. Out of Disk Space
```bash
# Clean up Docker resources
docker system prune -a --volumes -f

# Check disk usage
df -h
```

#### 5. Build Fails
```bash
# Check Docker daemon
sudo systemctl status docker

# Try building base image separately
docker build -f Dockerfile.base -t storage-python-base:latest .
```

## Performance Monitoring

### Service URLs (replace with your EC2 IP):
- **Main Services**: http://YOUR_EC2_IP:8001 (tenant-node)
- **Monitoring**: http://YOUR_EC2_IP:3000 (Grafana - admin/admin)
- **Metrics**: http://YOUR_EC2_IP:9090 (Prometheus)

### Management Commands:
```bash
# View all logs
docker-compose logs

# Follow logs in real-time
docker-compose logs -f

# Restart all services
docker-compose restart

# Stop all services
docker-compose down

# Full rebuild
docker-compose build --no-cache && docker-compose up -d
```

## Success Criteria

✅ **Deployment Successful When**:
- All 10 containers show "Up" status
- No error messages in logs
- Service endpoints respond
- Build time < 15 minutes
- Memory usage < 4GB
- All ports accessible

🎉 **You're Done!** Your optimized storage system is now running on EC2 with improved build performance and better resource utilization.
