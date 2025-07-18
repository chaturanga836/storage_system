package ingestion

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"storage-engine/internal/config"
)

// Service handles ingestion business logic
type Service struct {
	config   *config.Config
	wal      WALManager      // Write-Ahead Log manager
	memtable MemtableManager // Memtable manager
	// Add DB connection or other dependencies here
}

// WALManager interface for WAL operations
// You should implement this interface elsewhere
// and inject a concrete implementation when creating Service
// Example: file-based WAL, in-memory WAL, etc.
type WALManager interface {
	Write(record map[string]interface{}) error
}

// MemtableManager interface for memtable operations
// You should implement this interface elsewhere
// and inject a concrete implementation when creating Service
type MemtableManager interface {
	Add(record map[string]interface{}) error
}

// NewService creates a new ingestion service
func NewService(cfg *config.Config, wal WALManager, memtable MemtableManager) *Service {
	return &Service{
		config:   cfg,
		wal:      wal,
		memtable: memtable,
	}
}

// IngestRecord ingests a single record
func (s *Service) IngestRecord(ctx context.Context, record interface{}) error {
	log.Println("📝 Ingesting single record...")

	// 1. Validate record
	if err := s.validateRecord(ctx, record); err != nil {
		log.Printf("❌ Record validation failed: %v", err)
		return err
	}

	// Cast record to map[string]interface{} for WAL/memtable
	recordData, ok := record.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid record format for WAL/memtable")
	}

	// 2. Write to WAL
	if err := s.wal.Write(recordData); err != nil {
		log.Printf("❌ WAL write failed: %v", err)
		return fmt.Errorf("wal write failed: %w", err)
	}
	log.Println("🗃️ Record written to WAL")

	// 3. Add to memtable
	if err := s.memtable.Add(recordData); err != nil {
		log.Printf("❌ Memtable add failed: %v", err)
		return fmt.Errorf("memtable add failed: %w", err)
	}
	log.Println("📋 Record added to memtable")

	// 4. Return acknowledgment
	log.Println("✅ Record ingested successfully")
	return nil
}

// validateRecord performs comprehensive validation on incoming records
func (s *Service) validateRecord(ctx context.Context, record interface{}) error {
	// Extract basic record info
	recordData, ok := record.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid record format: expected map[string]interface{}")
	}

	// 1. Extract and validate TenantID
	tenantID, err := s.validateTenantID(recordData)
	if err != nil {
		return fmt.Errorf("tenant validation failed: %w", err)
	}

	// 2. Validate authentication & authorization
	if err := s.validatePermissions(ctx, tenantID); err != nil {
		return fmt.Errorf("permission validation failed: %w", err)
	}

	// 3. Validate required fields
	if err := s.validateRequiredFields(recordData); err != nil {
		return fmt.Errorf("required fields validation failed: %w", err)
	}

	// 4. Validate data types and constraints
	if err := s.validateDataTypes(recordData); err != nil {
		return fmt.Errorf("data type validation failed: %w", err)
	}

	// 5. Validate against schema
	if err := s.validateSchema(recordData, tenantID); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	// 6. Validate business rules
	if err := s.validateBusinessRules(recordData, tenantID); err != nil {
		return fmt.Errorf("business rules validation failed: %w", err)
	}

	// 7. Validate system limits
	if err := s.validateSystemLimits(recordData, tenantID); err != nil {
		return fmt.Errorf("system limits validation failed: %w", err)
	}

	return nil
}

// IngestBatch ingests multiple records
func (s *Service) IngestBatch(ctx context.Context, records []interface{}) error {
	log.Printf("📦 Ingesting batch of %d records...", len(records))
	if len(records) == 0 {
		return fmt.Errorf("no records provided for batch ingestion")
	}

	var batchErrs []error
	for i, record := range records {
		// 1. Validate each record
		if err := s.validateRecord(ctx, record); err != nil {
			log.Printf("❌ Record %d validation failed: %v", i, err)
			batchErrs = append(batchErrs, fmt.Errorf("record %d validation failed: %w", i, err))
			continue
		}

		// Cast record to map[string]interface{} for WAL/memtable
		recordData, ok := record.(map[string]interface{})
		if !ok {
			log.Printf("❌ Record %d format invalid for WAL/memtable", i)
			batchErrs = append(batchErrs, fmt.Errorf("record %d format invalid for WAL/memtable", i))
			continue
		}

		// 2. Write to WAL
		if err := s.wal.Write(recordData); err != nil {
			log.Printf("❌ WAL write failed for record %d: %v", i, err)
			batchErrs = append(batchErrs, fmt.Errorf("wal write failed for record %d: %w", i, err))
			continue
		}

		// 3. Add to memtable
		if err := s.memtable.Add(recordData); err != nil {
			log.Printf("❌ Memtable add failed for record %d: %v", i, err)
			batchErrs = append(batchErrs, fmt.Errorf("memtable add failed for record %d: %w", i, err))
			continue
		}
	}

	if len(batchErrs) > 0 {
		return fmt.Errorf("batch ingestion completed with %d errors: %v", len(batchErrs), batchErrs)
	}

	log.Println("✅ Batch ingested successfully")
	return nil
}

