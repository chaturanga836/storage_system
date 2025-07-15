# Testing Documentation

## Overview

This directory contains comprehensive test suites for the storage system, including integration tests and performance benchmarks.

## Test Structure

### Integration Tests (`integration/`)
End-to-end tests that verify the interaction between different system components.

**Key Test Scenarios:**
- Multi-service data flow (ingestion → storage → query)
- MVCC transaction handling
- WAL replay and recovery
- Schema evolution compatibility
- Multi-tenant isolation

**Running Integration Tests:**
```bash
go test ./tests/integration/...
```

### Performance Tests (`performance/`)
Benchmarks and load tests to validate system performance targets.

**Performance Metrics:**
- Ingestion throughput (target: 1M+ records/sec)
- Query latency (target: <10ms)
- Concurrent operations handling
- Memory and CPU utilization
- Storage efficiency

**Running Performance Tests:**
```bash
go test ./tests/performance/...
```

## Test Configuration

### Environment Setup
Tests require certain services to be running:

```bash
# Start required services for integration tests
./scripts/run_local.sh

# Or use Docker Compose
docker-compose -f deployments/docker/docker-compose.yml up
```

### Test Data
Tests use generated test data with configurable parameters:

```go
// Example test configuration
config := &TestConfig{
    RecordCount: 100000,
    BatchSize:   1000,
    Concurrency: 10,
    TableName:   "test_table",
}
```

## Key Test Files

### Integration Tests
- `integration_test.go`: Main integration test suite
- `multi_tenant_test.go`: Multi-tenant isolation tests
- `transaction_test.go`: MVCC transaction tests
- `recovery_test.go`: WAL recovery tests

### Performance Tests
- `performance_test.go`: Main performance benchmark suite
- `ingestion_benchmark_test.go`: Ingestion performance tests
- `query_benchmark_test.go`: Query performance tests
- `concurrent_ops_test.go`: Concurrency tests

## Test Utilities

### Test Helpers
Common utilities for test setup and data generation:

```go
// Setup test environment
func setupTestEnvironment() (*TestSuite, error)

// Generate test data
func generateTestRecords(count int) []TestRecord

// Cleanup test resources
func cleanupTestResources() error
```

### Mock Services
Mock implementations for isolated unit testing:

- Mock WAL manager
- Mock catalog service
- Mock MVCC manager
- Mock schema registry

## Running Tests

### All Tests
```bash
go test ./...
```

### Specific Test Suites
```bash
# Integration tests only
go test ./tests/integration/...

# Performance tests only
go test ./tests/performance/...

# With verbose output
go test -v ./tests/...

# With race detection
go test -race ./tests/...
```

### Continuous Integration
Tests are automatically run in CI/CD pipeline:

```yaml
# GitHub Actions example
- name: Run Tests
  run: |
    go test -race -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
```

## Performance Benchmarks

### Benchmark Results Format
```
BenchmarkInsertThroughput-8          1000000      1200 ns/op
BenchmarkQueryLatency-8             5000000       250 ns/op
BenchmarkConcurrentOperations-8      500000      2400 ns/op
```

### Performance Thresholds
Tests fail if performance degrades below thresholds:

- Ingestion: <900k records/sec (90% of target)
- Query latency: >15ms (150% of target)
- Error rate: >1%
- Memory growth: >10% per hour

## Troubleshooting Tests

### Common Issues

1. **Service Connectivity**: Tests fail to connect to services
   - Ensure services are running on expected ports
   - Check firewall and network configuration

2. **Resource Exhaustion**: Tests consume too much memory/CPU
   - Reduce test data size or concurrency
   - Monitor system resources during tests

3. **Timing Issues**: Tests fail due to timing/race conditions
   - Increase timeouts for slow operations
   - Use proper synchronization in tests

### Debug Mode
Enable verbose logging for troubleshooting:

```bash
go test -v -args -debug ./tests/...
```

## Contributing Tests

### Test Guidelines
1. Each major feature should have integration tests
2. Performance-critical paths need benchmarks
3. Use table-driven tests for multiple scenarios
4. Include negative test cases
5. Clean up resources in test teardown

### Test Naming Convention
```go
func TestIntegration_MultiTenant_DataIsolation(t *testing.T)
func BenchmarkIngestion_BatchSize1000(b *testing.B)
func TestUnit_SchemaEvolution_BackwardCompatibility(t *testing.T)
```
