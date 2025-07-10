package catalog

import (
	"time"
	"storage-engine/internal/storage/parquet"
)

// FileMetadata contains metadata about data files
type FileMetadata struct {
	// File identification
	Path      string    `json:"path" db:"path"`
	TenantID  string    `json:"tenant_id" db:"tenant_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Version   int       `json:"version" db:"version"`

	// File properties
	Size             int64                 `json:"size" db:"size"`
	RecordCount      int64                 `json:"record_count" db:"record_count"`
	RowGroupCount    int                   `json:"row_group_count" db:"row_group_count"`
	CompressionType  string                `json:"compression_type" db:"compression_type"`
	FormatVersion    string                `json:"format_version" db:"format_version"`
	
	// Schema information
	SchemaVersion    int                   `json:"schema_version" db:"schema_version"`
	SchemaFingerprint string               `json:"schema_fingerprint" db:"schema_fingerprint"`
	
	// Statistics
	MinValues        map[string]interface{} `json:"min_values"`
	MaxValues        map[string]interface{} `json:"max_values"`
	NullCounts       map[string]int64      `json:"null_counts"`
	
	// Lifecycle
	Status           FileStatus            `json:"status" db:"status"`
	CompactionLevel  int                   `json:"compaction_level" db:"compaction_level"`
	LastAccessed     *time.Time            `json:"last_accessed,omitempty" db:"last_accessed"`
	ExpiresAt        *time.Time            `json:"expires_at,omitempty" db:"expires_at"`
	
	// Relationships
	ParentFiles      []string              `json:"parent_files,omitempty"`
	ChildFiles       []string              `json:"child_files,omitempty"`
	
	// Custom metadata
	Tags             map[string]string     `json:"tags,omitempty"`
	Attributes       map[string]interface{} `json:"attributes,omitempty"`
}

// SchemaMetadata contains metadata about schema versions
type SchemaMetadata struct {
	TenantID      string                 `json:"tenant_id" db:"tenant_id"`
	Version       int                    `json:"version" db:"version"`
	Fingerprint   string                 `json:"fingerprint" db:"fingerprint"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	CreatedBy     string                 `json:"created_by" db:"created_by"`
	
	// Schema definition
	Fields        []*FieldMetadata       `json:"fields"`
	Description   string                 `json:"description,omitempty"`
	
	// Compatibility
	PreviousVersion *int                 `json:"previous_version,omitempty" db:"previous_version"`
	CompatibleWith  []int                `json:"compatible_with,omitempty"`
	Breaking        bool                 `json:"breaking" db:"breaking"`
	
	// Evolution tracking
	Changes       []*SchemaChange        `json:"changes,omitempty"`
	
	// Status
	Status        SchemaStatus           `json:"status" db:"status"`
	ValidatedAt   *time.Time             `json:"validated_at,omitempty" db:"validated_at"`
}

// FieldMetadata contains metadata about schema fields
type FieldMetadata struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Nullable     bool                   `json:"nullable"`
	Description  string                 `json:"description,omitempty"`
	DefaultValue interface{}            `json:"default_value,omitempty"`
	Tags         map[string]string      `json:"tags,omitempty"`
	
	// For complex types
	Children     []*FieldMetadata       `json:"children,omitempty"`
	
	// Statistics
	Cardinality  *int64                 `json:"cardinality,omitempty"`
	MinLength    *int                   `json:"min_length,omitempty"`
	MaxLength    *int                   `json:"max_length,omitempty"`
}

// SchemaChange represents a change in schema evolution
type SchemaChange struct {
	Type        ChangeType            `json:"type"`
	FieldName   string                `json:"field_name,omitempty"`
	OldValue    interface{}           `json:"old_value,omitempty"`
	NewValue    interface{}           `json:"new_value,omitempty"`
	Description string                `json:"description,omitempty"`
}

