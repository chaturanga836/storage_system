package schema

import (
	"fmt"
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

// GenericSchema represents a generic schema that can be converted to various formats
type GenericSchema struct {
	Name     string            `json:"name"`
	Fields   []GenericField    `json:"fields"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// GenericField represents a field in a generic schema
type GenericField struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Nullable    bool              `json:"nullable"`
	Description string            `json:"description,omitempty"`
	Fields      []GenericField    `json:"fields,omitempty"` // For complex types
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ToGenericSchema converts internal table schema to a generic schema representation
func (st *SchemaTranslator) ToGenericSchema(tableSchema *TableSchema) (*GenericSchema, error) {
	if err := tableSchema.Validate(); err != nil {
		return nil, fmt.Errorf("invalid table schema: %w", err)
	}

	genericSchema := &GenericSchema{
		Name:     tableSchema.Name,
		Fields:   make([]GenericField, 0, len(tableSchema.Columns)),
		Metadata: make(map[string]string),
	}

	// Add metadata
	genericSchema.Metadata["source_namespace"] = tableSchema.Namespace
	genericSchema.Metadata["schema_version"] = fmt.Sprintf("%d", tableSchema.Version)
	if tableSchema.Description != "" {
		genericSchema.Metadata["description"] = tableSchema.Description
	}

	// Convert columns to generic fields
	for _, column := range tableSchema.Columns {
		field, err := st.columnToGenericField(&column)
		if err != nil {
			return nil, fmt.Errorf("failed to convert column %s: %w", column.Name, err)
		}
		genericSchema.Fields = append(genericSchema.Fields, *field)
	}

	// Add primary key information as metadata
	if len(tableSchema.PrimaryKey) > 0 {
		genericSchema.Metadata["primary_key"] = strings.Join(tableSchema.PrimaryKey, ",")
	}

	return genericSchema, nil
}

// columnToGenericField converts a column schema to a generic field
func (st *SchemaTranslator) columnToGenericField(column *ColumnSchema) (*GenericField, error) {
	field := &GenericField{
		Name:        column.Name,
		Type:        string(column.Type),
		Nullable:    column.Nullable,
		Description: column.Description,
		Metadata:    make(map[string]string),
	}

	// Handle default value
	if column.DefaultValue != nil {
		field.Metadata["default"] = fmt.Sprintf("%v", column.DefaultValue)
	}

	// Handle complex types (struct, array, map)
	if len(column.Fields) > 0 {
		field.Fields = make([]GenericField, 0, len(column.Fields))
		for _, subColumn := range column.Fields {
			subField, err := st.columnToGenericField(&subColumn)
			if err != nil {
				return nil, fmt.Errorf("failed to convert sub-field %s: %w", subColumn.Name, err)
			}
			field.Fields = append(field.Fields, *subField)
		}
	}

	return field, nil
}

// ValidateSchema validates a generic schema
func (st *SchemaTranslator) ValidateSchema(schema *GenericSchema) error {
	if schema.Name == "" {
		return fmt.Errorf("schema name cannot be empty")
	}

	if len(schema.Fields) == 0 {
		return fmt.Errorf("schema must have at least one field")
	}

	// Check for duplicate field names
	fieldNames := make(map[string]bool)
	for _, field := range schema.Fields {
		if fieldNames[field.Name] {
			return fmt.Errorf("duplicate field name: %s", field.Name)
		}
		fieldNames[field.Name] = true

		if err := st.validateField(&field); err != nil {
			return fmt.Errorf("invalid field %s: %w", field.Name, err)
		}
	}

	return nil
}

// validateField validates a single field
func (st *SchemaTranslator) validateField(field *GenericField) error {
	if field.Name == "" {
		return fmt.Errorf("field name cannot be empty")
	}

	if field.Type == "" {
		return fmt.Errorf("field type cannot be empty")
	}

	// Validate supported types
	supportedTypes := map[string]bool{
		"boolean": true, "bool": true,
		"int32": true, "integer": true,
		"int64": true, "bigint": true, "long": true,
		"float32": true, "float": true,
		"float64": true, "double": true,
		"string": true, "varchar": true, "text": true,
		"bytes": true, "binary": true,
		"date": true, "time": true, "timestamp": true, "datetime": true,
		"struct": true, "array": true, "map": true,
	}

	if !supportedTypes[strings.ToLower(field.Type)] {
		return fmt.Errorf("unsupported field type: %s", field.Type)
	}

	// Validate nested fields for complex types
	for _, subField := range field.Fields {
		if err := st.validateField(&subField); err != nil {
			return fmt.Errorf("invalid nested field %s: %w", subField.Name, err)
		}
	}

	return nil
}
