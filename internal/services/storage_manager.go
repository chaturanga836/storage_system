package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"storage-engine/internal/catalog"
	"storage-engine/internal/common"
	"storage-engine/internal/config"
	"storage-engine/internal/messaging"
	"storage-engine/internal/schema"
	"storage-engine/internal/storage/block"
	"storage-engine/internal/storage/compaction"
	"storage-engine/internal/storage/index"
	"storage-engine/internal/storage/memtable"
	"storage-engine/internal/storage/mvcc"
	"storage-engine/internal/storage/parquet"
	"storage-engine/internal/wal"
)

// StorageManager orchestrates all storage operations and components
type StorageManager struct {
	config         *config.StorageConfig
	catalog        *catalog.Catalog
	schemaRegistry *schema.Registry
	walManager     *wal.Manager
	memtables      map[string]*memtable.Memtable
	blockStorage   block.Storage
	indexManager   *index.Manager
	compactor      *compaction.Compactor
	mvccResolver   *mvcc.Resolver
	parquetWriter  *parquet.Writer
	parquetReader  *parquet.Reader
	publisher      messaging.Publisher

	// Synchronization and lifecycle
	mu            sync.RWMutex
	running       bool
	stopChan      chan struct{}
	flushTicker   *time.Ticker
	compactTicker *time.Ticker

	// Metrics and monitoring
	metrics *StorageMetrics
}

// StorageMetrics tracks storage system metrics
type StorageMetrics struct {
	IngestedRecords    int64
	ProcessedRecords   int64
	CompactedFiles     int64
	IndexUpdates       int64
	MemtableFlushes    int64
	QueryLatency       time.Duration
	StorageUtilization float64
	ErrorCount         int64
	mu                 sync.RWMutex
}

// NewStorageManager creates a new storage manager
func NewStorageManager(cfg *config.StorageConfig, publisher messaging.Publisher) (*StorageManager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("storage config is required")
	}

	// Initialize catalog
	catalogInstance, err := catalog.NewCatalog(cfg.CatalogConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create catalog: %w", err)
	}

	// Initialize schema registry
	schemaRegistry := schema.NewRegistry()

	// Initialize WAL manager
	walManager, err := wal.NewManager(cfg.WALConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create WAL manager: %w", err)
	}

	// Initialize block storage
	var blockStorage block.Storage
	switch cfg.BlockStorageType {
	case "local":
		blockStorage = block.NewLocalFS(cfg.LocalStoragePath)
	case "s3":
		blockStorage, err = block.NewS3FS(cfg.S3Config)
		if err != nil {
			return nil, fmt.Errorf("failed to create S3 storage: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported block storage type: %s", cfg.BlockStorageType)
	}

	// Initialize index manager
	indexManager := index.NewManager(blockStorage)

	// Initialize compactor
	compactor := compaction.NewCompactor(cfg.CompactionConfig, blockStorage)

	// Initialize MVCC resolver
	mvccResolver := mvcc.NewResolver(cfg.MVCCConfig)

	// Initialize Parquet components
	parquetWriter := parquet.NewWriter(blockStorage)
	parquetReader := parquet.NewReader(blockStorage)

	sm := &StorageManager{
		config:         cfg,
		catalog:        catalogInstance,
		schemaRegistry: schemaRegistry,
		walManager:     walManager,
		memtables:      make(map[string]*memtable.Memtable),
		blockStorage:   blockStorage,
		indexManager:   indexManager,
		compactor:      compactor,
		mvccResolver:   mvccResolver,
		parquetWriter:  parquetWriter,
		parquetReader:  parquetReader,
		publisher:      publisher,
		stopChan:       make(chan struct{}),
		metrics:        &StorageMetrics{},
	}

	return sm, nil
}

// Start starts the storage manager
func (sm *StorageManager) Start(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.running {
		return fmt.Errorf("storage manager is already running")
	}

	// Start background processes
	sm.flushTicker = time.NewTicker(time.Duration(sm.config.FlushIntervalMs) * time.Millisecond)
	sm.compactTicker = time.NewTicker(time.Duration(sm.config.CompactionIntervalMs) * time.Millisecond)

	sm.running = true

	// Start background goroutines
	go sm.backgroundFlush(ctx)
	go sm.backgroundCompaction(ctx)
	go sm.backgroundMetrics(ctx)

	// Publish startup event
	if sm.publisher != nil {
		eventData := map[string]interface{}{
			"component": "storage_manager",
			"status":    "started",
		}
		sm.publishEvent(ctx, messaging.EventDataProcessed, eventData)
	}

	return nil
}

