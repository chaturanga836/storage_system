package schema

import (
	"context"
	"fmt"
	"sync"
	"time"

	"storage-engine/internal/storage/block"
)

// SchemaRegistry manages table schemas and their evolution
type SchemaRegistry struct {
	mu             sync.RWMutex
	schemas        map[string]*TableSchema   // table_name -> schema
	schemaVersions map[string][]*TableSchema // table_name -> versions
	evolutions     []SchemaEvolution

	// Storage backend for persistence
	storage  block.StorageBackend
	basePath string

	// Configuration
	maxVersionsPerTable int
	autoEvolution       bool

	// Background tasks
	stopCh chan struct{}
	done   chan struct{}

	// Statistics
	totalTables       int
	totalEvolutions   int
	lastEvolutionTime time.Time
}

// NewSchemaRegistry creates a new schema registry
func NewSchemaRegistry(storage block.StorageBackend) *SchemaRegistry {
	return &SchemaRegistry{
		schemas:             make(map[string]*TableSchema),
		schemaVersions:      make(map[string][]*TableSchema),
		evolutions:          make([]SchemaEvolution, 0),
		storage:             storage,
		basePath:            "schemas",
		maxVersionsPerTable: 10,
		autoEvolution:       false,
		stopCh:              make(chan struct{}),
		done:                make(chan struct{}),
	}
}

// Start initializes the schema registry
func (sr *SchemaRegistry) Start(ctx context.Context) error {
	// Load existing schemas from storage
	err := sr.loadSchemas(ctx)
	if err != nil {
		return fmt.Errorf("failed to load schemas: %w", err)
	}

	// Start background tasks
	go sr.backgroundTasks(ctx)

	return nil
}

// Stop gracefully shuts down the schema registry
func (sr *SchemaRegistry) Stop() error {
	close(sr.stopCh)
	<-sr.done

	// Save current state
	ctx := context.Background()
	return sr.saveAllSchemas(ctx)
}

// RegisterSchema registers a new table schema
func (sr *SchemaRegistry) RegisterSchema(schema *TableSchema) error {
	if err := schema.Validate(); err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()

	tableName := schema.GetFullyQualifiedName()

	// Check if schema already exists
	if existing, exists := sr.schemas[tableName]; exists {
		// Check if this is an evolution
		if sr.isSchemaEvolution(existing, schema) {
			return sr.evolveSchemaUnsafe(tableName, schema)
		} else if sr.isSameSchema(existing, schema) {
			// Same schema, no changes needed
			return nil
		} else {
			return fmt.Errorf("schema already exists for table %s with different structure", tableName)
		}
	}

	// Register new schema
	schema.Version = 1
	schema.CreatedAt = time.Now()
	schema.UpdatedAt = time.Now()

	sr.schemas[tableName] = schema
	sr.schemaVersions[tableName] = []*TableSchema{schema.Clone()}
	sr.totalTables++

	// Persist schema
	ctx := context.Background()
	return sr.saveSchema(ctx, tableName, schema)
}

// GetSchema retrieves the current schema for a table
func (sr *SchemaRegistry) GetSchema(tableName string) (*TableSchema, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	schema, exists := sr.schemas[tableName]
	if !exists {
		return nil, fmt.Errorf("schema not found for table %s", tableName)
	}

	return schema.Clone(), nil
}

// GetSchemaVersion retrieves a specific version of a table schema
func (sr *SchemaRegistry) GetSchemaVersion(tableName string, version int) (*TableSchema, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	versions, exists := sr.schemaVersions[tableName]
	if !exists {
		return nil, fmt.Errorf("no schema versions found for table %s", tableName)
	}

	for _, schema := range versions {
		if schema.Version == version {
			return schema.Clone(), nil
		}
	}

	return nil, fmt.Errorf("schema version %d not found for table %s", version, tableName)
}

// ListSchemas returns all registered table schemas
func (sr *SchemaRegistry) ListSchemas() map[string]*TableSchema {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	result := make(map[string]*TableSchema)
	for name, schema := range sr.schemas {
		result[name] = schema.Clone()
	}

	return result
}

// ListSchemaVersions returns all versions of a table schema
func (sr *SchemaRegistry) ListSchemaVersions(tableName string) ([]*TableSchema, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	versions, exists := sr.schemaVersions[tableName]
	if !exists {
		return nil, fmt.Errorf("no schema versions found for table %s", tableName)
	}

	result := make([]*TableSchema, len(versions))
	for i, schema := range versions {
		result[i] = schema.Clone()
	}

	return result, nil
}