// StartMemtableFlush starts the memtable flush process
func (s *Service) StartMemtableFlush(ctx context.Context) error {
	log.Println("💾 Starting memtable flush process...")
	// TODO: Implement memtable flush logic
	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check if memtables need flushing
			// Flush to Parquet files if needed
			log.Println("Checking memtables for flush...")
		}
	}
}

// Health check helpers
func (s *Service) CheckWALHealth() error {
	if s.wal == nil {
		return fmt.Errorf("WAL manager not initialized")
	}

	// Use WAL directory from config.WAL.Path
	walDir := s.config.WAL.Path
	if walDir == "" {
		walDir = "./wal" // fallback default
	}

	// 1. Check if WAL directory exists
	info, err := os.Stat(walDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("WAL directory does not exist: %s", walDir)
		}
		return fmt.Errorf("error accessing WAL directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("WAL path is not a directory: %s", walDir)
	}

	// 2. List WAL files
	files, err := os.ReadDir(walDir)
	if err != nil {
		return fmt.Errorf("error reading WAL directory: %w", err)
	}
	walFiles := []string{}
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".wal" {
			walFiles = append(walFiles, f.Name())
		}
	}
	if len(walFiles) == 0 {
		return fmt.Errorf("no WAL files found in directory: %s", walDir)
	}

	// 3. Check read access to first WAL file
	firstFile := filepath.Join(walDir, walFiles[0])
	file, err := os.Open(firstFile)
	if err != nil {
		return fmt.Errorf("cannot open WAL file for reading: %s, error: %w", firstFile, err)
	}
	defer file.Close()

	// Optionally, check file size or last modified time
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("cannot stat WAL file: %s, error: %w", firstFile, err)
	}
	if stat.Size() == 0 {
		return fmt.Errorf("WAL file is empty: %s", firstFile)
	}

	// All checks passed
	return nil
}

func (s *Service) CheckMemtableHealth() error {
	if s.memtable == nil {
		return fmt.Errorf("Memtable manager not initialized")
	}
	// Example: Add more memtable health logic here
	return nil
}

func (s *Service) CheckDBHealth() error {
	// TODO: Implement DB health check (e.g., ping, simple query)
	// return nil if healthy, error if not
	return nil
}

// --- Example WALManager and MemtableManager implementations ---

// InMemoryWAL is a simple in-memory WAL stub for demonstration
// Replace with file-based or persistent WAL in production

type InMemoryWAL struct {
	entries []map[string]interface{}
}

func NewInMemoryWAL() *InMemoryWAL {
	return &InMemoryWAL{
		entries: make([]map[string]interface{}, 0),
	}
}

func (w *InMemoryWAL) Write(record map[string]interface{}) error {
	w.entries = append(w.entries, record)
	log.Printf("[WAL] Record appended. Total entries: %d", len(w.entries))
	return nil
}

// InMemoryMemtable is a simple in-memory memtable stub for demonstration
// Replace with a more advanced structure in production

type InMemoryMemtable struct {
	entries []map[string]interface{}
}

func NewInMemoryMemtable() *InMemoryMemtable {
	return &InMemoryMemtable{
		entries: make([]map[string]interface{}, 0),
	}
}

func (m *InMemoryMemtable) Add(record map[string]interface{}) error {
	m.entries = append(m.entries, record)
	log.Printf("[Memtable] Record added. Total entries: %d", len(m.entries))
	return nil
}

// --- Example usage ---
// service := NewService(cfg, NewInMemoryWAL(), NewInMemoryMemtable())

// Validation helper methods

// validateTenantID extracts and validates the tenant ID from record data
func (s *Service) validateTenantID(recordData map[string]interface{}) (string, error) {
	tenantID, exists := recordData["tenant_id"]
	if !exists {
		return "", fmt.Errorf("tenant_id is required in record data")
	}

	tenantStr, ok := tenantID.(string)
	if !ok {
		return "", fmt.Errorf("tenant_id must be a string")
	}

	if len(tenantStr) == 0 {
		return "", fmt.Errorf("tenant_id cannot be empty")
	}

	// Additional tenant validation logic would go here
	// For example, checking if tenant exists in catalog
	return tenantStr, nil
}

