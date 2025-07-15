# Schema Module

## Overview

The Schema module provides schema management, validation, and evolution capabilities for the storage system. It handles data type definitions, schema parsing, transformation, and compatibility checking across different schema versions.

## Architecture

### Core Components

- **Schema Parser**: Parses schema definitions from various formats
- **Type System**: Defines data types and their operations
- **Schema Registry**: Manages schema versions and compatibility
- **Schema Translator**: Converts between different schema formats
- **Validation Engine**: Validates data against schema definitions

### Type System

```go
type DataType int

const (
    TypeNull DataType = iota
    TypeBoolean
    TypeInt32
    TypeInt64
    TypeFloat32
    TypeFloat64
    TypeString
    TypeBytes
    TypeTimestamp
    TypeArray
    TypeMap
    TypeStruct
)
```

## Features

### Schema Definition

- **Multiple Formats**: Support for JSON, Avro, Protocol Buffers
- **Nested Types**: Complex types including arrays, maps, and structs
- **Constraints**: Field-level constraints and validations
- **Metadata**: Attach metadata to schemas and fields

### Schema Evolution

- **Backward Compatibility**: Ensure existing data remains readable
- **Forward Compatibility**: Handle newer schemas with older readers
- **Schema Migration**: Automated data transformation during evolution
- **Version Management**: Track and manage schema versions

### Data Validation

- **Type Checking**: Validate data types against schema
- **Constraint Validation**: Check field constraints and rules
- **Null Handling**: Proper null value validation
- **Custom Validators**: Extensible validation framework

### Format Translation

- **JSON Schema**: Convert to/from JSON Schema format
- **Avro Schema**: Support for Apache Avro schemas
- **Parquet Schema**: Generate Parquet-compatible schemas
- **Protocol Buffers**: Support for protobuf schemas

## API Reference

### Schema Parser Interface

```go
type Parser interface {
    // Parse schema from different formats
    ParseJSON(data []byte) (*Schema, error)
    ParseAvro(data []byte) (*Schema, error)
    ParseProto(data []byte) (*Schema, error)
    
    // Generate schema from data
    InferSchema(data []map[string]interface{}) (*Schema, error)
    
    // Schema conversion
    ToJSON(*Schema) ([]byte, error)
    ToAvro(*Schema) ([]byte, error)
    ToParquet(*Schema) (*parquet.Schema, error)
}
```

### Schema Structure

```go
type Schema struct {
    Name        string
    Namespace   string
    Version     int
    Fields      []Field
    Metadata    map[string]string
    CreatedAt   time.Time
    ModifiedAt  time.Time
}

type Field struct {
    Name         string
    Type         DataType
    Nullable     bool
    DefaultValue interface{}
    Constraints  []Constraint
    Metadata     map[string]string
}

type Constraint struct {
    Type        ConstraintType
    Value       interface{}
    Message     string
}

type ConstraintType int

const (
    ConstraintRequired ConstraintType = iota
    ConstraintMinLength
    ConstraintMaxLength
    ConstraintPattern
    ConstraintMin
    ConstraintMax
    ConstraintEnum
)
```

### Schema Registry Interface

```go
type Registry interface {
    // Register and retrieve schemas
    Register(schema *Schema) (int, error)
    GetByID(id int) (*Schema, error)
    GetLatest(subject string) (*Schema, error)
    GetVersions(subject string) ([]int, error)
    
    // Compatibility checking
    CheckCompatibility(schema *Schema, version int) (bool, error)
    SetCompatibilityLevel(subject string, level CompatibilityLevel) error
    
    // Schema evolution
    Evolve(oldSchema, newSchema *Schema) (*Migration, error)
}
```

## Configuration

### Schema Settings

```yaml
schema:
  registry_url: "http://localhost:8081"
  compatibility_level: "backward"  # none, backward, forward, full
  cache_size: 1000
  validation_enabled: true
  auto_register: true
```

### Compatibility Levels

- **None**: No compatibility checking
- **Backward**: New schema can read old data
- **Forward**: Old schema can read new data
- **Full**: Both backward and forward compatibility

## Operations

### Basic Usage

```go
// Create schema parser
parser := schema.NewParser()

// Parse schema from JSON
schemaJSON := `{
    "name": "User",
    "type": "record",
    "fields": [
        {"name": "id", "type": "string"},
        {"name": "email", "type": "string"},
        {"name": "age", "type": ["null", "int"], "default": null}
    ]
}`

userSchema, err := parser.ParseJSON([]byte(schemaJSON))
if err != nil {
    return err
}

// Register schema
registry := schema.NewRegistry(config)
version, err := registry.Register(userSchema)
if err != nil {
    return err
}

// Validate data against schema
validator := schema.NewValidator()
record := map[string]interface{}{
    "id":    "user-123",
    "email": "user@example.com",
    "age":   25,
}

err = validator.Validate(record, userSchema)
if err != nil {
    return err
}
```

### Schema Evolution Example

```go
// Original schema
v1Schema := &Schema{
    Name: "User",
    Version: 1,
    Fields: []Field{
        {Name: "id", Type: TypeString, Nullable: false},
        {Name: "email", Type: TypeString, Nullable: false},
    },
}

// Evolved schema (backward compatible)
v2Schema := &Schema{
    Name: "User",
    Version: 2,
    Fields: []Field{
        {Name: "id", Type: TypeString, Nullable: false},
        {Name: "email", Type: TypeString, Nullable: false},
        {Name: "age", Type: TypeInt32, Nullable: true, DefaultValue: nil},
    },
}

// Check compatibility
compatible, err := registry.CheckCompatibility(v2Schema, 1)
if !compatible {
    return fmt.Errorf("schema evolution not compatible")
}

// Generate migration
migration, err := registry.Evolve(v1Schema, v2Schema)
if err != nil {
    return err
}
```

## Testing

### Unit Tests

```bash
go test ./internal/schema/...
```

### Integration Tests

```bash
go test ./tests/integration/schema/...
```

### Compatibility Tests

```bash
go test ./tests/schema-evolution/...
```

## Monitoring

### Key Metrics

- **Schema Count**: Number of registered schemas
- **Validation Rate**: Schema validations per second
- **Evolution Events**: Schema evolution frequency
- **Parse Errors**: Schema parsing failures
- **Compatibility Failures**: Failed compatibility checks

### Health Checks

- Validate schema registry connectivity
- Check schema consistency
- Monitor parsing performance
- Verify compatibility settings

## Troubleshooting

### Common Issues

1. **Parse Errors**: Invalid schema definitions
   - Solution: Validate schema syntax and structure

2. **Compatibility Failures**: Incompatible schema changes
   - Solution: Follow schema evolution best practices

3. **Validation Errors**: Data doesn't match schema
   - Solution: Check data format and schema constraints

4. **Performance Issues**: Slow schema operations
   - Solution: Optimize caching and validation logic

### Debug Commands

```bash
# Validate schema
storage-admin schema validate --schema-id 123

# Check compatibility
storage-admin schema check-compatibility --schema schema.json --version 1

# Show schema evolution
storage-admin schema evolution --from 1 --to 2
```

## Implementation Files

- `parser.go`: Schema parsing logic
- `types.go`: Type system definitions
- `registry.go`: Schema registry implementation
- `translator.go`: Format translation
- `validator.go`: Data validation engine
- `evolution.go`: Schema evolution logic
