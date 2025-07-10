package parquet

import (
	"context"
	"fmt"
	"io"

	"github.com/apache/arrow/go/v14/arrow"
	"github.com/apache/arrow/go/v14/arrow/array"
	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/apache/arrow/go/v14/parquet/pqarrow"

	"storage-engine/internal/storage"
	"storage-engine/internal/storage/block"
	"storage-engine/internal/schema"
)

// Reader handles reading records from Parquet files
type Reader struct {
	storage     block.Storage
	schema      *schema.Schema
	arrowSchema *arrow.Schema
	allocator   memory.Allocator
}

// ReadOptions configures how records are read from Parquet files
type ReadOptions struct {
	Columns    []string           // Specific columns to read (column pruning)
	Filters    []*FilterCondition // Predicate pushdown filters
	Limit      int64              // Maximum number of records to read
	Offset     int64              // Number of records to skip
	BatchSize  int                // Size of batches for streaming reads
}

// FilterCondition represents a filter condition for predicate pushdown
type FilterCondition struct {
	Column   string
	Operator FilterOperator
	Value    interface{}
}

// FilterOperator defines the type of filter operation
type FilterOperator int

const (
	Equal FilterOperator = iota
	NotEqual
	LessThan
	LessThanOrEqual
	GreaterThan
	GreaterThanOrEqual
	In
	NotIn
	IsNull
	IsNotNull
)

// NewReader creates a new Parquet reader
func NewReader(storage block.Storage, schema *schema.Schema) (*Reader, error) {
	// Convert schema to Arrow schema
	arrowSchema, err := schema.ToArrowSchema()
	if err != nil {
		return nil, fmt.Errorf("failed to convert schema to Arrow: %w", err)
	}

	return &Reader{
		storage:     storage,
		schema:      schema,
		arrowSchema: arrowSchema,
		allocator:   memory.NewGoAllocator(),
	}, nil
}

// ReadAllRecords reads all records from a Parquet file
func (r *Reader) ReadAllRecords(ctx context.Context, path string) ([]*storage.Record, error) {
	options := &ReadOptions{
		BatchSize: 10000, // Default batch size
	}
	
	return r.ReadRecords(ctx, path, options)
}

// ReadRecords reads records from a Parquet file with the given options
func (r *Reader) ReadRecords(ctx context.Context, path string, options *ReadOptions) ([]*storage.Record, error) {
	// Open the file for reading
	reader, err := r.storage.Reader(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for reading: %w", err)
	}
	defer reader.Close()

	// Create Parquet file reader
	pqReader, err := pqarrow.NewFileReader(reader, pqarrow.ArrowReadProperties{}, memory.DefaultAllocator)
	if err != nil {
		return nil, fmt.Errorf("failed to create Parquet reader: %w", err)
	}

	// Determine which columns to read
	columnIndices, err := r.getColumnIndices(options.Columns)
	if err != nil {
		return nil, fmt.Errorf("failed to get column indices: %w", err)
	}

	var allRecords []*storage.Record
	recordsRead := int64(0)

	// Read row groups
	for i := 0; i < pqReader.NumRowGroups(); i++ {
		if options.Limit > 0 && recordsRead >= options.Limit {
			break
		}

		rowGroupReader, err := pqReader.RowGroup(i)
		if err != nil {
			return nil, fmt.Errorf("failed to get row group %d: %w", i, err)
		}

		// Read the row group into Arrow table
		table, err := rowGroupReader.ReadTable(ctx)
		if err != nil {
			rowGroupReader.Release()
			return nil, fmt.Errorf("failed to read row group %d: %w", i, err)
		}

		// Convert Arrow table to records
		records, err := r.arrowTableToRecords(table, columnIndices, options)
		if err != nil {
			table.Release()
			rowGroupReader.Release()
			return nil, fmt.Errorf("failed to convert Arrow table to records: %w", err)
		}

		// Apply offset and limit
		startIdx := int64(0)
		if options.Offset > recordsRead {
			startIdx = options.Offset - recordsRead
			if startIdx >= int64(len(records)) {
				recordsRead += int64(len(records))
				table.Release()
				rowGroupReader.Release()
				continue
			}
		}

		endIdx := int64(len(records))
		if options.Limit > 0 {
			remainingLimit := options.Limit - recordsRead
			if startIdx+remainingLimit < endIdx {
				endIdx = startIdx + remainingLimit
			}
		}

		if startIdx < endIdx {
			allRecords = append(allRecords, records[startIdx:endIdx]...)
			recordsRead += endIdx - startIdx
		}

		table.Release()
		rowGroupReader.Release()
	}

	return allRecords, nil
}

