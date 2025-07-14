# HTTP Wrapper API Specification

## Overview
This document provides a detailed specification for the Storage System HTTP Wrapper REST API.

## Base Information
- **Base URL**: `http://localhost:8082`
- **API Version**: v1
- **Content-Type**: `application/json`
- **Authentication**: None (development mode)

## Endpoints

### 1. Health Check

**Endpoint**: `GET /health`

**Description**: Returns the health status of the HTTP wrapper service.

**Request**:
- Method: GET
- Headers: None required
- Body: None

**Response**:
```json
{
  "status": "healthy",
  "service": "ingestion-http-wrapper", 
  "timestamp": "2025-07-12T14:30:00Z",
  "version": "1.0.0"
}
```

**Status Codes**:
- `200 OK`: Service is healthy
- `500 Internal Server Error`: Service is experiencing issues

---

### 2. Service Status

**Endpoint**: `GET /api/v1/status`

**Description**: Returns detailed service metrics and operational status.

**Request**:
- Method: GET
- Headers: None required
- Body: None

**Response**:
```json
{
  "service": "ingestion-http-wrapper",
  "status": "operational",
  "uptime": "2h30m45s",
  "requests_processed": 1250,
  "errors": 12,
  "ingestion_service_status": "connected",
  "timestamp": "2025-07-12T14:30:00Z"
}
```

**Status Codes**:
- `200 OK`: Status retrieved successfully
- `500 Internal Server Error`: Unable to retrieve status

---

### 3. Single Record Ingestion

**Endpoint**: `POST /api/v1/ingest/record`

**Description**: Ingests a single data record into the storage system.

**Request**:
- Method: POST
- Headers: `Content-Type: application/json`
- Body:
```json
{
  "tenant_id": "string",
  "record_id": "string", 
  "timestamp": "string (RFC3339)",
  "data": {
    "tenant_id": "string",
    "id": "string",
    "timestamp": "string (RFC3339)",
    "data": {
      "field1": "value1",
      "field2": "value2"
    }
  },
  "metadata": {
    "source": "string",
    "tags": ["tag1", "tag2"]
  }
}
```

**Field Descriptions**:
- `tenant_id`: Unique identifier for the tenant (required)
- `record_id`: Unique identifier for the record (required)
- `timestamp`: ISO 8601 timestamp (required)
- `data`: The actual record data (required)
- `metadata`: Additional metadata (optional)

**Response (Success)**:
```json
{
  "success": true,
  "message": "Record ingested successfully",
  "record_id": "record-456",
  "tenant_id": "tenant-123",
  "timestamp": "2025-07-12T14:30:00Z"
}
```

**Response (Error)**:
```json
{
  "error": "Validation failed: tenant_id is required"
}
```

**Status Codes**:
- `200 OK`: Record ingested successfully
- `400 Bad Request`: Invalid request data
- `500 Internal Server Error`: Server processing error

**Validation Rules**:
- `tenant_id`: Must be non-empty string
- `record_id`: Must be non-empty string
- `timestamp`: Must be valid RFC3339 format
- `data`: Must be valid JSON object

---

### 4. Batch Record Ingestion

**Endpoint**: `POST /api/v1/ingest/batch`

**Description**: Ingests multiple records in a single transaction.

**Request**:
- Method: POST
- Headers: `Content-Type: application/json`
- Body:
```json
{
  "records": [
    {
      "tenant_id": "string",
      "record_id": "string",
      "timestamp": "string (RFC3339)",
      "data": {
        "tenant_id": "string",
        "id": "string", 
        "timestamp": "string (RFC3339)",
        "data": {
          "field1": "value1",
          "field2": "value2"
        }
      },
      "metadata": {
        "source": "string",
        "tags": ["tag1", "tag2"]
      }
    }
  ],
  "transactional": true
}
```

**Field Descriptions**:
- `records`: Array of record objects (required, min 1)
- `transactional`: Whether to process as single transaction (optional, default: false)

**Response (Success)**:
```json
{
  "success": true,
  "message": "Batch processed successfully",
  "records_processed": 3,
  "results": [
    {
      "record_id": "record-1",
      "status": "success"
    },
    {
      "record_id": "record-2", 
      "status": "success"
    },
    {
      "record_id": "record-3",
      "status": "success"
    }
  ],
  "timestamp": "2025-07-12T14:30:00Z"
}
```

**Response (Partial Failure)**:
```json
{
  "success": false,
  "message": "Batch processing completed with errors",
  "records_processed": 3,
  "results": [
    {
      "record_id": "record-1",
      "status": "success"
    },
    {
      "record_id": "record-2",
      "status": "error",
      "error": "Validation failed: invalid timestamp format"
    },
    {
      "record_id": "record-3", 
      "status": "success"
    }
  ],
  "timestamp": "2025-07-12T14:30:00Z"
}
```

**Status Codes**:
- `200 OK`: Batch processed (may contain individual record errors)
- `400 Bad Request`: Invalid request format
- `500 Internal Server Error`: Server processing error

**Validation Rules**:
- Same validation rules as single record ingestion apply to each record
- Batch size limit: 1000 records per request
- If `transactional` is true, all records must succeed or entire batch fails

---

## Error Response Format

All error responses follow this structure:
```json
{
  "error": "Error description",
  "code": "ERROR_CODE",
  "timestamp": "2025-07-12T14:30:00Z",
  "details": {
    "field": "additional_context"
  }
}
```

## Common Error Codes

| Code | Description |
|------|-------------|
| `VALIDATION_ERROR` | Request validation failed |
| `MISSING_FIELD` | Required field is missing |
| `INVALID_FORMAT` | Field format is invalid |
| `SERVICE_ERROR` | Internal service error |
| `TIMEOUT_ERROR` | Request timeout |

## Rate Limiting

Currently no rate limiting is implemented. This will be added in future versions.

## CORS

CORS is enabled for all origins in development mode. In production, this should be restricted to specific domains.

## Examples

See `test_examples/postman_http_examples.md` for comprehensive testing examples including:
- Postman collection setup
- cURL command examples  
- Expected responses
- Error scenarios
- Performance testing guidelines
