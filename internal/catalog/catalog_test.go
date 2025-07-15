package catalog

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockPersistenceLayer implements the PersistenceLayer interface for testing
// This allows us to test the catalog without needing a real database
type MockPersistenceLayer struct {
	files   map[string]*FileMetadata
	schemas map[string]*SchemaMetadata
	stats   map[string]*ColumnStatistics
}

// NewMockPersistenceLayer creates a new mock persistence layer
func NewMockPersistenceLayer() *MockPersistenceLayer {
	return &MockPersistenceLayer{
		files:   make(map[string]*FileMetadata),
		schemas: make(map[string]*SchemaMetadata),
		stats:   make(map[string]*ColumnStatistics),
	}
}

// Implement all required PersistenceLayer interface methods
func (m *MockPersistenceLayer) Save(ctx context.Context) error                       { return nil }
func (m *MockPersistenceLayer) Load(ctx context.Context) error                       { return nil }
func (m *MockPersistenceLayer) Backup(ctx context.Context) error                     { return nil }
func (m *MockPersistenceLayer) Restore(ctx context.Context, backupPath string) error { return nil }
func (m *MockPersistenceLayer) Health(ctx context.Context) error                     { return nil }
func (m *MockPersistenceLayer) Close() error                                         { return nil }
func (m *MockPersistenceLayer) Compact(ctx context.Context) error                    { return nil }

// File metadata operations
func (m *MockPersistenceLayer) StoreFileMetadata(ctx context.Context, metadata *FileMetadata) error {
	m.files[metadata.Path] = metadata
	return nil
}

