package performance

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"storage-engine/internal/api/client"
	"storage-engine/internal/schema"
)

// BenchmarkConfig holds configuration for performance tests
type BenchmarkConfig struct {
	BaseURL     string
	TableName   string
	RecordCount int
	BatchSize   int
	Concurrency int
	Duration    time.Duration
}

// DefaultBenchmarkConfig returns default benchmark configuration
func DefaultBenchmarkConfig() *BenchmarkConfig {
	return &BenchmarkConfig{
		BaseURL:     "http://localhost:8080",
		TableName:   "perf_test_table",
		RecordCount: 100000,
		BatchSize:   1000,
		Concurrency: 10,
		Duration:    5 * time.Minute,
	}
}

// PerformanceTestSuite contains performance tests
type PerformanceTestSuite struct {
	config *BenchmarkConfig
	client *client.Client
}

// NewPerformanceTestSuite creates a new performance test suite
func NewPerformanceTestSuite(config *BenchmarkConfig) *PerformanceTestSuite {
	if config == nil {
		config = DefaultBenchmarkConfig()
	}

	clientConfig := &client.ClientConfig{
		BaseURL: config.BaseURL,
		Timeout: 30 * time.Second,
	}

	return &PerformanceTestSuite{
		config: config,
		client: client.NewClient(clientConfig),
	}
}

// BenchmarkInsertThroughput benchmarks insert throughput
func BenchmarkInsertThroughput(b *testing.B) {
	suite := NewPerformanceTestSuite(nil)
	ctx := context.Background()

	// Setup test table
	err := suite.setupTestTable(ctx)
	require.NoError(b, err)
	defer suite.cleanupTestTable(ctx)

	b.ResetTimer()
	b.ReportAllocs()

	// Generate test data
	records := suite.generateTestRecords(b.N)

	b.RunParallel(func(pb *testing.PB) {
		recordIndex := 0
		for pb.Next() {
			if recordIndex >= len(records) {
				recordIndex = 0
			}

			_, err := suite.client.InsertRecord(ctx, suite.config.TableName, records[recordIndex])
			require.NoError(b, err)
			recordIndex++
		}
	})
}

// BenchmarkBatchInsertThroughput benchmarks batch insert throughput
func BenchmarkBatchInsertThroughput(b *testing.B) {
	suite := NewPerformanceTestSuite(nil)
	ctx := context.Background()

	// Setup test table
	err := suite.setupTestTable(ctx)
	require.NoError(b, err)
	defer suite.cleanupTestTable(ctx)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		records := suite.generateTestRecords(suite.config.BatchSize)
		_, err := suite.client.InsertRecords(ctx, suite.config.TableName, records)
		require.NoError(b, err)
	}
}

// BenchmarkQueryLatency benchmarks query latency
func BenchmarkQueryLatency(b *testing.B) {
	suite := NewPerformanceTestSuite(nil)
	ctx := context.Background()

	// Setup test table and data
	err := suite.setupTestTable(ctx)
	require.NoError(b, err)
	defer suite.cleanupTestTable(ctx)

	// Insert test data
	records := suite.generateTestRecords(10000)
	batches := suite.batchRecords(records, suite.config.BatchSize)
	for _, batch := range batches {
		_, err := suite.client.InsertRecords(ctx, suite.config.TableName, batch)
		require.NoError(b, err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Random query
			queryRequest := &client.QueryRequest{
				TableName: suite.config.TableName,
				Filters: map[string]interface{}{
					"category": suite.randomCategory(),
				},
				Limit: 100,
			}

			_, err := suite.client.QueryRecords(ctx, queryRequest)
			require.NoError(b, err)
		}
	})
}

// BenchmarkConcurrentOperations benchmarks concurrent mixed operations
func BenchmarkConcurrentOperations(b *testing.B) {
	suite := NewPerformanceTestSuite(nil)
	ctx := context.Background()

	// Setup test table and initial data
	err := suite.setupTestTable(ctx)
	require.NoError(b, err)
	defer suite.cleanupTestTable(ctx)

	// Insert initial data
	initialRecords := suite.generateTestRecords(1000)
	batches := suite.batchRecords(initialRecords, suite.config.BatchSize)
	for _, batch := range batches {
		_, err := suite.client.InsertRecords(ctx, suite.config.TableName, batch)
		require.NoError(b, err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Mixed workload: 70% reads, 30% writes
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if rand.Float32() < 0.7 {
				// Read operation
				queryRequest := &client.QueryRequest{
					TableName: suite.config.TableName,
					Filters: map[string]interface{}{
						"category": suite.randomCategory(),
					},
					Limit: 50,
				}
				_, err := suite.client.QueryRecords(ctx, queryRequest)
				require.NoError(b, err)
			} else {
				// Write operation
				record := suite.generateTestRecord()
				_, err := suite.client.InsertRecord(ctx, suite.config.TableName, record)
				require.NoError(b, err)
			}
		}
	})
}

