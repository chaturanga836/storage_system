package storage

import (
	"storage-engine/internal/common"
	"time"
)

// Record represents a data record in the storage system
type Record struct {
	ID        common.RecordID            `json:"id"`
	Data      map[string]interface{}     `json:"data"`
	Schema    common.SchemaID           `json:"schema"`
	Timestamp common.Timestamp          `json:"timestamp"`
	Version   int64                     `json:"version"`
	Metadata  map[string]string         `json:"metadata,omitempty"`
}

// Location represents the physical location of data
type Location struct {
	FileID     common.FileID `json:"file_id"`
	Offset     int64         `json:"offset"`
	Length     int64         `json:"length"`
	RowGroup   int           `json:"row_group"`
	Compressed bool          `json:"compressed"`
}

// Version represents a version of a record with MVCC support
type Version struct {
	VersionID   int64                `json:"version_id"`
	RecordID    common.RecordID     `json:"record_id"`
	Location    Location            `json:"location"`
	Timestamp   common.Timestamp    `json:"timestamp"`
	IsDeleted   bool                `json:"is_deleted"`
	Transaction string              `json:"transaction,omitempty"`
}

// FileMetadata represents metadata about a storage file
type FileMetadata struct {
	FileID      common.FileID    `json:"file_id"`
	Path        string           `json:"path"`
	Size        int64            `json:"size"`
	RecordCount int64            `json:"record_count"`
	MinKey      string           `json:"min_key"`
	MaxKey      string           `json:"max_key"`
	Schema      common.SchemaID  `json:"schema"`
	CreatedAt   common.Timestamp `json:"created_at"`
	UpdatedAt   common.Timestamp `json:"updated_at"`
	Checksum    string           `json:"checksum"`
	Compressed  bool             `json:"compressed"`
	Format      string           `json:"format"` // "parquet", "delta", etc.
}

// IndexEntry represents an entry in an index
type IndexEntry struct {
	Key      string          `json:"key"`
	RecordID common.RecordID `json:"record_id"`
	Location Location        `json:"location"`
	Version  int64           `json:"version"`
}

// QueryFilter represents a filter condition for queries
type QueryFilter struct {
	Column   string      `json:"column"`
	Operator string      `json:"operator"` // "eq", "ne", "gt", "gte", "lt", "lte", "in", "not_in"
	Value    interface{} `json:"value"`
}

// QueryRequest represents a query request
type QueryRequest struct {
	TenantID    common.TenantID     `json:"tenant_id"`
	Schema      common.SchemaID     `json:"schema"`
	Filters     []QueryFilter       `json:"filters"`
	Projection  []string            `json:"projection"`
	OrderBy     []string            `json:"order_by"`
	Limit       int                 `json:"limit"`
	Offset      int                 `json:"offset"`
	TimeRange   *TimeRange          `json:"time_range,omitempty"`
}

// TimeRange represents a time range filter
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// QueryResult represents the result of a query
type QueryResult struct {
	Records     []Record `json:"records"`
	TotalCount  int64    `json:"total_count"`
	HasMore     bool     `json:"has_more"`
	QueryTime   time.Duration `json:"query_time"`
	ScannedRows int64    `json:"scanned_rows"`
}

// CompactionJob represents a compaction job
type CompactionJob struct {
	JobID       string              `json:"job_id"`
	TenantID    common.TenantID     `json:"tenant_id"`
	InputFiles  []common.FileID     `json:"input_files"`
	OutputFile  common.FileID       `json:"output_file"`
	Strategy    string              `json:"strategy"`
	StartTime   common.Timestamp    `json:"start_time"`
	EndTime     *common.Timestamp   `json:"end_time,omitempty"`
	Status      string              `json:"status"` // "pending", "running", "completed", "failed"
	BytesRead   int64               `json:"bytes_read"`
	BytesWritten int64              `json:"bytes_written"`
	RecordsProcessed int64          `json:"records_processed"`
}

// Stats represents storage statistics
type Stats struct {
	TotalFiles      int64 `json:"total_files"`
	TotalSize       int64 `json:"total_size"`
	TotalRecords    int64 `json:"total_records"`
	CompressedSize  int64 `json:"compressed_size"`
	IndexSize       int64 `json:"index_size"`
	LastCompaction  *common.Timestamp `json:"last_compaction,omitempty"`
	LastUpdate      common.Timestamp `json:"last_update"`
}
