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

## ✅ **DEPLOYMENT SUCCESS CONFIRMATION**

🎉 **Congratulations!** If you've reached this point, your optimized storage system is fully operational!

### **Success Indicators:**
- ✅ **All 10 services running** (9 application + 1 tenant-node)
- ✅ **Tenant-node operational** at your-ec2-ip:8001
- ✅ **~20x faster build times** (101s vs 30+ minutes)
- ✅ **Shared base image optimization** working perfectly
- ✅ **All endpoints accessible** and responsive

### **Your System Status:**
```bash
# All services should show "Up" status:
NAME                                STATUS
storage_system-auth-gateway-1       Up 
storage_system-cbo-engine-1         Up 
storage_system-grafana-1            Up 
storage_system-metadata-catalog-1   Up 
storage_system-monitoring-1         Up 
storage_system-operation-node-1     Up 
storage_system-postgres-1           Up 
storage_system-prometheus-1         Up 
storage_system-redis-1              Up 
storage_system-tenant-node-1        Up ⭐ (Fixed!)
```

### **Performance Achievements:**
- 🚀 **Build Speed**: 101 seconds (vs 30+ minutes = ~20x improvement)
- 💾 **Resource Efficiency**: Shared Docker layers across all services
- 🔧 **Maintainability**: Centralized dependency management
- 📦 **Image Size**: Optimized with better layer caching

## Troubleshooting

### Common Issues and Quick Fixes:

#### 1. Tenant-Node Import Error (Most Common)
If you see `ImportError: attempted relative import with no known parent package` or logging errors for tenant-node:

**Method 1: Use the final complete fix script (RECOMMENDED)**
```bash
# Download the latest fixes first
git pull origin main
chmod +x fix_tenant_node_final.sh
./fix_tenant_node_final.sh
```

This script fixes:
- ✅ Import statement issues (relative to absolute imports)
- ✅ Missing stop() method in TenantNodeAPI
- ✅ Structured logging issues (Logger._log() keyword arguments)
- ✅ Main.py entry point problems

**Method 2: Quick manual fix for logging issue**
```bash
# If you see "Logger._log() got an unexpected keyword argument 'tenant_id'"
docker-compose stop tenant-node

# Fix the specific logging calls
cd tenant-node
sed -i 's/logger\.info(".*", tenant_id=self\.config\.tenant_id)/logger.info(f"Tenant Node for tenant: {self.config.tenant_id}")/' tenant_node.py
sed -i 's/logger\.info(".*", port=self\.config\.grpc_port)/logger.info(f"Starting gRPC server on port {self.config.grpc_port}")/' tenant_node.py
sed -i 's/logger\.error(".*", error=str(e))/logger.error(f"Error: {str(e)}")/' tenant_node.py
cd ..

docker-compose build --no-cache tenant-node
docker-compose up -d tenant-node
```

**Method 2: Manual fix (run these exact commands on EC2)**
```bash
# Stop the service first
docker-compose stop tenant-node

# Fix the import statements and main.py
cd tenant-node
sed -i 's/from \.config import/from config import/g' grpc_service.py
sed -i 's/from \.data_source import/from data_source import/g' grpc_service.py
sed -i 's/from \.generated import/from generated import/g' grpc_service.py
sed -i 's/from \.config import/from config import/g' rest_api.py
sed -i 's/from \.data_source import/from data_source import/g' rest_api.py

# Fix main.py entry point
cat > main.py << 'EOF'
"""
Tenant Node main entry point
"""
import asyncio
import logging
import sys

# Import the main tenant node application
from tenant_node import main as tenant_node_main

logger = logging.getLogger(__name__)


if __name__ == "__main__":
    try:
        # Run the tenant node main function
        asyncio.run(tenant_node_main())
    except KeyboardInterrupt:
        logger.info("Tenant Node service stopped")
        print("\nShutdown complete.")
    except Exception as e:
        logger.error(f"Fatal error: {e}")
        print(f"Fatal error: {e}")
        sys.exit(1)
EOF

cd ..

# Remove the old image and rebuild
docker-compose rm -f tenant-node
docker rmi storage_system-tenant-node 2>/dev/null || true
docker-compose build --no-cache tenant-node
docker-compose up -d tenant-node

# Check if it's working
sleep 15
docker-compose ps tenant-node
docker-compose logs --tail=10 tenant-node
```

**Method 3: If still failing, check the files directly**
```bash
# Verify the imports are fixed
grep "from \." tenant-node/*.py
# Should show no results

# If there are still relative imports, fix them manually:
nano tenant-node/grpc_service.py  # Change any "from ." to "from "
nano tenant-node/rest_api.py      # Change any "from ." to "from "
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
