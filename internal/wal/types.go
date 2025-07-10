package wal

import (
	"storage-engine/internal/common"
	"time"
)

// EntryType represents the type of WAL entry
type EntryType int

const (
	EntryTypeInsert EntryType = iota + 1
	EntryTypeUpdate
	EntryTypeDelete
	EntryTypeCheckpoint
	EntryTypeTransaction
)

// String returns the string representation of EntryType
func (t EntryType) String() string {
	switch t {
	case EntryTypeInsert:
		return "INSERT"
	case EntryTypeUpdate:
		return "UPDATE"
	case EntryTypeDelete:
		return "DELETE"
	case EntryTypeCheckpoint:
		return "CHECKPOINT"
	case EntryTypeTransaction:
		return "TRANSACTION"
	default:
		return "UNKNOWN"
	}
}

// Entry represents a single entry in the Write-Ahead Log
type Entry struct {
	ID          int64                  `json:"id"`
	Type        EntryType              `json:"type"`
	TenantID    common.TenantID        `json:"tenant_id"`
	RecordID    common.RecordID        `json:"record_id"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Schema      common.SchemaID        `json:"schema"`
	Timestamp   common.Timestamp       `json:"timestamp"`
	Transaction string                 `json:"transaction,omitempty"`
	Checksum    string                 `json:"checksum"`
	Size        int                    `json:"size"`
}

// Segment represents a WAL segment file
type Segment struct {
	ID           common.SegmentID   `json:"id"`
	Path         string             `json:"path"`
	Size         int64              `json:"size"`
	EntryCount   int64              `json:"entry_count"`
	FirstEntry   int64              `json:"first_entry"`
	LastEntry    int64              `json:"last_entry"`
	CreatedAt    common.Timestamp   `json:"created_at"`
	ClosedAt     *common.Timestamp  `json:"closed_at,omitempty"`
	Checksum     string             `json:"checksum"`
	IsClosed     bool               `json:"is_closed"`
	IsCheckpoint bool               `json:"is_checkpoint"`
}

// SegmentReader represents metadata for reading a segment
type SegmentReader struct {
	Segment *Segment `json:"segment"`
	Offset  int64    `json:"offset"`
	EOF     bool     `json:"eof"`
}

// Checkpoint represents a checkpoint in the WAL
type Checkpoint struct {
	ID            string             `json:"id"`
	SegmentID     common.SegmentID   `json:"segment_id"`
	EntryID       int64              `json:"entry_id"`
	Timestamp     common.Timestamp   `json:"timestamp"`
	MemtableCount int                `json:"memtable_count"`
	ProcessedTo   int64              `json:"processed_to"`
	Metadata      map[string]string  `json:"metadata,omitempty"`
}

// ReplayPosition represents the position during WAL replay
type ReplayPosition struct {
	SegmentID common.SegmentID `json:"segment_id"`
	EntryID   int64            `json:"entry_id"`
	Offset    int64            `json:"offset"`
	Timestamp common.Timestamp `json:"timestamp"`
}

// ReplayResult represents the result of WAL replay
type ReplayResult struct {
	StartPosition     ReplayPosition   `json:"start_position"`
	EndPosition       ReplayPosition   `json:"end_position"`
	EntriesReplayed   int64            `json:"entries_replayed"`
	BytesReplayed     int64            `json:"bytes_replayed"`
	Duration          time.Duration    `json:"duration"`
	LastCheckpoint    *Checkpoint      `json:"last_checkpoint,omitempty"`
	ErrorCount        int              `json:"error_count"`
	SkippedEntries    int64            `json:"skipped_entries"`
}

// Config represents WAL configuration
type Config struct {
	Path            string        `json:"path"`
	SegmentSize     int64         `json:"segment_size"`
	RetentionPeriod time.Duration `json:"retention_period"`
	SyncInterval    time.Duration `json:"sync_interval"`
	BufferSize      int           `json:"buffer_size"`
	CompressionType string        `json:"compression_type"`
	EnableChecksum  bool          `json:"enable_checksum"`
}

// Stats represents WAL statistics
type Stats struct {
	TotalSegments     int                `json:"total_segments"`
	ActiveSegments    int                `json:"active_segments"`
	ClosedSegments    int                `json:"closed_segments"`
	TotalSize         int64              `json:"total_size"`
	TotalEntries      int64              `json:"total_entries"`
	LastEntry         int64              `json:"last_entry"`
	LastCheckpoint    *Checkpoint        `json:"last_checkpoint,omitempty"`
	OldestSegment     *common.Timestamp  `json:"oldest_segment,omitempty"`
	NewestSegment     *common.Timestamp  `json:"newest_segment,omitempty"`
}