// ReadRecordStream returns a streaming reader for large files
func (r *Reader) ReadRecordStream(ctx context.Context, path string, options *ReadOptions) (*RecordStream, error) {
	// Open the file for reading
	reader, err := r.storage.Reader(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for reading: %w", err)
	}

	// Create Parquet file reader
	pqReader, err := pqarrow.NewFileReader(reader, pqarrow.ArrowReadProperties{}, memory.DefaultAllocator)
	if err != nil {
		reader.Close()
		return nil, fmt.Errorf("failed to create Parquet reader: %w", err)
	}

	columnIndices, err := r.getColumnIndices(options.Columns)
	if err != nil {
		reader.Close()
		pqReader.Close()
		return nil, fmt.Errorf("failed to get column indices: %w", err)
	}

	return &RecordStream{
		reader:        reader,
		pqReader:      pqReader,
		parquetReader: r,
		options:       options,
		columnIndices: columnIndices,
		currentGroup:  0,
		recordsRead:   0,
	}, nil
}

// ReadMetadata reads metadata from a Parquet file without reading the data
func (r *Reader) ReadMetadata(ctx context.Context, path string) (*FileMetadata, error) {
	// Open the file for reading
	reader, err := r.storage.Reader(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for reading: %w", err)
	}
	defer reader.Close()

	// Create Parquet file reader
	pqReader, err := pqarrow.NewFileReader(reader, pqarrow.ArrowReadProperties{}, memory.DefaultAllocator)
	if err != nil {
		return nil, fmt.Errorf("failed to create Parquet reader: %w", err)
	}
	defer pqReader.Close()

	// Get file metadata
	parquetMetadata := pqReader.ParquetReader().MetaData()

	metadata := &FileMetadata{
		Path:             path,
		RecordCount:      parquetMetadata.NumRows,
		RowGroups:        parquetMetadata.NumRowGroups,
		Schema:           r.schema,
		CreatedBy:        parquetMetadata.CreatedBy,
		UncompressedSize: 0, // Would need to calculate
		CompressedSize:   0, // Would need to calculate
	}

	// Extract min/max statistics if available
	metadata.MinValues = make(map[string]interface{})
	metadata.MaxValues = make(map[string]interface{})

	for i := 0; i < parquetMetadata.NumRowGroups; i++ {
		rowGroup := parquetMetadata.RowGroup(i)
		for j := 0; j < rowGroup.NumColumns(); j++ {
			columnChunk := rowGroup.ColumnChunk(j)
			if columnChunk.HasStatistics() {
				stats := columnChunk.Statistics()
				columnName := r.arrowSchema.Field(j).Name

				// This would need proper type handling based on column type
				if stats.HasMinMax() {
					metadata.MinValues[columnName] = stats.EncodeMin()
					metadata.MaxValues[columnName] = stats.EncodeMax()
				}
			}
		}
	}

	return metadata, nil
}

// ScanForCondition scans a file for records matching the given condition
func (r *Reader) ScanForCondition(ctx context.Context, path string, condition *FilterCondition) ([]*storage.Record, error) {
	options := &ReadOptions{
		Filters: []*FilterCondition{condition},
	}
	
	return r.ReadRecords(ctx, path, options)
}

