package parquet

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/apache/arrow/go/v14/arrow"
	"github.com/apache/arrow/go/v14/arrow/array"
	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/apache/arrow/go/v14/parquet"
	"github.com/apache/arrow/go/v14/parquet/compress"
	"github.com/apache/arrow/go/v14/parquet/pqarrow"

	"storage-engine/internal/common"
	"storage-engine/internal/schema"
	"storage-engine/internal/storage"
	"storage-engine/internal/storage/block"
)

// Writer handles writing records to Parquet files
type Writer struct {
	storage      block.Storage
	schema       *schema.Schema
	arrowSchema  *arrow.Schema
	compression  compress.Compression
	rowGroupSize int64
	pageSize     int64
	allocator    memory.Allocator
}

// Config holds configuration for the Parquet writer
type Config struct {
	Compression  compress.Compression
	RowGroupSize int64
	PageSize     int64
}

// NewWriter creates a new Parquet writer
func NewWriter(storage block.Storage, schema *schema.Schema, config Config) (*Writer, error) {
	// Convert schema to Arrow schema
	arrowSchema, err := schema.ToArrowSchema()
	if err != nil {
		return nil, fmt.Errorf("failed to convert schema to Arrow: %w", err)
	}

	return &Writer{
		storage:      storage,
		schema:       schema,
		arrowSchema:  arrowSchema,
		compression:  config.Compression,
		rowGroupSize: config.RowGroupSize,
		pageSize:     config.PageSize,
		allocator:    memory.NewGoAllocator(),
	}, nil
}

// WriteRecords writes a batch of records to a Parquet file
func (w *Writer) WriteRecords(ctx context.Context, path string, records []*storage.Record) (*FileMetadata, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("no records to write")
	}

	// Create the output stream
	outputStream, err := w.storage.Writer(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to create output stream: %w", err)
	}
	defer outputStream.Close()

	// Create Parquet writer
	props := parquet.NewWriterProperties(
		parquet.WithCompression(w.compression),
		parquet.WithDataPageSize(w.pageSize),
		parquet.WithMaxRowGroupLength(w.rowGroupSize),
	)

	pqWriter, err := pqarrow.NewFileWriter(w.arrowSchema, outputStream, props, pqarrow.DefaultWriterProps())
	if err != nil {
		return nil, fmt.Errorf("failed to create Parquet writer: %w", err)
	}
	defer pqWriter.Close()

	// Convert records to Arrow record batch
	recordBatch, err := w.recordsToArrowBatch(records)
	if err != nil {
		return nil, fmt.Errorf("failed to convert records to Arrow batch: %w", err)
	}
	defer recordBatch.Release()

	// Write the record batch
	if err := pqWriter.Write(recordBatch); err != nil {
		return nil, fmt.Errorf("failed to write record batch: %w", err)
	}

	// Close the writer to finalize the file
	if err := pqWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close Parquet writer: %w", err)
	}

	// Gather metadata about the written file
	metadata := &FileMetadata{
		Path:             path,
		RecordCount:      int64(len(records)),
		UncompressedSize: 0, // Would need to calculate this
		CompressedSize:   0, // Would need to calculate this
		CreatedAt:        time.Now(),
		Schema:           w.schema,
		RowGroups:        1, // Simplified - in reality would depend on data size
		Compression:      w.compression,
	}

	// Calculate min/max values for each column (simplified)
	if len(records) > 0 {
		metadata.MinValues = make(map[string]interface{})
		metadata.MaxValues = make(map[string]interface{})

		// This is a simplified implementation
		// In practice, you'd scan through all records to find actual min/max
		firstRecord := records[0]
		lastRecord := records[len(records)-1]

		metadata.MinValues["id"] = firstRecord.ID.String()
		metadata.MaxValues["id"] = lastRecord.ID.String()
		metadata.MinValues["version"] = firstRecord.Version
		metadata.MaxValues["version"] = lastRecord.Version
		metadata.MinValues["timestamp"] = firstRecord.Timestamp
		metadata.MaxValues["timestamp"] = lastRecord.Timestamp
	}

	return metadata, nil
}

// WriteMemtable writes an entire memtable to a Parquet file
func (w *Writer) WriteMemtable(ctx context.Context, memtable MemtableReader, outputPath string) (*FileMetadata, error) {
	var records []*storage.Record

	// Read all records from the memtable
	iterator := memtable.Iterator()
	defer iterator.Close()

	for {
		record, err := iterator.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read from memtable: %w", err)
		}
		if record == nil {
			break
		}

		records = append(records, record)
	}

	return w.WriteRecords(ctx, outputPath, records)
}

// AppendToFile appends records to an existing Parquet file (creates new row group)
func (w *Writer) AppendToFile(ctx context.Context, existingPath string, records []*storage.Record) error {
	// For now, we'll implement this as reading the existing file and rewriting with new data
	// In a production system, you might want to implement true append functionality

	// Read existing records
	reader, err := NewReader(w.storage, w.schema)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}

	existingRecords, err := reader.ReadAllRecords(ctx, existingPath)
	if err != nil {
		return fmt.Errorf("failed to read existing records: %w", err)
	}

	// Combine existing and new records
	allRecords := append(existingRecords, records...)

	// Write back to the file
	_, err = w.WriteRecords(ctx, existingPath, allRecords)
	return err
}