// LoadTest performs sustained load testing
func (suite *PerformanceTestSuite) LoadTest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), suite.config.Duration)
	defer cancel()

	// Setup test table
	err := suite.setupTestTable(ctx)
	require.NoError(t, err)
	defer suite.cleanupTestTable(ctx)

	// Metrics tracking
	var totalOps int64
	var totalLatency time.Duration
	var errorCount int64
	var mu sync.Mutex

	// Start load generators
	var wg sync.WaitGroup
	for i := 0; i < suite.config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			suite.runLoadWorker(ctx, workerID, &totalOps, &totalLatency, &errorCount, &mu)
		}(i)
	}

	// Wait for completion
	wg.Wait()

	// Report results
	mu.Lock()
	avgLatency := totalLatency / time.Duration(totalOps)
	opsPerSecond := float64(totalOps) / suite.config.Duration.Seconds()
	errorRate := float64(errorCount) / float64(totalOps) * 100
	mu.Unlock()

	t.Logf("Load Test Results:")
	t.Logf("  Duration: %v", suite.config.Duration)
	t.Logf("  Total Operations: %d", totalOps)
	t.Logf("  Operations/sec: %.2f", opsPerSecond)
	t.Logf("  Average Latency: %v", avgLatency)
	t.Logf("  Error Rate: %.2f%%", errorRate)

	// Assert performance thresholds
	require.Less(t, avgLatency, 100*time.Millisecond, "Average latency too high")
	require.Less(t, errorRate, 1.0, "Error rate too high")
	require.Greater(t, opsPerSecond, 100.0, "Throughput too low")
}

// StressTest performs stress testing with increasing load
func (suite *PerformanceTestSuite) StressTest(t *testing.T) {
	ctx := context.Background()

	// Setup test table
	err := suite.setupTestTable(ctx)
	require.NoError(t, err)
	defer suite.cleanupTestTable(ctx)

	concurrencyLevels := []int{1, 5, 10, 20, 50, 100}
	results := make([]StressTestResult, 0, len(concurrencyLevels))

	for _, concurrency := range concurrencyLevels {
		t.Logf("Running stress test with concurrency: %d", concurrency)

		result := suite.runStressTestLevel(ctx, concurrency, 30*time.Second)
		results = append(results, result)

		t.Logf("  Concurrency %d: %.2f ops/sec, %.2fms avg latency, %.2f%% error rate",
			concurrency, result.OpsPerSecond, result.AvgLatency.Seconds()*1000, result.ErrorRate)

		// Break if performance degrades significantly
		if result.ErrorRate > 5.0 || result.AvgLatency > 500*time.Millisecond {
			t.Logf("Performance degradation detected at concurrency %d", concurrency)
			break
		}
	}

	// Report final results
	t.Logf("\nStress Test Summary:")
	for _, result := range results {
		t.Logf("  Concurrency %d: %.2f ops/sec", result.Concurrency, result.OpsPerSecond)
	}
}

// Helper methods

// setupTestTable creates a test table for benchmarking
func (suite *PerformanceTestSuite) setupTestTable(ctx context.Context) error {
	tableSchema := &schema.TableSchema{
		Name: suite.config.TableName,
		Columns: []schema.ColumnSchema{
			{Name: "id", Type: schema.TypeString, Nullable: false},
			{Name: "name", Type: schema.TypeString, Nullable: false},
			{Name: "category", Type: schema.TypeString, Nullable: false},
			{Name: "value", Type: schema.TypeFloat64, Nullable: false},
			{Name: "active", Type: schema.TypeBoolean, Nullable: false},
			{Name: "created_at", Type: schema.TypeTimestamp, Nullable: false},
			{Name: "metadata", Type: schema.TypeString, Nullable: true},
		},
	}

	return suite.client.CreateTable(ctx, tableSchema)
}

// cleanupTestTable drops the test table
func (suite *PerformanceTestSuite) cleanupTestTable(ctx context.Context) {
	_ = suite.client.DropTable(ctx, suite.config.TableName)
}