// GetColumnStatistics returns statistics for a specific column
func (r *Reader) GetColumnStatistics(ctx context.Context, path string, columnName string) (*ColumnStatistics, error) {
	// Open the file for reading
	reader, err := r.storage.Reader(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for reading: %w", err)
	}
	defer reader.Close()

	// Create Parquet file reader
	pqReader, err := pqarrow.NewFileReader(reader, pqarrow.ArrowReadProperties{}, memory.DefaultAllocator)
	if err != nil {
		return nil, fmt.Errorf("failed to create Parquet reader: %w", err)
	}
	defer pqReader.Close()

	// Find column index
	columnIndex := -1
	for i, field := range r.arrowSchema.Fields() {
		if field.Name == columnName {
			columnIndex = i
			break
		}
	}

	if columnIndex == -1 {
		return nil, fmt.Errorf("column %s not found in schema", columnName)
	}

	stats := &ColumnStatistics{
		ColumnName: columnName,
	}

	// Aggregate statistics across all row groups
	parquetMetadata := pqReader.ParquetReader().MetaData()
	for i := 0; i < parquetMetadata.NumRowGroups; i++ {
		rowGroup := parquetMetadata.RowGroup(i)
		if columnIndex < rowGroup.NumColumns() {
			columnChunk := rowGroup.ColumnChunk(columnIndex)
			if columnChunk.HasStatistics() {
				chunkStats := columnChunk.Statistics()
				stats.NullCount += chunkStats.NullCount()
				stats.ValueCount += chunkStats.NumValues()

				// Update min/max values (simplified)
				if chunkStats.HasMinMax() {
					if stats.MinValue == nil {
						stats.MinValue = chunkStats.EncodeMin()
						stats.MaxValue = chunkStats.EncodeMax()
					}
					// Additional logic would be needed to properly compare min/max values
				}
			}
		}
	}

	return stats, nil
}

// Helper methods

func (r *Reader) getColumnIndices(columnNames []string) ([]int, error) {
	if len(columnNames) == 0 {
		// Return all columns
		indices := make([]int, len(r.arrowSchema.Fields()))
		for i := range indices {
			indices[i] = i
		}
		return indices, nil
	}

	var indices []int
	for _, name := range columnNames {
		index := -1
		for i, field := range r.arrowSchema.Fields() {
			if field.Name == name {
				index = i
				break
			}
		}
		if index == -1 {
			return nil, fmt.Errorf("column %s not found in schema", name)
		}
		indices = append(indices, index)
	}

	return indices, nil
}

func (r *Reader) arrowTableToRecords(table arrow.Table, columnIndices []int, options *ReadOptions) ([]*storage.Record, error) {
	if table.NumRows() == 0 {
		return nil, nil
	}

	var records []*storage.Record

	// Create a record reader
	reader := array.NewTableReader(table, 0)
	defer reader.Release()

	for reader.Next() {
		record := reader.Record()
		
		// Convert Arrow record to storage records
		batchRecords, err := r.arrowRecordToStorageRecords(record, columnIndices)
		if err != nil {
			return nil, fmt.Errorf("failed to convert Arrow record: %w", err)
		}

		// Apply filters
		for _, storageRecord := range batchRecords {
			if r.applyFilters(storageRecord, options.Filters) {
				records = append(records, storageRecord)
			}
		}
	}

	return records, nil
}

func (r *Reader) arrowRecordToStorageRecords(record arrow.Record, columnIndices []int) ([]*storage.Record, error) {
	numRows := int(record.NumRows())
	records := make([]*storage.Record, numRows)

	for i := 0; i < numRows; i++ {
		storageRecord := &storage.Record{}

		// Extract values from each column
		for j, colIndex := range columnIndices {
			if colIndex >= int(record.NumCols()) {
				continue
			}

			column := record.Column(colIndex)
			fieldName := r.arrowSchema.Field(colIndex).Name

			// Extract value based on column type (simplified)
			switch fieldName {
			case "tenant_id":
				if stringArray, ok := column.(*array.String); ok && i < stringArray.Len() {
					storageRecord.TenantID = stringArray.Value(i)
				}
			case "entity_id":
				if stringArray, ok := column.(*array.String); ok && i < stringArray.Len() {
					storageRecord.EntityID = stringArray.Value(i)
				}
			case "version":
				if uint64Array, ok := column.(*array.Uint64); ok && i < uint64Array.Len() {
					storageRecord.Version = uint64Array.Value(i)
				}
			case "timestamp":
				if uint64Array, ok := column.(*array.Uint64); ok && i < uint64Array.Len() {
					storageRecord.Timestamp = uint64Array.Value(i)
				}
			case "data":
				if binaryArray, ok := column.(*array.Binary); ok && i < binaryArray.Len() {
					storageRecord.Data = binaryArray.Value(i)
				}
			}
		}

		records[i] = storageRecord
	}

	return records, nil
}

