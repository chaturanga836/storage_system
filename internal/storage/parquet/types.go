package parquet

import (
	"time"

	"github.com/apache/arrow/go/v14/parquet/compress"
	"storage-engine/internal/schema"
)

// FileMetadata contains metadata about a Parquet file
type FileMetadata struct {
	Path             string                 `json:"path"`
	RecordCount      int64                  `json:"record_count"`
	UncompressedSize int64                  `json:"uncompressed_size"`
	CompressedSize   int64                  `json:"compressed_size"`
	RowGroups        int                    `json:"row_groups"`
	CreatedAt        time.Time              `json:"created_at"`
	CreatedBy        string                 `json:"created_by"`
	Schema           *schema.Schema         `json:"schema"`
	Compression      compress.Compression   `json:"compression"`
	MinValues        map[string]interface{} `json:"min_values"`
	MaxValues        map[string]interface{} `json:"max_values"`
	ColumnStats      map[string]*ColumnStatistics `json:"column_stats,omitempty"`
	Version          string                 `json:"version"`
	KeyValueMetadata map[string]string      `json:"key_value_metadata,omitempty"`
}

// RowGroupMetadata contains metadata about a row group within a Parquet file
type RowGroupMetadata struct {
	Index            int                    `json:"index"`
	RecordCount      int64                  `json:"record_count"`
	UncompressedSize int64                  `json:"uncompressed_size"`
	CompressedSize   int64                  `json:"compressed_size"`
	Columns          []*ColumnChunkMetadata `json:"columns"`
	MinValues        map[string]interface{} `json:"min_values"`
	MaxValues        map[string]interface{} `json:"max_values"`
}

// ColumnChunkMetadata contains metadata about a column chunk within a row group
type ColumnChunkMetadata struct {
	ColumnName       string               `json:"column_name"`
	ColumnIndex      int                  `json:"column_index"`
	DataType         string               `json:"data_type"`
	Encoding         string               `json:"encoding"`
	Compression      compress.Compression `json:"compression"`
	UncompressedSize int64                `json:"uncompressed_size"`
	CompressedSize   int64                `json:"compressed_size"`
	ValueCount       int64                `json:"value_count"`
	NullCount        int64                `json:"null_count"`
	MinValue         interface{}          `json:"min_value,omitempty"`
	MaxValue         interface{}          `json:"max_value,omitempty"`
	HasStatistics    bool                 `json:"has_statistics"`
}

// WriteResult contains the result of a write operation
type WriteResult struct {
	FilePath      string        `json:"file_path"`
	RecordsWritten int64         `json:"records_written"`
	BytesWritten   int64         `json:"bytes_written"`
	Duration       time.Duration `json:"duration"`
	Metadata       *FileMetadata `json:"metadata"`
}

// ReadResult contains the result of a read operation
type ReadResult struct {
	FilePath     string        `json:"file_path"`
	RecordsRead  int64         `json:"records_read"`
	BytesRead    int64         `json:"bytes_read"`
	Duration     time.Duration `json:"duration"`
	Metadata     *FileMetadata `json:"metadata"`
}

// CompactionResult contains the result of a compaction operation
type CompactionResult struct {
	InputFiles    []string      `json:"input_files"`
	OutputFile    string        `json:"output_file"`
	RecordsIn     int64         `json:"records_in"`
	RecordsOut    int64         `json:"records_out"`
	SizeReduction int64         `json:"size_reduction"` // Bytes saved
	Duration      time.Duration `json:"duration"`
}

// SchemaEvolution represents changes between schema versions
type SchemaEvolution struct {
	FromVersion   int                     `json:"from_version"`
	ToVersion     int                     `json:"to_version"`
	AddedColumns  []string                `json:"added_columns"`
	DroppedColumns []string               `json:"dropped_columns"`
	ChangedColumns map[string]interface{} `json:"changed_columns"`
	Compatible     bool                   `json:"compatible"`
}

// FileFormat represents different Parquet file format options
type FileFormat struct {
	Version      string               `json:"version"`      // Parquet format version
	Compression  compress.Compression `json:"compression"`  // Compression algorithm
	PageSize     int64                `json:"page_size"`    // Page size in bytes
	RowGroupSize int64                `json:"row_group_size"` // Row group size in bytes
	EnableStats  bool                 `json:"enable_stats"` // Whether to generate statistics
	EnableDict   bool                 `json:"enable_dict"`  // Whether to enable dictionary encoding
}

// QueryPushdown represents predicate pushdown information
type QueryPushdown struct {
	Filters         []*FilterCondition `json:"filters"`
	ProjectedColumns []string          `json:"projected_columns"`
	Limit           int64             `json:"limit"`
	Offset          int64             `json:"offset"`
}

// ScanStatistics contains statistics from scanning Parquet files
type ScanStatistics struct {
	FilesScanned    int                          `json:"files_scanned"`
	RowGroupsScanned int                         `json:"row_groups_scanned"`
	RecordsScanned  int64                        `json:"records_scanned"`
	RecordsFiltered int64                        `json:"records_filtered"`
	BytesScanned    int64                        `json:"bytes_scanned"`
	ScanDuration    time.Duration                `json:"scan_duration"`
	FilterStats     map[string]*FilterStatistics `json:"filter_stats"`
}

// FilterStatistics contains statistics about filter application
type FilterStatistics struct {
	FilterType       string    `json:"filter_type"`
	RecordsEvaluated int64     `json:"records_evaluated"`
	RecordsMatched   int64     `json:"records_matched"`
	EvaluationTime   time.Duration `json:"evaluation_time"`
}

