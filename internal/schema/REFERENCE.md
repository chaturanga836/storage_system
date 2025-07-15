# Schema Module Reference

## Quick Reference

### Key Files
- `parser.go` - Schema parsing from various formats
- `types.go` - Type system and data type definitions
- `registry.go` - Schema registry implementation
- `translator.go` - Format translation between schema types
- `validator.go` - Data validation against schemas (if exists)

### Core Types
```go
type Parser interface {
    ParseJSON(data []byte) (*Schema, error)
    ParseAvro(data []byte) (*Schema, error)
    InferSchema(data []map[string]interface{}) (*Schema, error)
    ToJSON(*Schema) ([]byte, error)
    ToParquet(*Schema) (*parquet.Schema, error)
}

type Schema struct {
    Name        string
    Namespace   string
    Version     int
    Fields      []Field
    Metadata    map[string]string
    CreatedAt   time.Time
}

type Field struct {
    Name         string
    Type         DataType
    Nullable     bool
    DefaultValue interface{}
    Constraints  []Constraint
}

// Data types
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

### Configuration
```yaml
schema:
  registry_url: "http://localhost:8081"
  compatibility_level: "backward"  # none, backward, forward, full
  cache_size: 1000
  validation_enabled: true
  auto_register: true
```

### Common Operations
```go
// Parse schema from JSON
parser := schema.NewParser()
userSchema, err := parser.ParseJSON([]byte(schemaJSON))

// Register schema
registry := schema.NewRegistry(config)
version, err := registry.Register(userSchema)

// Validate data
validator := schema.NewValidator()
err = validator.Validate(record, userSchema)

// Check compatibility
compatible, err := registry.CheckCompatibility(newSchema, oldVersion)
```

### Schema Evolution Example
```go
// Original schema v1
v1Schema := &Schema{
    Name: "User",
    Fields: []Field{
        {Name: "id", Type: TypeString, Nullable: false},
        {Name: "email", Type: TypeString, Nullable: false},
    },
}

// Evolved schema v2 (backward compatible)
v2Schema := &Schema{
    Name: "User",
    Fields: []Field{
        {Name: "id", Type: TypeString, Nullable: false},
        {Name: "email", Type: TypeString, Nullable: false},
        {Name: "age", Type: TypeInt32, Nullable: true, DefaultValue: nil},
    },
}
```

### Error Types
- `ErrInvalidSchema` - Schema parsing or validation failed
- `ErrIncompatibleSchema` - Schema evolution not compatible
- `ErrSchemaNotFound` - Schema not found in registry
- `ErrValidationFailed` - Data doesn't match schema

### Testing
```bash
# Unit tests
go test ./internal/schema/...

# Schema evolution tests
go test ./tests/schema-evolution/...

# Compatibility tests
go test ./tests/integration/schema/...
```

### Monitoring Metrics
- `schema_registrations_total` - Total schema registrations
- `schema_validations_total` - Total data validations
- `schema_parse_errors_total` - Schema parsing failures
- `schema_compatibility_checks_total` - Compatibility checks performed
- `schema_cache_hit_ratio` - Schema cache hit percentage

### Admin Commands
```bash
# Validate schema
storage-admin schema validate --schema-id 123

# Check compatibility
storage-admin schema check-compatibility --schema schema.json --version 1

# List schemas
storage-admin schema list --tenant tenant-123

# Show evolution
storage-admin schema evolution --from 1 --to 2
```

### Supported Formats
- **JSON Schema** - Standard JSON schema format
- **Apache Avro** - Avro schema format
- **Protocol Buffers** - Protobuf schema definitions
- **Parquet Schema** - Parquet-compatible schemas

### Dependencies
- `internal/catalog` - Schema storage and retrieval
- `internal/common` - Error types and utilities
- `internal/config` - Configuration management
- External: JSON, Avro, Parquet libraries