// validatePermissions checks if the tenant has write permissions
func (s *Service) validatePermissions(ctx context.Context, tenantID string) error {
	// TODO: Implement actual permission checking
	// This would typically involve:
	// 1. Checking tenant status (active/inactive)
	// 2. Validating write permissions
	// 3. Checking quota limits
	log.Printf("Validating permissions for tenant: %s", tenantID)
	return nil
}

// validateRequiredFields ensures all mandatory fields are present
func (s *Service) validateRequiredFields(recordData map[string]interface{}) error {
	requiredFields := []string{
		"id",
		"timestamp",
		"data",
	}

	for _, field := range requiredFields {
		if _, exists := recordData[field]; !exists {
			return fmt.Errorf("required field '%s' is missing", field)
		}
	}

	return nil
}

// validateDataTypes checks if field types match expected schema
func (s *Service) validateDataTypes(recordData map[string]interface{}) error {
	// Example type validations
	if id, exists := recordData["id"]; exists {
		if _, ok := id.(string); !ok {
			return fmt.Errorf("field 'id' must be a string")
		}
	}

	if timestamp, exists := recordData["timestamp"]; exists {
		switch v := timestamp.(type) {
		case string:
			// Try to parse as time
			if _, err := time.Parse(time.RFC3339, v); err != nil {
				return fmt.Errorf("timestamp format is invalid: %v", err)
			}
		case int64:
			// Unix timestamp is acceptable
		default:
			return fmt.Errorf("timestamp must be a string (RFC3339) or int64 (unix)")
		}
	}

	return nil
}

// validateSchema performs comprehensive schema validation
func (s *Service) validateSchema(recordData map[string]interface{}, tenantID string) error {
	// TODO: Implement schema registry lookup and validation
	// This would involve:
	// 1. Looking up schema for the tenant
	// 2. Validating record against schema
	// 3. Checking field constraints (min/max, patterns, etc.)
	log.Printf("Validating schema for tenant: %s", tenantID)

	// Basic validation for now
	if data, exists := recordData["data"]; exists {
		if nested, ok := data.(map[string]interface{}); ok {
			return s.validateNestedData(nested)
		}
	}

	return nil
}

// validateNestedData validates nested data structures
func (s *Service) validateNestedData(data map[string]interface{}) error {
	// Implement nested validation logic
	for key, value := range data {
		if value == nil {
			log.Printf("Warning: null value for field '%s'", key)
		}
	}
	return nil
}

// validateBusinessRules applies business-specific validation logic
func (s *Service) validateBusinessRules(recordData map[string]interface{}, tenantID string) error {
	// TODO: Implement business rule validation
	// Examples:
	// 1. Check for duplicate records
	// 2. Validate referential integrity
	// 3. Apply tenant-specific business rules
	log.Printf("Validating business rules for tenant: %s", tenantID)

	// Example: Check for future timestamps
	if timestampVal, exists := recordData["timestamp"]; exists {
		if timestampStr, ok := timestampVal.(string); ok {
			if parsedTime, err := time.Parse(time.RFC3339, timestampStr); err == nil {
				if parsedTime.After(time.Now().Add(24 * time.Hour)) {
					return fmt.Errorf("timestamp cannot be more than 24 hours in the future")
				}
			}
		}
	}

	return nil
}

// validateSystemLimits checks system-level constraints
func (s *Service) validateSystemLimits(recordData map[string]interface{}, tenantID string) error {
	// TODO: Implement system limit checks
	// Examples:
	// 1. Check storage capacity
	// 2. Validate record size limits
	// 3. Check ingestion rate limits
	log.Printf("Validating system limits for tenant: %s", tenantID)

	// Example: Check record size
	recordSize := s.calculateRecordSize(recordData)
	maxRecordSize := 10 * 1024 * 1024 // 10MB limit

	if recordSize > maxRecordSize {
		return fmt.Errorf("record size (%d bytes) exceeds maximum limit (%d bytes)",
			recordSize, maxRecordSize)
	}

	return nil
}

// calculateRecordSize estimates the size of a record
func (s *Service) calculateRecordSize(recordData map[string]interface{}) int {
	// Simple estimation - in production, use more accurate serialization
	size := 0
	for key, value := range recordData {
		size += len(key)
		if str, ok := value.(string); ok {
			size += len(str)
		} else {
			size += 100 // Rough estimate for other types
		}
	}
	return size
}
