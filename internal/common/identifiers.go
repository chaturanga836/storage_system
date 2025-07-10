package common

import (
	"fmt"
	"time"
)

// TenantID represents a unique tenant identifier
type TenantID string

// EntityID represents a unique entity identifier within a tenant
type EntityID string

// RecordID represents a unique record identifier
type RecordID struct {
	TenantID TenantID `json:"tenant_id"`
	EntityID EntityID `json:"entity_id"`
	Version  int64    `json:"version"`
}

// String returns a string representation of RecordID
func (r RecordID) String() string {
	return fmt.Sprintf("%s/%s#%d", r.TenantID, r.EntityID, r.Version)
}

// Location represents a physical location of data
type Location struct {
	FilePath string `json:"file_path"`
	Offset   int64  `json:"offset"`
	Length   int64  `json:"length"`
}

// Timestamp represents a point in time
type Timestamp time.Time

// Now returns the current timestamp
func Now() Timestamp {
	return Timestamp(time.Now())
}

// Unix returns the Unix timestamp
func (t Timestamp) Unix() int64 {
	return time.Time(t).Unix()
}

// String returns a string representation of the timestamp
func (t Timestamp) String() string {
	return time.Time(t).Format(time.RFC3339)
}

// SchemaID represents a schema identifier
type SchemaID struct {
	TenantID TenantID `json:"tenant_id"`
	Name     string   `json:"name"`
	Version  int      `json:"version"`
}

// String returns a string representation of SchemaID
func (s SchemaID) String() string {
	return fmt.Sprintf("%s/%s:v%d", s.TenantID, s.Name, s.Version)
}

// FileID represents a unique file identifier
type FileID string

// SegmentID represents a WAL segment identifier
type SegmentID string

// IndexID represents an index identifier
type IndexID string

// Constants for system limits
const (
	MaxTenantIDLength = 128
	MaxEntityIDLength = 256
	MaxSchemaNameLength = 128
	MaxBatchSize = 10000
	DefaultTimeout = 30 * time.Second
)
