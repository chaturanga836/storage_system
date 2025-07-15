# Tests Reference

## Quick Reference

### Test Structure
```
tests/
├── integration/        # End-to-end integration tests
├── performance/        # Performance benchmarks and load tests
└── README.md          # Testing documentation
```

### Running Tests

#### All Tests
```bash
go test ./...                    # All tests
go test -v ./...                 # Verbose output
go test -race ./...              # Race condition detection
go test -cover ./...             # Coverage analysis
```

#### Specific Test Suites
```bash
go test ./tests/integration/...   # Integration tests only
go test ./tests/performance/...   # Performance tests only
go test -bench=. ./tests/performance/...  # Benchmarks only
```

#### Test Filtering
```bash
go test -run TestIntegration ./tests/integration/...
go test -bench BenchmarkInsert ./tests/performance/...
go test -short ./...             # Skip long-running tests
```

### Integration Tests (`integration/`)

#### Key Files
- `integration_test.go` - Main integration test suite
- `multi_tenant_test.go` - Multi-tenant isolation tests (if exists)
- `transaction_test.go` - MVCC transaction tests (if exists)
- `recovery_test.go` - WAL recovery tests (if exists)

#### Test Configuration
```go
type TestConfig struct {
    ServiceEndpoints map[string]string
    TestTimeout      time.Duration
    CleanupEnabled   bool
    LogLevel         string
}

var defaultConfig = &TestConfig{
    ServiceEndpoints: map[string]string{
        "ingestion": "localhost:8080",
        "query":     "localhost:8081",
    },
    TestTimeout:    5 * time.Minute,
    CleanupEnabled: true,
    LogLevel:      "info",
}
```

#### Common Test Patterns
```go
func TestIntegration_DataFlow(t *testing.T) {
    // Setup test environment
    suite := setupTestSuite(t)
    defer suite.cleanup()
    
    // Create test table
    schema := generateTestSchema()
    err := suite.ingestionClient.CreateTable(ctx, schema)
    require.NoError(t, err)
    
    // Insert test data
    records := generateTestRecords(1000)
    err = suite.ingestionClient.InsertRecords(ctx, records)
    require.NoError(t, err)
    
    // Query data
    results, err := suite.queryClient.QueryRecords(ctx, query)
    require.NoError(t, err)
    require.Len(t, results, 1000)
}
```

### Performance Tests (`performance/`)

#### Key Files
- `performance_test.go` - Main performance benchmark suite
- `ingestion_benchmark_test.go` - Ingestion performance (if exists)
- `query_benchmark_test.go` - Query performance (if exists)
- `concurrent_ops_test.go` - Concurrency tests (if exists)

#### Benchmark Configuration
```go
type BenchmarkConfig struct {
    BaseURL     string
    TableName   string
    RecordCount int
    BatchSize   int
    Concurrency int
    Duration    time.Duration
}
```

#### Performance Targets
- **Ingestion**: 1M+ records/second
- **Query Latency**: <10ms for indexed queries
- **Concurrent Operations**: Support 1000+ concurrent clients
- **Memory Usage**: Stable memory consumption under load

#### Benchmark Examples
```go
func BenchmarkInsertThroughput(b *testing.B) {
    suite := NewPerformanceTestSuite(nil)
    // ... setup ...
    
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            suite.client.InsertRecord(ctx, record)
        }
    })
}

func BenchmarkQueryLatency(b *testing.B) {
    // ... setup with test data ...
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        suite.client.QueryRecords(ctx, query)
    }
}
```

### Test Utilities

#### Test Data Generation
```go
// Generate test records
func generateTestRecords(count int) []TestRecord

// Generate test schema
func generateTestSchema() *schema.TableSchema

// Random data generators
func randomString(length int) string
func randomTimestamp() time.Time
func randomCategory() string
```

#### Test Environment Setup
```go
// Setup test environment
func setupTestEnvironment() (*TestSuite, error)

// Cleanup resources
func cleanupTestResources() error

// Wait for services
func waitForServices(endpoints map[string]string, timeout time.Duration) error
```

#### Mock Services (for unit tests)
```go
type MockWALManager struct{}
type MockCatalogService struct{}
type MockMVCCManager struct{}
type MockSchemaRegistry struct{}
```

### Test Data

#### Schema Examples
```go
userSchema := &schema.TableSchema{
    Name: "users",
    Columns: []schema.ColumnSchema{
        {Name: "id", Type: schema.TypeString, Nullable: false},
        {Name: "email", Type: schema.TypeString, Nullable: false},
        {Name: "created_at", Type: schema.TypeTimestamp, Nullable: false},
    },
}
```

#### Record Examples
```go
userRecord := map[string]interface{}{
    "id":         "user-123",
    "email":      "user@example.com", 
    "created_at": time.Now(),
}
```

### Environment Variables
- `TEST_ENDPOINTS` - Service endpoints for testing
- `TEST_TIMEOUT` - Test timeout duration
- `TEST_LOG_LEVEL` - Logging level for tests
- `TEST_CLEANUP` - Enable/disable cleanup
- `TEST_DATA_SIZE` - Size of test datasets

### CI/CD Integration
```yaml
# GitHub Actions example
- name: Run Tests
  run: |
    go test -race -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

- name: Run Performance Tests
  run: |
    go test -bench=. -benchmem ./tests/performance/...
```

### Performance Thresholds
```go
const (
    MaxLatency = 100 * time.Millisecond
    MinThroughput = 100.0  // ops/sec
    MaxErrorRate = 1.0     // percent
    MaxMemoryGrowth = 10.0 // percent per hour
)
```

### Troubleshooting Tests

#### Common Issues
1. **Service connectivity** - Ensure services are running
2. **Resource exhaustion** - Monitor memory/CPU during tests
3. **Timing issues** - Increase timeouts for slow operations
4. **Data conflicts** - Use unique test data or proper cleanup

#### Debug Commands
```bash
# Verbose test output
go test -v ./tests/...

# Debug specific test
go test -run TestSpecific -v ./tests/integration/

# Profile performance tests
go test -bench=. -cpuprofile=cpu.prof ./tests/performance/
```

### Dependencies
- Running storage system services
- Test data generation utilities  
- Network connectivity to services
- Sufficient resources (CPU, memory, disk)
