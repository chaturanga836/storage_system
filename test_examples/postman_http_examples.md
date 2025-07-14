# Postman HTTP REST API Examples

## Setup
1. Start the ingestion server: `go run cmd/ingestion-server/main.go` (runs on port 8001)
2. Start the HTTP wrapper: `go run cmd/http-wrapper/main.go` (runs on port 8082)
3. Import these examples into Postman

## Base URL
```
http://localhost:8082
```

## 1. Health Check

**Method:** GET
**URL:** `{{base_url}}/health`
**Headers:** None needed

**Expected Response:**
```json
{
  "status": "healthy",
  "service": "ingestion-http-wrapper",
  "timestamp": "2025-07-12T14:30:00Z",
  "version": "1.0.0"
}
```

---

## 2. Ingest Single Record (Valid)

**Method:** POST
**URL:** `{{base_url}}/api/v1/ingest/record`
**Headers:**
```
Content-Type: application/json
```

**Body (JSON):**
```json
{
  "tenant_id": "tenant-123",
  "record_id": "record-456",
  "timestamp": "2025-07-12T14:30:00Z",
  "data": {
    "tenant_id": "tenant-123",
    "id": "record-456",
    "timestamp": "2025-07-12T14:30:00Z",
    "data": {
      "user_id": "user-789",
      "event_type": "page_view",
      "page_url": "https://example.com/home",
      "session_id": "session-abc123",
      "browser": "Chrome",
      "ip_address": "192.168.1.100"
    }
  },
  "metadata": {
    "source": "web_analytics",
    "version": "1.0"
  }
}
```

**Expected Response:**
```json
{
  "status": "success",
  "record_id": "record-456",
  "tenant_id": "tenant-123",
  "timestamp": "2025-07-12T14:30:00Z"
}
```

---

## 3. Ingest Single Record (Missing tenant_id - Should Fail)

**Method:** POST
**URL:** `{{base_url}}/api/v1/ingest/record`
**Headers:**
```
Content-Type: application/json
```

**Body (JSON):**
```json
{
  "record_id": "record-456",
  "data": {
    "id": "record-456",
    "timestamp": "2025-07-12T14:30:00Z",
    "data": {
      "user_id": "user-789",
      "event_type": "page_view"
    }
  }
}
```

**Expected Response:**
```json
{
  "error": "Invalid request format",
  "details": "Key: 'IngestRecordRequest.TenantID' Error:Field validation for 'TenantID' failed on the 'required' tag"
}
```

---

## 4. Ingest Single Record (Invalid timestamp - Should Fail)

**Method:** POST
**URL:** `{{base_url}}/api/v1/ingest/record`
**Headers:**
```
Content-Type: application/json
```

**Body (JSON):**
```json
{
  "tenant_id": "tenant-123",
  "record_id": "record-456",
  "timestamp": "invalid-timestamp",
  "data": {
    "tenant_id": "tenant-123",
    "id": "record-456",
    "timestamp": "invalid-timestamp",
    "data": {
      "user_id": "user-789",
      "event_type": "page_view"
    }
  }
}
```

**Expected Response:**
```json
{
  "error": "Ingestion failed",
  "details": "rpc error: code = InvalidArgument desc = timestamp format is invalid: ..."
}
```

---

## 5. Ingest Batch Records

**Method:** POST
**URL:** `{{base_url}}/api/v1/ingest/batch`
**Headers:**
```
Content-Type: application/json
```

**Body (JSON):**
```json
{
  "transactional": true,
  "records": [
    {
      "tenant_id": "tenant-123",
      "record_id": "record-001",
      "timestamp": "2025-07-12T14:30:00Z",
      "data": {
        "tenant_id": "tenant-123",
        "id": "record-001",
        "timestamp": "2025-07-12T14:30:00Z",
        "data": {
          "user_id": "user-100",
          "event_type": "click",
          "element": "signup_button",
          "page": "/signup"
        }
      },
      "metadata": {
        "source": "web_tracking"
      }
    },
    {
      "tenant_id": "tenant-123",
      "record_id": "record-002",
      "timestamp": "2025-07-12T14:31:00Z",
      "data": {
        "tenant_id": "tenant-123",
        "id": "record-002",
        "timestamp": "2025-07-12T14:31:00Z",
        "data": {
          "user_id": "user-101",
          "event_type": "purchase",
          "amount": 29.99,
          "currency": "USD",
          "product_id": "prod-abc123"
        }
      },
      "metadata": {
        "source": "ecommerce"
      }
    },
    {
      "tenant_id": "tenant-123",
      "record_id": "record-003",
      "timestamp": "2025-07-12T14:32:00Z",
      "data": {
        "tenant_id": "tenant-123",
        "id": "record-003",
        "timestamp": "2025-07-12T14:32:00Z",
        "data": {
          "user_id": "user-102",
          "event_type": "logout",
          "session_duration": 1800
        }
      }
    }
  ]
}
```

**Expected Response:**
```json
{
  "status": "success",
  "records_count": 3,
  "transactional": true,
  "timestamp": "2025-07-12T14:32:00Z"
}
```

---

## 6. Get Ingestion Status

**Method:** GET
**URL:** `{{base_url}}/api/v1/status`
**Headers:** None needed

**Expected Response:**
```json
{
  "status": "running",
  "service": "ingestion-service",
  "timestamp": "2025-07-12T14:32:00Z",
  "metrics": {
    "uptime": "active",
    "health": "good"
  }
}
```

---

## Environment Variables

Create a Postman environment with:

| Variable | Initial Value | Current Value |
|----------|---------------|---------------|
| `base_url` | `http://localhost:8082` | `http://localhost:8082` |
| `tenant_id` | `tenant-123` | `tenant-123` |
| `timestamp` | `{{$isoTimestamp}}` | (auto-generated) |

---

## Test Scenarios Collection

Create a Postman collection with these test cases:

### 🟢 Positive Tests
1. **Health Check** - Verify service is running
2. **Valid Single Record** - Basic ingestion
3. **Valid Batch Records** - Batch ingestion
4. **Large Record** - Test size limits
5. **Special Characters** - Unicode/UTF-8 data

### 🔴 Negative Tests
6. **Missing Tenant ID** - Required field validation
7. **Missing Record ID** - Required field validation  
8. **Missing Data** - Required field validation
9. **Invalid Timestamp** - Data type validation
10. **Future Timestamp** - Business rule validation
11. **Empty Batch** - Edge case testing
12. **Malformed JSON** - Format validation

### 📊 Performance Tests
13. **Large Batch** - 100+ records
14. **Concurrent Requests** - Load testing
15. **Stress Test** - Maximum throughput

---

## cURL Examples (Alternative)

If you prefer command line testing:

```bash
# Health check
curl -X GET http://localhost:8082/health

# Single record
curl -X POST http://localhost:8082/api/v1/ingest/record \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "tenant-123",
    "record_id": "record-456",
    "data": {
      "tenant_id": "tenant-123",
      "id": "record-456",
      "timestamp": "2025-07-12T14:30:00Z",
      "data": {"user_id": "user-789", "event_type": "test"}
    }
  }'

# Status
curl -X GET http://localhost:8082/api/v1/status
```
