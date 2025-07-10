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

// PrimaryIndexEntry represents an entry in the primary index
type PrimaryIndexEntry struct {
	Key    []byte // Sorting key
	Offset uint64 // Offset within the data block
	Size   uint32 // Size of the record
	Hash   uint32 // Hash of the key for quick comparison
}

// PrimaryIndex provides efficient lookups and range scans for primary keys
type PrimaryIndex struct {
	mu      sync.RWMutex
	entries []PrimaryIndexEntry
	sorted  bool
	
	// Storage backend for persistence
	storage block.StorageBackend
	path    string
	
	// Statistics
	totalEntries uint64
	totalSize    uint64
}

// NewPrimaryIndex creates a new primary index
func NewPrimaryIndex(storage block.StorageBackend, path string) *PrimaryIndex {
	return &PrimaryIndex{
		entries: make([]PrimaryIndexEntry, 0, 1024),
		sorted:  true,
		storage: storage,
		path:    path,
	}
}

// Insert adds a new entry to the primary index
func (pi *PrimaryIndex) Insert(key []byte, offset uint64, size uint32) error {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	
	hash := common.HashBytes(key)
	entry := PrimaryIndexEntry{
		Key:    make([]byte, len(key)),
		Offset: offset,
		Size:   size,
		Hash:   hash,
	}
	copy(entry.Key, key)
	
	pi.entries = append(pi.entries, entry)
	pi.sorted = false
	pi.totalEntries++
	pi.totalSize += uint64(size)
	
	return nil
}

// Find locates an entry by exact key match
func (pi *PrimaryIndex) Find(key []byte) (*PrimaryIndexEntry, error) {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	
	if !pi.sorted {
		pi.sortEntriesUnsafe()
	}
	
	hash := common.HashBytes(key)
	
	// Binary search with hash optimization
	left, right := 0, len(pi.entries)-1
	
	for left <= right {
		mid := (left + right) / 2
		entry := &pi.entries[mid]
		
		// Quick hash comparison first
		if entry.Hash == hash {
			cmp := common.CompareBytes(entry.Key, key)
			if cmp == 0 {
				return entry, nil
			} else if cmp < 0 {
				left = mid + 1
			} else {
				right = mid - 1
			}
		} else if entry.Hash < hash {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	
	return nil, common.ErrKeyNotFound
}

// RangeScan returns entries within the specified key range
func (pi *PrimaryIndex) RangeScan(startKey, endKey []byte, limit int) ([]PrimaryIndexEntry, error) {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	
	if !pi.sorted {
		pi.sortEntriesUnsafe()
	}
	
	var results []PrimaryIndexEntry
	count := 0
	
	for _, entry := range pi.entries {
		if limit > 0 && count >= limit {
			break
		}
		
		// Check if key is within range
		if common.CompareBytes(entry.Key, startKey) >= 0 && 
		   (endKey == nil || common.CompareBytes(entry.Key, endKey) <= 0) {
			results = append(results, entry)
			count++
		}
	}
	
	return results, nil
}

// Persist saves the index to storage
func (pi *PrimaryIndex) Persist(ctx context.Context) error {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	
	if !pi.sorted {
		pi.sortEntriesUnsafe()
	}
	
	// Serialize index data
	data, err := pi.serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize index: %w", err)
	}
	
	// Write to storage
	err = pi.storage.WriteBlock(ctx, pi.path, data)
	if err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}
	
	return nil
}

// Load restores the index from storage
func (pi *PrimaryIndex) Load(ctx context.Context) error {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	
	// Read from storage
	data, err := pi.storage.ReadBlock(ctx, pi.path)
	if err != nil {
		if err == common.ErrNotFound {
			// Index doesn't exist yet, start fresh
			return nil
		}
		return fmt.Errorf("failed to read index: %w", err)
	}
	
	// Deserialize index data
	err = pi.deserialize(data)
	if err != nil {
		return fmt.Errorf("failed to deserialize index: %w", err)
	}
	
	pi.sorted = true
	return nil
}

