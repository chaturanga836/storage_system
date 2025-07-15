package index

import (
	"context"
	"fmt"
	"io"
	"sync"

	"storage-engine/internal/storage/block"
)

// storageAdapter adapts block.Storage to block.StorageBackend for index compatibility
type storageAdapter struct {
	storage block.Storage
}

func (sa *storageAdapter) Read(ctx context.Context, path string) ([]byte, error) {
	reader, err := sa.storage.Reader(ctx, path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	// Read all data (this is a simple implementation)
	data := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		data = append(data, buf[:n]...)
		if err != nil {
			break
		}
	}
	return data, nil
}

func (sa *storageAdapter) Write(ctx context.Context, path string, data []byte) error {
	writer, err := sa.storage.Writer(ctx, path)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = writer.Write(data)
	return err
}

func (sa *storageAdapter) ReadBlock(ctx context.Context, path string) ([]byte, error) {
	return sa.Read(ctx, path)
}

func (sa *storageAdapter) WriteBlock(ctx context.Context, path string, data []byte) error {
	return sa.Write(ctx, path, data)
}

func (sa *storageAdapter) Delete(ctx context.Context, path string) error {
	return sa.storage.Delete(ctx, path)
}

func (sa *storageAdapter) Exists(ctx context.Context, path string) (bool, error) {
	_, err := sa.storage.Stat(ctx, path)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (sa *storageAdapter) List(ctx context.Context, prefix string) ([]string, error) {
	metadata, err := sa.storage.List(ctx, prefix)
	if err != nil {
		return nil, err
	}

	paths := make([]string, len(metadata))
	for i, meta := range metadata {
		paths[i] = meta.Path
	}
	return paths, nil
}

func (sa *storageAdapter) Size(ctx context.Context, path string) (int64, error) {
	metadata, err := sa.storage.Stat(ctx, path)
	if err != nil {
		return 0, err
	}
	return metadata.Size, nil
}

func (sa *storageAdapter) OpenReader(ctx context.Context, path string) (io.ReadCloser, error) {
	return sa.storage.Reader(ctx, path)
}

func (sa *storageAdapter) OpenWriter(ctx context.Context, path string) (io.WriteCloser, error) {
	return sa.storage.Writer(ctx, path)
}

func (sa *storageAdapter) Copy(ctx context.Context, srcPath, dstPath string) error {
	return sa.storage.Copy(ctx, srcPath, dstPath)
}

func (sa *storageAdapter) Health(ctx context.Context) error {
	return sa.storage.Health(ctx)
}

func (sa *storageAdapter) Close() error {
	// Storage doesn't have a Close method, so this is a no-op
	return nil
}

// Manager manages all indexes for tables
type Manager struct {
	mu               sync.RWMutex
	primaryIndexes   map[string]*PrimaryIndex
	secondaryIndexes map[string]map[string]*SecondaryIndex
	storage          block.Storage
}

// NewManager creates a new index manager
func NewManager(storage block.Storage) *Manager {
	return &Manager{
		primaryIndexes:   make(map[string]*PrimaryIndex),
		secondaryIndexes: make(map[string]map[string]*SecondaryIndex),
		storage:          storage,
	}
}

// UpdateIndexes updates indexes for a table with new records
func (m *Manager) UpdateIndexes(tableID string, records interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get or create primary index for table
	primaryIndex, exists := m.primaryIndexes[tableID]
	if !exists {
		// Create new primary index
		indexPath := fmt.Sprintf("indexes/%s/primary.idx", tableID)
		adapter := &storageAdapter{storage: m.storage}
		primaryIndex = NewPrimaryIndex(adapter, indexPath)
		m.primaryIndexes[tableID] = primaryIndex
	}

	// For now, just acknowledge the update
	// TODO: Implement actual index updating logic based on record type

	return nil
}

// GetPrimaryIndex returns the primary index for a table
func (m *Manager) GetPrimaryIndex(tableID string) (*PrimaryIndex, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	index, exists := m.primaryIndexes[tableID]
	if !exists {
		return nil, fmt.Errorf("primary index not found for table %s", tableID)
	}

	return index, nil
}

// CreateSecondaryIndex creates a secondary index for a table column
func (m *Manager) CreateSecondaryIndex(tableID, columnName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get or create secondary indexes map for table
	if m.secondaryIndexes[tableID] == nil {
		m.secondaryIndexes[tableID] = make(map[string]*SecondaryIndex)
	}

	// Create secondary index
	indexPath := fmt.Sprintf("indexes/%s/secondary_%s.idx", tableID, columnName)
	adapter := &storageAdapter{storage: m.storage}
	secondaryIndex := NewSecondaryIndex(BTreeIndex, columnName, adapter, indexPath)
	m.secondaryIndexes[tableID][columnName] = secondaryIndex

	return nil
}

// GetSecondaryIndex returns a secondary index for a table column
func (m *Manager) GetSecondaryIndex(tableID, columnName string) (*SecondaryIndex, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tableIndexes, exists := m.secondaryIndexes[tableID]
	if !exists {
		return nil, fmt.Errorf("no secondary indexes found for table %s", tableID)
	}

	index, exists := tableIndexes[columnName]
	if !exists {
		return nil, fmt.Errorf("secondary index not found for column %s in table %s", columnName, tableID)
	}

	return index, nil
}

// ListIndexes returns all indexes for a table
func (m *Manager) ListIndexes(tableID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var indexes []string

	// Add primary index
	if _, exists := m.primaryIndexes[tableID]; exists {
		indexes = append(indexes, "primary")
	}

	// Add secondary indexes
	if tableIndexes, exists := m.secondaryIndexes[tableID]; exists {
		for columnName := range tableIndexes {
			indexes = append(indexes, fmt.Sprintf("secondary_%s", columnName))
		}
	}

	return indexes, nil
}

// DropIndex drops an index for a table
func (m *Manager) DropIndex(tableID, indexName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if indexName == "primary" {
		delete(m.primaryIndexes, tableID)
		return nil
	}

	// Handle secondary indexes
	if tableIndexes, exists := m.secondaryIndexes[tableID]; exists {
		for columnName := range tableIndexes {
			if fmt.Sprintf("secondary_%s", columnName) == indexName {
				delete(tableIndexes, columnName)
				return nil
			}
		}
	}

	return fmt.Errorf("index %s not found for table %s", indexName, tableID)
}

// Close closes all indexes and releases resources
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close primary indexes
	for _, index := range m.primaryIndexes {
		// TODO: Implement Close method on PrimaryIndex if needed
		_ = index
	}

	// Close secondary indexes
	for _, tableIndexes := range m.secondaryIndexes {
		for _, index := range tableIndexes {
			// TODO: Implement Close method on SecondaryIndex if needed
			_ = index
		}
	}

	return nil
}
