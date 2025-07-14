# Query HTTP Wrapper API Specification

## Overview
This document provides a detailed specification for the Storage System Query HTTP Wrapper REST API.

## Base Information
- **Base URL**: `http://localhost:8083`
- **API Version**: v1
- **Content-Type**: `application/json`
- **Authentication**: None (development mode)

## Endpoints

### 1. Health Check

**Endpoint**: `GET /health`

**Description**: Returns the health status of the query HTTP wrapper service.

**Request**:
- Method: GET
- Headers: None required
- Body: None

**Response**:
```json
{
  "status": "healthy",
  "service": "query-http-wrapper", 
  "timestamp": "2025-07-13T14:30:00Z",
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
  "status": "running",
  "service": "query-service",
  "timestamp": "2025-07-13T14:30:00Z",
  "metrics": {
    "uptime": "active",
    "health": "good",
    "cache_status": "enabled",
    "query_count": 42,
    "avg_latency": "150ms"
  }
}
```

**Status Codes**:
- `200 OK`: Status retrieved successfully
- `500 Internal Server Error`: Unable to retrieve status

---

### 3. Execute Query

**Endpoint**: `POST /api/v1/query`

**Description**: Executes a query against the storage system with filters, projections, and ordering.

**Request**:
- Method: POST
- Headers: `Content-Type: application/json`
- Body:
```json
{
  "tenant_id": "string",
  "filters": [
    {
      "field": "string",
      "operator": "string",
      "value": "any"
    }
  ],
  "projection": ["field1", "field2"],
  "order_by": [
    {
      "field": "string",
      "direction": "asc|desc"
    }
  ],
  "limit": 100,
  "offset": 0,
  "time_range": {
    "start": "2025-07-13T00:00:00Z",
    "end": "2025-07-13T23:59:59Z"
  },
  "options": {
    "include_metadata": "true"
  }
}
```

**Field Descriptions**:
- `tenant_id`: Unique identifier for the tenant (required)
- `filters`: Array of filter conditions (optional)
  - `field`: Field name to filter on
  - `operator`: Filter operator (eq, ne, gt, lt, gte, lte, in, contains)
  - `value`: Filter value
- `projection`: Array of field names to return (optional)
- `order_by`: Array of ordering specifications (optional)
- `limit`: Maximum number of results to return (optional)
- `offset`: Number of results to skip (optional)
- `time_range`: Time range filter (optional)
- `options`: Additional query options (optional)

**Response (Success)**:
```json
{
  "status": "success",
  "results": [
    {
      "tenant_id": "tenant-123",
      "record_id": "record-456",
      "timestamp": "2025-07-13T14:30:00Z",
      "data": {
        "field1": "value1",
        "field2": "value2"
      }
    }
  ],
  "tenant_id": "tenant-123",
  "timestamp": "2025-07-13T14:30:00Z",
  "metadata": {
    "total_results": 1,
    "execution_time": "45ms",
    "cached": false
  }
}
```

**Response (Error)**:
```json
{
  "error": "Query execution failed",
  "details": "Invalid filter operator: 'invalid_op'"
}
```

**Status Codes**:
- `200 OK`: Query executed successfully
- `400 Bad Request`: Invalid query parameters
- `500 Internal Server Error`: Query processing error

---

### 4. Get Record by ID

**Endpoint**: `GET /api/v1/record/{tenant_id}/{record_id}`

**Description**: Retrieves a specific record by tenant ID and record ID.

**Request**:
- Method: GET
- Headers: None required
- Body: None
- Path Parameters:
  - `tenant_id`: Tenant identifier (required)
  - `record_id`: Record identifier (required)

**Response (Success)**:
```json
{
  "status": "success",
  "record": {
    "tenant_id": "tenant-123",
    "record_id": "record-456",
    "timestamp": "2025-07-13T14:30:00Z",
    "data": {
      "field1": "value1",
      "field2": "value2"
    },
    "metadata": {
      "version": 1,
      "source": "api"
    }
  },
  "tenant_id": "tenant-123",
  "record_id": "record-456",
  "timestamp": "2025-07-13T14:30:00Z"
}
```

**Response (Not Found)**:
```json
{
  "error": "Record not found",
  "details": "No record found with ID 'record-456' for tenant 'tenant-123'"
}
```

**Status Codes**:
- `200 OK`: Record retrieved successfully
- `400 Bad Request`: Invalid parameters
- `404 Not Found`: Record not found
- `500 Internal Server Error`: Retrieval error

---

### 5. Execute Aggregation Query

**Endpoint**: `POST /api/v1/query/aggregate`

**Description**: Executes aggregation queries (count, sum, avg, min, max) with optional grouping.

