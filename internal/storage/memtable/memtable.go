package memtable

import (
	"fmt"
	"sync"
	"time"

	"storage-engine/internal/common"
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
	recordSize := record.EstimatedSize()

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
	var latestVersion uint64

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
	// Create a composite key for ordering: TenantID#EntityID#Version#Timestamp
	return fmt.Sprintf("%s#%s#%016x#%016x",
		record.TenantID,
		record.EntityID,
		record.Version,
		record.Timestamp,
	)
}

func (mt *Memtable) createKeyForLookup(tenantID, entityID string, version uint64) string {
	return fmt.Sprintf("%s#%s#%016x",
		tenantID,
		entityID,
		version,
	)
}
