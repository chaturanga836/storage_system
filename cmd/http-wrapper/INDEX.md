# HTTP Wrapper Documentation Index

## 📋 Overview
The HTTP Wrapper provides a RESTful HTTP interface for the Storage System's ingestion service. This directory contains comprehensive documentation for setup, configuration, deployment, and troubleshooting.

## 📚 Documentation Structure

### 🚀 Getting Started
| Document | Description | Use When |
|----------|-------------|----------|
| **[README.md](README.md)** | Quick start guide and overview | First time setup |
| **[API_SPEC.md](API_SPEC.md)** | Complete API reference | Integrating with the service |

### ⚙️ Configuration & Deployment  
| Document | Description | Use When |
|----------|-------------|----------|
| **[CONFIGURATION.md](CONFIGURATION.md)** | Configuration options and environment variables | Customizing service behavior |
| **[DEPLOYMENT.md](DEPLOYMENT.md)** | Deployment guides for all environments | Moving to production |

### 🔧 Testing & Troubleshooting
| Document | Description | Use When |
|----------|-------------|----------|
| **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)** | Common issues and solutions | Service not working as expected |
| **[Test Examples](../../test_examples/)** | Postman collections and cURL examples | Testing functionality |

## 🎯 Quick Navigation

### I Want To...

#### **Get Started Quickly**
1. Read [README.md](README.md) - Overview and quick start
2. Follow the "Quick Start" section
3. Test with [Postman examples](../../test_examples/postman_http_examples.md)

#### **Understand the API**
1. Read [API_SPEC.md](API_SPEC.md) - Complete API reference
2. Try the examples in [test_examples](../../test_examples/)
3. Use the health endpoint to verify connectivity

#### **Deploy to Production**
1. Read [DEPLOYMENT.md](DEPLOYMENT.md) - Deployment strategies
2. Configure using [CONFIGURATION.md](CONFIGURATION.md)
3. Set up monitoring and health checks

#### **Debug Issues**
1. Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Common solutions
2. Enable debug logging per configuration guide
3. Use the diagnostic tools in troubleshooting guide

#### **Integrate with My Application**
1. Study [API_SPEC.md](API_SPEC.md) - Request/response formats
2. Test with [Postman examples](../../test_examples/postman_http_examples.md)
3. Implement error handling based on API responses

## 🏗️ Architecture Overview

```
┌─────────────────┐    HTTP/REST    ┌──────────────────┐    gRPC    ┌─────────────────┐
│   HTTP Client   │ ──────────────→ │   HTTP Wrapper   │ ─────────→ │ Ingestion Service│
│ (Your App/Tool) │                 │   (Port 8082)    │            │   (Port 8001)   │
└─────────────────┘                 └──────────────────┘            └─────────────────┘
```

## 📖 Document Dependencies

### Read First
- [README.md](README.md) - Essential overview

### Reference Documents
- [API_SPEC.md](API_SPEC.md) - When implementing clients
- [CONFIGURATION.md](CONFIGURATION.md) - When customizing behavior

### Implementation Guides
- [DEPLOYMENT.md](DEPLOYMENT.md) - When deploying
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - When debugging

## 🔄 Common Workflows

### Development Workflow
1. **Setup**: Follow [README.md](README.md) quick start
2. **Configure**: Customize using [CONFIGURATION.md](CONFIGURATION.md)
3. **Test**: Use [test examples](../../test_examples/)
4. **Debug**: Reference [TROUBLESHOOTING.md](TROUBLESHOOTING.md) as needed

### Production Workflow  
1. **Plan**: Review [DEPLOYMENT.md](DEPLOYMENT.md) strategies
2. **Configure**: Set production config per [CONFIGURATION.md](CONFIGURATION.md)
3. **Deploy**: Follow environment-specific deployment guide
4. **Monitor**: Implement health checks and monitoring

### Integration Workflow
1. **Learn**: Study [API_SPEC.md](API_SPEC.md) endpoints
2. **Test**: Try [Postman examples](../../test_examples/postman_http_examples.md)
3. **Implement**: Build client using API specification
4. **Debug**: Use [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for issues

## 📝 Quick Reference

### Essential Endpoints
- **Health**: `GET /health` - Service status
- **Single Ingest**: `POST /api/v1/ingest/record` - Ingest one record
- **Batch Ingest**: `POST /api/v1/ingest/batch` - Ingest multiple records
- **Status**: `GET /api/v1/status` - Detailed service metrics

### Essential Commands
```bash
# Start services
go run cmd/ingestion-server/main.go     # Start ingestion service (port 8001)
go run cmd/http-wrapper/main.go         # Start HTTP wrapper (port 8082)

# Test health
curl http://localhost:8082/health

# Test single record
curl -X POST http://localhost:8082/api/v1/ingest/record \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"test","record_id":"test","timestamp":"2025-07-12T14:30:00Z","data":{"tenant_id":"test","id":"test","timestamp":"2025-07-12T14:30:00Z","data":{}}}'
```

### Configuration Essentials
```bash
# Key environment variables
HTTP_WRAPPER_PORT=8082                    # HTTP server port
INGESTION_SERVICE_ADDRESS=localhost:8001  # gRPC service address
LOG_LEVEL=info                           # Logging level
ENABLE_CORS=true                         # CORS support
```

## 🆘 Need Help?

### Quick Troubleshooting
1. **Service won't start**: Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Service Won't Start
2. **Connection refused**: Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Cannot Connect to Ingestion Service  
3. **Validation errors**: Check [API_SPEC.md](API_SPEC.md) - Field Descriptions
4. **Performance issues**: Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Performance Issues

### Documentation Updates
- All documentation is version controlled
- Check timestamps for latest updates
- Cross-references are maintained between documents

## 📊 Document Status

| Document | Last Updated | Status | Dependencies |
|----------|--------------|--------|--------------|
| README.md | Latest | ✅ Complete | None |
| API_SPEC.md | Latest | ✅ Complete | README.md |
| CONFIGURATION.md | Latest | ✅ Complete | README.md |
| DEPLOYMENT.md | Latest | ✅ Complete | CONFIGURATION.md |
| TROUBLESHOOTING.md | Latest | ✅ Complete | All above |

---

**Next Steps**: Start with [README.md](README.md) for your first HTTP Wrapper experience!