// Stop stops the storage manager
func (sm *StorageManager) Stop(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.running {
		return nil
	}

	sm.running = false

	// Stop tickers
	if sm.flushTicker != nil {
		sm.flushTicker.Stop()
	}
	if sm.compactTicker != nil {
		sm.compactTicker.Stop()
	}

	// Signal stop
	close(sm.stopChan)

	// Flush remaining data
	if err := sm.flushAllMemtables(ctx); err != nil {
		return fmt.Errorf("failed to flush memtables during shutdown: %w", err)
	}

	// Publish shutdown event
	if sm.publisher != nil {
		eventData := map[string]interface{}{
			"component": "storage_manager",
			"status":    "stopped",
		}
		sm.publishEvent(ctx, messaging.EventDataProcessed, eventData)
	}

	return nil
}

// IngestRecord ingests a single record into the storage system
func (sm *StorageManager) IngestRecord(ctx context.Context, tableID string, record map[string]interface{}) error {
	sm.mu.RLock()
	if !sm.running {
		sm.mu.RUnlock()
		return fmt.Errorf("storage manager is not running")
	}
	sm.mu.RUnlock()

	// Get or create memtable for table
	memTable, err := sm.getOrCreateMemtable(tableID)
	if err != nil {
		return fmt.Errorf("failed to get memtable for table %s: %w", tableID, err)
	}

	// Generate record ID and version
	recordID := common.GenerateID()
	version := sm.mvccResolver.NewVersion()

	// Create versioned record
	versionedRecord := &mvcc.VersionedRecord{
		ID:        recordID,
		Version:   version,
		Data:      record,
		Timestamp: time.Now(),
	}

	// Write to WAL first (durability)
	walEntry := &wal.Entry{
		ID:        recordID,
		TableID:   tableID,
		Operation: wal.OperationInsert,
		Data:      record,
		Timestamp: time.Now(),
	}

	if err := sm.walManager.Append(ctx, walEntry); err != nil {
		return fmt.Errorf("failed to write to WAL: %w", err)
	}

	// Insert into memtable
	if err := memTable.Insert(recordID, versionedRecord); err != nil {
		return fmt.Errorf("failed to insert into memtable: %w", err)
	}

	// Update metrics
	sm.updateMetrics(func(m *StorageMetrics) {
		m.IngestedRecords++
	})

	// Publish ingestion event
	if sm.publisher != nil {
		eventData := map[string]interface{}{
			"table_id":  tableID,
			"record_id": recordID,
			"size":      len(record),
		}
		sm.publishEvent(ctx, messaging.EventDataIngested, eventData)
	}

	// Check if memtable needs flushing
	if memTable.Size() >= sm.config.MemtableFlushThreshold {
		return sm.flushMemtable(ctx, tableID)
	}

	return nil
}

// QueryRecords queries records from the storage system
func (sm *StorageManager) QueryRecords(ctx context.Context, query *QueryRequest) (*QueryResponse, error) {
	sm.mu.RLock()
	if !sm.running {
		sm.mu.RUnlock()
		return nil, fmt.Errorf("storage manager is not running")
	}
	sm.mu.RUnlock()

	startTime := time.Now()
	defer func() {
		sm.updateMetrics(func(m *StorageMetrics) {
			m.QueryLatency = time.Since(startTime)
		})
	}()

	// Get table schema
	tableSchema, err := sm.catalog.GetTableSchema(query.TableID)
	if err != nil {
		return nil, fmt.Errorf("failed to get table schema: %w", err)
	}

	// Query memtable first (latest data)
	var memtableResults []*mvcc.VersionedRecord
	if memTable, exists := sm.memtables[query.TableID]; exists {
		memtableResults = memTable.Scan(query.StartKey, query.EndKey, query.Limit)
	}

	// Query persistent storage
	persistentResults, err := sm.queryPersistentStorage(ctx, query, tableSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to query persistent storage: %w", err)
	}

	// Merge results using MVCC resolver
	mergedResults := sm.mvccResolver.MergeResults(memtableResults, persistentResults)

	// Apply filters and projections
	filteredResults := sm.applyFilters(mergedResults, query.Filters)
	projectedResults := sm.applyProjections(filteredResults, query.Projections)

	response := &QueryResponse{
		Records:   projectedResults,
		Total:     len(projectedResults),
		TableID:   query.TableID,
		Timestamp: time.Now(),
	}

	// Publish query event
	if sm.publisher != nil {
		eventData := map[string]interface{}{
			"table_id":     query.TableID,
			"result_count": len(projectedResults),
			"latency_ms":   time.Since(startTime).Milliseconds(),
		}
		sm.publishEvent(ctx, messaging.EventQueryExecuted, eventData)
	}

	return response, nil
}

