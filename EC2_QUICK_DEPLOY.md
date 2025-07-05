# Quick EC2 Deployment Commands

## 1. Commit and Push Changes (on your local machine)

```powershell
# In your local storage_system directory
cd d:\python\storage_system
git add .
git commit -m "Add optimized Docker setup with shared base image and EC2 deployment scripts"
git push origin main
```

## 2. Connect to EC2 and Deploy

```bash
# SSH to your EC2 instance
ssh -i your-key.pem ubuntu@your-ec2-ip

# Navigate to your project directory
cd storage_system  # or wherever you cloned the repo

# Pull the latest changes
git pull origin main

# Make deployment script executable
chmod +x deploy_ec2.sh

# Run the automated deployment
./deploy_ec2.sh
```

## 3. Alternative: Manual Commands

If you prefer to run commands manually:

```bash
# Pull changes
git pull origin main

# Stop existing services
docker-compose down --volumes --remove-orphans

# Clean up (optional)
docker system prune -a --volumes -f

# Build base image
docker build -f Dockerfile.base -t storage-python-base:latest .

# Build all services
docker-compose build --no-cache

# Start services
docker-compose up -d

# Check status
docker-compose ps
```

## 4. Verify Deployment

```bash
# Check all containers are running
docker-compose ps

# Check logs if any service fails
docker-compose logs [service-name]

# Test endpoints (replace with your EC2 public IP)
curl http://your-ec2-ip:8080/health  # auth-gateway
curl http://your-ec2-ip:8001/health  # tenant-node
```

## 5. Monitor Performance

```bash
# Watch resource usage
docker stats

# Monitor logs in real-time
docker-compose logs -f

# Check disk usage
docker system df
```

## Expected Results

- **Build time**: ~8-12 minutes (first time), ~3-5 minutes (subsequent)
- **Services**: All 10 services running (6 Python + 4 external)
- **Memory usage**: ~2-4GB total
- **Disk usage**: ~3-5GB Docker images

## Troubleshooting

### If deployment fails:
1. Check Docker is running: `sudo systemctl status docker`
2. Check permissions: `groups $USER` (should include docker)
3. Check disk space: `df -h`
4. Check logs: `docker-compose logs`

### If services don't start:
1. Check individual service logs: `docker-compose logs [service]`
2. Restart specific service: `docker-compose restart [service]`
3. Check port conflicts: `netstat -tulpn`

### Performance issues:
1. Monitor resources: `docker stats`
2. Check EC2 instance size (recommend t3.large or larger)
3. Ensure adequate disk space (8GB+ free)