func (m *MockPersistenceLayer) GetFileMetadata(ctx context.Context, path string) (*FileMetadata, error) {
	if file, exists := m.files[path]; exists {
		return file, nil
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

func (m *MockPersistenceLayer) DeleteFileMetadata(ctx context.Context, path string) error {
	delete(m.files, path)
	return nil
}

func (m *MockPersistenceLayer) ListAllFiles(ctx context.Context) ([]*FileMetadata, error) {
	var result []*FileMetadata
	for _, file := range m.files {
		result = append(result, file)
	}
	return result, nil
}

// Schema metadata operations
func (m *MockPersistenceLayer) StoreSchemaMetadata(ctx context.Context, metadata *SchemaMetadata) error {
	key := fmt.Sprintf("%s:%d", metadata.TenantID, metadata.Version)
	m.schemas[key] = metadata
	return nil
}

func (m *MockPersistenceLayer) GetSchemaMetadata(ctx context.Context, tenantID string, version int) (*SchemaMetadata, error) {
	key := fmt.Sprintf("%s:%d", tenantID, version)
	if schema, exists := m.schemas[key]; exists {
		return schema, nil
	}
	return nil, fmt.Errorf("schema not found: %s:%d", tenantID, version)
}

func (m *MockPersistenceLayer) GetLatestSchemaMetadata(ctx context.Context, tenantID string) (*SchemaMetadata, error) {
	var latest *SchemaMetadata
	for _, schema := range m.schemas {
		if schema.TenantID == tenantID {
			if latest == nil || schema.Version > latest.Version {
				latest = schema
			}
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("no schema found for tenant: %s", tenantID)
	}
	return latest, nil
}

func (m *MockPersistenceLayer) ListSchemaMetadata(ctx context.Context, tenantID string) ([]*SchemaMetadata, error) {
	var result []*SchemaMetadata
	for _, schema := range m.schemas {
		if schema.TenantID == tenantID {
			result = append(result, schema)
		}
	}
	return result, nil
}

func (m *MockPersistenceLayer) ListAllSchemas(ctx context.Context) ([]*SchemaMetadata, error) {
	var result []*SchemaMetadata
	for _, schema := range m.schemas {
		result = append(result, schema)
	}
	return result, nil
}

// Statistics operations
func (m *MockPersistenceLayer) StoreColumnStats(ctx context.Context, stats *ColumnStatistics) error {
	key := fmt.Sprintf("%s:%s", stats.TenantID, stats.ColumnName)
	m.stats[key] = stats
	return nil
}

func (m *MockPersistenceLayer) GetColumnStats(ctx context.Context, tenantID, column string) (*ColumnStatistics, error) {
	key := fmt.Sprintf("%s:%s", tenantID, column)
	if stats, exists := m.stats[key]; exists {
		return stats, nil
	}
	return nil, fmt.Errorf("stats not found: %s:%s", tenantID, column)
}

func (m *MockPersistenceLayer) GetTableStats(ctx context.Context, tenantID string) (*TableStatistics, error) {
	return nil, fmt.Errorf("table stats not found: %s", tenantID)
}

// Compaction operations
func (m *MockPersistenceLayer) StoreCompactionJob(ctx context.Context, job *CompactionJob) error {
	return nil
}

func (m *MockPersistenceLayer) GetCompactionCandidates(ctx context.Context, maxFiles int) ([]*CompactionJob, error) {
	return nil, nil
}

// Transaction operations
func (m *MockPersistenceLayer) BeginTransaction(ctx context.Context) (Transaction, error) {
	return &TransactionImpl{
		id:        "mock-tx-123",
		startTime: time.Now(),
		active:    true,
	}, nil
}

// Helper function to create a test catalog with mock persistence
func createTestCatalog(t testing.TB) *CatalogImpl {
	mockPersistence := NewMockPersistenceLayer()
	config := Config{
		CacheSize:         1024 * 1024, // 1MB cache
		CacheTTL:          5 * time.Minute,
		CompactionWorkers: 2,
		StatsTTL:          10 * time.Minute,
		BatchSize:         100,
	}

	catalog, err := NewCatalog(mockPersistence, config)
	require.NoError(t, err)
	return catalog
}

// TEST: Register a file in the catalog and verify it can be retrieved
func TestCatalog_RegisterFile(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	ctx := context.Background()

	// Create file metadata
	fileMetadata := &FileMetadata{
		Path:              "/data/test-table/file-001.parquet",
		TenantID:          "tenant-123",
		Size:              1024000, // 1MB file
		RecordCount:       10000,   // 10K records
		RowGroupCount:     10,      // 10 row groups
		CompressionType:   "snappy",
		FormatVersion:     "1.0",
		SchemaVersion:     1,
		SchemaFingerprint: "abc123",
		Status:            FileStatusActive,
		CompactionLevel:   0,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Version:           1,
	}

	// Register the file
	err := catalog.RegisterFile(ctx, fileMetadata)
	require.NoError(t, err, "Should be able to register file")

	// Retrieve the file and verify it matches
	retrievedFile, err := catalog.GetFile(ctx, fileMetadata.Path)
	require.NoError(t, err, "Should be able to retrieve registered file")
	require.NotNil(t, retrievedFile, "Retrieved file should not be nil")

	// Verify all important fields match
	assert.Equal(t, fileMetadata.Path, retrievedFile.Path)
	assert.Equal(t, fileMetadata.TenantID, retrievedFile.TenantID)
	assert.Equal(t, fileMetadata.Size, retrievedFile.Size)
	assert.Equal(t, fileMetadata.RecordCount, retrievedFile.RecordCount)
	assert.Equal(t, FileStatusActive, retrievedFile.Status)
}

// TEST: Register a schema and verify it can be retrieved
func TestCatalog_RegisterSchema(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	ctx := context.Background()

	// Create schema metadata
	schemaMetadata := &SchemaMetadata{
		TenantID:    "tenant-123",
		Version:     1,
		Fingerprint: "schema-fingerprint-123",
		CreatedAt:   time.Now(),
		CreatedBy:   "test-user",
		Description: "Test table schema",
		Fields: []*FieldMetadata{
			{
				Name:        "id",
				Type:        "string",
				Nullable:    false,
				Description: "Primary key",
			},
			{
				Name:        "email",
				Type:        "string",
				Nullable:    false,
				Description: "User email",
			},
			{
				Name:        "created_at",
				Type:        "timestamp",
				Nullable:    false,
				Description: "Creation timestamp",
			},
		},
		Status:   SchemaStatusActive,
		Breaking: false,
	}

	// Register the schema
	err := catalog.RegisterSchema(ctx, schemaMetadata)
	require.NoError(t, err, "Should be able to register schema")

	// Retrieve the schema and verify it matches
	retrievedSchema, err := catalog.GetSchema(ctx, "tenant-123", 1)
	require.NoError(t, err, "Should be able to retrieve registered schema")
	require.NotNil(t, retrievedSchema, "Retrieved schema should not be nil")

	// Verify all important fields match
	assert.Equal(t, "tenant-123", retrievedSchema.TenantID)
	assert.Equal(t, 1, retrievedSchema.Version)
	assert.Equal(t, "schema-fingerprint-123", retrievedSchema.Fingerprint)
	assert.Len(t, retrievedSchema.Fields, 3, "Should have 3 fields")
	assert.Equal(t, SchemaStatusActive, retrievedSchema.Status)
}

// TEST: Try to get a schema that doesn't exist
func TestCatalog_GetSchema_NotFound(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	ctx := context.Background()

	// Try to get a non-existent schema
	schema, err := catalog.GetSchema(ctx, "non-existent-tenant", 1)
	assert.Error(t, err, "Should get error for non-existent schema")
	assert.Nil(t, schema, "Schema should be nil when not found")
	assert.Contains(t, err.Error(), "not found", "Error should mention 'not found'")
}

// TEST: List files with and without filters
func TestCatalog_ListFiles(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	ctx := context.Background()
	tenantID := "tenant-123"

	// Create multiple test files
	files := []*FileMetadata{
		{
			Path:            "/data/table1/file-001.parquet",
			TenantID:        tenantID,
			Size:            1000,
			RecordCount:     100,
			Status:          FileStatusActive,
			CompactionLevel: 0,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			Version:         1,
		},
		{
			Path:            "/data/table1/file-002.parquet",
			TenantID:        tenantID,
			Size:            2000,
			RecordCount:     200,
			Status:          FileStatusActive,
			CompactionLevel: 0,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			Version:         1,
		},
		{
			Path:            "/data/table2/file-003.parquet",
			TenantID:        "different-tenant",
			Size:            3000,
			RecordCount:     300,
			Status:          FileStatusActive,
			CompactionLevel: 0,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			Version:         1,
		},
	}

	// Register all files
	for _, file := range files {
		err := catalog.RegisterFile(ctx, file)
		require.NoError(t, err)
	}

	// List all files (no filter)
	allFiles, err := catalog.ListFiles(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, allFiles, 3, "Should list all 3 files")

	// List with tenant filter
	filter := &FileFilter{
		TenantID: tenantID,
	}
	tenantFiles, err := catalog.ListFiles(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, tenantFiles, 2, "Should list only 2 files for tenant-123")

	// Verify all returned files belong to the correct tenant
	for _, file := range tenantFiles {
		assert.Equal(t, tenantID, file.TenantID)
	}
}

// TEST: Update column statistics
func TestCatalog_UpdateColumnStats(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	ctx := context.Background()

	// Create column statistics
	distinctCount := int64(9995)
	stats := &ColumnStatistics{
		TenantID:      "tenant-123",
		ColumnName:    "email",
		RecordCount:   10000,
		NullCount:     5,
		DistinctCount: &distinctCount,
		UpdatedAt:     time.Now(),
	}

	// Update the statistics
	err := catalog.UpdateColumnStats(ctx, stats)
	require.NoError(t, err, "Should be able to update column stats")

	// Retrieve and verify the statistics
	retrievedStats, err := catalog.GetColumnStats(ctx, "tenant-123", "email")
	require.NoError(t, err, "Should be able to retrieve column stats")
	require.NotNil(t, retrievedStats, "Retrieved stats should not be nil")

	// Verify the statistics match
	assert.Equal(t, "tenant-123", retrievedStats.TenantID)
	assert.Equal(t, "email", retrievedStats.ColumnName)
	assert.Equal(t, int64(10000), retrievedStats.RecordCount)
	assert.Equal(t, int64(5), retrievedStats.NullCount)
}

// TEST: File metadata validation
func TestFileMetadata_Validate(t *testing.T) {
	tests := []struct {
		name     string
		metadata *FileMetadata
		wantErr  bool
		reason   string
	}{
		{
			name: "valid metadata",
			metadata: &FileMetadata{
				Path:            "/data/file.parquet",
				TenantID:        "tenant-123",
				Size:            1000,
				RecordCount:     100,
				Status:          FileStatusActive,
				CompactionLevel: 0,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
				Version:         1,
			},
			wantErr: false,
			reason:  "Valid metadata should pass validation",
		},
		{
			name: "empty path",
			metadata: &FileMetadata{
				TenantID:        "tenant-123",
				Size:            1000,
				RecordCount:     100,
				Status:          FileStatusActive,
				CompactionLevel: 0,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
				Version:         1,
			},
			wantErr: true,
			reason:  "File path is required",
		},
		{
			name: "empty tenant ID",
			metadata: &FileMetadata{
				Path:            "/data/file.parquet",
				Size:            1000,
				RecordCount:     100,
				Status:          FileStatusActive,
				CompactionLevel: 0,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
				Version:         1,
			},
			wantErr: true,
			reason:  "Tenant ID is required",
		},
		{
			name: "negative size",
			metadata: &FileMetadata{
				Path:            "/data/file.parquet",
				TenantID:        "tenant-123",
				Size:            -1000,
				RecordCount:     100,
				Status:          FileStatusActive,
				CompactionLevel: 0,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
				Version:         1,
			},
			wantErr: true,
			reason:  "File size cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metadata.Validate()
			if tt.wantErr {
				assert.Error(t, err, tt.reason)
			} else {
				assert.NoError(t, err, tt.reason)
			}
		})
	}
}

// BENCHMARK: Test performance of registering files
func BenchmarkCatalog_RegisterFile(b *testing.B) {
	catalog := createTestCatalog(b)
	defer catalog.Close()

	ctx := context.Background()

	// Base file template
	baseFile := &FileMetadata{
		TenantID:          "tenant-123",
		Size:              1000,
		RecordCount:       100,
		Status:            FileStatusActive,
		CompactionLevel:   0,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Version:           1,
		CompressionType:   "snappy",
		FormatVersion:     "1.0",
		SchemaVersion:     1,
		SchemaFingerprint: "abc123",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		file := *baseFile // Copy the template
		file.Path = fmt.Sprintf("/data/file-%d.parquet", i)
		err := catalog.RegisterFile(ctx, &file)
		require.NoError(b, err)
	}
}

// BENCHMARK: Test performance of getting files
func BenchmarkCatalog_GetFile(b *testing.B) {
	catalog := createTestCatalog(b)
	defer catalog.Close()

	ctx := context.Background()

	// Setup: Register a file first
	fileMetadata := &FileMetadata{
		Path:              "/data/benchmark-file.parquet",
		TenantID:          "tenant-123",
		Size:              1000,
		RecordCount:       100,
		Status:            FileStatusActive,
		CompactionLevel:   0,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Version:           1,
		CompressionType:   "snappy",
		FormatVersion:     "1.0",
		SchemaVersion:     1,
		SchemaFingerprint: "abc123",
	}

	err := catalog.RegisterFile(ctx, fileMetadata)
	require.NoError(b, err)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := catalog.GetFile(ctx, "/data/benchmark-file.parquet")
		require.NoError(b, err)
	}
}