// ColumnStatistics contains statistics about a column
type ColumnStatistics struct {
	TenantID     string                 `json:"tenant_id" db:"tenant_id"`
	ColumnName   string                 `json:"column_name" db:"column_name"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
	
	// Basic statistics
	RecordCount  int64                  `json:"record_count" db:"record_count"`
	NullCount    int64                  `json:"null_count" db:"null_count"`
	DistinctCount *int64                `json:"distinct_count,omitempty" db:"distinct_count"`
	
	// Value statistics
	MinValue     interface{}            `json:"min_value,omitempty"`
	MaxValue     interface{}            `json:"max_value,omitempty"`
	AvgLength    *float64               `json:"avg_length,omitempty" db:"avg_length"`
	
	// Distribution
	Histogram    *Histogram             `json:"histogram,omitempty"`
	TopValues    []*ValueCount          `json:"top_values,omitempty"`
	
	// Quality metrics
	NullRate     float64                `json:"null_rate" db:"null_rate"`
	Cardinality  float64                `json:"cardinality" db:"cardinality"`
}

// TableStatistics contains statistics about an entire table
type TableStatistics struct {
	TenantID      string                `json:"tenant_id" db:"tenant_id"`
	UpdatedAt     time.Time             `json:"updated_at" db:"updated_at"`
	
	// File statistics
	FileCount     int                   `json:"file_count" db:"file_count"`
	TotalSize     int64                 `json:"total_size" db:"total_size"`
	AvgFileSize   int64                 `json:"avg_file_size" db:"avg_file_size"`
	
	// Record statistics
	RecordCount   int64                 `json:"record_count" db:"record_count"`
	AvgRecordSize int64                 `json:"avg_record_size" db:"avg_record_size"`
	
	// Time range
	MinTimestamp  *int64                `json:"min_timestamp,omitempty" db:"min_timestamp"`
	MaxTimestamp  *int64                `json:"max_timestamp,omitempty" db:"max_timestamp"`
	
	// Compaction statistics
	CompactionLevel0Files int          `json:"compaction_level_0_files" db:"compaction_level_0_files"`
	CompactionLevel1Files int          `json:"compaction_level_1_files" db:"compaction_level_1_files"`
	CompactionLevel2Files int          `json:"compaction_level_2_files" db:"compaction_level_2_files"`
	
	// Performance metrics
	AvgQueryTime     time.Duration      `json:"avg_query_time,omitempty"`
	AvgScanRate      int64              `json:"avg_scan_rate,omitempty"`
}

// CompactionJob represents a compaction job
type CompactionJob struct {
	ID          string            `json:"id" db:"id"`
	Files       []string          `json:"files"`
	OutputFile  string            `json:"output_file,omitempty" db:"output_file"`
	Priority    int               `json:"priority" db:"priority"`
	Status      CompactionStatus  `json:"status" db:"status"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty" db:"completed_at"`
	
	// Progress tracking
	Progress    float64           `json:"progress" db:"progress"`
	Error       string            `json:"error,omitempty" db:"error"`
	
	// Metrics
	InputSize   int64             `json:"input_size,omitempty" db:"input_size"`
	OutputSize  int64             `json:"output_size,omitempty" db:"output_size"`
	RecordsIn   int64             `json:"records_in,omitempty" db:"records_in"`
	RecordsOut  int64             `json:"records_out,omitempty" db:"records_out"`
}

// Histogram represents a histogram for column value distribution
type Histogram struct {
	Buckets   []*HistogramBucket    `json:"buckets"`
	BucketCount int                 `json:"bucket_count"`
	SampleRate  float64             `json:"sample_rate"`
}

// HistogramBucket represents a single bucket in a histogram
type HistogramBucket struct {
	LowerBound interface{}          `json:"lower_bound"`
	UpperBound interface{}          `json:"upper_bound"`
	Count      int64                `json:"count"`
	Frequency  float64              `json:"frequency"`
}

// ValueCount represents a value and its count
type ValueCount struct {
	Value interface{}              `json:"value"`
	Count int64                    `json:"count"`
}

// FileFilter represents filtering criteria for files
type FileFilter struct {
	TenantID        string                `json:"tenant_id,omitempty"`
	Status          *FileStatus           `json:"status,omitempty"`
	MinSize         *int64                `json:"min_size,omitempty"`
	MaxSize         *int64                `json:"max_size,omitempty"`
	MinRecordCount  *int64                `json:"min_record_count,omitempty"`
	MaxRecordCount  *int64                `json:"max_record_count,omitempty"`
	CompactionLevel *int                  `json:"compaction_level,omitempty"`
	CreatedAfter    *time.Time            `json:"created_after,omitempty"`
	CreatedBefore   *time.Time            `json:"created_before,omitempty"`
	Tags            map[string]string     `json:"tags,omitempty"`
	PathPattern     string                `json:"path_pattern,omitempty"`
}

