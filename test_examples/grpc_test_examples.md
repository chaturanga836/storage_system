# gRPC Testing Examples for Ingestion Server

## Install grpcurl (if not installed)
```bash
# Windows (using go)
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# Or download from: https://github.com/fullstorydev/grpcurl/releases
```

## Test Examples

### 1. Health Check
```bash
grpcurl -plaintext localhost:8001 storage.IngestionService/HealthCheck
```

### 2. Ingest Single Record
```bash
grpcurl -plaintext \
  -d '{
    "record": {
      "id": {
        "tenant_id": {"id": "tenant-123"},
        "entity_id": {"id": "record-456"},
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
        "tenant_id": {"id": "tenant-123"},
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
  }' \
  localhost:8001 storage.IngestionService/IngestRecord
```

### 3. Ingest Batch Records
```bash
grpcurl -plaintext \
  -d '{
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
  }' \
  localhost:8001 storage.IngestionService/IngestBatch
```

### 4. Get Service Info (Reflection)
```bash
# List all services
grpcurl -plaintext localhost:8001 list

# List methods for IngestionService
grpcurl -plaintext localhost:8001 list storage.IngestionService

# Describe a specific method
grpcurl -plaintext localhost:8001 describe storage.IngestionService.IngestRecord
```

### 5. Test Validation Failures

#### Missing tenant_id:
```bash
grpcurl -plaintext \
  -d '{
    "record": {
      "id": {
        "entity_id": {"id": "record-456"},
        "version": 1
      },
      "data": {
        "id": "record-456",
        "timestamp": "2025-07-12T14:30:00Z",
        "data": {"test": "data"}
      }
    }
  }' \
  localhost:8001 storage.IngestionService/IngestRecord
```

#### Invalid timestamp format:
```bash
grpcurl -plaintext \
  -d '{
    "record": {
      "id": {
        "tenant_id": {"id": "tenant-123"},
        "entity_id": {"id": "record-456"},
        "version": 1
      },
      "data": {
        "tenant_id": "tenant-123",
        "id": "record-456",
        "timestamp": "invalid-timestamp",
        "data": {"test": "data"}
      }
    }
  }' \
  localhost:8001 storage.IngestionService/IngestRecord
```

#### Future timestamp (business rule violation):
```bash
grpcurl -plaintext \
  -d '{
    "record": {
      "id": {
        "tenant_id": {"id": "tenant-123"},
        "entity_id": {"id": "record-456"},
        "version": 1
      },
      "data": {
        "tenant_id": "tenant-123",
        "id": "record-456",
        "timestamp": "2025-07-15T14:30:00Z",
        "data": {"test": "data"}
      }
    }
  }' \
  localhost:8001 storage.IngestionService/IngestRecord
```