// EvolveSchema evolves a table schema to a new version
func (sr *SchemaRegistry) EvolveSchema(tableName string, newSchema *TableSchema) error {
	if err := newSchema.Validate(); err != nil {
		return fmt.Errorf("invalid new schema: %w", err)
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()

	return sr.evolveSchemaUnsafe(tableName, newSchema)
}

// evolveSchemaUnsafe performs schema evolution (caller must hold write lock)
func (sr *SchemaRegistry) evolveSchemaUnsafe(tableName string, newSchema *TableSchema) error {
	currentSchema, exists := sr.schemas[tableName]
	if !exists {
		return fmt.Errorf("table %s does not exist", tableName)
	}

	// Validate evolution compatibility
	evolutions, err := sr.planSchemaEvolution(currentSchema, newSchema)
	if err != nil {
		return fmt.Errorf("incompatible schema evolution: %w", err)
	}

	// Apply evolution
	newSchema.Version = currentSchema.Version + 1
	newSchema.CreatedAt = currentSchema.CreatedAt
	newSchema.UpdatedAt = time.Now()

	// Update registry
	sr.schemas[tableName] = newSchema

	// Add to version history
	versions := sr.schemaVersions[tableName]
	versions = append(versions, newSchema.Clone())

	// Limit number of versions
	if len(versions) > sr.maxVersionsPerTable {
		versions = versions[len(versions)-sr.maxVersionsPerTable:]
	}
	sr.schemaVersions[tableName] = versions

	// Record evolutions
	for _, evolution := range evolutions {
		evolution.Timestamp = time.Now()
		sr.evolutions = append(sr.evolutions, evolution)
	}

	sr.totalEvolutions += len(evolutions)
	sr.lastEvolutionTime = time.Now()

	// Persist new schema
	ctx := context.Background()
	return sr.saveSchema(ctx, tableName, newSchema)
}

// planSchemaEvolution analyzes the differences between schemas and plans evolution steps
func (sr *SchemaRegistry) planSchemaEvolution(current, new *TableSchema) ([]SchemaEvolution, error) {
	var evolutions []SchemaEvolution

	// Map current columns for easy lookup
	currentColumns := make(map[string]*ColumnSchema)
	for i, col := range current.Columns {
		currentColumns[col.Name] = &current.Columns[i]
	}

	// Map new columns for easy lookup
	newColumns := make(map[string]*ColumnSchema)
	for i, col := range new.Columns {
		newColumns[col.Name] = &new.Columns[i]
	}

	// Check for added columns
	for _, newCol := range new.Columns {
		if _, exists := currentColumns[newCol.Name]; !exists {
			evolution := SchemaEvolution{
				Type:       "add_column",
				TableName:  current.Name,
				ColumnName: newCol.Name,
				NewSchema:  &newCol,
			}
			evolutions = append(evolutions, evolution)
		}
	}

	// Check for removed columns
	for _, currentCol := range current.Columns {
		if _, exists := newColumns[currentCol.Name]; !exists {
			evolution := SchemaEvolution{
				Type:       "drop_column",
				TableName:  current.Name,
				ColumnName: currentCol.Name,
				OldSchema:  &currentCol,
			}
			evolutions = append(evolutions, evolution)
		}
	}

	// Check for modified columns
	for _, newCol := range new.Columns {
		if currentCol, exists := currentColumns[newCol.Name]; exists {
			if !sr.isSameColumnSchema(currentCol, &newCol) {
				// Check if modification is compatible
				if !currentCol.IsCompatible(&newCol) {
					return nil, fmt.Errorf("incompatible column modification: %s", newCol.Name)
				}

				evolution := SchemaEvolution{
					Type:       "modify_column",
					TableName:  current.Name,
					ColumnName: newCol.Name,
					OldSchema:  currentCol,
					NewSchema:  &newCol,
				}
				evolutions = append(evolutions, evolution)
			}
		}
	}

	// Check for primary key changes
	if !sr.isSamePrimaryKey(current.PrimaryKey, new.PrimaryKey) {
		evolution := SchemaEvolution{
			Type:      "modify_primary_key",
			TableName: current.Name,
			Properties: map[string]interface{}{
				"old_primary_key": current.PrimaryKey,
				"new_primary_key": new.PrimaryKey,
			},
		}
		evolutions = append(evolutions, evolution)
	}

	return evolutions, nil
}

// isSchemaEvolution checks if the new schema represents an evolution of the current schema
func (sr *SchemaRegistry) isSchemaEvolution(current, new *TableSchema) bool {
	// Basic checks for evolution vs. replacement
	if current.Name != new.Name || current.Namespace != new.Namespace {
		return false
	}

	// If schemas are identical, it's not an evolution
	if sr.isSameSchema(current, new) {
		return false
	}

	// Check if evolution is compatible
	_, err := sr.planSchemaEvolution(current, new)
	return err == nil
}

// isSameSchema compares two schemas for equality
func (sr *SchemaRegistry) isSameSchema(schema1, schema2 *TableSchema) bool {
	if schema1.Name != schema2.Name || schema1.Namespace != schema2.Namespace {
		return false
	}

	if len(schema1.Columns) != len(schema2.Columns) {
		return false
	}

	// Compare columns
	for i, col1 := range schema1.Columns {
		col2 := schema2.Columns[i]
		if !sr.isSameColumnSchema(&col1, &col2) {
			return false
		}
	}

	// Compare primary key
	if !sr.isSamePrimaryKey(schema1.PrimaryKey, schema2.PrimaryKey) {
		return false
	}

	return true
}

// isSameColumnSchema compares two column schemas for equality
func (sr *SchemaRegistry) isSameColumnSchema(col1, col2 *ColumnSchema) bool {
	if col1.Name != col2.Name || col1.Type != col2.Type || col1.Nullable != col2.Nullable {
		return false
	}

	// Compare type-specific properties
	if (col1.Length == nil) != (col2.Length == nil) {
		return false
	}
	if col1.Length != nil && col2.Length != nil && *col1.Length != *col2.Length {
		return false
	}

	if (col1.Precision == nil) != (col2.Precision == nil) {
		return false
	}
	if col1.Precision != nil && col2.Precision != nil && *col1.Precision != *col2.Precision {
		return false
	}

	if (col1.Scale == nil) != (col2.Scale == nil) {
		return false
	}
	if col1.Scale != nil && col2.Scale != nil && *col1.Scale != *col2.Scale {
		return false
	}

	return true
}

// isSamePrimaryKey compares two primary key definitions
func (sr *SchemaRegistry) isSamePrimaryKey(pk1, pk2 []string) bool {
	if len(pk1) != len(pk2) {
		return false
	}

	for i, col := range pk1 {
		if col != pk2[i] {
			return false
		}
	}

	return true
}

// GetEvolutionHistory returns the evolution history for a table
func (sr *SchemaRegistry) GetEvolutionHistory(tableName string) []SchemaEvolution {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	var history []SchemaEvolution
	for _, evolution := range sr.evolutions {
		if evolution.TableName == tableName {
			history = append(history, evolution)
		}
	}

	return history
}

// ValidateSchemaCompatibility checks if data with old schema can be read with new schema
func (sr *SchemaRegistry) ValidateSchemaCompatibility(oldSchema, newSchema *TableSchema) error {
	// Plan the evolution to check compatibility
	_, err := sr.planSchemaEvolution(oldSchema, newSchema)
	return err
}

// GetSchemaStats returns statistics about the schema registry
func (sr *SchemaRegistry) GetSchemaStats() map[string]interface{} {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	totalVersions := 0
	for _, versions := range sr.schemaVersions {
		totalVersions += len(versions)
	}

	return map[string]interface{}{
		"total_tables":           sr.totalTables,
		"total_evolutions":       sr.totalEvolutions,
		"total_versions":         totalVersions,
		"last_evolution_time":    sr.lastEvolutionTime,
		"auto_evolution":         sr.autoEvolution,
		"max_versions_per_table": sr.maxVersionsPerTable,
	}
}

// saveSchema persists a schema to storage
func (sr *SchemaRegistry) saveSchema(ctx context.Context, tableName string, schema *TableSchema) error {
	data, err := schema.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize schema: %w", err)
	}

	path := fmt.Sprintf("%s/%s.json", sr.basePath, tableName)
	return sr.storage.WriteBlock(ctx, path, data)
}

