package index

import (
	"context"
	"fmt"
	"sync"

	"storage-engine/internal/common"
	"storage-engine/internal/storage/block"
)

// IndexType represents the type of index
type IndexType string

const (
	Primary   IndexType = "primary"
	Secondary IndexType = "secondary"
)

// IndexMetadata contains metadata about an index
type IndexMetadata struct {
	Name      string    `json:"name"`
	Type      IndexType `json:"type"`
	Column    string    `json:"column"`
	TableName string    `json:"table_name"`
	Path      string    `json:"path"`
	Size      uint64    `json:"size"`
	Entries   uint64    `json:"entries"`
}

// IndexSerializer handles serialization and deserialization of index metadata and data
type IndexSerializer struct {
	storage block.StorageBackend
	mu      sync.RWMutex
	
	// Cache of loaded indexes
	primaryIndexes   map[string]*PrimaryIndex
	secondaryIndexes map[string]*SecondaryIndex
	metadata         map[string]*IndexMetadata
}

// NewIndexSerializer creates a new index serializer
func NewIndexSerializer(storage block.StorageBackend) *IndexSerializer {
	return &IndexSerializer{
		storage:          storage,
		primaryIndexes:   make(map[string]*PrimaryIndex),
		secondaryIndexes: make(map[string]*SecondaryIndex),
		metadata:         make(map[string]*IndexMetadata),
	}
}

// LoadPrimaryIndex loads or creates a primary index
func (is *IndexSerializer) LoadPrimaryIndex(tableName string) (*PrimaryIndex, error) {
	is.mu.Lock()
	defer is.mu.Unlock()
	
	indexKey := fmt.Sprintf("%s_primary", tableName)
	
	// Check if already loaded
	if idx, exists := is.primaryIndexes[indexKey]; exists {
		return idx, nil
	}
	
	// Create new index
	path := fmt.Sprintf("indexes/%s/primary.idx", tableName)
	idx := NewPrimaryIndex(is.storage, path)
	
	// Try to load from storage
	ctx := context.Background()
	err := idx.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load primary index for %s: %w", tableName, err)
	}
	
	// Cache the index
	is.primaryIndexes[indexKey] = idx
	
	// Update metadata
	is.metadata[indexKey] = &IndexMetadata{
		Name:      indexKey,
		Type:      Primary,
		Column:    "primary_key",
		TableName: tableName,
		Path:      path,
		Size:      idx.TotalDataSize(),
		Entries:   idx.Size(),
	}
	
	return idx, nil
}

// LoadSecondaryIndex loads or creates a secondary index
func (is *IndexSerializer) LoadSecondaryIndex(tableName, column string, indexType SecondaryIndexType) (*SecondaryIndex, error) {
	is.mu.Lock()
	defer is.mu.Unlock()
	
	indexKey := fmt.Sprintf("%s_%s_secondary", tableName, column)
	
	// Check if already loaded
	if idx, exists := is.secondaryIndexes[indexKey]; exists {
		return idx, nil
	}
	
	// Create new index
	path := fmt.Sprintf("indexes/%s/%s.idx", tableName, column)
	idx := NewSecondaryIndex(indexType, column, is.storage, path)
	
	// Try to load from storage
	ctx := context.Background()
	err := idx.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load secondary index for %s.%s: %w", tableName, column, err)
	}
	
	// Cache the index
	is.secondaryIndexes[indexKey] = idx
	
	// Update metadata
	stats := idx.GetStats()
	is.metadata[indexKey] = &IndexMetadata{
		Name:      indexKey,
		Type:      Secondary,
		Column:    column,
		TableName: tableName,
		Path:      path,
		Size:      0, // TODO: Calculate actual size
		Entries:   stats["total_entries"].(uint64),
	}
	
	return idx, nil
}