**Request**:
- Method: POST
- Headers: `Content-Type: application/json`
- Body:
```json
{
  "tenant_id": "string",
  "aggregation": "count|sum|avg|min|max",
  "field": "string",
  "group_by": ["field1", "field2"],
  "filters": [
    {
      "field": "string",
      "operator": "string",
      "value": "any"
    }
  ],
  "time_range": {
    "start": "2025-07-13T00:00:00Z",
    "end": "2025-07-13T23:59:59Z"
  },
  "options": {
    "precision": "2"
  }
}
```

**Field Descriptions**:
- `tenant_id`: Unique identifier for the tenant (required)
- `aggregation`: Type of aggregation to perform (required)
- `field`: Field to aggregate on (required for sum, avg, min, max)
- `group_by`: Fields to group results by (optional)
- `filters`: Array of filter conditions (optional)
- `time_range`: Time range filter (optional)
- `options`: Additional aggregation options (optional)

**Response (Success)**:
```json
{
  "status": "success",
  "results": [
    {
      "group": {
        "field1": "value1",
        "field2": "value2"
      },
      "aggregation": "count",
      "value": 150
    }
  ],
  "aggregation": "count",
  "tenant_id": "tenant-123",
  "timestamp": "2025-07-13T14:30:00Z",
  "metadata": {
    "execution_time": "75ms",
    "groups_count": 1
  }
}
```

**Status Codes**:
- `200 OK`: Aggregation executed successfully
- `400 Bad Request`: Invalid aggregation parameters
- `500 Internal Server Error`: Aggregation processing error

---

### 6. Explain Query

**Endpoint**: `POST /api/v1/query/explain`

**Description**: Returns the execution plan for a query without executing it.

**Request**:
- Method: POST
- Headers: `Content-Type: application/json`
- Body: Same as query request (see Execute Query endpoint)

**Response**:
```json
{
  "status": "success",
  "plan": {
    "query_type": "scan",
    "estimated_cost": 100,
    "estimated_rows": 1000,
    "steps": [
      {
        "step": 1,
        "operation": "index_scan",
        "table": "records",
        "estimated_cost": 50,
        "estimated_rows": 1000
      },
      {
        "step": 2,
        "operation": "filter",
        "condition": "tenant_id = 'tenant-123'",
        "estimated_cost": 30,
        "estimated_rows": 500
      },
      {
        "step": 3,
        "operation": "project",
        "fields": ["field1", "field2"],
        "estimated_cost": 20,
        "estimated_rows": 500
      }
    ]
  },
  "timestamp": "2025-07-13T14:30:00Z"
}
```

**Status Codes**:
- `200 OK`: Query plan generated successfully
- `400 Bad Request`: Invalid query parameters
- `500 Internal Server Error`: Plan generation error

---

## Query Filter Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `eq` | Equal to | `{"field": "status", "operator": "eq", "value": "active"}` |
| `ne` | Not equal to | `{"field": "status", "operator": "ne", "value": "deleted"}` |
| `gt` | Greater than | `{"field": "age", "operator": "gt", "value": 18}` |
| `lt` | Less than | `{"field": "price", "operator": "lt", "value": 100.0}` |
| `gte` | Greater than or equal | `{"field": "score", "operator": "gte", "value": 80}` |
| `lte` | Less than or equal | `{"field": "count", "operator": "lte", "value": 50}` |
| `in` | In array | `{"field": "category", "operator": "in", "value": ["A", "B"]}` |
| `contains` | Contains substring | `{"field": "name", "operator": "contains", "value": "test"}` |

## Error Response Format

All error responses follow this structure:
```json
{
  "error": "Error description",
  "code": "ERROR_CODE",
  "timestamp": "2025-07-13T14:30:00Z",
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
| `INVALID_OPERATOR` | Invalid filter operator |
| `INVALID_FORMAT` | Field format is invalid |
| `QUERY_ERROR` | Query execution error |
| `TIMEOUT_ERROR` | Query timeout |
| `NOT_FOUND` | Record not found |

## Performance Considerations

- **Pagination**: Use `limit` and `offset` for large result sets
- **Projections**: Specify only needed fields to reduce data transfer
- **Filters**: Use indexed fields in filters for better performance
- **Caching**: Identical queries may be cached for improved response times
- **Timeouts**: Queries have a default 30-second timeout

## Rate Limiting

Currently no rate limiting is implemented. This will be added in future versions.

## CORS

CORS is enabled for all origins in development mode. In production, this should be restricted to specific domains.

## Examples

See `test_examples/query_http_examples.md` for comprehensive testing examples including:
- Postman collection setup
- cURL command examples  
- Expected responses
- Error scenarios
- Performance testing guidelines