// loadSchemas loads all schemas from storage
func (sr *SchemaRegistry) loadSchemas(ctx context.Context) error {
	// This is a simplified implementation
	// In a real system, you would scan the storage for schema files
	return nil
}

// saveAllSchemas saves all schemas to storage
func (sr *SchemaRegistry) saveAllSchemas(ctx context.Context) error {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	for tableName, schema := range sr.schemas {
		err := sr.saveSchema(ctx, tableName, schema)
		if err != nil {
			return fmt.Errorf("failed to save schema for %s: %w", tableName, err)
		}
	}

	return nil
}

// backgroundTasks runs background maintenance tasks
func (sr *SchemaRegistry) backgroundTasks(ctx context.Context) {
	defer close(sr.done)

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sr.stopCh:
			return
		case <-ticker.C:
			sr.performMaintenance(ctx)
		}
	}
}

// performMaintenance performs background maintenance tasks
func (sr *SchemaRegistry) performMaintenance(ctx context.Context) {
	// Save all schemas periodically
	err := sr.saveAllSchemas(ctx)
	if err != nil {
		// Log error but continue
		fmt.Printf("Failed to save schemas during maintenance: %v\n", err)
	}

	// Cleanup old evolution history
	sr.cleanupOldEvolutions()
}

// cleanupOldEvolutions removes old evolution records
func (sr *SchemaRegistry) cleanupOldEvolutions() {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	// Keep only last 1000 evolutions
	maxEvolutions := 1000
	if len(sr.evolutions) > maxEvolutions {
		sr.evolutions = sr.evolutions[len(sr.evolutions)-maxEvolutions:]
	}
}
