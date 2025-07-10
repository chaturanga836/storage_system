package index

import (
	"context"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"

	"github.com/storage-system/internal/common"
	"github.com/storage-system/internal/storage/block"
)

// SecondaryIndexType defines the type of secondary index
type SecondaryIndexType int

const (
	BTreeIndex SecondaryIndexType = iota
	BloomFilterIndex
	BitMapIndex
)

// SecondaryIndexEntry represents an entry in a secondary index
type SecondaryIndexEntry struct {
	Value     []byte   // Indexed value
	PrimaryKeys [][]byte // List of primary keys that have this value
	Hash      uint32   // Hash of the value for quick comparison
}

// SecondaryIndex provides efficient lookups on non-primary key columns
type SecondaryIndex struct {
	mu       sync.RWMutex
	indexType SecondaryIndexType
	column   string
	entries  map[string]*SecondaryIndexEntry // Value hash -> entry
	
	// Storage backend for persistence
	storage block.StorageBackend
	path    string
	
	// Statistics
	totalEntries   uint64
	uniqueValues   uint64
	totalKeyRefs   uint64
}

// NewSecondaryIndex creates a new secondary index
func NewSecondaryIndex(indexType SecondaryIndexType, column string, storage block.StorageBackend, path string) *SecondaryIndex {
	return &SecondaryIndex{
		indexType: indexType,
		column:    column,
		entries:   make(map[string]*SecondaryIndexEntry),
		storage:   storage,
		path:      path,
	}
}

// Insert adds or updates an entry in the secondary index
func (si *SecondaryIndex) Insert(value []byte, primaryKey []byte) error {
	si.mu.Lock()
	defer si.mu.Unlock()
	
	hash := common.HashBytes(value)
	hashStr := fmt.Sprintf("%x", hash)
	
	entry, exists := si.entries[hashStr]
	if !exists {
		entry = &SecondaryIndexEntry{
			Value:       make([]byte, len(value)),
			PrimaryKeys: make([][]byte, 0, 1),
			Hash:        hash,
		}
		copy(entry.Value, value)
		si.entries[hashStr] = entry
		si.uniqueValues++
	}
	
	// Check if primary key already exists
	for _, existing := range entry.PrimaryKeys {
		if common.CompareBytes(existing, primaryKey) == 0 {
			return nil // Already exists
		}
	}
	
	// Add new primary key reference
	pkCopy := make([]byte, len(primaryKey))
	copy(pkCopy, primaryKey)
	entry.PrimaryKeys = append(entry.PrimaryKeys, pkCopy)
	
	si.totalEntries++
	si.totalKeyRefs++
	
	return nil
}

// Remove removes a primary key reference from the secondary index
func (si *SecondaryIndex) Remove(value []byte, primaryKey []byte) error {
	si.mu.Lock()
	defer si.mu.Unlock()
	
	hash := common.HashBytes(value)
	hashStr := fmt.Sprintf("%x", hash)
	
	entry, exists := si.entries[hashStr]
	if !exists {
		return common.ErrKeyNotFound
	}
	
	// Find and remove the primary key
	for i, pk := range entry.PrimaryKeys {
		if common.CompareBytes(pk, primaryKey) == 0 {
			// Remove by swapping with last element
			entry.PrimaryKeys[i] = entry.PrimaryKeys[len(entry.PrimaryKeys)-1]
			entry.PrimaryKeys = entry.PrimaryKeys[:len(entry.PrimaryKeys)-1]
			si.totalKeyRefs--
			
			// If no more references, remove the entire entry
			if len(entry.PrimaryKeys) == 0 {
				delete(si.entries, hashStr)
				si.uniqueValues--
			}
			
			return nil
		}
	}
	
	return common.ErrKeyNotFound
}

// FindExact finds all primary keys that have the exact value
func (si *SecondaryIndex) FindExact(value []byte) ([][]byte, error) {
	si.mu.RLock()
	defer si.mu.RUnlock()
	
	hash := common.HashBytes(value)
	hashStr := fmt.Sprintf("%x", hash)
	
	entry, exists := si.entries[hashStr]
	if !exists {
		return nil, common.ErrKeyNotFound
	}
	
	// Verify exact match (hash collision protection)
	if common.CompareBytes(entry.Value, value) != 0 {
		return nil, common.ErrKeyNotFound
	}
	
	// Return copy of primary keys
	result := make([][]byte, len(entry.PrimaryKeys))
	for i, pk := range entry.PrimaryKeys {
		result[i] = make([]byte, len(pk))
		copy(result[i], pk)
	}
	
	return result, nil
}