// CreateTable creates a new table with the given schema
func (sm *StorageManager) CreateTable(ctx context.Context, tableSchema *schema.TableSchema) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Validate schema
	if err := tableSchema.Validate(); err != nil {
		return fmt.Errorf("invalid table schema: %w", err)
	}

	// Register schema
	if err := sm.schemaRegistry.RegisterSchema(tableSchema); err != nil {
		return fmt.Errorf("failed to register schema: %w", err)
	}

	// Create table in catalog
	if err := sm.catalog.CreateTable(tableSchema); err != nil {
		return fmt.Errorf("failed to create table in catalog: %w", err)
	}

	// Create memtable for the table
	memTable := memtable.NewMemtable(sm.config.MemtableConfig)
	sm.memtables[tableSchema.Name] = memTable

	// Publish table creation event
	if sm.publisher != nil {
		eventData := map[string]interface{}{
			"table_name": tableSchema.Name,
			"columns":    len(tableSchema.Columns),
		}
		sm.publishEvent(ctx, messaging.EventTableCreated, eventData)
	}

	return nil
}

// Background processes

// backgroundFlush handles periodic memtable flushing
func (sm *StorageManager) backgroundFlush(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-sm.stopChan:
			return
		case <-sm.flushTicker.C:
			if err := sm.flushAllMemtables(ctx); err != nil {
				sm.updateMetrics(func(m *StorageMetrics) {
					m.ErrorCount++
				})
			}
		}
	}
}

// backgroundCompaction handles periodic compaction
func (sm *StorageManager) backgroundCompaction(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-sm.stopChan:
			return
		case <-sm.compactTicker.C:
			if err := sm.runCompaction(ctx); err != nil {
				sm.updateMetrics(func(m *StorageMetrics) {
					m.ErrorCount++
				})
			}
		}
	}
}

// backgroundMetrics handles periodic metrics collection
func (sm *StorageManager) backgroundMetrics(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sm.stopChan:
			return
		case <-ticker.C:
			sm.collectMetrics(ctx)
		}
	}
}

// Helper methods

// getOrCreateMemtable gets or creates a memtable for a table
func (sm *StorageManager) getOrCreateMemtable(tableID string) (*memtable.Memtable, error) {
	sm.mu.RLock()
	memTable, exists := sm.memtables[tableID]
	sm.mu.RUnlock()

	if exists {
		return memTable, nil
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Double-check after acquiring write lock
	if memTable, exists := sm.memtables[tableID]; exists {
		return memTable, nil
	}

	// Create new memtable
	memTable = memtable.NewMemtable(sm.config.MemtableConfig)
	sm.memtables[tableID] = memTable

	return memTable, nil
}

// flushMemtable flushes a specific memtable to persistent storage
func (sm *StorageManager) flushMemtable(ctx context.Context, tableID string) error {
	sm.mu.Lock()
	memTable, exists := sm.memtables[tableID]
	if !exists {
		sm.mu.Unlock()
		return nil
	}

	// Create new memtable and replace current one
	newMemtable := memtable.NewMemtable(sm.config.MemtableConfig)
	sm.memtables[tableID] = newMemtable
	sm.mu.Unlock()

	// Flush old memtable to Parquet
	records := memTable.GetAllRecords()
	if len(records) == 0 {
		return nil
	}

	// Get table schema
	tableSchema, err := sm.catalog.GetTableSchema(tableID)
	if err != nil {
		return fmt.Errorf("failed to get table schema: %w", err)
	}

	// Write to Parquet
	filename := fmt.Sprintf("%s_%d.parquet", tableID, time.Now().Unix())
	if err := sm.parquetWriter.WriteRecords(ctx, filename, records, tableSchema); err != nil {
		return fmt.Errorf("failed to write Parquet file: %w", err)
	}

	// Update indexes
	if err := sm.indexManager.UpdateIndexes(tableID, records); err != nil {
		return fmt.Errorf("failed to update indexes: %w", err)
	}

	// Update metrics
	sm.updateMetrics(func(m *StorageMetrics) {
		m.MemtableFlushes++
		m.ProcessedRecords += int64(len(records))
	})

	return nil
}

// flushAllMemtables flushes all memtables
func (sm *StorageManager) flushAllMemtables(ctx context.Context) error {
	sm.mu.RLock()
	tableIDs := make([]string, 0, len(sm.memtables))
	for tableID := range sm.memtables {
		tableIDs = append(tableIDs, tableID)
	}
	sm.mu.RUnlock()

	for _, tableID := range tableIDs {
		if err := sm.flushMemtable(ctx, tableID); err != nil {
			return fmt.Errorf("failed to flush memtable for table %s: %w", tableID, err)
		}
	}

	return nil
}

// runCompaction runs compaction on storage files
func (sm *StorageManager) runCompaction(ctx context.Context) error {
	tables, err := sm.catalog.ListTables()
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}

	for _, table := range tables {
		if err := sm.compactor.CompactTable(ctx, table.Name); err != nil {
			return fmt.Errorf("failed to compact table %s: %w", table.Name, err)
		}
	}

	sm.updateMetrics(func(m *StorageMetrics) {
		m.CompactedFiles++
	})

	return nil
}