// Enums

// FileStatus represents the status of a file
type FileStatus int

const (
	FileStatusActive FileStatus = iota
	FileStatusDeleted
	FileStatusArchived
	FileStatusCorrupted
	FileStatusPendingCompaction
)

// SchemaStatus represents the status of a schema
type SchemaStatus int

const (
	SchemaStatusActive SchemaStatus = iota
	SchemaStatusDeprecated
	SchemaStatusInvalid
)

// CompactionStatus represents the status of a compaction job
type CompactionStatus int

const (
	CompactionStatusPending CompactionStatus = iota
	CompactionStatusRunning
	CompactionStatusCompleted
	CompactionStatusFailed
	CompactionStatusCancelled
)

// ChangeType represents the type of schema change
type ChangeType int

const (
	ChangeTypeFieldAdded ChangeType = iota
	ChangeTypeFieldRemoved
	ChangeTypeFieldRenamed
	ChangeTypeFieldTypeChanged
	ChangeTypeFieldNullabilityChanged
	ChangeTypeFieldDefaultChanged
)

// Methods

// Validate validates file metadata
func (fm *FileMetadata) Validate() error {
	if fm.Path == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	if fm.TenantID == "" {
		return fmt.Errorf("tenant ID cannot be empty")
	}
	if fm.Size < 0 {
		return fmt.Errorf("file size cannot be negative")
	}
	if fm.RecordCount < 0 {
		return fmt.Errorf("record count cannot be negative")
	}
	return nil
}

// Clone creates a deep copy of file metadata
func (fm *FileMetadata) Clone() *FileMetadata {
	clone := *fm
	
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
	
	if fm.NullCounts != nil {
		clone.NullCounts = make(map[string]int64)
		for k, v := range fm.NullCounts {
			clone.NullCounts[k] = v
		}
	}
	
	if fm.Tags != nil {
		clone.Tags = make(map[string]string)
		for k, v := range fm.Tags {
			clone.Tags[k] = v
		}
	}
	
	if fm.Attributes != nil {
		clone.Attributes = make(map[string]interface{})
		for k, v := range fm.Attributes {
			clone.Attributes[k] = v
		}
	}
	
	// Deep copy slices
	if fm.ParentFiles != nil {
		clone.ParentFiles = make([]string, len(fm.ParentFiles))
		copy(clone.ParentFiles, fm.ParentFiles)
	}
	
	if fm.ChildFiles != nil {
		clone.ChildFiles = make([]string, len(fm.ChildFiles))
		copy(clone.ChildFiles, fm.ChildFiles)
	}
	
	return &clone
}

// Clone creates a deep copy of schema metadata
func (sm *SchemaMetadata) Clone() *SchemaMetadata {
	clone := *sm
	
	// Deep copy fields
	if sm.Fields != nil {
		clone.Fields = make([]*FieldMetadata, len(sm.Fields))
		for i, field := range sm.Fields {
			clone.Fields[i] = field.Clone()
		}
	}
	
	// Deep copy compatible versions
	if sm.CompatibleWith != nil {
		clone.CompatibleWith = make([]int, len(sm.CompatibleWith))
		copy(clone.CompatibleWith, sm.CompatibleWith)
	}
	
	// Deep copy changes
	if sm.Changes != nil {
		clone.Changes = make([]*SchemaChange, len(sm.Changes))
		for i, change := range sm.Changes {
			clone.Changes[i] = &(*change)
		}
	}
	
	return &clone
}

// Clone creates a deep copy of field metadata
func (fm *FieldMetadata) Clone() *FieldMetadata {
	clone := *fm
	
	// Deep copy tags
	if fm.Tags != nil {
		clone.Tags = make(map[string]string)
		for k, v := range fm.Tags {
			clone.Tags[k] = v
		}
	}
	
	// Deep copy children
	if fm.Children != nil {
		clone.Children = make([]*FieldMetadata, len(fm.Children))
		for i, child := range fm.Children {
			clone.Children[i] = child.Clone()
		}
	}
	
	return &clone
}

