package parquet

import (
	"fmt"
	"sort"
	"strings"
)

// SchemaTranslator translates between different schema representations
type SchemaTranslator struct {
	strictMode          bool
	preserveNullability bool
	defaultStringLength int
}

// NewSchemaTranslator creates a new schema translator
func NewSchemaTranslator() *SchemaTranslator {
	return &SchemaTranslator{
		strictMode:          false,
		preserveNullability: true,
		defaultStringLength: 255,
	}
}

// ParquetSchema represents a Parquet schema (placeholder)
type ParquetSchema struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace,omitempty"`
	Fields    []ParquetField    `json:"fields"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ParquetField represents a field in a Parquet schema
type ParquetField struct {
	Name        string              `json:"name"`
	Type        ParquetPhysicalType `json:"type"`
	LogicalType ParquetLogicalType  `json:"logical_type,omitempty"`
	Repetition  RepetitionType      `json:"repetition"`
	Fields      []ParquetField      `json:"fields,omitempty"`
	Metadata    map[string]string   `json:"metadata,omitempty"`
}

// ParquetPhysicalType represents Parquet physical types
type ParquetPhysicalType string

const (
	TypeBoolean   ParquetPhysicalType = "BOOLEAN"
	TypeInt32     ParquetPhysicalType = "INT32"
	TypeInt64     ParquetPhysicalType = "INT64"
	TypeFloat     ParquetPhysicalType = "FLOAT"
	TypeDouble    ParquetPhysicalType = "DOUBLE"
	TypeByteArray ParquetPhysicalType = "BYTE_ARRAY"
)

// ParquetLogicalType represents Parquet logical types
type ParquetLogicalType string

const (
	LogicalTypeNone      ParquetLogicalType = ""
	LogicalTypeString    ParquetLogicalType = "STRING"
	LogicalTypeEnum      ParquetLogicalType = "ENUM"
	LogicalTypeUTF8      ParquetLogicalType = "UTF8"
	LogicalTypeDate      ParquetLogicalType = "DATE"
	LogicalTypeTime      ParquetLogicalType = "TIME"
	LogicalTypeTimestamp ParquetLogicalType = "TIMESTAMP"
)

// RepetitionType represents Parquet repetition types
type RepetitionType string

const (
	RepetitionRequired RepetitionType = "REQUIRED"
	RepetitionOptional RepetitionType = "OPTIONAL"
	RepetitionRepeated RepetitionType = "REPEATED"
)

// Simple schema types to avoid circular dependency
type TableSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Columns     []ColumnSchema `json:"columns"`
}

type ColumnSchema struct {
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	Nullable     bool           `json:"nullable"`
	Description  string         `json:"description,omitempty"`
	DefaultValue interface{}    `json:"default_value,omitempty"`
	Fields       []ColumnSchema `json:"fields,omitempty"` // For complex types
}

// Validate validates the table schema
func (ts *TableSchema) Validate() error {
	if ts.Name == "" {
		return fmt.Errorf("table name cannot be empty")
	}
	if len(ts.Columns) == 0 {
		return fmt.Errorf("table must have at least one column")
	}
	return nil
}

// ToParquetSchema converts internal table schema to Parquet schema
func (st *SchemaTranslator) ToParquetSchema(tableSchema *TableSchema) (*ParquetSchema, error) {
	if err := tableSchema.Validate(); err != nil {
		return nil, fmt.Errorf("invalid table schema: %w", err)
	}

	parquetSchema := &ParquetSchema{
		Name:     tableSchema.Name,
		Fields:   make([]ParquetField, 0, len(tableSchema.Columns)),
		Metadata: make(map[string]string),
	}

	// Convert columns to Parquet fields
	for _, column := range tableSchema.Columns {
		field, err := st.columnToParquetField(&column)
		if err != nil {
			return nil, fmt.Errorf("failed to convert column %s: %w", column.Name, err)
		}
		parquetSchema.Fields = append(parquetSchema.Fields, *field)
	}

	// Sort fields by name for consistency
	sort.Slice(parquetSchema.Fields, func(i, j int) bool {
		return parquetSchema.Fields[i].Name < parquetSchema.Fields[j].Name
	})

	return parquetSchema, nil
}

// columnToParquetField converts a column schema to a Parquet field
func (st *SchemaTranslator) columnToParquetField(column *ColumnSchema) (*ParquetField, error) {
	field := &ParquetField{
		Name:     column.Name,
		Metadata: make(map[string]string),
	}

	// Set repetition based on nullability
	if column.Nullable {
		field.Repetition = RepetitionOptional
	} else {
		field.Repetition = RepetitionRequired
	}

	// Convert data type
	physicalType, logicalType, err := st.dataTypeToParquet(column)
	if err != nil {
		return nil, err
	}

	field.Type = physicalType
	field.LogicalType = logicalType

	// Add description as metadata
	if column.Description != "" {
		field.Metadata["description"] = column.Description
	}

	// Handle default value
	if column.DefaultValue != nil {
		field.Metadata["default"] = fmt.Sprintf("%v", column.DefaultValue)
	}

	// Handle complex types (struct, array, map)
	if len(column.Fields) > 0 {
		field.Fields = make([]ParquetField, 0, len(column.Fields))
		for _, subColumn := range column.Fields {
			subField, err := st.columnToParquetField(&subColumn)
			if err != nil {
				return nil, fmt.Errorf("failed to convert sub-field %s: %w", subColumn.Name, err)
			}
			field.Fields = append(field.Fields, *subField)
		}
	}

	return field, nil
}

// dataTypeToParquet maps internal data types to Parquet types
func (st *SchemaTranslator) dataTypeToParquet(column *ColumnSchema) (ParquetPhysicalType, ParquetLogicalType, error) {
	switch strings.ToLower(column.Type) {
	case "boolean", "bool":
		return TypeBoolean, LogicalTypeNone, nil
	case "int32", "integer":
		return TypeInt32, LogicalTypeNone, nil
	case "int64", "bigint", "long":
		return TypeInt64, LogicalTypeNone, nil
	case "float32", "float":
		return TypeFloat, LogicalTypeNone, nil
	case "float64", "double":
		return TypeDouble, LogicalTypeNone, nil
	case "string", "varchar", "text":
		return TypeByteArray, LogicalTypeString, nil
	case "bytes", "binary":
		return TypeByteArray, LogicalTypeNone, nil
	case "date":
		return TypeInt32, LogicalTypeDate, nil
	case "time":
		return TypeInt64, LogicalTypeTime, nil
	case "timestamp", "datetime":
		return TypeInt64, LogicalTypeTimestamp, nil
	default:
		return "", "", fmt.Errorf("unsupported data type: %s", column.Type)
	}
}
