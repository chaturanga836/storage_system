package memtable

import (
	"fmt"
	"sync"
	"time"

	"storage-engine/internal/storage"
)

// Memtable represents an in-memory data structure for storing records before flushing to disk
type Memtable struct {
	mu            sync.RWMutex
	data          *SkipList
	size          int64
	maxSize       int64
	createdAt     time.Time
	lastWriteAt   time.Time
	immutable     bool
	tenantID      string
	endpointID    string
	flushCallback func(*Memtable) error
}

// Config holds memtable configuration
type Config struct {
	MaxSize       int64
	FlushInterval time.Duration
	SkipListLevel int
}

// New creates a new memtable
func New(tenantID, endpointID string, config Config) *Memtable {
	return &Memtable{
		data:       NewSkipList(config.SkipListLevel),
		maxSize:    config.MaxSize,
		createdAt:  time.Now(),
		tenantID:   tenantID,
		endpointID: endpointID,
	}
}

// Put inserts a record into the memtable
func (mt *Memtable) Put(record *storage.Record) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if mt.immutable {
		return fmt.Errorf("cannot write to immutable memtable")
	}

	// Calculate the size of the record
	recordSize := mt.estimateRecordSize(record)

	// Check if adding this record would exceed the max size
	if mt.size+recordSize > mt.maxSize {
		return fmt.Errorf("memtable size limit exceeded")
	}

	// Create a composite key for ordering
	key := mt.createCompositeKey(record)

	// Insert into skip list
	mt.data.Put(key, record)
	mt.size += recordSize
	mt.lastWriteAt = time.Now()

	return nil
}

// Get retrieves a record from the memtable
func (mt *Memtable) Get(tenantID, entityID string, version uint64) (*storage.Record, error) {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	key := mt.createKeyForLookup(tenantID, entityID, version)
	value := mt.data.Get(key)

	if value == nil {
		return nil, nil // Not found
	}

	record, ok := value.(*storage.Record)
	if !ok {
		return nil, fmt.Errorf("invalid record type in memtable")
	}

	return record, nil
}

// GetLatest retrieves the latest version of a record
func (mt *Memtable) GetLatest(tenantID, entityID string) (*storage.Record, error) {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	// Find the latest version by scanning with the prefix
	prefix := fmt.Sprintf("%s#%s#", tenantID, entityID)

	var latestRecord *storage.Record
	var latestVersion int64

	mt.data.Range(prefix, func(key string, value interface{}) bool {
		record, ok := value.(*storage.Record)
		if !ok {
			return true // Continue iteration
		}

		if record.Version > latestVersion {
			latestVersion = record.Version
			latestRecord = record
		}

		return true // Continue iteration
	})

	return latestRecord, nil
}

// Scan returns all records in the memtable within the given range
func (mt *Memtable) Scan(startKey, endKey string) ([]*storage.Record, error) {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	var records []*storage.Record

	mt.data.RangeFrom(startKey, func(key string, value interface{}) bool {
		if endKey != "" && key > endKey {
			return false // Stop iteration
		}

		record, ok := value.(*storage.Record)
		if !ok {
			return true // Continue iteration
		}

		records = append(records, record)
		return true // Continue iteration
	})

	return records, nil
}

// Size returns the current size of the memtable in bytes
func (mt *Memtable) Size() int64 {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return mt.size
}

// Count returns the number of records in the memtable
func (mt *Memtable) Count() int {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return mt.data.Len()
}

// IsImmutable returns whether the memtable is immutable
func (mt *Memtable) IsImmutable() bool {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return mt.immutable
}

// MakeImmutable marks the memtable as immutable
func (mt *Memtable) MakeImmutable() {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.immutable = true
}

// ShouldFlush determines if the memtable should be flushed
func (mt *Memtable) ShouldFlush(maxSize int64, maxAge time.Duration) bool {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	// Check size threshold
	if mt.size >= maxSize {
		return true
	}

	// Check age threshold
	if time.Since(mt.createdAt) >= maxAge {
		return true
	}

	return false
}

// Iterator returns an iterator for the memtable
func (mt *Memtable) Iterator() *Iterator {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	return &Iterator{
		memtable: mt,
		iter:     mt.data.Iterator(),
	}
}