// Clone creates a deep copy of column statistics
func (cs *ColumnStatistics) Clone() *ColumnStatistics {
	clone := *cs
	
	// Deep copy histogram
	if cs.Histogram != nil {
		clone.Histogram = &Histogram{
			BucketCount: cs.Histogram.BucketCount,
			SampleRate:  cs.Histogram.SampleRate,
		}
		
		if cs.Histogram.Buckets != nil {
			clone.Histogram.Buckets = make([]*HistogramBucket, len(cs.Histogram.Buckets))
			for i, bucket := range cs.Histogram.Buckets {
				clone.Histogram.Buckets[i] = &(*bucket)
			}
		}
	}
	
	// Deep copy top values
	if cs.TopValues != nil {
		clone.TopValues = make([]*ValueCount, len(cs.TopValues))
		for i, value := range cs.TopValues {
			clone.TopValues[i] = &(*value)
		}
	}
	
	return &clone
}

// Matches checks if a file matches the filter criteria
func (ff *FileFilter) Matches(metadata *FileMetadata) bool {
	if ff.TenantID != "" && ff.TenantID != metadata.TenantID {
		return false
	}
	
	if ff.Status != nil && *ff.Status != metadata.Status {
		return false
	}
	
	if ff.MinSize != nil && metadata.Size < *ff.MinSize {
		return false
	}
	
	if ff.MaxSize != nil && metadata.Size > *ff.MaxSize {
		return false
	}
	
	if ff.MinRecordCount != nil && metadata.RecordCount < *ff.MinRecordCount {
		return false
	}
	
	if ff.MaxRecordCount != nil && metadata.RecordCount > *ff.MaxRecordCount {
		return false
	}
	
	if ff.CompactionLevel != nil && metadata.CompactionLevel != *ff.CompactionLevel {
		return false
	}
	
	if ff.CreatedAfter != nil && metadata.CreatedAt.Before(*ff.CreatedAfter) {
		return false
	}
	
	if ff.CreatedBefore != nil && metadata.CreatedAt.After(*ff.CreatedBefore) {
		return false
	}
	
	// Check tags
	if ff.Tags != nil {
		for key, value := range ff.Tags {
			if metadata.Tags == nil {
				return false
			}
			if tagValue, exists := metadata.Tags[key]; !exists || tagValue != value {
				return false
			}
		}
	}
	
	// Path pattern matching would require regex or glob matching
	// Simplified implementation for now
	if ff.PathPattern != "" {
		// This would typically use filepath.Match or regex
		// For now, just check if pattern is contained in path
		if !strings.Contains(metadata.Path, ff.PathPattern) {
			return false
		}
	}
	
	return true
}

// String methods for enums

func (fs FileStatus) String() string {
	switch fs {
	case FileStatusActive:
		return "active"
	case FileStatusDeleted:
		return "deleted"
	case FileStatusArchived:
		return "archived"
	case FileStatusCorrupted:
		return "corrupted"
	case FileStatusPendingCompaction:
		return "pending_compaction"
	default:
		return "unknown"
	}
}

func (ss SchemaStatus) String() string {
	switch ss {
	case SchemaStatusActive:
		return "active"
	case SchemaStatusDeprecated:
		return "deprecated"
	case SchemaStatusInvalid:
		return "invalid"
	default:
		return "unknown"
	}
}

func (cs CompactionStatus) String() string {
	switch cs {
	case CompactionStatusPending:
		return "pending"
	case CompactionStatusRunning:
		return "running"
	case CompactionStatusCompleted:
		return "completed"
	case CompactionStatusFailed:
		return "failed"
	case CompactionStatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

func (ct ChangeType) String() string {
	switch ct {
	case ChangeTypeFieldAdded:
		return "field_added"
	case ChangeTypeFieldRemoved:
		return "field_removed"
	case ChangeTypeFieldRenamed:
		return "field_renamed"
	case ChangeTypeFieldTypeChanged:
		return "field_type_changed"
	case ChangeTypeFieldNullabilityChanged:
		return "field_nullability_changed"
	case ChangeTypeFieldDefaultChanged:
		return "field_default_changed"
	default:
		return "unknown"
	}
}
