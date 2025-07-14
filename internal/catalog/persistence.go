package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"storage-engine/internal/storage/block"
)

// PersistenceConfig configures how catalog data is persisted
type PersistenceConfig struct {
	BackupInterval     time.Duration `json:"backup_interval"`
	MaxBackupFiles     int           `json:"max_backup_files"`
	CompressionEnabled bool          `json:"compression_enabled"`
	EncryptionEnabled  bool          `json:"encryption_enabled"`
	ChecksumValidation bool          `json:"checksum_validation"`
}

// DefaultPersistenceConfig returns default persistence configuration
func DefaultPersistenceConfig() *PersistenceConfig {
	return &PersistenceConfig{
		BackupInterval:     30 * time.Minute,
		MaxBackupFiles:     10,
		CompressionEnabled: true,
		EncryptionEnabled:  false,
		ChecksumValidation: true,
	}
}

// CatalogPersistence handles persistence of catalog metadata
type CatalogPersistence struct {
	mu      sync.RWMutex
	storage block.StorageBackend
	config  *PersistenceConfig
	catalog *CatalogImpl

	// File paths
	primaryPath    string
	backupBasePath string

	// Background persistence
	stopCh           chan struct{}
	done             chan struct{}
	lastBackupTime   time.Time
	backupInProgress bool

	// Statistics
	totalBackups    uint64
	totalRestores   uint64
	lastBackupSize  uint64
	lastRestoreTime time.Time
}

// NewCatalogPersistence creates a new catalog persistence manager
func NewCatalogPersistence(storage block.StorageBackend, catalog *CatalogImpl, config *PersistenceConfig) *CatalogPersistence {
	if config == nil {
		config = DefaultPersistenceConfig()
	}

	return &CatalogPersistence{
		storage:        storage,
		config:         config,
		catalog:        catalog,
		primaryPath:    "catalog/metadata.json",
		backupBasePath: "catalog/backups/metadata",
		stopCh:         make(chan struct{}),
		done:           make(chan struct{}),
		lastBackupTime: time.Now(),
	}
}

// Start begins background persistence operations
func (cp *CatalogPersistence) Start(ctx context.Context) error {
	// Load existing catalog data
	err := cp.LoadCatalog(ctx)
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w", err)
	}

	// Start background backup routine
	go cp.backgroundBackup(ctx)

	return nil
}

// Stop gracefully shuts down persistence operations
func (cp *CatalogPersistence) Stop() error {
	close(cp.stopCh)
	<-cp.done

	// Perform final backup
	ctx := context.Background()
	return cp.SaveCatalog(ctx)
}

// SaveCatalog saves the current catalog state to storage
func (cp *CatalogPersistence) SaveCatalog(ctx context.Context) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Serialize catalog data
	data, err := cp.serializeCatalog()
	if err != nil {
		return fmt.Errorf("failed to serialize catalog: %w", err)
	}

	// Apply compression if enabled
	if cp.config.CompressionEnabled {
		data, err = cp.compressData(data)
		if err != nil {
			return fmt.Errorf("failed to compress catalog data: %w", err)
		}
	}

	// Apply encryption if enabled
	if cp.config.EncryptionEnabled {
		data, err = cp.encryptData(data)
		if err != nil {
			return fmt.Errorf("failed to encrypt catalog data: %w", err)
		}
	}

	// Write to primary location
	err = cp.storage.WriteBlock(ctx, cp.primaryPath, data)
	if err != nil {
		return fmt.Errorf("failed to write catalog to storage: %w", err)
	}

	cp.lastBackupSize = uint64(len(data))
	cp.lastBackupTime = time.Now()
	cp.totalBackups++

	return nil
}

// LoadCatalog loads catalog data from storage
func (cp *CatalogPersistence) LoadCatalog(ctx context.Context) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Try to load from primary location first
	data, err := cp.storage.ReadBlock(ctx, cp.primaryPath)
	if err != nil {
		// Simple error check instead of specific error type
		// No existing catalog, try backup or start fresh
		return cp.loadFromBackup(ctx)
	}

	return cp.deserializeCatalogData(data)
}

// loadFromBackup attempts to load catalog from the most recent backup
func (cp *CatalogPersistence) loadFromBackup(ctx context.Context) error {
	// Try to find the most recent backup
	for i := 0; i < cp.config.MaxBackupFiles; i++ {
		backupPath := fmt.Sprintf("%s_%d.json", cp.backupBasePath, i)

		data, err := cp.storage.ReadBlock(ctx, backupPath)
		if err == nil {
			err = cp.deserializeCatalogData(data)
			if err == nil {
				cp.totalRestores++
				cp.lastRestoreTime = time.Now()
				return nil
			}
		}
	}

	// No valid backup found, start fresh
	return nil
}

