# Postman gRPC Testing Guide

## Setup Postman for gRPC (Postman v9.12+)

### 1. Import Proto Files
1. Open Postman
2. Click "New" → "gRPC Request"
3. Enter server URL: `localhost:8001`
4. Import proto files:
   - `proto/storage/ingestion.proto`
   - `proto/storage/common.proto`
5. Select "storage.IngestionService"

### 2. Test Single Record Ingestion

**Service:** `storage.IngestionService`
**Method:** `IngestRecord`
**Server URL:** `localhost:8001`

**Request Body (JSON):**
```json
{
  "record": {
    "id": {
      "tenant_id": {
        "id": "tenant-123"
      },
      "entity_id": {
        "id": "record-456"
      },
      "version": 1
    },
    "data": {
      "tenant_id": "tenant-123",
      "id": "record-456",
      "timestamp": "2025-07-12T14:30:00Z",
      "data": {
        "user_id": "user-789",
        "event_type": "page_view",
        "page_url": "https://example.com/home",
        "session_id": "session-abc123"
      }
    },
    "schema": {
      "tenant_id": {
        "id": "tenant-123"
      },
      "name": "page_events",
      "version": 1
    },
    "timestamp": "2025-07-12T14:30:00Z",
    "version": 1,
    "metadata": {
      "source": "web_analytics",
      "ip_address": "192.168.1.100"
    }
  },
  "options": {
    "async": "false",
    "validate_schema": "true"
  }
}
```

### 3. Test Batch Ingestion

**Method:** `IngestBatch`

**Request Body:**
```json
{
  "records": [
    {
      "id": {
        "tenant_id": {"id": "tenant-123"},
        "entity_id": {"id": "record-001"},
        "version": 1
      },
      "data": {
        "tenant_id": "tenant-123",
        "id": "record-001",
        "timestamp": "2025-07-12T14:30:00Z",
        "data": {
          "user_id": "user-100",
          "event_type": "click",
          "element": "signup_button"
        }
      },
      "timestamp": "2025-07-12T14:30:00Z"
    },
    {
      "id": {
        "tenant_id": {"id": "tenant-123"},
        "entity_id": {"id": "record-002"},
        "version": 1
      },
      "data": {
        "tenant_id": "tenant-123",
        "id": "record-002",
        "timestamp": "2025-07-12T14:31:00Z",
        "data": {
          "user_id": "user-101",
          "event_type": "purchase",
          "amount": 29.99,
          "currency": "USD"
        }
      },
      "timestamp": "2025-07-12T14:31:00Z"
    }
  ],
  "options": {
    "batch_size": "100",
    "validate_all": "true"
  },
  "transactional": true
}
```

### 4. Health Check

**Method:** `HealthCheck`

**Request Body:**
```json
{}
```

### 5. Test Collection Setup

Create a collection with these test scenarios:

1. **Valid Single Record** - Should succeed
2. **Missing Tenant ID** - Should fail validation
3. **Invalid Timestamp** - Should fail validation  
4. **Future Timestamp** - Should fail business rules
5. **Large Record** - Should test size limits
6. **Batch Valid Records** - Should succeed
7. **Batch Mixed Valid/Invalid** - Should show partial success

## Environment Variables for Postman

Create environment with:
- `server_url`: `localhost:8001`
- `tenant_id`: `tenant-123`
- `test_user_id`: `user-789`
- `timestamp`: `{{$isoTimestamp}}`