// FindRange finds all primary keys with values in the specified range
func (si *SecondaryIndex) FindRange(startValue, endValue []byte, limit int) ([][]byte, error) {
	si.mu.RLock()
	defer si.mu.RUnlock()
	
	var allPrimaryKeys [][]byte
	count := 0
	
	// Collect all entries that fall within the range
	var matchingEntries []*SecondaryIndexEntry
	for _, entry := range si.entries {
		if common.CompareBytes(entry.Value, startValue) >= 0 &&
		   (endValue == nil || common.CompareBytes(entry.Value, endValue) <= 0) {
			matchingEntries = append(matchingEntries, entry)
		}
	}
	
	// Sort entries by value for consistent ordering
	sort.Slice(matchingEntries, func(i, j int) bool {
		return common.CompareBytes(matchingEntries[i].Value, matchingEntries[j].Value) < 0
	})
	
	// Collect primary keys from matching entries
	for _, entry := range matchingEntries {
		for _, pk := range entry.PrimaryKeys {
			if limit > 0 && count >= limit {
				break
			}
			
			pkCopy := make([]byte, len(pk))
			copy(pkCopy, pk)
			allPrimaryKeys = append(allPrimaryKeys, pkCopy)
			count++
		}
		
		if limit > 0 && count >= limit {
			break
		}
	}
	
	return allPrimaryKeys, nil
}

// FindPrefix finds all primary keys with values that start with the given prefix
func (si *SecondaryIndex) FindPrefix(prefix []byte, limit int) ([][]byte, error) {
	si.mu.RLock()
	defer si.mu.RUnlock()
	
	var allPrimaryKeys [][]byte
	count := 0
	
	for _, entry := range si.entries {
		if len(entry.Value) >= len(prefix) {
			match := true
			for i, b := range prefix {
				if entry.Value[i] != b {
					match = false
					break
				}
			}
			
			if match {
				for _, pk := range entry.PrimaryKeys {
					if limit > 0 && count >= limit {
						break
					}
					
					pkCopy := make([]byte, len(pk))
					copy(pkCopy, pk)
					allPrimaryKeys = append(allPrimaryKeys, pkCopy)
					count++
				}
			}
		}
		
		if limit > 0 && count >= limit {
			break
		}
	}
	
	return allPrimaryKeys, nil
}

// Persist saves the index to storage
func (si *SecondaryIndex) Persist(ctx context.Context) error {
	si.mu.RLock()
	defer si.mu.RUnlock()
	
	// Serialize index data
	data, err := si.serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize secondary index: %w", err)
	}
	
	// Write to storage
	err = si.storage.WriteBlock(ctx, si.path, data)
	if err != nil {
		return fmt.Errorf("failed to write secondary index: %w", err)
	}
	
	return nil
}

// Load restores the index from storage
func (si *SecondaryIndex) Load(ctx context.Context) error {
	si.mu.Lock()
	defer si.mu.Unlock()
	
	// Read from storage
	data, err := si.storage.ReadBlock(ctx, si.path)
	if err != nil {
		if err == common.ErrNotFound {
			// Index doesn't exist yet, start fresh
			return nil
		}
		return fmt.Errorf("failed to read secondary index: %w", err)
	}
	
	// Deserialize index data
	err = si.deserialize(data)
	if err != nil {
		return fmt.Errorf("failed to deserialize secondary index: %w", err)
	}
	
	return nil
}

// GetStats returns statistics about the index
func (si *SecondaryIndex) GetStats() map[string]interface{} {
	si.mu.RLock()
	defer si.mu.RUnlock()
	
	return map[string]interface{}{
		"type":           si.indexType,
		"column":         si.column,
		"total_entries":  si.totalEntries,
		"unique_values":  si.uniqueValues,
		"total_key_refs": si.totalKeyRefs,
		"avg_refs_per_value": float64(si.totalKeyRefs) / float64(si.uniqueValues),
	}
}

