# Query HTTP Wrapper Documentation Index

## 📋 Overview
The Query HTTP Wrapper provides a RESTful HTTP interface for the Storage System's query service. This directory contains comprehensive documentation for setup, configuration, deployment, and troubleshooting.

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
3. Test with [API examples](API_SPEC.md#examples)

#### **Understand the Query API**
1. Read [API_SPEC.md](API_SPEC.md) - Complete API reference
2. Try the query examples
3. Use the health endpoint to verify connectivity

#### **Deploy to Production**
1. Read [DEPLOYMENT.md](DEPLOYMENT.md) - Deployment strategies
2. Configure using [CONFIGURATION.md](CONFIGURATION.md)
3. Set up monitoring and health checks

#### **Debug Query Issues**
1. Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Common solutions
2. Enable debug logging per configuration guide
3. Use the diagnostic tools in troubleshooting guide

#### **Integrate Query API with My Application**
1. Study [API_SPEC.md](API_SPEC.md) - Request/response formats
2. Test with query examples
3. Implement error handling based on API responses

## 🏗️ Architecture Overview

```
┌─────────────────┐    HTTP/REST    ┌──────────────────┐    gRPC    ┌─────────────────┐
│   HTTP Client   │ ──────────────→ │ Query HTTP       │ ─────────→ │ Query Service   │
│ (Your App/Tool) │                 │ Wrapper (8083)   │            │   (Port 8002)   │
└─────────────────┘                 └──────────────────┘            └─────────────────┘
```

## 📖 Document Dependencies

### Read First
- [README.md](README.md) - Essential overview

### Reference Documents
- [API_SPEC.md](API_SPEC.md) - When implementing query clients
- [CONFIGURATION.md](CONFIGURATION.md) - When customizing behavior

### Implementation Guides
- [DEPLOYMENT.md](DEPLOYMENT.md) - When deploying
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - When debugging

## 🔄 Common Workflows

### Development Workflow
1. **Setup**: Follow [README.md](README.md) quick start
2. **Configure**: Customize using [CONFIGURATION.md](CONFIGURATION.md)
3. **Test**: Use query examples from [API_SPEC.md](API_SPEC.md)
4. **Debug**: Reference [TROUBLESHOOTING.md](TROUBLESHOOTING.md) as needed

### Production Workflow  
1. **Plan**: Review [DEPLOYMENT.md](DEPLOYMENT.md) strategies
2. **Configure**: Set production config per [CONFIGURATION.md](CONFIGURATION.md)
3. **Deploy**: Follow environment-specific deployment guide
4. **Monitor**: Implement health checks and query performance monitoring

### Integration Workflow
1. **Learn**: Study [API_SPEC.md](API_SPEC.md) endpoints
2. **Test**: Try query examples with different parameters
3. **Implement**: Build query client using API specification
4. **Debug**: Use [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for issues

## 📝 Quick Reference

### Essential Endpoints
- **Health**: `GET /health` - Service status
- **Query**: `POST /api/v1/query` - Execute queries with filters
- **Get Record**: `GET /api/v1/record/{tenant_id}/{record_id}` - Get specific record
- **Aggregate**: `POST /api/v1/query/aggregate` - Execute aggregations
- **Explain**: `POST /api/v1/query/explain` - Get query execution plan
- **Status**: `GET /api/v1/status` - Detailed service metrics

### Essential Commands
```bash
# Start services
go run cmd/query-server/main.go          # Start query service (port 8002)
go run cmd/query-http-wrapper/main.go    # Start HTTP wrapper (port 8083)

# Test health
curl http://localhost:8083/health

# Test simple query
curl -X POST http://localhost:8083/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "test",
    "filters": [
      {"field": "status", "operator": "eq", "value": "active"}
    ],
    "limit": 10
  }'

# Get specific record
curl http://localhost:8083/api/v1/record/test/record-123
```

### Configuration Essentials
```bash
# Key environment variables
QUERY_HTTP_WRAPPER_PORT=8083              # HTTP server port
QUERY_SERVICE_ADDRESS=localhost:8002      # gRPC service address
LOG_LEVEL=info                           # Logging level
ENABLE_CORS=true                         # CORS support
QUERY_CACHE_ENABLED=true                 # Enable query caching
```

## 🆘 Need Help?

### Quick Troubleshooting
1. **Service won't start**: Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Service Won't Start
2. **Connection refused**: Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Cannot Connect to Query Service  
3. **Query errors**: Check [API_SPEC.md](API_SPEC.md) - Query Filter Operators
4. **Performance issues**: Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Performance Issues

### Query-Specific Help
1. **Invalid filters**: Check [API_SPEC.md](API_SPEC.md) - Query Filter Operators
2. **Slow queries**: Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Slow Query Response Times
3. **Large results**: Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Large Result Set Memory Issues
4. **Cache issues**: Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Caching Issues

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

## 🔄 Related Services

### Ingestion Pipeline
- **[Ingestion HTTP Wrapper](../http-wrapper/)** - REST API for data ingestion
- **[Ingestion Server](../ingestion-server/)** - gRPC ingestion service

### Query Pipeline
- **[Query Server](../query-server/)** - gRPC query service
- **[Query HTTP Wrapper](.)** - REST API for queries (this service)

## 🎨 API Feature Highlights

### 🔍 **Flexible Querying**
- Multiple filter operators (eq, ne, gt, lt, gte, lte, in, contains)
- Field projections to reduce data transfer
- Sorting and pagination support
- Time range filtering

### 📊 **Aggregations**
- Count, sum, average, min, max
- Group by multiple fields
- Filtered aggregations
- Performance optimized

### 🚀 **Performance Features**
- Query result caching
- Connection pooling
- Parallel query execution
- Result size limits

### 🛡️ **Production Ready**
- Comprehensive error handling
- Request validation
- Timeout management
- Health monitoring

---

**Next Steps**: Start with [README.md](README.md) for your first Query HTTP Wrapper experience!