// collectMetrics collects and publishes metrics
func (sm *StorageManager) collectMetrics(ctx context.Context) {
	if sm.publisher == nil {
		return
	}

	sm.metrics.mu.RLock()
	metricsData := map[string]interface{}{
		"ingested_records":    sm.metrics.IngestedRecords,
		"processed_records":   sm.metrics.ProcessedRecords,
		"compacted_files":     sm.metrics.CompactedFiles,
		"memtable_flushes":    sm.metrics.MemtableFlushes,
		"query_latency_ms":    sm.metrics.QueryLatency.Milliseconds(),
		"storage_utilization": sm.metrics.StorageUtilization,
		"error_count":         sm.metrics.ErrorCount,
	}
	sm.metrics.mu.RUnlock()

	sm.publishEvent(ctx, messaging.EventMetricsUpdated, metricsData)
}

// updateMetrics safely updates metrics
func (sm *StorageManager) updateMetrics(fn func(*StorageMetrics)) {
	sm.metrics.mu.Lock()
	defer sm.metrics.mu.Unlock()
	fn(sm.metrics)
}

// publishEvent publishes an event
func (sm *StorageManager) publishEvent(ctx context.Context, eventType messaging.EventType, data map[string]interface{}) {
	if sm.publisher == nil {
		return
	}

	event := &messaging.Event{
		Type:      eventType,
		Source:    "storage_manager",
		Data:      data,
		Timestamp: time.Now(),
		TraceID:   common.GetTraceID(ctx),
	}

	message, err := event.ToMessage()
	if err != nil {
		return
	}

	sm.publisher.Publish(ctx, string(eventType), message)
}

// queryPersistentStorage queries persistent Parquet files
func (sm *StorageManager) queryPersistentStorage(ctx context.Context, query *QueryRequest, tableSchema *schema.TableSchema) ([]*mvcc.VersionedRecord, error) {
	// List Parquet files for the table
	files, err := sm.blockStorage.ListFiles(ctx, query.TableID)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	var allResults []*mvcc.VersionedRecord

	// Query each file
	for _, file := range files {
		records, err := sm.parquetReader.ReadRecords(ctx, file, query.StartKey, query.EndKey)
		if err != nil {
			return nil, fmt.Errorf("failed to read Parquet file %s: %w", file, err)
		}

		allResults = append(allResults, records...)
	}

	return allResults, nil
}

// applyFilters applies query filters to results
func (sm *StorageManager) applyFilters(records []*mvcc.VersionedRecord, filters map[string]interface{}) []*mvcc.VersionedRecord {
	if len(filters) == 0 {
		return records
	}

	var filtered []*mvcc.VersionedRecord
	for _, record := range records {
		if sm.matchesFilters(record, filters) {
			filtered = append(filtered, record)
		}
	}

	return filtered
}

// applyProjections applies projections to results
func (sm *StorageManager) applyProjections(records []*mvcc.VersionedRecord, projections []string) []map[string]interface{} {
	result := make([]map[string]interface{}, len(records))

	for i, record := range records {
		projected := make(map[string]interface{})

		if len(projections) == 0 {
			// No projections, return all fields
			projected = record.Data
		} else {
			// Apply projections
			for _, field := range projections {
				if value, exists := record.Data[field]; exists {
					projected[field] = value
				}
			}
		}

		result[i] = projected
	}

	return result
}

// matchesFilters checks if a record matches the given filters
func (sm *StorageManager) matchesFilters(record *mvcc.VersionedRecord, filters map[string]interface{}) bool {
	for field, expectedValue := range filters {
		if actualValue, exists := record.Data[field]; !exists || actualValue != expectedValue {
			return false
		}
	}
	return true
}

// Request/Response types

// QueryRequest represents a query request
type QueryRequest struct {
	TableID     string                 `json:"table_id"`
	StartKey    string                 `json:"start_key,omitempty"`
	EndKey      string                 `json:"end_key,omitempty"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
	Projections []string               `json:"projections,omitempty"`
	Limit       int                    `json:"limit,omitempty"`
}

// QueryResponse represents a query response
type QueryResponse struct {
	Records   []map[string]interface{} `json:"records"`
	Total     int                      `json:"total"`
	TableID   string                   `json:"table_id"`
	Timestamp time.Time                `json:"timestamp"`
}