// CompactFiles merges multiple Parquet files into one
func (w *Writer) CompactFiles(ctx context.Context, inputPaths []string, outputPath string) (*FileMetadata, error) {
	reader, err := NewReader(w.storage, w.schema)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader: %w", err)
	}

	var allRecords []*storage.Record

	// Read records from all input files
	for _, inputPath := range inputPaths {
		records, err := reader.ReadAllRecords(ctx, inputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read records from %s: %w", inputPath, err)
		}
		allRecords = append(allRecords, records...)
	}

	// Sort records if needed (implementation depends on your sorting requirements)
	// sortRecords(allRecords)

	// Write compacted file
	return w.WriteRecords(ctx, outputPath, allRecords)
}

// GetEstimatedSize estimates the size of records when written to Parquet
func (w *Writer) GetEstimatedSize(records []*storage.Record) int64 {
	if len(records) == 0 {
		return 0
	}

	// Rough estimation based on record size and compression ratio
	totalRecordSize := int64(0)
	for _, record := range records {
		// Estimate record size based on its contents
		recordSize := int64(len(record.ID.String())) +
			int64(8) + // Version (int64)
			int64(8) + // Timestamp
			int64(len(record.Schema.String()))

		// Add estimated size of Data map
		for key, value := range record.Data {
			recordSize += int64(len(key))
			if str, ok := value.(string); ok {
				recordSize += int64(len(str))
			} else {
				recordSize += 32 // Rough estimate for non-string values
			}
		}

		totalRecordSize += recordSize
	}

	// Assume 70% compression ratio for Parquet with compression
	compressionRatio := 0.7
	if w.compression == compress.Codecs.Uncompressed {
		compressionRatio = 1.0
	}

	return int64(float64(totalRecordSize) * compressionRatio)
}

// ValidateSchema checks if the records match the expected schema
func (w *Writer) ValidateSchema(records []*storage.Record) error {
	if len(records) == 0 {
		return nil
	}

	// TODO: Implement proper schema validation
	// For now, just check basic structure
	for i, record := range records {
		if record == nil {
			return fmt.Errorf("record %d is nil", i)
		}
		if record.Data == nil {
			return fmt.Errorf("record %d has nil Data field", i)
		}
	}

	return nil
}

// recordsToArrowBatch converts storage records to Arrow record batch
func (w *Writer) recordsToArrowBatch(records []*storage.Record) (arrow.Record, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("no records to convert")
	}

	// Create builders for each column
	builders := make([]array.Builder, len(w.arrowSchema.Fields()))
	for i, field := range w.arrowSchema.Fields() {
		builders[i] = array.NewBuilder(w.allocator, field.Type)
	}
	defer func() {
		for _, builder := range builders {
			builder.Release()
		}
	}()

	// Populate builders with data from records
	for _, record := range records {
		if err := w.appendRecordToBuilders(builders, record); err != nil {
			return nil, fmt.Errorf("failed to append record to builders: %w", err)
		}
	}

	// Build arrays
	arrays := make([]arrow.Array, len(builders))
	for i, builder := range builders {
		arrays[i] = builder.NewArray()
	}
	defer func() {
		for _, arr := range arrays {
			arr.Release()
		}
	}()

	// Create record batch
	return array.NewRecord(w.arrowSchema, arrays, int64(len(records))), nil
}

// appendRecordToBuilders appends a single record to the Arrow builders
func (w *Writer) appendRecordToBuilders(builders []array.Builder, record *storage.Record) error {
	// This is a simplified implementation that assumes a specific schema
	// In practice, you'd need to map your record fields to the schema fields dynamically

	fieldIndex := 0

	// ID field
	if fieldIndex < len(builders) {
		if builder, ok := builders[fieldIndex].(*array.StringBuilder); ok {
			builder.Append(record.ID.String())
		}
		fieldIndex++
	}

	// Data field (serialized as binary)
	if fieldIndex < len(builders) && record.Data != nil {
		if builder, ok := builders[fieldIndex].(*array.BinaryBuilder); ok {
			// TODO: Serialize the map to JSON or another format
			builder.Append([]byte("{}")) // Placeholder
		}
		fieldIndex++
	}

	// Timestamp field (convert to uint64)
	if fieldIndex < len(builders) {
		if builder, ok := builders[fieldIndex].(*array.Uint64Builder); ok {
			builder.Append(uint64(common.Timestamp(record.Timestamp).Unix()))
		}
		fieldIndex++
	}

	// Version field (convert to int64 builder)
	if fieldIndex < len(builders) {
		if builder, ok := builders[fieldIndex].(*array.Int64Builder); ok {
			builder.Append(record.Version)
		}
		fieldIndex++
	}

	return nil
}

// MemtableReader interface for reading from memtables
type MemtableReader interface {
	Iterator() MemtableIterator
}

// MemtableIterator interface for iterating over memtable records
type MemtableIterator interface {
	Next() (*storage.Record, error)
	Close() error
}
