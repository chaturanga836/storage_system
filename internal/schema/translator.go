package schema

import (
	"fmt"
	"sort"
	"strings"

	"github.com/storage-system/internal/storage/parquet"
)

// SchemaTranslator translates between different schema representations
type SchemaTranslator struct {
	strictMode        bool
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

// ToParquetSchema converts internal table schema to Parquet schema
func (st *SchemaTranslator) ToParquetSchema(tableSchema *TableSchema) (*parquet.Schema, error) {
	if err := tableSchema.Validate(); err != nil {
		return nil, fmt.Errorf("invalid table schema: %w", err)
	}
	
	parquetSchema := &parquet.Schema{
		Name:    tableSchema.Name,
		Fields:  make([]parquet.Field, 0, len(tableSchema.Columns)),
		Metadata: make(map[string]string),
	}
	
	// Add metadata
	parquetSchema.Metadata["source_namespace"] = tableSchema.Namespace
	parquetSchema.Metadata["schema_version"] = fmt.Sprintf("%d", tableSchema.Version)
	if tableSchema.Description != "" {
		parquetSchema.Metadata["description"] = tableSchema.Description
	}
	
	// Convert columns to Parquet fields
	for _, column := range tableSchema.Columns {
		field, err := st.columnToParquetField(&column)
		if err != nil {
			return nil, fmt.Errorf("failed to convert column %s: %w", column.Name, err)
		}
		parquetSchema.Fields = append(parquetSchema.Fields, *field)
	}
	
	// Add primary key information as metadata
	if len(tableSchema.PrimaryKey) > 0 {
		parquetSchema.Metadata["primary_key"] = strings.Join(tableSchema.PrimaryKey, ",")
	}
	
	return parquetSchema, nil
}

// columnToParquetField converts a column schema to a Parquet field
func (st *SchemaTranslator) columnToParquetField(column *ColumnSchema) (*parquet.Field, error) {
	field := &parquet.Field{
		Name:        column.Name,
		Nullable:    column.Nullable,
		Description: column.Description,
		Metadata:    make(map[string]string),
	}
	
	// Add tags as metadata
	for k, v := range column.Tags {
		field.Metadata[k] = v
	}
	
	// Convert data type
	parquetType, logicalType, err := st.dataTypeToParquet(column)
	if err != nil {
		return nil, err
	}
	
	field.Type = parquetType
	field.LogicalType = logicalType
	
	// Handle type-specific properties
	switch column.Type {
	case TypeString, TypeBytes:
		if column.Length != nil {
			field.Metadata["max_length"] = fmt.Sprintf("%d", *column.Length)
		}
	case TypeDecimal:
		if column.Precision != nil {
			field.Metadata["precision"] = fmt.Sprintf("%d", *column.Precision)
		}
		if column.Scale != nil {
			field.Metadata["scale"] = fmt.Sprintf("%d", *column.Scale)
		}
	case TypeTimestamp:
		if column.Format != "" {
			field.Metadata["format"] = column.Format
		}
	}
	
	// Handle complex types
	switch column.Type {
	case TypeArray:
		if column.ElementType != nil {
			elementField, err := st.columnToParquetField(column.ElementType)
			if err != nil {
				return nil, fmt.Errorf("failed to convert array element type: %w", err)
			}
			field.ElementType = elementField
		}
	case TypeMap:
		if column.KeyType != nil {
			keyField, err := st.columnToParquetField(column.KeyType)
			if err != nil {
				return nil, fmt.Errorf("failed to convert map key type: %w", err)
			}
			field.KeyType = keyField
		}
		if column.ValueType != nil {
			valueField, err := st.columnToParquetField(column.ValueType)
			if err != nil {
				return nil, fmt.Errorf("failed to convert map value type: %w", err)
			}
			field.ValueType = valueField
		}
	case TypeStruct:
		if len(column.Fields) > 0 {
			field.Fields = make([]parquet.Field, 0, len(column.Fields))
			for _, subField := range column.Fields {
				subParquetField, err := st.columnToParquetField(&subField)
				if err != nil {
					return nil, fmt.Errorf("failed to convert struct field %s: %w", subField.Name, err)
				}
				field.Fields = append(field.Fields, *subParquetField)
			}
		}
	}
	
	return field, nil
}

// dataTypeToParquet converts internal data types to Parquet types
func (st *SchemaTranslator) dataTypeToParquet(column *ColumnSchema) (parquet.PhysicalType, parquet.LogicalType, error) {
	switch column.Type {
	case TypeBoolean:
		return parquet.TypeBoolean, parquet.LogicalTypeNone, nil
	case TypeInt32:
		return parquet.TypeInt32, parquet.LogicalTypeNone, nil
	case TypeInt64:
		return parquet.TypeInt64, parquet.LogicalTypeNone, nil
	case TypeFloat32:
		return parquet.TypeFloat, parquet.LogicalTypeNone, nil
	case TypeFloat64:
		return parquet.TypeDouble, parquet.LogicalTypeNone, nil
	case TypeString:
		return parquet.TypeByteArray, parquet.LogicalTypeString, nil
	case TypeBytes:
		return parquet.TypeByteArray, parquet.LogicalTypeNone, nil
	case TypeTimestamp:
		return parquet.TypeInt64, parquet.LogicalTypeTimestamp, nil
	case TypeDate:
		return parquet.TypeInt32, parquet.LogicalTypeDate, nil
	case TypeTime:
		return parquet.TypeInt64, parquet.LogicalTypeTime, nil
	case TypeUUID:
		return parquet.TypeFixedLenByteArray, parquet.LogicalTypeUUID, nil
	case TypeDecimal:
		// Use FIXED_LEN_BYTE_ARRAY for decimal
		return parquet.TypeFixedLenByteArray, parquet.LogicalTypeDecimal, nil
	case TypeArray:
		return parquet.TypeGroup, parquet.LogicalTypeList, nil
	case TypeMap:
		return parquet.TypeGroup, parquet.LogicalTypeMap, nil
	case TypeStruct:
		return parquet.TypeGroup, parquet.LogicalTypeNone, nil
	default:
		return "", "", fmt.Errorf("unsupported data type for Parquet: %s", column.Type)
	}
}

// FromParquetSchema converts Parquet schema to internal table schema
func (st *SchemaTranslator) FromParquetSchema(parquetSchema *parquet.Schema) (*TableSchema, error) {
	tableSchema := NewTableSchema(parquetSchema.Name, "")
	
	// Extract metadata
	if namespace, exists := parquetSchema.Metadata["source_namespace"]; exists {
		tableSchema.Namespace = namespace
	}
	if description, exists := parquetSchema.Metadata["description"]; exists {
		tableSchema.Description = description
	}
	
	// Convert Parquet fields to columns
	for _, field := range parquetSchema.Fields {
		column, err := st.parquetFieldToColumn(&field)
		if err != nil {
			return nil, fmt.Errorf("failed to convert field %s: %w", field.Name, err)
		}
		err = tableSchema.AddColumn(*column)
		if err != nil {
			return nil, err
		}
	}
	
	// Extract primary key information
	if pkStr, exists := parquetSchema.Metadata["primary_key"]; exists && pkStr != "" {
		primaryKey := strings.Split(pkStr, ",")
		for i, col := range primaryKey {
			primaryKey[i] = strings.TrimSpace(col)
		}
		err := tableSchema.SetPrimaryKey(primaryKey)
		if err != nil {
			return nil, fmt.Errorf("failed to set primary key: %w", err)
		}
	}
	
	return tableSchema, nil
}

// parquetFieldToColumn converts a Parquet field to a column schema
func (st *SchemaTranslator) parquetFieldToColumn(field *parquet.Field) (*ColumnSchema, error) {
	// Convert Parquet type to internal type
	dataType, err := st.parquetToDataType(field.Type, field.LogicalType)
	if err != nil {
		return nil, err
	}
	
	column, err := NewColumnSchema(field.Name, dataType, field.Nullable)
	if err != nil {
		return nil, err
	}
	
	column.Description = field.Description
	
	// Copy metadata as tags
	for k, v := range field.Metadata {
		if column.Tags == nil {
			column.Tags = make(map[string]string)
		}
		column.Tags[k] = v
	}
	
	// Handle type-specific properties from metadata
	if maxLengthStr, exists := field.Metadata["max_length"]; exists {
		if maxLength := parseInt(maxLengthStr); maxLength > 0 {
			column.Length = &maxLength
		}
	}
	
	if precisionStr, exists := field.Metadata["precision"]; exists {
		if precision := parseInt(precisionStr); precision > 0 {
			column.Precision = &precision
		}
	}
	
	if scaleStr, exists := field.Metadata["scale"]; exists {
		if scale := parseInt(scaleStr); scale >= 0 {
			column.Scale = &scale
		}
	}
	
	if format, exists := field.Metadata["format"]; exists {
		column.Format = format
	}
	
	// Handle complex types
	switch dataType {
	case TypeArray:
		if field.ElementType != nil {
			elementColumn, err := st.parquetFieldToColumn(field.ElementType)
			if err != nil {
				return nil, fmt.Errorf("failed to convert array element: %w", err)
			}
			column.ElementType = elementColumn
		}
	case TypeMap:
		if field.KeyType != nil {
			keyColumn, err := st.parquetFieldToColumn(field.KeyType)
			if err != nil {
				return nil, fmt.Errorf("failed to convert map key: %w", err)
			}
			column.KeyType = keyColumn
		}
		if field.ValueType != nil {
			valueColumn, err := st.parquetFieldToColumn(field.ValueType)
			if err != nil {
				return nil, fmt.Errorf("failed to convert map value: %w", err)
			}
			column.ValueType = valueColumn
		}
	case TypeStruct:
		if len(field.Fields) > 0 {
			column.Fields = make([]ColumnSchema, 0, len(field.Fields))
			for _, subField := range field.Fields {
				subColumn, err := st.parquetFieldToColumn(&subField)
				if err != nil {
					return nil, fmt.Errorf("failed to convert struct field %s: %w", subField.Name, err)
				}
				column.Fields = append(column.Fields, *subColumn)
			}
		}
	}
	
	return column, nil
}

// parquetToDataType converts Parquet types to internal data types
func (st *SchemaTranslator) parquetToDataType(physicalType parquet.PhysicalType, logicalType parquet.LogicalType) (DataType, error) {
	switch physicalType {
	case parquet.TypeBoolean:
		return TypeBoolean, nil
	case parquet.TypeInt32:
		switch logicalType {
		case parquet.LogicalTypeDate:
			return TypeDate, nil
		default:
			return TypeInt32, nil
		}
	case parquet.TypeInt64:
		switch logicalType {
		case parquet.LogicalTypeTimestamp:
			return TypeTimestamp, nil
		case parquet.LogicalTypeTime:
			return TypeTime, nil
		default:
			return TypeInt64, nil
		}
	case parquet.TypeFloat:
		return TypeFloat32, nil
	case parquet.TypeDouble:
		return TypeFloat64, nil
	case parquet.TypeByteArray:
		switch logicalType {
		case parquet.LogicalTypeString:
			return TypeString, nil
		default:
			return TypeBytes, nil
		}
	case parquet.TypeFixedLenByteArray:
		switch logicalType {
		case parquet.LogicalTypeUUID:
			return TypeUUID, nil
		case parquet.LogicalTypeDecimal:
			return TypeDecimal, nil
		default:
			return TypeBytes, nil
		}
	case parquet.TypeGroup:
		switch logicalType {
		case parquet.LogicalTypeList:
			return TypeArray, nil
		case parquet.LogicalTypeMap:
			return TypeMap, nil
		default:
			return TypeStruct, nil
		}
	default:
		return "", fmt.Errorf("unsupported Parquet type: %s", physicalType)
	}
}

// ToJSONSchema converts internal table schema to JSON Schema
func (st *SchemaTranslator) ToJSONSchema(tableSchema *TableSchema) (map[string]interface{}, error) {
	jsonSchema := map[string]interface{}{
		"$schema":    "https://json-schema.org/draft/2020-12/schema",
		"type":       "object",
		"title":      tableSchema.Name,
		"properties": make(map[string]interface{}),
	}
	
	if tableSchema.Description != "" {
		jsonSchema["description"] = tableSchema.Description
	}
	
	properties := jsonSchema["properties"].(map[string]interface{})
	var required []string
	
	for _, column := range tableSchema.Columns {
		propSchema, err := st.columnToJSONProperty(&column)
		if err != nil {
			return nil, fmt.Errorf("failed to convert column %s: %w", column.Name, err)
		}
		
		properties[column.Name] = propSchema
		
		if !column.Nullable && column.DefaultValue == nil {
			required = append(required, column.Name)
		}
	}
	
	if len(required) > 0 {
		sort.Strings(required)
		jsonSchema["required"] = required
	}
	
	return jsonSchema, nil
}

// columnToJSONProperty converts a column schema to JSON Schema property
func (st *SchemaTranslator) columnToJSONProperty(column *ColumnSchema) (map[string]interface{}, error) {
	property := make(map[string]interface{})
	
	if column.Description != "" {
		property["description"] = column.Description
	}
	
	// Convert data type
	switch column.Type {
	case TypeString:
		property["type"] = "string"
		if column.Length != nil {
			property["maxLength"] = *column.Length
		}
		if column.Pattern != "" {
			property["pattern"] = column.Pattern
		}
	case TypeInt32, TypeInt64:
		property["type"] = "integer"
		if column.MinValue != nil {
			property["minimum"] = column.MinValue
		}
		if column.MaxValue != nil {
			property["maximum"] = column.MaxValue
		}
	case TypeFloat32, TypeFloat64:
		property["type"] = "number"
		if column.MinValue != nil {
			property["minimum"] = column.MinValue
		}
		if column.MaxValue != nil {
			property["maximum"] = column.MaxValue
		}
	case TypeBoolean:
		property["type"] = "boolean"
	case TypeBytes:
		property["type"] = "string"
		property["contentEncoding"] = "base64"
	case TypeTimestamp, TypeDate, TypeTime:
		property["type"] = "string"
		if column.Format != "" {
			property["format"] = column.Format
		} else {
			switch column.Type {
			case TypeTimestamp:
				property["format"] = "date-time"
			case TypeDate:
				property["format"] = "date"
			case TypeTime:
				property["format"] = "time"
			}
		}
	case TypeUUID:
		property["type"] = "string"
		property["format"] = "uuid"
	case TypeArray:
		property["type"] = "array"
		if column.ElementType != nil {
			itemSchema, err := st.columnToJSONProperty(column.ElementType)
			if err != nil {
				return nil, fmt.Errorf("failed to convert array element: %w", err)
			}
			property["items"] = itemSchema
		}
	case TypeStruct:
		property["type"] = "object"
		if len(column.Fields) > 0 {
			properties := make(map[string]interface{})
			for _, field := range column.Fields {
				fieldSchema, err := st.columnToJSONProperty(&field)
				if err != nil {
					return nil, fmt.Errorf("failed to convert struct field %s: %w", field.Name, err)
				}
				properties[field.Name] = fieldSchema
			}
			property["properties"] = properties
		}
	default:
		// Fallback for unsupported types
		property["type"] = "string"
	}
	
	// Handle nullable
	if column.Nullable {
		if existingType, exists := property["type"]; exists {
			property["type"] = []interface{}{existingType, "null"}
		}
	}
	
	// Handle default value
	if column.DefaultValue != nil {
		property["default"] = column.DefaultValue
	}
	
	// Handle enum
	if len(column.Enum) > 0 {
		property["enum"] = column.Enum
	}
	
	return property, nil
}

// GenerateCreateTableSQL generates SQL CREATE TABLE statement from schema
func (st *SchemaTranslator) GenerateCreateTableSQL(tableSchema *TableSchema) (string, error) {
	var sql strings.Builder
	
	// Start CREATE TABLE
	sql.WriteString("CREATE TABLE ")
	if tableSchema.Namespace != "" {
		sql.WriteString(tableSchema.Namespace)
		sql.WriteString(".")
	}
	sql.WriteString(tableSchema.Name)
	sql.WriteString(" (\n")
	
	// Add columns
	for i, column := range tableSchema.Columns {
		if i > 0 {
			sql.WriteString(",\n")
		}
		
		sql.WriteString("  ")
		sql.WriteString(column.Name)
		sql.WriteString(" ")
		
		// Add data type
		sqlType, err := st.dataTypeToSQL(&column)
		if err != nil {
			return "", fmt.Errorf("failed to convert column %s: %w", column.Name, err)
		}
		sql.WriteString(sqlType)
		
		// Add constraints
		if !column.Nullable {
			sql.WriteString(" NOT NULL")
		}
		
		if column.DefaultValue != nil {
			sql.WriteString(" DEFAULT ")
			sql.WriteString(fmt.Sprintf("'%v'", column.DefaultValue))
		}
	}
	
	// Add primary key constraint
	if len(tableSchema.PrimaryKey) > 0 {
		sql.WriteString(",\n  PRIMARY KEY (")
		sql.WriteString(strings.Join(tableSchema.PrimaryKey, ", "))
		sql.WriteString(")")
	}
	
	// Add indexes
	for _, index := range tableSchema.Indexes {
		sql.WriteString(",\n  ")
		if index.Unique {
			sql.WriteString("UNIQUE ")
		}
		sql.WriteString("INDEX ")
		sql.WriteString(index.Name)
		sql.WriteString(" (")
		sql.WriteString(strings.Join(index.Columns, ", "))
		sql.WriteString(")")
	}
	
	sql.WriteString("\n);")
	
	return sql.String(), nil
}

// dataTypeToSQL converts internal data types to SQL types
func (st *SchemaTranslator) dataTypeToSQL(column *ColumnSchema) (string, error) {
	switch column.Type {
	case TypeString:
		if column.Length != nil {
			return fmt.Sprintf("VARCHAR(%d)", *column.Length), nil
		}
		return "TEXT", nil
	case TypeInt32:
		return "INTEGER", nil
	case TypeInt64:
		return "BIGINT", nil
	case TypeFloat32:
		return "FLOAT", nil
	case TypeFloat64:
		return "DOUBLE", nil
	case TypeBoolean:
		return "BOOLEAN", nil
	case TypeBytes:
		return "BYTEA", nil
	case TypeTimestamp:
		return "TIMESTAMP", nil
	case TypeDate:
		return "DATE", nil
	case TypeTime:
		return "TIME", nil
	case TypeUUID:
		return "UUID", nil
	case TypeDecimal:
		if column.Precision != nil && column.Scale != nil {
			return fmt.Sprintf("DECIMAL(%d,%d)", *column.Precision, *column.Scale), nil
		} else if column.Precision != nil {
			return fmt.Sprintf("DECIMAL(%d)", *column.Precision), nil
		}
		return "DECIMAL", nil
	default:
		return "", fmt.Errorf("unsupported data type for SQL: %s", column.Type)
	}
}

// Helper function to parse integer from string
func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}