// serialize converts the index to bytes for persistence
func (si *SecondaryIndex) serialize() ([]byte, error) {
	// Calculate total size needed
	totalSize := 4 + 4 + len(si.column) + 8 + 8 + 8 // header fields
	
	for _, entry := range si.entries {
		totalSize += 4 + len(entry.Value) + 4 + 4 // value_len + value + pk_count + hash
		for _, pk := range entry.PrimaryKeys {
			totalSize += 4 + len(pk) // pk_len + pk
		}
	}
	
	data := make([]byte, totalSize)
	offset := 0
	
	// Write header
	binary.LittleEndian.PutUint32(data[offset:], uint32(si.indexType))
	offset += 4
	binary.LittleEndian.PutUint32(data[offset:], uint32(len(si.column)))
	offset += 4
	copy(data[offset:], si.column)
	offset += len(si.column)
	binary.LittleEndian.PutUint64(data[offset:], si.totalEntries)
	offset += 8
	binary.LittleEndian.PutUint64(data[offset:], si.uniqueValues)
	offset += 8
	binary.LittleEndian.PutUint64(data[offset:], si.totalKeyRefs)
	offset += 8
	
	// Write entries
	for _, entry := range si.entries {
		// Value
		binary.LittleEndian.PutUint32(data[offset:], uint32(len(entry.Value)))
		offset += 4
		copy(data[offset:], entry.Value)
		offset += len(entry.Value)
		
		// Primary key count and hash
		binary.LittleEndian.PutUint32(data[offset:], uint32(len(entry.PrimaryKeys)))
		offset += 4
		binary.LittleEndian.PutUint32(data[offset:], entry.Hash)
		offset += 4
		
		// Primary keys
		for _, pk := range entry.PrimaryKeys {
			binary.LittleEndian.PutUint32(data[offset:], uint32(len(pk)))
			offset += 4
			copy(data[offset:], pk)
			offset += len(pk)
		}
	}
	
	return data, nil
}

// deserialize restores the index from bytes
func (si *SecondaryIndex) deserialize(data []byte) error {
	if len(data) < 32 {
		return fmt.Errorf("invalid secondary index data: too short")
	}
	
	offset := 0
	
	// Read header
	si.indexType = SecondaryIndexType(binary.LittleEndian.Uint32(data[offset:]))
	offset += 4
	
	columnLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4
	si.column = string(data[offset : offset+int(columnLen)])
	offset += int(columnLen)
	
	si.totalEntries = binary.LittleEndian.Uint64(data[offset:])
	offset += 8
	si.uniqueValues = binary.LittleEndian.Uint64(data[offset:])
	offset += 8
	si.totalKeyRefs = binary.LittleEndian.Uint64(data[offset:])
	offset += 8
	
	// Read entries
	si.entries = make(map[string]*SecondaryIndexEntry)
	
	for i := uint64(0); i < si.uniqueValues; i++ {
		if offset+4 > len(data) {
			return fmt.Errorf("invalid secondary index data: truncated value length")
		}
		
		// Value
		valueLen := binary.LittleEndian.Uint32(data[offset:])
		offset += 4
		
		if offset+int(valueLen) > len(data) {
			return fmt.Errorf("invalid secondary index data: truncated value")
		}
		
		value := make([]byte, valueLen)
		copy(value, data[offset:offset+int(valueLen)])
		offset += int(valueLen)
		
		if offset+8 > len(data) {
			return fmt.Errorf("invalid secondary index data: truncated entry header")
		}
		
		// Primary key count and hash
		pkCount := binary.LittleEndian.Uint32(data[offset:])
		offset += 4
		hash := binary.LittleEndian.Uint32(data[offset:])
		offset += 4
		
		// Primary keys
		primaryKeys := make([][]byte, 0, pkCount)
		for j := uint32(0); j < pkCount; j++ {
			if offset+4 > len(data) {
				return fmt.Errorf("invalid secondary index data: truncated pk length")
			}
			
			pkLen := binary.LittleEndian.Uint32(data[offset:])
			offset += 4
			
			if offset+int(pkLen) > len(data) {
				return fmt.Errorf("invalid secondary index data: truncated pk")
			}
			
			pk := make([]byte, pkLen)
			copy(pk, data[offset:offset+int(pkLen)])
			offset += int(pkLen)
			
			primaryKeys = append(primaryKeys, pk)
		}
		
		entry := &SecondaryIndexEntry{
			Value:       value,
			PrimaryKeys: primaryKeys,
			Hash:        hash,
		}
		
		hashStr := fmt.Sprintf("%x", hash)
		si.entries[hashStr] = entry
	}
	
	return nil
}

// Clear removes all entries from the index
func (si *SecondaryIndex) Clear() {
	si.mu.Lock()
	defer si.mu.Unlock()
	
	si.entries = make(map[string]*SecondaryIndexEntry)
	si.totalEntries = 0
	si.uniqueValues = 0
	si.totalKeyRefs = 0
}