// CreateBackup creates a numbered backup of the current catalog
func (cp *CatalogPersistence) CreateBackup(ctx context.Context) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.backupInProgress {
		return fmt.Errorf("backup already in progress")
	}

	cp.backupInProgress = true
	defer func() {
		cp.backupInProgress = false
	}()

	// Rotate existing backups
	for i := cp.config.MaxBackupFiles - 1; i > 0; i-- {
		oldPath := fmt.Sprintf("%s_%d.json", cp.backupBasePath, i-1)
		newPath := fmt.Sprintf("%s_%d.json", cp.backupBasePath, i)

		// Check if old backup exists
		_, err := cp.storage.ReadBlock(ctx, oldPath)
		if err == nil {
			// Copy old backup to new location
			data, err := cp.storage.ReadBlock(ctx, oldPath)
			if err == nil {
				err = cp.storage.WriteBlock(ctx, newPath, data)
				if err != nil {
					// Log error but continue
					fmt.Printf("Warning: failed to rotate backup %s to %s: %v\n", oldPath, newPath, err)
				}
			}
		}
	}

	// Create new backup
	data, err := cp.serializeCatalog()
	if err != nil {
		return fmt.Errorf("failed to serialize catalog for backup: %w", err)
	}

	if cp.config.CompressionEnabled {
		data, err = cp.compressData(data)
		if err != nil {
			return fmt.Errorf("failed to compress backup data: %w", err)
		}
	}

	backupPath := fmt.Sprintf("%s_0.json", cp.backupBasePath)
	err = cp.storage.WriteBlock(ctx, backupPath, data)
	if err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	return nil
}

// backgroundBackup runs periodic backup operations
func (cp *CatalogPersistence) backgroundBackup(ctx context.Context) {
	defer close(cp.done)

	ticker := time.NewTicker(cp.config.BackupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cp.stopCh:
			return
		case <-ticker.C:
			err := cp.CreateBackup(ctx)
			if err != nil {
				// Log error but continue
				fmt.Printf("Background backup failed: %v\n", err)
			}
		}
	}
}

// serializeCatalog converts catalog data to JSON bytes
func (cp *CatalogPersistence) serializeCatalog() ([]byte, error) {
	catalogData := struct {
		Tables   map[string]*TableInfo  `json:"tables"`
		Schemas  map[string]*SchemaInfo `json:"schemas"`
		Indexes  map[string]*IndexInfo  `json:"indexes"`
		Stats    *CatalogStats          `json:"stats"`
		Metadata *CatalogMetadata       `json:"metadata"`
		Version  string                 `json:"version"`
		SavedAt  time.Time              `json:"saved_at"`
	}{
		Tables:   cp.catalog.tables,
		Schemas:  cp.catalog.schemas,
		Indexes:  cp.catalog.indexes,
		Stats:    cp.catalog.stats,
		Metadata: cp.catalog.metadata,
		Version:  "1.0",
		SavedAt:  time.Now(),
	}

	return json.MarshalIndent(catalogData, "", "  ")
}

// deserializeCatalogData restores catalog from JSON bytes
func (cp *CatalogPersistence) deserializeCatalogData(data []byte) error {
	// Apply decryption if enabled
	if cp.config.EncryptionEnabled {
		decrypted, err := cp.decryptData(data)
		if err != nil {
			return fmt.Errorf("failed to decrypt catalog data: %w", err)
		}
		data = decrypted
	}

	// Apply decompression if enabled
	if cp.config.CompressionEnabled {
		decompressed, err := cp.decompressData(data)
		if err != nil {
			return fmt.Errorf("failed to decompress catalog data: %w", err)
		}
		data = decompressed
	}

	// Validate checksum if enabled
	if cp.config.ChecksumValidation {
		if !cp.validateChecksum(data) {
			return fmt.Errorf("catalog data checksum validation failed")
		}
	}

	var catalogData struct {
		Tables   map[string]*TableInfo  `json:"tables"`
		Schemas  map[string]*SchemaInfo `json:"schemas"`
		Indexes  map[string]*IndexInfo  `json:"indexes"`
		Stats    *CatalogStats          `json:"stats"`
		Metadata *CatalogMetadata       `json:"metadata"`
		Version  string                 `json:"version"`
		SavedAt  time.Time              `json:"saved_at"`
	}

	err := json.Unmarshal(data, &catalogData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal catalog data: %w", err)
	}

	// Restore catalog state
	cp.catalog.mu.Lock()
	defer cp.catalog.mu.Unlock()

	if catalogData.Tables != nil {
		cp.catalog.tables = catalogData.Tables
	}
	if catalogData.Schemas != nil {
		cp.catalog.schemas = catalogData.Schemas
	}
	if catalogData.Indexes != nil {
		cp.catalog.indexes = catalogData.Indexes
	}
	if catalogData.Stats != nil {
		cp.catalog.stats = catalogData.Stats
	}
	if catalogData.Metadata != nil {
		cp.catalog.metadata = catalogData.Metadata
	}

	return nil
}