// PersistIndex saves an index to storage
func (is *IndexSerializer) PersistIndex(indexKey string) error {
	is.mu.RLock()
	defer is.mu.RUnlock()
	
	ctx := context.Background()
	
	// Try primary index first
	if idx, exists := is.primaryIndexes[indexKey]; exists {
		err := idx.Persist(ctx)
		if err != nil {
			return fmt.Errorf("failed to persist primary index %s: %w", indexKey, err)
		}
		
		// Update metadata
		if meta, exists := is.metadata[indexKey]; exists {
			meta.Size = idx.TotalDataSize()
			meta.Entries = idx.Size()
		}
		
		return nil
	}
	
	// Try secondary index
	if idx, exists := is.secondaryIndexes[indexKey]; exists {
		err := idx.Persist(ctx)
		if err != nil {
			return fmt.Errorf("failed to persist secondary index %s: %w", indexKey, err)
		}
		
		// Update metadata
		if meta, exists := is.metadata[indexKey]; exists {
			stats := idx.GetStats()
			meta.Entries = stats["total_entries"].(uint64)
		}
		
		return nil
	}
	
	return fmt.Errorf("index %s not found", indexKey)
}

// PersistAllIndexes saves all loaded indexes to storage
func (is *IndexSerializer) PersistAllIndexes() error {
	is.mu.RLock()
	keys := make([]string, 0, len(is.primaryIndexes)+len(is.secondaryIndexes))
	
	for key := range is.primaryIndexes {
		keys = append(keys, key)
	}
	for key := range is.secondaryIndexes {
		keys = append(keys, key)
	}
	is.mu.RUnlock()
	
	for _, key := range keys {
		err := is.PersistIndex(key)
		if err != nil {
			return fmt.Errorf("failed to persist index %s: %w", key, err)
		}
	}
	
	return nil
}

// ListIndexes returns metadata for all indexes of a table
func (is *IndexSerializer) ListIndexes(tableName string) ([]*IndexMetadata, error) {
	is.mu.RLock()
	defer is.mu.RUnlock()
	
	var indexes []*IndexMetadata
	
	for _, meta := range is.metadata {
		if meta.TableName == tableName {
			indexes = append(indexes, meta)
		}
	}
	
	return indexes, nil
}

// DropIndex removes an index from memory and storage
func (is *IndexSerializer) DropIndex(indexKey string) error {
	is.mu.Lock()
	defer is.mu.Unlock()
	
	ctx := context.Background()
	
	// Get metadata for path
	meta, exists := is.metadata[indexKey]
	if !exists {
		return fmt.Errorf("index %s not found", indexKey)
	}
	
	// Remove from storage
	err := is.storage.DeleteBlock(ctx, meta.Path)
	if err != nil && err != common.ErrNotFound {
		return fmt.Errorf("failed to delete index file: %w", err)
	}
	
	// Remove from memory
	delete(is.primaryIndexes, indexKey)
	delete(is.secondaryIndexes, indexKey)
	delete(is.metadata, indexKey)
	
	return nil
}

// GetIndexStats returns statistics for all loaded indexes
func (is *IndexSerializer) GetIndexStats() map[string]interface{} {
	is.mu.RLock()
	defer is.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total_primary_indexes":   len(is.primaryIndexes),
		"total_secondary_indexes": len(is.secondaryIndexes),
		"metadata_entries":        len(is.metadata),
	}
	
	var totalEntries uint64
	var totalSize uint64
	
	for _, meta := range is.metadata {
		totalEntries += meta.Entries
		totalSize += meta.Size
	}
	
	stats["total_entries"] = totalEntries
	stats["total_size"] = totalSize
	
	return stats
}

// Close performs cleanup and persists all indexes
func (is *IndexSerializer) Close() error {
	return is.PersistAllIndexes()
}

// RebuildIndex rebuilds an index from scratch
func (is *IndexSerializer) RebuildIndex(indexKey string) error {
	is.mu.Lock()
	defer is.mu.Unlock()
	
	// Clear the index from memory
	if idx, exists := is.primaryIndexes[indexKey]; exists {
		idx.Clear()
	}
	
	if idx, exists := is.secondaryIndexes[indexKey]; exists {
		idx.Clear()
	}
	
	// TODO: Implement data re-reading and index rebuilding
	// This would involve:
	// 1. Reading all data blocks for the table
	// 2. Re-extracting keys and values
	// 3. Re-inserting into the index
	
	return fmt.Errorf("index rebuilding not yet implemented")
}

// ValidateIndex performs integrity checks on an index
func (is *IndexSerializer) ValidateIndex(indexKey string) error {
	is.mu.RLock()
	defer is.mu.RUnlock()
	
	// TODO: Implement index validation
	// This would involve:
	// 1. Checking that all referenced data blocks exist
	// 2. Verifying key ordering and uniqueness
	// 3. Validating metadata consistency
	
	return fmt.Errorf("index validation not yet implemented")
}