// Size returns the number of entries in the index
func (pi *PrimaryIndex) Size() uint64 {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.totalEntries
}

// TotalDataSize returns the total size of data referenced by the index
func (pi *PrimaryIndex) TotalDataSize() uint64 {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.totalSize
}

// sortEntriesUnsafe sorts entries by key (caller must hold write lock)
func (pi *PrimaryIndex) sortEntriesUnsafe() {
	if pi.sorted {
		return
	}
	
	sort.Slice(pi.entries, func(i, j int) bool {
		return common.CompareBytes(pi.entries[i].Key, pi.entries[j].Key) < 0
	})
	
	pi.sorted = true
}

// serialize converts the index to bytes for persistence
func (pi *PrimaryIndex) serialize() ([]byte, error) {
	// Calculate total size needed
	totalSize := 8 + 8 // header: entry count + total data size
	for _, entry := range pi.entries {
		totalSize += 4 + len(entry.Key) + 8 + 4 + 4 // key_len + key + offset + size + hash
	}
	
	data := make([]byte, totalSize)
	offset := 0
	
	// Write header
	binary.LittleEndian.PutUint64(data[offset:], pi.totalEntries)
	offset += 8
	binary.LittleEndian.PutUint64(data[offset:], pi.totalSize)
	offset += 8
	
	// Write entries
	for _, entry := range pi.entries {
		// Key length and key
		binary.LittleEndian.PutUint32(data[offset:], uint32(len(entry.Key)))
		offset += 4
		copy(data[offset:], entry.Key)
		offset += len(entry.Key)
		
		// Offset, size, hash
		binary.LittleEndian.PutUint64(data[offset:], entry.Offset)
		offset += 8
		binary.LittleEndian.PutUint32(data[offset:], entry.Size)
		offset += 4
		binary.LittleEndian.PutUint32(data[offset:], entry.Hash)
		offset += 4
	}
	
	return data, nil
}

// deserialize restores the index from bytes
func (pi *PrimaryIndex) deserialize(data []byte) error {
	if len(data) < 16 {
		return fmt.Errorf("invalid index data: too short")
	}
	
	offset := 0
	
	// Read header
	pi.totalEntries = binary.LittleEndian.Uint64(data[offset:])
	offset += 8
	pi.totalSize = binary.LittleEndian.Uint64(data[offset:])
	offset += 8
	
	// Read entries
	pi.entries = make([]PrimaryIndexEntry, 0, pi.totalEntries)
	
	for i := uint64(0); i < pi.totalEntries; i++ {
		if offset+4 > len(data) {
			return fmt.Errorf("invalid index data: truncated key length")
		}
		
		// Key length and key
		keyLen := binary.LittleEndian.Uint32(data[offset:])
		offset += 4
		
		if offset+int(keyLen) > len(data) {
			return fmt.Errorf("invalid index data: truncated key")
		}
		
		key := make([]byte, keyLen)
		copy(key, data[offset:offset+int(keyLen)])
		offset += int(keyLen)
		
		if offset+16 > len(data) {
			return fmt.Errorf("invalid index data: truncated entry data")
		}
		
		// Offset, size, hash
		entryOffset := binary.LittleEndian.Uint64(data[offset:])
		offset += 8
		size := binary.LittleEndian.Uint32(data[offset:])
		offset += 4
		hash := binary.LittleEndian.Uint32(data[offset:])
		offset += 4
		
		entry := PrimaryIndexEntry{
			Key:    key,
			Offset: entryOffset,
			Size:   size,
			Hash:   hash,
		}
		
		pi.entries = append(pi.entries, entry)
	}
	
	return nil
}

// Clear removes all entries from the index
func (pi *PrimaryIndex) Clear() {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	
	pi.entries = pi.entries[:0]
	pi.sorted = true
	pi.totalEntries = 0
	pi.totalSize = 0
}
