package schema

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLParser_ParseCreateTable(t *testing.T) {
	parser := NewSQLParser()

	createTableSQL := `CREATE TABLE users (id STRING NOT NULL, email STRING NOT NULL, age INT32, created_at TIMESTAMP)`

	schema, err := parser.ParseCreateTable(createTableSQL)
	require.NoError(t, err)
	require.NotNil(t, schema)

	assert.Equal(t, "users", schema.Name)
	assert.Len(t, schema.Columns, 4)

	// Check first column
	assert.Equal(t, "id", schema.Columns[0].Name)
	assert.Equal(t, TypeString, schema.Columns[0].Type)
	assert.False(t, schema.Columns[0].Nullable)

	// Check nullable column
	assert.Equal(t, "age", schema.Columns[2].Name)
	assert.Equal(t, TypeInt32, schema.Columns[2].Type)
	assert.True(t, schema.Columns[2].Nullable)
}

func TestSQLParser_ParseCreateTable_InvalidSQL(t *testing.T) {
	parser := NewSQLParser()

	invalidSQL := `INVALID SQL STATEMENT`

	schema, err := parser.ParseCreateTable(invalidSQL)
	assert.Error(t, err)
	assert.Nil(t, schema)
	assert.Contains(t, err.Error(), "invalid")
}

func TestSQLParser_ParseCreateTable_WithNamespace(t *testing.T) {
	parser := NewSQLParser()

	createTableSQL := `CREATE TABLE company.employees (employee_id STRING NOT NULL, name STRING NOT NULL, department STRING, salary FLOAT64)`

	schema, err := parser.ParseCreateTable(createTableSQL)
	require.NoError(t, err)
	require.NotNil(t, schema)

	assert.Equal(t, "employees", schema.Name)
	assert.Equal(t, "company", schema.Namespace)
	assert.Len(t, schema.Columns, 4)
}

func TestColumnSchema_Validate(t *testing.T) {
	tests := []struct {
		name    string
		column  ColumnSchema
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid column",
			column: ColumnSchema{
				Name:      "user_id",
				Type:      TypeString,
				Nullable:  false,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "empty name",
			column: ColumnSchema{
				Type:      TypeString,
				Nullable:  false,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "invalid type",
			column: ColumnSchema{
				Name:      "user_id",
				Type:      DataType("invalid_type"),
				Nullable:  false,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.column.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTableSchema_Validate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		schema  TableSchema
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid table schema",
			schema: TableSchema{
				Name: "users",
				Columns: []ColumnSchema{
					{
						Name:      "id",
						Type:      TypeString,
						Nullable:  false,
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						Name:      "email",
						Type:      TypeString,
						Nullable:  false,
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
				CreatedAt: now,
				UpdatedAt: now,
				Version:   1,
			},
			wantErr: false,
		},
		{
			name: "empty table name",
			schema: TableSchema{
				Columns: []ColumnSchema{
					{
						Name:      "id",
						Type:      TypeString,
						Nullable:  false,
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
				CreatedAt: now,
				UpdatedAt: now,
				Version:   1,
			},
			wantErr: true,
			errMsg:  "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDataType_IsValid(t *testing.T) {
	validTypes := []DataType{
		TypeString,
		TypeInt32,
		TypeInt64,
		TypeFloat32,
		TypeFloat64,
		TypeBoolean,
		TypeBytes,
		TypeTimestamp,
		TypeUUID,
	}

	for _, dataType := range validTypes {
		t.Run(string(dataType), func(t *testing.T) {
			assert.True(t, IsValidDataType(dataType), "Type %s should be valid", dataType)
		})
	}

	// Test invalid type
	invalidType := DataType("invalid_type")
	assert.False(t, IsValidDataType(invalidType), "Invalid type should return false")
}

func TestTableSchema_GetColumn(t *testing.T) {
	now := time.Now()
	schema := TableSchema{
		Name: "test_table",
		Columns: []ColumnSchema{
			{
				Name:      "id",
				Type:      TypeString,
				Nullable:  false,
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				Name:      "email",
				Type:      TypeString,
				Nullable:  true,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}

	// Test existing column
	column, err := schema.GetColumn("email")
	assert.NoError(t, err)
	assert.Equal(t, "email", column.Name)
	assert.Equal(t, TypeString, column.Type)
	assert.True(t, column.Nullable)

	// Test non-existing column
	_, err = schema.GetColumn("non_existent")
	assert.Error(t, err)
}

// Benchmark tests
func BenchmarkSQLParser_ParseCreateTable(b *testing.B) {
	parser := NewSQLParser()
	createTableSQL := `CREATE TABLE benchmark_table (id STRING NOT NULL, name STRING, age INT32, email STRING, created_at TIMESTAMP)`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := parser.ParseCreateTable(createTableSQL)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTableSchema_Validate(b *testing.B) {
	now := time.Now()
	schema := TableSchema{
		Name: "benchmark_table",
		Columns: []ColumnSchema{
			{Name: "id", Type: TypeString, Nullable: false, CreatedAt: now, UpdatedAt: now},
			{Name: "name", Type: TypeString, Nullable: true, CreatedAt: now, UpdatedAt: now},
			{Name: "age", Type: TypeInt32, Nullable: true, CreatedAt: now, UpdatedAt: now},
			{Name: "email", Type: TypeString, Nullable: true, CreatedAt: now, UpdatedAt: now},
			{Name: "created_at", Type: TypeTimestamp, Nullable: false, CreatedAt: now, UpdatedAt: now},
		},
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := schema.Validate()
		if err != nil {
			b.Fatal(err)
		}
	}
}
