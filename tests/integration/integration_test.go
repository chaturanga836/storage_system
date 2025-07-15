package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"storage-engine/internal/api/client"
	"storage-engine/internal/schema"
)

// IntegrationTestSuite contains all integration tests
type IntegrationTestSuite struct {
	suite.Suite
	client    *client.Client
	testTable string
}

// SetupSuite runs once before all tests
func (suite *IntegrationTestSuite) SetupSuite() {
	// Initialize client
	config := &client.ClientConfig{
		BaseURL: "http://localhost:8080",
		Timeout: 30 * time.Second,
	}
	suite.client = client.NewClient(config)
	suite.testTable = "integration_test_table"

	// Wait for services to be ready
	suite.waitForServices()
}

// TearDownSuite runs once after all tests
func (suite *IntegrationTestSuite) TearDownSuite() {
	// Clean up test table
	ctx := context.Background()
	_ = suite.client.DropTable(ctx, suite.testTable)
}

// SetupTest runs before each test
func (suite *IntegrationTestSuite) SetupTest() {
	// Create test table
	ctx := context.Background()
	tableSchema := &schema.TableSchema{
		Name: suite.testTable,
		Columns: []schema.ColumnSchema{
			{
				Name:     "id",
				Type:     schema.TypeString,
				Nullable: false,
			},
			{
				Name:     "name",
				Type:     schema.TypeString,
				Nullable: false,
			},
			{
				Name:     "age",
				Type:     schema.TypeInt64,
				Nullable: true,
			},
			{
				Name:     "created_at",
				Type:     schema.TypeTimestamp,
				Nullable: false,
			},
		},
	}

	err := suite.client.CreateTable(ctx, tableSchema)
	require.NoError(suite.T(), err)
}

// TearDownTest runs after each test
func (suite *IntegrationTestSuite) TearDownTest() {
	// Drop test table
	ctx := context.Background()
	_ = suite.client.DropTable(ctx, suite.testTable)
}

// TestTableOperations tests basic table operations
func (suite *IntegrationTestSuite) TestTableOperations() {
	ctx := context.Background()

	// Test listing tables
	tables, err := suite.client.ListTables(ctx)
	require.NoError(suite.T(), err)
	assert.Contains(suite.T(), tableNames(tables), suite.testTable)

	// Test getting table info
	tableInfo, err := suite.client.GetTable(ctx, suite.testTable)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), suite.testTable, tableInfo.Name)
	assert.Len(suite.T(), tableInfo.Schema.Columns, 4)
}

// TestDataIngestion tests data ingestion and retrieval
func (suite *IntegrationTestSuite) TestDataIngestion() {
	ctx := context.Background()

	// Insert single record
	record := map[string]interface{}{
		"id":         "test-001",
		"name":       "Test User",
		"age":        25,
		"created_at": time.Now(),
	}

	insertResult, err := suite.client.InsertRecord(ctx, suite.testTable, record)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, insertResult.InsertedCount)
	assert.Len(suite.T(), insertResult.RecordIDs, 1)

	// Query records
	queryRequest := &client.QueryRequest{
		TableName: suite.testTable,
		Filters: map[string]interface{}{
			"id": "test-001",
		},
	}

	queryResult, err := suite.client.QueryRecords(ctx, queryRequest)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, queryResult.Total)
	assert.Len(suite.T(), queryResult.Records, 1)

	retrievedRecord := queryResult.Records[0]
	assert.Equal(suite.T(), "test-001", retrievedRecord["id"])
	assert.Equal(suite.T(), "Test User", retrievedRecord["name"])
	assert.Equal(suite.T(), float64(25), retrievedRecord["age"]) // JSON numbers are float64
}

// TestBatchIngestion tests batch data ingestion
func (suite *IntegrationTestSuite) TestBatchIngestion() {
	ctx := context.Background()

	// Insert multiple records
	records := []map[string]interface{}{
		{
			"id":         "batch-001",
			"name":       "Batch User 1",
			"age":        30,
			"created_at": time.Now(),
		},
		{
			"id":         "batch-002",
			"name":       "Batch User 2",
			"age":        35,
			"created_at": time.Now(),
		},
		{
			"id":         "batch-003",
			"name":       "Batch User 3",
			"age":        40,
			"created_at": time.Now(),
		},
	}

	insertResult, err := suite.client.InsertRecords(ctx, suite.testTable, records)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 3, insertResult.InsertedCount)
	assert.Len(suite.T(), insertResult.RecordIDs, 3)

	// Query all records
	queryRequest := &client.QueryRequest{
		TableName: suite.testTable,
	}

	queryResult, err := suite.client.QueryRecords(ctx, queryRequest)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 3, queryResult.Total)
	assert.Len(suite.T(), queryResult.Records, 3)
}