// generateTestRecords generates test records for benchmarking
func (suite *PerformanceTestSuite) generateTestRecords(count int) []map[string]interface{} {
	records := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		records[i] = suite.generateTestRecord()
	}
	return records
}

// generateTestRecord generates a single test record
func (suite *PerformanceTestSuite) generateTestRecord() map[string]interface{} {
	return map[string]interface{}{
		"id":         fmt.Sprintf("perf-test-%d-%d", time.Now().UnixNano(), rand.Intn(1000000)),
		"name":       suite.randomName(),
		"category":   suite.randomCategory(),
		"value":      rand.Float64() * 1000,
		"active":     rand.Intn(2) == 1,
		"created_at": time.Now(),
		"metadata":   suite.randomMetadata(),
	}
}

// randomName generates a random name
func (suite *PerformanceTestSuite) randomName() string {
	names := []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Henry"}
	return names[rand.Intn(len(names))] + fmt.Sprintf("-%d", rand.Intn(1000))
}

// randomCategory generates a random category
func (suite *PerformanceTestSuite) randomCategory() string {
	categories := []string{"Electronics", "Books", "Clothing", "Home", "Sports", "Toys", "Food", "Health"}
	return categories[rand.Intn(len(categories))]
}

// randomMetadata generates random metadata JSON
func (suite *PerformanceTestSuite) randomMetadata() string {
	metadata := []string{
		`{"source": "api", "version": "1.0"}`,
		`{"source": "batch", "priority": "high"}`,
		`{"source": "stream", "processed": true}`,
		`{"source": "manual", "verified": false}`,
	}
	return metadata[rand.Intn(len(metadata))]
}

// batchRecords splits records into batches
func (suite *PerformanceTestSuite) batchRecords(records []map[string]interface{}, batchSize int) [][]map[string]interface{} {
	var batches [][]map[string]interface{}
	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batches = append(batches, records[i:end])
	}
	return batches
}

// runLoadWorker runs a load testing worker
func (suite *PerformanceTestSuite) runLoadWorker(
	ctx context.Context,
	workerID int,
	totalOps *int64,
	totalLatency *time.Duration,
	errorCount *int64,
	mu *sync.Mutex,
) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			start := time.Now()
			var err error

			// 70% reads, 30% writes
			if rand.Float32() < 0.7 {
				queryRequest := &client.QueryRequest{
					TableName: suite.config.TableName,
					Filters: map[string]interface{}{
						"category": suite.randomCategory(),
					},
					Limit: 10,
				}
				_, err = suite.client.QueryRecords(ctx, queryRequest)
			} else {
				record := suite.generateTestRecord()
				_, err = suite.client.InsertRecord(ctx, suite.config.TableName, record)
			}

			latency := time.Since(start)

			mu.Lock()
			*totalOps++
			*totalLatency += latency
			if err != nil {
				*errorCount++
			}
			mu.Unlock()
		}
	}
}

// runStressTestLevel runs stress test at a specific concurrency level
func (suite *PerformanceTestSuite) runStressTestLevel(ctx context.Context, concurrency int, duration time.Duration) StressTestResult {
	testCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	var totalOps int64
	var totalLatency time.Duration
	var errorCount int64
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			suite.runLoadWorker(testCtx, 0, &totalOps, &totalLatency, &errorCount, &mu)
		}()
	}

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	var avgLatency time.Duration
	if totalOps > 0 {
		avgLatency = totalLatency / time.Duration(totalOps)
	}

	return StressTestResult{
		Concurrency:  concurrency,
		TotalOps:     totalOps,
		OpsPerSecond: float64(totalOps) / duration.Seconds(),
		AvgLatency:   avgLatency,
		ErrorCount:   errorCount,
		ErrorRate:    float64(errorCount) / float64(totalOps) * 100,
	}
}

// StressTestResult represents the result of a stress test level
type StressTestResult struct {
	Concurrency  int
	TotalOps     int64
	OpsPerSecond float64
	AvgLatency   time.Duration
	ErrorCount   int64
	ErrorRate    float64
}

// Test functions that can be called from go test

func TestLoadTest(t *testing.T) {
	suite := NewPerformanceTestSuite(nil)
	suite.LoadTest(t)
}

func TestStressTest(t *testing.T) {
	suite := NewPerformanceTestSuite(nil)
	suite.StressTest(t)
}
