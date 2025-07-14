# HTTP Wrapper Documentation

## Overview

The HTTP Wrapper provides REST API endpoints for the ingestion storage engine. It acts as a bridge between HTTP/REST clients and the internal ingestion service, converting HTTP requests to internal service calls.

## Architecture

```
HTTP Client → HTTP Wrapper (Port 8082) → Ingestion Service → Storage Engine
```

## Features

- **RESTful API**: Standard HTTP/JSON interface
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

### Single Record Ingestion
- **Endpoint**: `POST /api/v1/ingest/record`
- **Purpose**: Ingest a single data record
- **Request**: JSON record with tenant, data, and metadata
- **Response**: Success confirmation with record details

### Batch Record Ingestion
- **Endpoint**: `POST /api/v1/ingest/batch`
- **Purpose**: Ingest multiple records in one request
- **Request**: JSON array of records with transactional flag
- **Response**: Batch processing results

### Service Status
- **Endpoint**: `GET /api/v1/status`
- **Purpose**: Get service metrics and status
- **Response**: Service health and operational metrics

## Quick Start

1. **Start the ingestion server** (port 8001):
   ```bash
   go run cmd/ingestion-server/main.go
   ```

2. **Start the HTTP wrapper** (port 8082):
   ```bash
   go run cmd/http-wrapper/main.go
   ```

3. **Test health endpoint**:
   ```bash
   curl -X GET http://localhost:8082/health
   ```

## Configuration

The HTTP wrapper uses the same configuration as the ingestion service:
- Config file: `config.json`
- Environment variables supported
- Default port: 8082

## Error Handling

The wrapper provides structured error responses:
- **400 Bad Request**: Invalid input data
- **500 Internal Server Error**: Service processing errors
- **Detailed messages**: Specific validation failure descriptions

## Logging

Request logging includes:
- 📥 Single record requests
- 📦 Batch requests with record details
- 🔍 Validation steps
- ✅ Success confirmations
- ❌ Error details

## Dependencies

- **Gin Web Framework**: HTTP routing and middleware
- **Internal Services**: Direct integration with ingestion service
- **Configuration**: Shared config system with main application

## Documentation

### Core Documentation
- **[README.md](README.md)**: This overview and quick start guide
- **[API_SPEC.md](API_SPEC.md)**: Complete API specification with request/response examples
- **[CONFIGURATION.md](CONFIGURATION.md)**: Configuration options and environment variables
- **[DEPLOYMENT.md](DEPLOYMENT.md)**: Deployment guides for various environments
- **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)**: Common issues and solutions

### Testing and Examples
- **[test_examples/postman_http_examples.md](../../test_examples/postman_http_examples.md)**: Postman collection and cURL examples
- **[test_examples/grpc_test_examples.md](../../test_examples/grpc_test_examples.md)**: Direct gRPC testing examples

### Implementation Details
- **[main.go](main.go)**: Main HTTP wrapper implementation
- **[internal/services/ingestion/](../../internal/services/ingestion/)**: Core ingestion service logic

## Development

### Local Development Setup
1. Read the [API specification](API_SPEC.md) to understand the endpoints
2. Review [configuration options](CONFIGURATION.md) for your environment
3. Follow the [deployment guide](DEPLOYMENT.md) for local setup
4. Use [test examples](../../test_examples/postman_http_examples.md) to verify functionality
5. Consult [troubleshooting guide](TROUBLESHOOTING.md) if issues arise

### Code Structure
```
cmd/http-wrapper/
├── main.go                 # HTTP server and routing
├── README.md              # This overview
├── API_SPEC.md           # Complete API documentation
├── CONFIGURATION.md      # Configuration guide
├── DEPLOYMENT.md         # Deployment instructions
└── TROUBLESHOOTING.md    # Common issues and solutions
```