// TestQueryFiltering tests query filtering capabilities
func (suite *IntegrationTestSuite) TestQueryFiltering() {
	ctx := context.Background()

	// Insert test data
	records := []map[string]interface{}{
		{
			"id":         "filter-001",
			"name":       "Young User",
			"age":        20,
			"created_at": time.Now(),
		},
		{
			"id":         "filter-002",
			"name":       "Old User",
			"age":        60,
			"created_at": time.Now(),
		},
		{
			"id":         "filter-003",
			"name":       "Middle User",
			"age":        35,
			"created_at": time.Now(),
		},
	}

	_, err := suite.client.InsertRecords(ctx, suite.testTable, records)
	require.NoError(suite.T(), err)

	// Test filtering by age
	queryRequest := &client.QueryRequest{
		TableName: suite.testTable,
		Filters: map[string]interface{}{
			"age": 35,
		},
	}

	queryResult, err := suite.client.QueryRecords(ctx, queryRequest)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, queryResult.Total)
	assert.Equal(suite.T(), "Middle User", queryResult.Records[0]["name"])

	// Test projection
	queryRequest = &client.QueryRequest{
		TableName:   suite.testTable,
		Projections: []string{"id", "name"},
	}

	queryResult, err = suite.client.QueryRecords(ctx, queryRequest)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 3, queryResult.Total)

	for _, record := range queryResult.Records {
		assert.Contains(suite.T(), record, "id")
		assert.Contains(suite.T(), record, "name")
		assert.NotContains(suite.T(), record, "age")
		assert.NotContains(suite.T(), record, "created_at")
	}
}

// TestSchemaEvolution tests schema evolution capabilities
func (suite *IntegrationTestSuite) TestSchemaEvolution() {
	ctx := context.Background()

	// Get current schema
	currentSchema, err := suite.client.GetSchema(ctx, suite.testTable)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), currentSchema.Columns, 4)

	// Add a new column
	newSchema := *currentSchema
	newSchema.Columns = append(newSchema.Columns, schema.ColumnSchema{
		Name:     "email",
		Type:     schema.TypeString,
		Nullable: true,
	})

	err = suite.client.UpdateSchema(ctx, suite.testTable, &newSchema)
	require.NoError(suite.T(), err)

	// Verify schema was updated
	updatedSchema, err := suite.client.GetSchema(ctx, suite.testTable)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), updatedSchema.Columns, 5)

	// Insert record with new column
	record := map[string]interface{}{
		"id":         "schema-001",
		"name":       "Schema Test User",
		"age":        30,
		"email":      "test@example.com",
		"created_at": time.Now(),
	}

	_, err = suite.client.InsertRecord(ctx, suite.testTable, record)
	require.NoError(suite.T(), err)
}

// TestIndexOperations tests index creation and usage
func (suite *IntegrationTestSuite) TestIndexOperations() {
	ctx := context.Background()

	// Create an index
	indexDef := &client.IndexDefinition{
		Name:      "idx_name",
		TableName: suite.testTable,
		Columns:   []string{"name"},
		Type:      "btree",
	}

	err := suite.client.CreateIndex(ctx, indexDef)
	require.NoError(suite.T(), err)

	// List indexes
	indexes, err := suite.client.ListIndexes(ctx, suite.testTable)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), indexes, 1)
	assert.Equal(suite.T(), "idx_name", indexes[0].Name)

	// Insert data and query (should use index)
	records := []map[string]interface{}{
		{
			"id":         "idx-001",
			"name":       "Alice",
			"age":        25,
			"created_at": time.Now(),
		},
		{
			"id":         "idx-002",
			"name":       "Bob",
			"age":        30,
			"created_at": time.Now(),
		},
	}

	_, err = suite.client.InsertRecords(ctx, suite.testTable, records)
	require.NoError(suite.T(), err)

	// Query using indexed column
	queryRequest := &client.QueryRequest{
		TableName: suite.testTable,
		Filters: map[string]interface{}{
			"name": "Alice",
		},
	}

	queryResult, err := suite.client.QueryRecords(ctx, queryRequest)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, queryResult.Total)
	assert.Equal(suite.T(), "Alice", queryResult.Records[0]["name"])

	// Drop index
	err = suite.client.DropIndex(ctx, "idx_name")
	require.NoError(suite.T(), err)
}

// TestSystemOperations tests system-level operations
func (suite *IntegrationTestSuite) TestSystemOperations() {
	ctx := context.Background()

	// Get system status
	status, err := suite.client.GetStatus(ctx)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "healthy", status.Status)

	// Get system metrics
	metrics, err := suite.client.GetMetrics(ctx)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), metrics)
	assert.GreaterOrEqual(suite.T(), metrics.IngestedRecords, int64(0))
}

// Helper methods

// waitForServices waits for all services to be ready
func (suite *IntegrationTestSuite) waitForServices() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			suite.T().Fatal("Services did not become ready in time")
		default:
			if suite.areServicesReady(ctx) {
				return
			}
			time.Sleep(5 * time.Second)
		}
	}
}

// areServicesReady checks if all services are ready
func (suite *IntegrationTestSuite) areServicesReady(ctx context.Context) bool {
	_, err := suite.client.GetStatus(ctx)
	return err == nil
}

// tableNames extracts table names from table info slice
func tableNames(tables []*client.TableInfo) []string {
	names := make([]string, len(tables))
	for i, table := range tables {
		names[i] = table.Name
	}
	return names
}

// TestIntegrationSuite runs the integration test suite
func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
