# 🐍 PYTHON SERVICES DEPLOYMENT GUIDE

## 📍 **System Overview**
- **Python Services EC2**: `65.0.150.75` (Docker microservices)
- **Go Control Plane EC2**: `15.207.184.150` (Go binary service)

---

## 🐍 **PYTHON SERVICES DEPLOYMENT**

### **Step 1: Update Environment Configuration**
```bash
# Navigate to storage system
cd /path/to/your/storage_system

# The config.env file is now configured with proper settings
# Verify it exists:
ls -la config.env

# If you need to update Go control plane IP in the future:
nano config.env
# Update: GO_CONTROL_PLANE_IP=15.207.184.150
```

### **Step 2: Deploy Python Services**
```bash
# Make sure all services are running
docker-compose ps

# If not running, start them:
docker-compose up -d

# Verify all services are healthy:
python health_check.py localhost
```

### **Step 3: Test Python Services**
```bash
# Test key endpoints
curl http://localhost:8080/health  # auth-gateway
curl http://localhost:8001/health  # tenant-node
curl http://localhost:8087/health  # metadata-catalog
```

---

## 🔒 **SECURITY GROUP CONFIGURATION**

### **Python EC2 (65.0.150.75) Security Group:**
```
Type        Protocol    Port Range    Source
HTTP        TCP         8080         15.207.184.150/32  # Go → Auth Gateway
HTTP        TCP         8001         15.207.184.150/32  # Go → Tenant Node
HTTP        TCP         8087         15.207.184.150/32  # Go → Metadata Catalog
HTTP        TCP         8086         15.207.184.150/32  # Go → Operation Node
HTTP        TCP         8088         15.207.184.150/32  # Go → CBO Engine
HTTP        TCP         8089         15.207.184.150/32  # Go → Monitoring
HTTP        TCP         8085         15.207.184.150/32  # Go → Query Interpreter

# Optional: Public access to main gateway
HTTP        TCP         8080         0.0.0.0/0          # Public → Auth Gateway
```

---

## 🧪 **VERIFICATION CHECKLIST**

### **✅ Python Services (65.0.150.75)**
- [ ] All Docker containers running (`docker-compose ps`)
- [ ] Health checks passing (`python health_check.py localhost`)
- [ ] Auth Gateway responding (`curl http://localhost:8080/health`)
- [ ] Tenant Node responding (`curl http://localhost:8001/health`)

---

## 🌐 **ACCESS URLS**

### **Public Access:**
- **Python Auth Gateway**: `http://65.0.150.75:8080`
- **Python Tenant Node**: `http://65.0.150.75:8001`
- **Python Metadata Catalog**: `http://65.0.150.75:8087`

### **System Management:**
```bash
# Python Services
docker-compose logs -f
docker-compose restart service-name
docker-compose ps
```

### **Cross-System Test:**
```bash
# Test Go control plane connectivity (from Python EC2)
curl http://15.207.184.150:8090/health
```

## 🎉 **SUCCESS INDICATORS**

When working correctly, you should see:

1. **All containers running**: `docker-compose ps` shows "Up" status
2. **Health checks passing**: All service health endpoints responding
3. **Go control plane can reach services**: Cross-system connectivity working

Your Python services are now fully deployed! 🚀