// CompressionInfo contains information about compression used
type CompressionInfo struct {
	Algorithm        compress.Compression `json:"algorithm"`
	OriginalSize     int64               `json:"original_size"`
	CompressedSize   int64               `json:"compressed_size"`
	CompressionRatio float64             `json:"compression_ratio"`
	CompressionTime  time.Duration       `json:"compression_time"`
}

// BloomFilterMetadata contains metadata about bloom filters
type BloomFilterMetadata struct {
	ColumnName       string  `json:"column_name"`
	NumHashFunctions int     `json:"num_hash_functions"`
	NumBytes         int64   `json:"num_bytes"`
	FalsePositiveRate float64 `json:"false_positive_rate"`
}

// PageHeader contains information about data pages
type PageHeader struct {
	Type                 string `json:"type"`
	UncompressedPageSize int32  `json:"uncompressed_page_size"`
	CompressedPageSize   int32  `json:"compressed_page_size"`
	CRC32                int32  `json:"crc32,omitempty"`
	NumValues            int32  `json:"num_values"`
}

// DictionaryPageHeader contains information about dictionary pages
type DictionaryPageHeader struct {
	NumValues    int32  `json:"num_values"`
	Encoding     string `json:"encoding"`
	IsSorted     bool   `json:"is_sorted"`
}

// ValidationResult contains the result of Parquet file validation
type ValidationResult struct {
	IsValid      bool     `json:"is_valid"`
	Errors       []string `json:"errors,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
	ChecksumValid bool    `json:"checksum_valid"`
	SchemaValid  bool     `json:"schema_valid"`
	MetadataValid bool    `json:"metadata_valid"`
}

// Constants for common values
const (
	// Default configuration values
	DefaultPageSize     = 1024 * 1024      // 1MB
	DefaultRowGroupSize = 128 * 1024 * 1024 // 128MB
	DefaultBatchSize    = 10000
	
	// File format versions
	ParquetVersion1_0 = "1.0"
	ParquetVersion2_0 = "2.0"
	ParquetVersion2_6 = "2.6"
	
	// Magic bytes for Parquet files
	ParquetMagicBytes = "PAR1"
)

// DefaultFileFormat returns a default file format configuration
func DefaultFileFormat() *FileFormat {
	return &FileFormat{
		Version:      ParquetVersion2_6,
		Compression:  compress.Codecs.Snappy,
		PageSize:     DefaultPageSize,
		RowGroupSize: DefaultRowGroupSize,
		EnableStats:  true,
		EnableDict:   true,
	}
}

// GetCompressionRatio calculates the compression ratio
func (fm *FileMetadata) GetCompressionRatio() float64 {
	if fm.UncompressedSize == 0 {
		return 0.0
	}
	return float64(fm.CompressedSize) / float64(fm.UncompressedSize)
}

// GetSpaceSavings calculates the space savings from compression
func (fm *FileMetadata) GetSpaceSavings() int64 {
	return fm.UncompressedSize - fm.CompressedSize
}

// GetSpaceSavingsPercent calculates the space savings percentage
func (fm *FileMetadata) GetSpaceSavingsPercent() float64 {
	if fm.UncompressedSize == 0 {
		return 0.0
	}
	return (1.0 - fm.GetCompressionRatio()) * 100.0
}

// IsEmpty returns true if the file contains no records
func (fm *FileMetadata) IsEmpty() bool {
	return fm.RecordCount == 0
}

// HasColumn returns true if the file contains the specified column
func (fm *FileMetadata) HasColumn(columnName string) bool {
	if fm.Schema == nil {
		return false
	}
	// This would need to be implemented based on your schema structure
	return fm.Schema.HasColumn(columnName)
}

// GetColumnNames returns all column names in the file
func (fm *FileMetadata) GetColumnNames() []string {
	if fm.Schema == nil {
		return nil
	}
	return fm.Schema.GetColumnNames()
}

// Validate validates the file metadata for consistency
func (fm *FileMetadata) Validate() error {
	if fm.Path == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	
	if fm.RecordCount < 0 {
		return fmt.Errorf("record count cannot be negative")
	}
	
	if fm.UncompressedSize < 0 {
		return fmt.Errorf("uncompressed size cannot be negative")
	}
	
	if fm.CompressedSize < 0 {
		return fmt.Errorf("compressed size cannot be negative")
	}
	
	if fm.RowGroups < 0 {
		return fmt.Errorf("row group count cannot be negative")
	}
	
	if fm.Schema == nil {
		return fmt.Errorf("schema cannot be nil")
	}
	
	return nil
}

// Clone creates a deep copy of the file metadata
func (fm *FileMetadata) Clone() *FileMetadata {
	clone := &FileMetadata{
		Path:             fm.Path,
		RecordCount:      fm.RecordCount,
		UncompressedSize: fm.UncompressedSize,
		CompressedSize:   fm.CompressedSize,
		RowGroups:        fm.RowGroups,
		CreatedAt:        fm.CreatedAt,
		CreatedBy:        fm.CreatedBy,
		Schema:           fm.Schema, // Note: This is a shallow copy
		Compression:      fm.Compression,
		Version:          fm.Version,
	}
	
	// Deep copy maps
	if fm.MinValues != nil {
		clone.MinValues = make(map[string]interface{})
		for k, v := range fm.MinValues {
			clone.MinValues[k] = v
		}
	}
	
	if fm.MaxValues != nil {
		clone.MaxValues = make(map[string]interface{})
		for k, v := range fm.MaxValues {
			clone.MaxValues[k] = v
		}
	}
	
	if fm.KeyValueMetadata != nil {
		clone.KeyValueMetadata = make(map[string]string)
		for k, v := range fm.KeyValueMetadata {
			clone.KeyValueMetadata[k] = v
		}
	}
	
	return clone
}
