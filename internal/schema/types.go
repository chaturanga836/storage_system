package schema

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// DataType represents the type of a column
type DataType string

const (
	// Basic types
	TypeString    DataType = "string"
	TypeInt32     DataType = "int32"
	TypeInt64     DataType = "int64"
	TypeFloat32   DataType = "float32"
	TypeFloat64   DataType = "float64"
	TypeBoolean   DataType = "boolean"
	TypeBytes     DataType = "bytes"
	TypeTimestamp DataType = "timestamp"
	TypeUUID      DataType = "uuid"
	
	// Complex types
	TypeArray     DataType = "array"
	TypeMap       DataType = "map"
	TypeStruct    DataType = "struct"
	TypeUnion     DataType = "union"
	
	// Decimal type
	TypeDecimal   DataType = "decimal"
	
	// Date/Time types
	TypeDate      DataType = "date"
	TypeTime      DataType = "time"
	TypeInterval  DataType = "interval"
)

// ColumnSchema defines the schema for a single column
type ColumnSchema struct {
	Name         string                 `json:"name"`
	Type         DataType               `json:"type"`
	Nullable     bool                   `json:"nullable"`
	DefaultValue interface{}            `json:"default_value,omitempty"`
	Description  string                 `json:"description,omitempty"`
	
	// Type-specific properties
	Length       *int                   `json:"length,omitempty"`        // For strings, bytes
	Precision    *int                   `json:"precision,omitempty"`     // For decimals
	Scale        *int                   `json:"scale,omitempty"`         // For decimals
	Format       string                 `json:"format,omitempty"`        // For timestamps, dates
	
	// Complex type definitions
	ElementType  *ColumnSchema          `json:"element_type,omitempty"`  // For arrays
	KeyType      *ColumnSchema          `json:"key_type,omitempty"`      // For maps
	ValueType    *ColumnSchema          `json:"value_type,omitempty"`    // For maps
	Fields       []ColumnSchema         `json:"fields,omitempty"`        // For structs
	UnionTypes   []ColumnSchema         `json:"union_types,omitempty"`   // For unions
	
	// Constraints
	MinValue     interface{}            `json:"min_value,omitempty"`
	MaxValue     interface{}            `json:"max_value,omitempty"`
	Pattern      string                 `json:"pattern,omitempty"`       // Regex pattern for strings
	Enum         []interface{}          `json:"enum,omitempty"`          // Allowed values
	
	// Metadata
	Tags         map[string]string      `json:"tags,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// TableSchema defines the schema for a table
type TableSchema struct {
	Name         string                 `json:"name"`
	Namespace    string                 `json:"namespace,omitempty"`
	Columns      []ColumnSchema         `json:"columns"`
	PrimaryKey   []string               `json:"primary_key,omitempty"`
	Indexes      []IndexSchema          `json:"indexes,omitempty"`
	Partitioning *PartitioningSchema    `json:"partitioning,omitempty"`
	
	// Metadata
	Description  string                 `json:"description,omitempty"`
	Tags         map[string]string      `json:"tags,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Version      int                    `json:"version"`
}