func (r *Reader) applyFilters(record *storage.Record, filters []*FilterCondition) bool {
	if len(filters) == 0 {
		return true
	}

	for _, filter := range filters {
		if !r.evaluateFilter(record, filter) {
			return false
		}
	}

	return true
}

func (r *Reader) evaluateFilter(record *storage.Record, filter *FilterCondition) bool {
	var recordValue interface{}

	// Get the value from the record based on column name
	switch filter.Column {
	case "tenant_id":
		recordValue = record.TenantID
	case "entity_id":
		recordValue = record.EntityID
	case "version":
		recordValue = record.Version
	case "timestamp":
		recordValue = record.Timestamp
	default:
		return true // Unknown column, skip filter
	}

	// Apply the filter operation
	switch filter.Operator {
	case Equal:
		return recordValue == filter.Value
	case NotEqual:
		return recordValue != filter.Value
	case LessThan:
		return compareValues(recordValue, filter.Value) < 0
	case LessThanOrEqual:
		return compareValues(recordValue, filter.Value) <= 0
	case GreaterThan:
		return compareValues(recordValue, filter.Value) > 0
	case GreaterThanOrEqual:
		return compareValues(recordValue, filter.Value) >= 0
	// Add more operators as needed
	default:
		return true
	}
}

// compareValues compares two values (simplified implementation)
func compareValues(a, b interface{}) int {
	// This is a very simplified comparison
	// In practice, you'd need proper type-aware comparison
	switch va := a.(type) {
	case string:
		if vb, ok := b.(string); ok {
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
	case uint64:
		if vb, ok := b.(uint64); ok {
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
	}
	return 0
}

// RecordStream provides streaming access to Parquet records
type RecordStream struct {
	reader        io.ReadCloser
	pqReader      *pqarrow.FileReader
	parquetReader *Reader
	options       *ReadOptions
	columnIndices []int
	currentGroup  int
	recordsRead   int64
	closed        bool
}

// Next returns the next batch of records
func (rs *RecordStream) Next() ([]*storage.Record, error) {
	if rs.closed {
		return nil, io.EOF
	}

	if rs.options.Limit > 0 && rs.recordsRead >= rs.options.Limit {
		return nil, io.EOF
	}

	if rs.currentGroup >= rs.pqReader.NumRowGroups() {
		return nil, io.EOF
	}

	rowGroupReader, err := rs.pqReader.RowGroup(rs.currentGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to get row group %d: %w", rs.currentGroup, err)
	}
	defer rowGroupReader.Release()

	table, err := rowGroupReader.ReadTable(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to read row group %d: %w", rs.currentGroup, err)
	}
	defer table.Release()

	records, err := rs.parquetReader.arrowTableToRecords(table, rs.columnIndices, rs.options)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Arrow table to records: %w", err)
	}

	rs.currentGroup++
	rs.recordsRead += int64(len(records))

	return records, nil
}

// Close closes the record stream
func (rs *RecordStream) Close() error {
	if rs.closed {
		return nil
	}

	rs.closed = true
	rs.pqReader.Close()
	return rs.reader.Close()
}

// ColumnStatistics contains statistics about a column
type ColumnStatistics struct {
	ColumnName string
	ValueCount int64
	NullCount  int64
	MinValue   interface{}
	MaxValue   interface{}
}