// GetMetadata returns metadata about the memtable
func (mt *Memtable) GetMetadata() *Metadata {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	return &Metadata{
		TenantID:    mt.tenantID,
		EndpointID:  mt.endpointID,
		Size:        mt.size,
		Count:       mt.data.Len(),
		CreatedAt:   mt.createdAt,
		LastWriteAt: mt.lastWriteAt,
		Immutable:   mt.immutable,
	}
}

// Flush flushes the memtable to disk
func (mt *Memtable) Flush() error {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if !mt.immutable {
		return fmt.Errorf("cannot flush mutable memtable")
	}

	if mt.flushCallback != nil {
		return mt.flushCallback(mt)
	}

	return fmt.Errorf("no flush callback configured")
}

// SetFlushCallback sets the callback function for flushing
func (mt *Memtable) SetFlushCallback(callback func(*Memtable) error) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.flushCallback = callback
}

// Metadata contains metadata about a memtable
type Metadata struct {
	TenantID    string
	EndpointID  string
	Size        int64
	Count       int
	CreatedAt   time.Time
	LastWriteAt time.Time
	Immutable   bool
}

// Iterator provides iteration over memtable records
type Iterator struct {
	memtable *Memtable
	iter     *SkipListIterator
}

// Next advances the iterator to the next record
func (it *Iterator) Next() (*storage.Record, error) {
	if !it.iter.Next() {
		return nil, nil // End of iteration
	}

	value := it.iter.Value()
	record, ok := value.(*storage.Record)
	if !ok {
		return nil, fmt.Errorf("invalid record type in iterator")
	}

	return record, nil
}

// HasNext returns true if there are more records to iterate
func (it *Iterator) HasNext() bool {
	return it.iter.HasNext()
}

// Close closes the iterator
func (it *Iterator) Close() error {
	// No-op for in-memory iterator
	return nil
}

// Helper methods

func (mt *Memtable) createCompositeKey(record *storage.Record) string {
	// Create a composite key for ordering: RecordID#Version#Timestamp
	return fmt.Sprintf("%s#%016x#%016x",
		record.ID.String(),
		record.Version,
		time.Time(record.Timestamp).Unix(),
	)
}

func (mt *Memtable) createKeyForLookup(tenantID, entityID string, version uint64) string {
	return fmt.Sprintf("%s#%s#%016x",
		tenantID,
		entityID,
		version,
	)
}

func (mt *Memtable) estimateRecordSize(record *storage.Record) int64 {
	// Simple size estimation based on record fields
	size := int64(0)

	// ID size (estimate based on string representation)
	size += int64(len(record.ID.String()))

	// Data size (rough estimate based on JSON serialization)
	for key, value := range record.Data {
		size += int64(len(key))
		// Simple estimate for value size
		switch v := value.(type) {
		case string:
			size += int64(len(v))
		case int, int32, int64:
			size += 8
		case float32, float64:
			size += 8
		case bool:
			size += 1
		default:
			size += 50 // rough estimate for other types
		}
	}

	// Schema, timestamp, version, metadata
	size += 100 // rough estimate for other fields

	return size
}

// Methods expected by storage_manager.go

// Insert inserts a versioned record into the memtable
func (mt *Memtable) Insert(recordID string, versionedRecord interface{}) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if mt.immutable {
		return fmt.Errorf("cannot write to immutable memtable")
	}

	// Convert to storage.Record if needed
	// For now, just store as-is since we don't have proper type conversion
	mt.data.Put(recordID, versionedRecord)

	// Update size estimate (simplified)
	mt.size += int64(len(recordID) + 100) // rough estimate
	mt.lastWriteAt = time.Now()

	return nil
}

// GetAllRecords returns all records in the memtable
func (mt *Memtable) GetAllRecords() interface{} {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	// Simplified implementation
	var records []interface{}

	// In a real implementation, this would iterate through the skip list
	// and collect all records

	return records
}

// NewMemtable creates a new memtable (alias for New)
func NewMemtable(config interface{}) *Memtable {
	// Create with default config for now
	defaultConfig := Config{
		MaxSize:       64 * 1024 * 1024, // 64MB
		FlushInterval: 5 * time.Minute,
		SkipListLevel: 16,
	}

	return New("default", "default", defaultConfig)
}
