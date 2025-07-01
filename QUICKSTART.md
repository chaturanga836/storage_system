# Quick Start Guide

This guide helps you get the microservices system up and running quickly.

## 🚀 Quick Start

### Option 1: Docker Compose (Easiest)
```bash
# Start all services with Docker
docker-compose up --build

# Wait for all services to start (30-60 seconds)
# Then run demos
python microservices_demo.py
```

### Option 2: Individual Services (Development)
```powershell
# Start all services
./start_services.ps1 -Individual

# Install demo dependencies
pip install -r demo_requirements.txt

# Run demos
python auth_demo.py
python microservices_demo.py
python scaling_demo.py
```

### Option 3: Manual (Step by step)
```bash
# 1. Start Auth Gateway
cd auth-gateway
pip install -r requirements.txt
python main.py &

# 2. Start Monitoring
cd ../monitoring
pip install -r requirements.txt
python main.py &

# 3. Start other services...
# (See start_services.ps1 for complete order)
```

## 📋 Service Health Checks

Check if services are running:

```bash
# Health checks
curl http://localhost:8080/health  # Auth Gateway
curl http://localhost:8000/health  # Tenant Node
curl http://localhost:8084/status  # Monitoring
```

## 🎯 Run Demos

### 1. Authentication Demo
```bash
python auth_demo.py
```
Shows JWT authentication, role-based access, token validation.

### 2. Full System Demo
```bash
python microservices_demo.py
```
Complete integration test with data operations across all services.

### 3. Scaling Demo
```bash
python scaling_demo.py
```
Load testing and auto-scaling demonstration.

## 🐛 Troubleshooting

### Services Won't Start
- Check if ports are available (8000, 8080, 8081, 8082, 8083, 8084)
- Install dependencies: `pip install -r <service>/requirements.txt`
- Check logs in each service directory

### Demos Fail
- Ensure all services are healthy first
- Install demo dependencies: `pip install -r demo_requirements.txt`
- Check authentication is working: `python auth_demo.py`

### Docker Issues
- Ensure Docker is running
- Try: `docker-compose down && docker-compose up --build`
- Check container logs: `docker-compose logs <service-name>`

## 📊 Service URLs

- **Auth Gateway**: http://localhost:8080
- **Tenant Node**: http://localhost:8000
- **Operation Node**: http://localhost:8081
- **CBO Engine**: http://localhost:8082
- **Metadata Catalog**: http://localhost:8083
- **Monitoring**: http://localhost:8084

## 🔧 Configuration

Each service has its own configuration files:
- `<service>/config.json` - Service-specific settings
- `<service>/requirements.txt` - Python dependencies
- `<service>/Dockerfile` - Container configuration

## 📚 Next Steps

1. **Explore APIs**: Check each service's REST endpoints
2. **Customize Configuration**: Modify service configs for your needs
3. **Add Data Sources**: Use the Tenant Node APIs to add your data
4. **Monitor Performance**: Use the monitoring dashboard
5. **Deploy to Production**: Use the Docker configs for deployment

## 🆘 Getting Help

- Check service logs for error details
- Review the MIGRATION.md for architecture details
- Each service has its own README.md with specific information
