# Query HTTP Wrapper Documentation

## Overview

The Query HTTP Wrapper provides REST API endpoints for the storage system's query service. It acts as a bridge between HTTP/REST clients and the internal query service, converting HTTP requests to internal query calls.

## Architecture

```
HTTP Client → Query HTTP Wrapper (Port 8083) → Query Service → Storage Engine
```

## Features

- **RESTful Query API**: Standard HTTP/JSON interface for data retrieval
- **Multiple Query Types**: Simple queries, aggregations, record lookups
- **Request Validation**: Comprehensive input validation
- **Error Handling**: Detailed error responses
- **CORS Support**: Cross-origin requests enabled
- **Logging**: Detailed request/response logging
- **Timeout Management**: Configurable request timeouts

## API Endpoints

### Health Check
- **Endpoint**: `GET /health`
- **Purpose**: Service health verification
- **Response**: Service status and metadata

### Query Execution
- **Endpoint**: `POST /api/v1/query`
- **Purpose**: Execute queries with filters, projections, and ordering
- **Request**: JSON query with tenant, filters, and options
- **Response**: Query results with metadata

### Record Retrieval
- **Endpoint**: `GET /api/v1/record/{tenant_id}/{record_id}`
- **Purpose**: Get a specific record by ID
- **Request**: Tenant and record IDs in URL path
- **Response**: Record data with metadata

### Aggregation Queries
- **Endpoint**: `POST /api/v1/query/aggregate`
- **Purpose**: Execute aggregation queries (count, sum, avg, etc.)
- **Request**: JSON aggregation specification
- **Response**: Aggregated results

### Query Plan Explanation
- **Endpoint**: `POST /api/v1/query/explain`
- **Purpose**: Get query execution plan and cost estimates
- **Request**: JSON query specification
- **Response**: Execution plan details

### Service Status
- **Endpoint**: `GET /api/v1/status`
- **Purpose**: Get service metrics and operational status
- **Response**: Service health and query statistics

## Quick Start

1. **Start the query server** (port 8002):
   ```bash
   go run cmd/query-server/main.go
   ```

2. **Start the query HTTP wrapper** (port 8083):
   ```bash
   go run cmd/query-http-wrapper/main.go
   ```

3. **Test health endpoint**:
   ```bash
   curl -X GET http://localhost:8083/health
   ```

## Configuration

The Query HTTP wrapper uses the same configuration as the query service:
- Config file: `config.json`
- Environment variables supported
- Default port: 8083

## Error Handling

The wrapper provides structured error responses:
- **400 Bad Request**: Invalid query parameters
- **500 Internal Server Error**: Query processing errors
- **Detailed messages**: Specific validation failure descriptions

## Logging

Request logging includes:
- 🔍 Query requests with filters and parameters
- 📄 Record retrieval requests
- 📊 Aggregation requests
- 📋 Query plan requests
- ✅ Success confirmations
- ❌ Error details

## Dependencies

- **Gin Web Framework**: HTTP routing and middleware
- **Internal Services**: Direct integration with query service
- **Configuration**: Shared config system with main application

## Documentation

### Core Documentation
- **[README.md](README.md)**: This overview and quick start guide
- **[API_SPEC.md](API_SPEC.md)**: Complete API specification with request/response examples
- **[CONFIGURATION.md](CONFIGURATION.md)**: Configuration options and environment variables
- **[DEPLOYMENT.md](DEPLOYMENT.md)**: Deployment guides for various environments
- **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)**: Common issues and solutions

### Testing and Examples
- **[test_examples/query_http_examples.md](../../test_examples/query_http_examples.md)**: Postman collection and cURL examples

### Implementation Details
- **[main.go](main.go)**: Main query HTTP wrapper implementation
- **[internal/services/query/](../../internal/services/query/)**: Core query service logic

## Development

### Local Development Setup
1. Read the [API specification](API_SPEC.md) to understand the endpoints
2. Review [configuration options](CONFIGURATION.md) for your environment
3. Follow the [deployment guide](DEPLOYMENT.md) for local setup
4. Use [test examples](../../test_examples/query_http_examples.md) to verify functionality
5. Consult [troubleshooting guide](TROUBLESHOOTING.md) if issues arise

### Code Structure
```
cmd/query-http-wrapper/
├── main.go                 # HTTP server and routing
├── README.md              # This overview
├── API_SPEC.md           # Complete API documentation
├── CONFIGURATION.md      # Configuration guide
├── DEPLOYMENT.md         # Deployment instructions
└── TROUBLESHOOTING.md    # Common issues and solutions
```