// IndexSchema defines the schema for an index
type IndexSchema struct {
	Name         string                 `json:"name"`
	Columns      []string               `json:"columns"`
	Type         string                 `json:"type"`         // btree, hash, bitmap, etc.
	Unique       bool                   `json:"unique"`
	Partial      string                 `json:"partial,omitempty"` // Partial index condition
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// PartitioningSchema defines how a table is partitioned
type PartitioningSchema struct {
	Type         string                 `json:"type"`         // range, hash, list
	Columns      []string               `json:"columns"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// SchemaEvolution represents a change to a schema
type SchemaEvolution struct {
	Type         string                 `json:"type"`         // add_column, drop_column, modify_column, etc.
	TableName    string                 `json:"table_name"`
	ColumnName   string                 `json:"column_name,omitempty"`
	OldSchema    *ColumnSchema          `json:"old_schema,omitempty"`
	NewSchema    *ColumnSchema          `json:"new_schema,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Applied      bool                   `json:"applied"`
}

// NewColumnSchema creates a new column schema with basic validation
func NewColumnSchema(name string, dataType DataType, nullable bool) (*ColumnSchema, error) {
	if name == "" {
		return nil, fmt.Errorf("column name cannot be empty")
	}
	
	if !IsValidDataType(dataType) {
		return nil, fmt.Errorf("invalid data type: %s", dataType)
	}
	
	return &ColumnSchema{
		Name:      name,
		Type:      dataType,
		Nullable:  nullable,
		Tags:      make(map[string]string),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// NewTableSchema creates a new table schema
func NewTableSchema(name string, namespace string) *TableSchema {
	return &TableSchema{
		Name:      name,
		Namespace: namespace,
		Columns:   make([]ColumnSchema, 0),
		Indexes:   make([]IndexSchema, 0),
		Tags:      make(map[string]string),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
	}
}

// AddColumn adds a column to the table schema
func (ts *TableSchema) AddColumn(column ColumnSchema) error {
	// Check for duplicate column names
	for _, existing := range ts.Columns {
		if existing.Name == column.Name {
			return fmt.Errorf("column %s already exists", column.Name)
		}
	}
	
	column.CreatedAt = time.Now()
	column.UpdatedAt = time.Now()
	ts.Columns = append(ts.Columns, column)
	ts.UpdatedAt = time.Now()
	ts.Version++
	
	return nil
}

// RemoveColumn removes a column from the table schema
func (ts *TableSchema) RemoveColumn(columnName string) error {
	for i, column := range ts.Columns {
		if column.Name == columnName {
			// Check if column is part of primary key
			for _, pkCol := range ts.PrimaryKey {
				if pkCol == columnName {
					return fmt.Errorf("cannot remove column %s: it is part of the primary key", columnName)
				}
			}
			
			// Remove the column
			ts.Columns = append(ts.Columns[:i], ts.Columns[i+1:]...)
			ts.UpdatedAt = time.Now()
			ts.Version++
			return nil
		}
	}
	
	return fmt.Errorf("column %s not found", columnName)
}

// GetColumn returns a column by name
func (ts *TableSchema) GetColumn(columnName string) (*ColumnSchema, error) {
	for i, column := range ts.Columns {
		if column.Name == columnName {
			return &ts.Columns[i], nil
		}
	}
	
	return nil, fmt.Errorf("column %s not found", columnName)
}

// UpdateColumn updates an existing column's schema
func (ts *TableSchema) UpdateColumn(columnName string, newSchema ColumnSchema) error {
	for i, column := range ts.Columns {
		if column.Name == columnName {
			// Preserve creation time
			newSchema.CreatedAt = column.CreatedAt
			newSchema.UpdatedAt = time.Now()
			ts.Columns[i] = newSchema
			ts.UpdatedAt = time.Now()
			ts.Version++
			return nil
		}
	}
	
	return fmt.Errorf("column %s not found", columnName)
}

// SetPrimaryKey sets the primary key columns
func (ts *TableSchema) SetPrimaryKey(columns []string) error {
	// Validate that all columns exist
	for _, pkCol := range columns {
		found := false
		for _, column := range ts.Columns {
			if column.Name == pkCol {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("primary key column %s does not exist", pkCol)
		}
	}
	
	ts.PrimaryKey = make([]string, len(columns))
	copy(ts.PrimaryKey, columns)
	ts.UpdatedAt = time.Now()
	ts.Version++
	
	return nil
}

// AddIndex adds an index to the table schema
func (ts *TableSchema) AddIndex(index IndexSchema) error {
	// Check for duplicate index names
	for _, existing := range ts.Indexes {
		if existing.Name == index.Name {
			return fmt.Errorf("index %s already exists", index.Name)
		}
	}
	
	// Validate that all indexed columns exist
	for _, idxCol := range index.Columns {
		found := false
		for _, column := range ts.Columns {
			if column.Name == idxCol {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("index column %s does not exist", idxCol)
		}
	}
	
	ts.Indexes = append(ts.Indexes, index)
	ts.UpdatedAt = time.Now()
	ts.Version++
	
	return nil
}

// RemoveIndex removes an index from the table schema
func (ts *TableSchema) RemoveIndex(indexName string) error {
	for i, index := range ts.Indexes {
		if index.Name == indexName {
			ts.Indexes = append(ts.Indexes[:i], ts.Indexes[i+1:]...)
			ts.UpdatedAt = time.Now()
			ts.Version++
			return nil
		}
	}
	
	return fmt.Errorf("index %s not found", indexName)
}

// Validate performs comprehensive validation of the table schema
func (ts *TableSchema) Validate() error {
	if ts.Name == "" {
		return fmt.Errorf("table name cannot be empty")
	}
	
	if len(ts.Columns) == 0 {
		return fmt.Errorf("table must have at least one column")
	}
	
	// Validate column names are unique
	columnNames := make(map[string]bool)
	for _, column := range ts.Columns {
		if columnNames[column.Name] {
			return fmt.Errorf("duplicate column name: %s", column.Name)
		}
		columnNames[column.Name] = true
		
		// Validate individual column
		if err := column.Validate(); err != nil {
			return fmt.Errorf("invalid column %s: %w", column.Name, err)
		}
	}
	
	// Validate primary key columns exist
	for _, pkCol := range ts.PrimaryKey {
		if !columnNames[pkCol] {
			return fmt.Errorf("primary key column %s does not exist", pkCol)
		}
	}
	
	// Validate index names are unique and columns exist
	indexNames := make(map[string]bool)
	for _, index := range ts.Indexes {
		if indexNames[index.Name] {
			return fmt.Errorf("duplicate index name: %s", index.Name)
		}
		indexNames[index.Name] = true
		
		for _, idxCol := range index.Columns {
			if !columnNames[idxCol] {
				return fmt.Errorf("index %s references non-existent column: %s", index.Name, idxCol)
			}
		}
	}
	
	return nil
}

// Validate performs validation of a column schema
func (cs *ColumnSchema) Validate() error {
	if cs.Name == "" {
		return fmt.Errorf("column name cannot be empty")
	}
	
	if !IsValidDataType(cs.Type) {
		return fmt.Errorf("invalid data type: %s", cs.Type)
	}
	
	// Type-specific validations
	switch cs.Type {
	case TypeString, TypeBytes:
		if cs.Length != nil && *cs.Length < 0 {
			return fmt.Errorf("length cannot be negative")
		}
	case TypeDecimal:
		if cs.Precision != nil && *cs.Precision <= 0 {
			return fmt.Errorf("precision must be positive")
		}
		if cs.Scale != nil && *cs.Scale < 0 {
			return fmt.Errorf("scale cannot be negative")
		}
		if cs.Precision != nil && cs.Scale != nil && *cs.Scale > *cs.Precision {
			return fmt.Errorf("scale cannot be greater than precision")
		}
	case TypeArray:
		if cs.ElementType == nil {
			return fmt.Errorf("array type must specify element type")
		}
		if err := cs.ElementType.Validate(); err != nil {
			return fmt.Errorf("invalid array element type: %w", err)
		}
	case TypeMap:
		if cs.KeyType == nil || cs.ValueType == nil {
			return fmt.Errorf("map type must specify both key and value types")
		}
		if err := cs.KeyType.Validate(); err != nil {
			return fmt.Errorf("invalid map key type: %w", err)
		}
		if err := cs.ValueType.Validate(); err != nil {
			return fmt.Errorf("invalid map value type: %w", err)
		}
	case TypeStruct:
		if len(cs.Fields) == 0 {
			return fmt.Errorf("struct type must have at least one field")
		}
		fieldNames := make(map[string]bool)
		for _, field := range cs.Fields {
			if fieldNames[field.Name] {
				return fmt.Errorf("duplicate field name in struct: %s", field.Name)
			}
			fieldNames[field.Name] = true
			if err := field.Validate(); err != nil {
				return fmt.Errorf("invalid struct field %s: %w", field.Name, err)
			}
		}
	case TypeUnion:
		if len(cs.UnionTypes) < 2 {
			return fmt.Errorf("union type must have at least two types")
		}
		for i, unionType := range cs.UnionTypes {
			if err := unionType.Validate(); err != nil {
				return fmt.Errorf("invalid union type %d: %w", i, err)
			}
		}
	}
	
	// Validate constraints
	if cs.MinValue != nil && cs.MaxValue != nil {
		// This is a simplified check - in reality you'd need type-aware comparison
		if fmt.Sprintf("%v", cs.MinValue) > fmt.Sprintf("%v", cs.MaxValue) {
			return fmt.Errorf("min_value cannot be greater than max_value")
		}
	}
	
	return nil
}

// IsValidDataType checks if a data type is valid
func IsValidDataType(dataType DataType) bool {
	validTypes := map[DataType]bool{
		TypeString:    true,
		TypeInt32:     true,
		TypeInt64:     true,
		TypeFloat32:   true,
		TypeFloat64:   true,
		TypeBoolean:   true,
		TypeBytes:     true,
		TypeTimestamp: true,
		TypeUUID:      true,
		TypeArray:     true,
		TypeMap:       true,
		TypeStruct:    true,
		TypeUnion:     true,
		TypeDecimal:   true,
		TypeDate:      true,
		TypeTime:      true,
		TypeInterval:  true,
	}
	
	return validTypes[dataType]
}

// GetTypeSizeBytes returns the size in bytes for fixed-size types
func (cs *ColumnSchema) GetTypeSizeBytes() int {
	switch cs.Type {
	case TypeBoolean:
		return 1
	case TypeInt32, TypeFloat32:
		return 4
	case TypeInt64, TypeFloat64, TypeTimestamp:
		return 8
	case TypeUUID:
		return 16
	case TypeString, TypeBytes:
		if cs.Length != nil {
			return *cs.Length
		}
		return -1 // Variable length
	default:
		return -1 // Variable or complex type
	}
}

// IsCompatible checks if two column schemas are compatible for data migration
func (cs *ColumnSchema) IsCompatible(other *ColumnSchema) bool {
	// Same type is always compatible
	if cs.Type == other.Type {
		return true
	}
	
	// Define compatible type conversions
	compatibleConversions := map[DataType][]DataType{
		TypeInt32:   {TypeInt64, TypeFloat32, TypeFloat64, TypeString},
		TypeInt64:   {TypeFloat64, TypeString},
		TypeFloat32: {TypeFloat64, TypeString},
		TypeFloat64: {TypeString},
		TypeString:  {TypeBytes},
		TypeBytes:   {TypeString},
	}
	
	compatible, exists := compatibleConversions[cs.Type]
	if !exists {
		return false
	}
	
	for _, compatibleType := range compatible {
		if other.Type == compatibleType {
			return true
		}
	}
	
	return false
}

// ToJSON serializes the table schema to JSON
func (ts *TableSchema) ToJSON() ([]byte, error) {
	return json.MarshalIndent(ts, "", "  ")
}

// FromJSON deserializes a table schema from JSON
func (ts *TableSchema) FromJSON(data []byte) error {
	return json.Unmarshal(data, ts)
}

// Clone creates a deep copy of the table schema
func (ts *TableSchema) Clone() *TableSchema {
	clone := &TableSchema{
		Name:        ts.Name,
		Namespace:   ts.Namespace,
		Description: ts.Description,
		CreatedAt:   ts.CreatedAt,
		UpdatedAt:   ts.UpdatedAt,
		Version:     ts.Version,
	}
	
	// Clone columns
	clone.Columns = make([]ColumnSchema, len(ts.Columns))
	for i, col := range ts.Columns {
		clone.Columns[i] = col.Clone()
	}
	
	// Clone primary key
	if ts.PrimaryKey != nil {
		clone.PrimaryKey = make([]string, len(ts.PrimaryKey))
		copy(clone.PrimaryKey, ts.PrimaryKey)
	}
	
	// Clone indexes
	clone.Indexes = make([]IndexSchema, len(ts.Indexes))
	copy(clone.Indexes, ts.Indexes)
	
	// Clone tags
	if ts.Tags != nil {
		clone.Tags = make(map[string]string)
		for k, v := range ts.Tags {
			clone.Tags[k] = v
		}
	}
	
	// Clone partitioning
	if ts.Partitioning != nil {
		clone.Partitioning = &PartitioningSchema{
			Type:    ts.Partitioning.Type,
			Columns: make([]string, len(ts.Partitioning.Columns)),
		}
		copy(clone.Partitioning.Columns, ts.Partitioning.Columns)
		
		if ts.Partitioning.Properties != nil {
			clone.Partitioning.Properties = make(map[string]interface{})
			for k, v := range ts.Partitioning.Properties {
				clone.Partitioning.Properties[k] = v
			}
		}
	}
	
	return clone
}

// Clone creates a deep copy of the column schema
func (cs *ColumnSchema) Clone() ColumnSchema {
	clone := ColumnSchema{
		Name:         cs.Name,
		Type:         cs.Type,
		Nullable:     cs.Nullable,
		DefaultValue: cs.DefaultValue,
		Description:  cs.Description,
		Format:       cs.Format,
		MinValue:     cs.MinValue,
		MaxValue:     cs.MaxValue,
		Pattern:      cs.Pattern,
		CreatedAt:    cs.CreatedAt,
		UpdatedAt:    cs.UpdatedAt,
	}
	
	// Clone pointers
	if cs.Length != nil {
		length := *cs.Length
		clone.Length = &length
	}
	if cs.Precision != nil {
		precision := *cs.Precision
		clone.Precision = &precision
	}
	if cs.Scale != nil {
		scale := *cs.Scale
		clone.Scale = &scale
	}
	
	// Clone complex types
	if cs.ElementType != nil {
		elementType := cs.ElementType.Clone()
		clone.ElementType = &elementType
	}
	if cs.KeyType != nil {
		keyType := cs.KeyType.Clone()
		clone.KeyType = &keyType
	}
	if cs.ValueType != nil {
		valueType := cs.ValueType.Clone()
		clone.ValueType = &valueType
	}
	
	// Clone fields
	if cs.Fields != nil {
		clone.Fields = make([]ColumnSchema, len(cs.Fields))
		for i, field := range cs.Fields {
			clone.Fields[i] = field.Clone()
		}
	}
	
	// Clone union types
	if cs.UnionTypes != nil {
		clone.UnionTypes = make([]ColumnSchema, len(cs.UnionTypes))
		for i, unionType := range cs.UnionTypes {
			clone.UnionTypes[i] = unionType.Clone()
		}
	}
	
	// Clone enum
	if cs.Enum != nil {
		clone.Enum = make([]interface{}, len(cs.Enum))
		copy(clone.Enum, cs.Enum)
	}
	
	// Clone tags
	if cs.Tags != nil {
		clone.Tags = make(map[string]string)
		for k, v := range cs.Tags {
			clone.Tags[k] = v
		}
	}
	
	return clone
}

// GetFullyQualifiedName returns the fully qualified name of the table
func (ts *TableSchema) GetFullyQualifiedName() string {
	if ts.Namespace != "" {
		return fmt.Sprintf("%s.%s", ts.Namespace, ts.Name)
	}
	return ts.Name
}

// GetColumnNames returns a slice of all column names
func (ts *TableSchema) GetColumnNames() []string {
	names := make([]string, len(ts.Columns))
	for i, col := range ts.Columns {
		names[i] = col.Name
	}
	return names
}

// HasColumn checks if a column exists in the schema
func (ts *TableSchema) HasColumn(columnName string) bool {
	for _, col := range ts.Columns {
		if col.Name == columnName {
			return true
		}
	}
	return false
}

// GetIndexByName returns an index by name
func (ts *TableSchema) GetIndexByName(indexName string) (*IndexSchema, error) {
	for i, index := range ts.Indexes {
		if index.Name == indexName {
			return &ts.Indexes[i], nil
		}
	}
	return nil, fmt.Errorf("index %s not found", indexName)
}

// GetColumnByPath returns a column by path (for nested structures)
func (cs *ColumnSchema) GetColumnByPath(path string) (*ColumnSchema, error) {
	parts := strings.Split(path, ".")
	current := cs
	
	for i, part := range parts {
		if i == 0 && part == current.Name {
			continue // Skip the root column name
		}
		
		if current.Type == TypeStruct {
			found := false
			for j, field := range current.Fields {
				if field.Name == part {
					current = &current.Fields[j]
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("field %s not found in struct", part)
			}
		} else {
			return nil, fmt.Errorf("cannot navigate path %s: %s is not a struct", path, current.Name)
		}
	}
	
	return current, nil
}