// compressData compresses data using a simple compression algorithm
func (cp *CatalogPersistence) compressData(data []byte) ([]byte, error) {
	// In a real implementation, you would use gzip, snappy, or another compression library
	// For now, we'll just return the data as-is with a compression header
	compressed := make([]byte, len(data)+4)
	compressed[0] = 'C' // Compression marker
	compressed[1] = 'M'
	compressed[2] = 'P'
	compressed[3] = '1' // Version
	copy(compressed[4:], data)
	return compressed, nil
}

// decompressData decompresses data
func (cp *CatalogPersistence) decompressData(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return data, nil // Not compressed
	}

	if string(data[0:3]) == "CMP" {
		// Remove compression header
		return data[4:], nil
	}

	return data, nil // Not compressed
}

// encryptData encrypts data (placeholder implementation)
func (cp *CatalogPersistence) encryptData(data []byte) ([]byte, error) {
	// In a real implementation, you would use AES or another encryption algorithm
	// For now, we'll just add an encryption header
	encrypted := make([]byte, len(data)+4)
	encrypted[0] = 'E' // Encryption marker
	encrypted[1] = 'N'
	encrypted[2] = 'C'
	encrypted[3] = '1' // Version
	copy(encrypted[4:], data)
	return encrypted, nil
}

// decryptData decrypts data (placeholder implementation)
func (cp *CatalogPersistence) decryptData(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return data, nil // Not encrypted
	}

	if string(data[0:3]) == "ENC" {
		// Remove encryption header
		return data[4:], nil
	}

	return data, nil // Not encrypted
}

// validateChecksum validates data integrity
func (cp *CatalogPersistence) validateChecksum(data []byte) bool {
	// In a real implementation, you would calculate and validate a checksum
	// For now, we'll just return true
	return true
}

// GetPersistenceStats returns statistics about persistence operations
func (cp *CatalogPersistence) GetPersistenceStats() map[string]interface{} {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	return map[string]interface{}{
		"total_backups":       cp.totalBackups,
		"total_restores":      cp.totalRestores,
		"last_backup_time":    cp.lastBackupTime,
		"last_backup_size":    cp.lastBackupSize,
		"last_restore_time":   cp.lastRestoreTime,
		"backup_in_progress":  cp.backupInProgress,
		"backup_interval":     cp.config.BackupInterval,
		"max_backup_files":    cp.config.MaxBackupFiles,
		"compression_enabled": cp.config.CompressionEnabled,
		"encryption_enabled":  cp.config.EncryptionEnabled,
	}
}

// ValidateCatalogIntegrity performs integrity checks on the catalog
func (cp *CatalogPersistence) ValidateCatalogIntegrity(ctx context.Context) error {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	// Check that all referenced tables exist
	for indexName, indexInfo := range cp.catalog.indexes {
		if _, exists := cp.catalog.tables[indexInfo.TableName]; !exists {
			return fmt.Errorf("index %s references non-existent table %s", indexName, indexInfo.TableName)
		}
	}

	// Check that all referenced schemas exist
	for tableName, tableInfo := range cp.catalog.tables {
		if _, exists := cp.catalog.schemas[tableInfo.SchemaName]; !exists {
			return fmt.Errorf("table %s references non-existent schema %s", tableName, tableInfo.SchemaName)
		}
	}

	// Validate table statistics consistency
	for tableName, tableInfo := range cp.catalog.tables {
		if tableInfo.RowCount < 0 {
			return fmt.Errorf("table %s has invalid row count: %d", tableName, tableInfo.RowCount)
		}

		if tableInfo.DataSize < 0 {
			return fmt.Errorf("table %s has invalid data size: %d", tableName, tableInfo.DataSize)
		}
	}

	return nil
}

// RepairCatalog attempts to repair catalog inconsistencies
func (cp *CatalogPersistence) RepairCatalog(ctx context.Context) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	repairCount := 0

	// Remove indexes for non-existent tables
	for indexName, indexInfo := range cp.catalog.indexes {
		if _, exists := cp.catalog.tables[indexInfo.TableName]; !exists {
			delete(cp.catalog.indexes, indexName)
			repairCount++
		}
	}

	// Remove tables with non-existent schemas
	for tableName, tableInfo := range cp.catalog.tables {
		if _, exists := cp.catalog.schemas[tableInfo.SchemaName]; !exists {
			delete(cp.catalog.tables, tableName)
			repairCount++
		}
	}

	// Fix negative statistics
	for _, tableInfo := range cp.catalog.tables {
		if tableInfo.RowCount < 0 {
			tableInfo.RowCount = 0
			repairCount++
		}
		if tableInfo.DataSize < 0 {
			tableInfo.DataSize = 0
			repairCount++
		}
	}

	if repairCount > 0 {
		// Save repaired catalog
		err := cp.SaveCatalog(ctx)
		if err != nil {
			return fmt.Errorf("failed to save repaired catalog: %w", err)
		}
	}

	return nil
}
