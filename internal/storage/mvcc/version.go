package mvcc

import (
	"encoding/binary"
	"fmt"
	"time"
)

// VersionMetadata contains metadata about a version
type VersionMetadata struct {
	Version     uint64    `json:"version"`
	Timestamp   time.Time `json:"timestamp"`
	TxnID       uint64    `json:"txn_id"`
	Deleted     bool      `json:"deleted"`
	Size        uint32    `json:"size"`
	Checksum    uint32    `json:"checksum"`
	CompactedAt time.Time `json:"compacted_at,omitempty"`
}

// VersionHistory maintains a history of all version operations
type VersionHistory struct {
	versions []VersionMetadata
	capacity int
}

// NewVersionHistory creates a new version history with the specified capacity
func NewVersionHistory(capacity int) *VersionHistory {
	return &VersionHistory{
		versions: make([]VersionMetadata, 0, capacity),
		capacity: capacity,
	}
}

// AddVersion adds a new version to the history
func (vh *VersionHistory) AddVersion(metadata VersionMetadata) {
	if len(vh.versions) >= vh.capacity {
		// Remove oldest version
		copy(vh.versions, vh.versions[1:])
		vh.versions = vh.versions[:len(vh.versions)-1]
	}

	vh.versions = append(vh.versions, metadata)
}

// GetVersion retrieves version metadata by version number
func (vh *VersionHistory) GetVersion(version uint64) (*VersionMetadata, error) {
	for i := len(vh.versions) - 1; i >= 0; i-- {
		if vh.versions[i].Version == version {
			return &vh.versions[i], nil
		}
	}
	return nil, fmt.Errorf("version %d not found", version)
}

// GetLatestVersion returns the most recent version metadata
func (vh *VersionHistory) GetLatestVersion() (*VersionMetadata, error) {
	if len(vh.versions) == 0 {
		return nil, fmt.Errorf("no versions available")
	}
	return &vh.versions[len(vh.versions)-1], nil
}

// GetVersionsInRange returns all versions within a time range
func (vh *VersionHistory) GetVersionsInRange(start, end time.Time) []VersionMetadata {
	var result []VersionMetadata

	for _, version := range vh.versions {
		if !version.Timestamp.Before(start) && !version.Timestamp.After(end) {
			result = append(result, version)
		}
	}

	return result
}

// GetVersionCount returns the number of versions stored
func (vh *VersionHistory) GetVersionCount() int {
	return len(vh.versions)
}

// PurgeOldVersions removes versions older than the specified duration
func (vh *VersionHistory) PurgeOldVersions(maxAge time.Duration) int {
	cutoff := time.Now().Add(-maxAge)
	purged := 0

	for i := 0; i < len(vh.versions); {
		if vh.versions[i].Timestamp.Before(cutoff) {
			// Remove this version
			copy(vh.versions[i:], vh.versions[i+1:])
			vh.versions = vh.versions[:len(vh.versions)-1]
			purged++
		} else {
			i++
		}
	}

	return purged
}

// GetStats returns statistics about the version history
func (vh *VersionHistory) GetStats() map[string]interface{} {
	if len(vh.versions) == 0 {
		return map[string]interface{}{
			"count":         0,
			"capacity":      vh.capacity,
			"oldest":        nil,
			"newest":        nil,
			"total_size":    0,
			"deleted_count": 0,
		}
	}

	oldest := vh.versions[0]
	newest := vh.versions[len(vh.versions)-1]

	var totalSize uint64
	var deletedCount int

	for _, version := range vh.versions {
		totalSize += uint64(version.Size)
		if version.Deleted {
			deletedCount++
		}
	}

	return map[string]interface{}{
		"count":         len(vh.versions),
		"capacity":      vh.capacity,
		"oldest":        oldest.Timestamp,
		"newest":        newest.Timestamp,
		"total_size":    totalSize,
		"deleted_count": deletedCount,
		"avg_size":      float64(totalSize) / float64(len(vh.versions)),
	}
}

// Serialize converts the version history to bytes
func (vh *VersionHistory) Serialize() ([]byte, error) {
	// Calculate total size
	totalSize := 4 + 4 // count + capacity

	for range vh.versions {
		totalSize += 8 + 8 + 8 + 1 + 4 + 4 + 8 // version + timestamp + txnid + deleted + size + checksum + compacted_at
	}

	data := make([]byte, totalSize)
	offset := 0

	// Write header
	binary.LittleEndian.PutUint32(data[offset:], uint32(len(vh.versions)))
	offset += 4
	binary.LittleEndian.PutUint32(data[offset:], uint32(vh.capacity))
	offset += 4

	// Write versions
	for _, version := range vh.versions {
		binary.LittleEndian.PutUint64(data[offset:], version.Version)
		offset += 8

		binary.LittleEndian.PutUint64(data[offset:], uint64(version.Timestamp.Unix()))
		offset += 8

		binary.LittleEndian.PutUint64(data[offset:], version.TxnID)
		offset += 8

		if version.Deleted {
			data[offset] = 1
		} else {
			data[offset] = 0
		}
		offset += 1

		binary.LittleEndian.PutUint32(data[offset:], version.Size)
		offset += 4

		binary.LittleEndian.PutUint32(data[offset:], version.Checksum)
		offset += 4

		compactedAt := int64(0)
		if !version.CompactedAt.IsZero() {
			compactedAt = version.CompactedAt.Unix()
		}
		binary.LittleEndian.PutUint64(data[offset:], uint64(compactedAt))
		offset += 8
	}

	return data, nil
}

// Deserialize restores the version history from bytes
func (vh *VersionHistory) Deserialize(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("invalid version history data: too short")
	}

	offset := 0

	// Read header
	count := binary.LittleEndian.Uint32(data[offset:])
	offset += 4
	capacity := binary.LittleEndian.Uint32(data[offset:])
	offset += 4

	vh.capacity = int(capacity)
	vh.versions = make([]VersionMetadata, 0, count)

	// Read versions
	for i := uint32(0); i < count; i++ {
		if offset+41 > len(data) {
			return fmt.Errorf("invalid version history data: truncated version")
		}

		version := VersionMetadata{}

		version.Version = binary.LittleEndian.Uint64(data[offset:])
		offset += 8

		version.Timestamp = time.Unix(int64(binary.LittleEndian.Uint64(data[offset:])), 0)
		offset += 8

		version.TxnID = binary.LittleEndian.Uint64(data[offset:])
		offset += 8

		version.Deleted = data[offset] == 1
		offset += 1

		version.Size = binary.LittleEndian.Uint32(data[offset:])
		offset += 4

		version.Checksum = binary.LittleEndian.Uint32(data[offset:])
		offset += 4

		compactedAt := int64(binary.LittleEndian.Uint64(data[offset:]))
		if compactedAt > 0 {
			version.CompactedAt = time.Unix(compactedAt, 0)
		}
		offset += 8

		vh.versions = append(vh.versions, version)
	}

	return nil
}

// VersionSnapshot represents a point-in-time snapshot of the database
type VersionSnapshot struct {
	Version   uint64    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	KeyCount  uint64    `json:"key_count"`
	DataSize  uint64    `json:"data_size"`
	Checksum  uint64    `json:"checksum"`
}

// SnapshotManager manages database snapshots for point-in-time recovery
type SnapshotManager struct {
	snapshots    []VersionSnapshot
	maxSnapshots int
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(maxSnapshots int) *SnapshotManager {
	return &SnapshotManager{
		snapshots:    make([]VersionSnapshot, 0, maxSnapshots),
		maxSnapshots: maxSnapshots,
	}
}

// CreateSnapshot creates a new snapshot
func (sm *SnapshotManager) CreateSnapshot(version uint64, keyCount, dataSize, checksum uint64) {
	snapshot := VersionSnapshot{
		Version:   version,
		Timestamp: time.Now(),
		KeyCount:  keyCount,
		DataSize:  dataSize,
		Checksum:  checksum,
	}

	if len(sm.snapshots) >= sm.maxSnapshots {
		// Remove oldest snapshot
		copy(sm.snapshots, sm.snapshots[1:])
		sm.snapshots = sm.snapshots[:len(sm.snapshots)-1]
	}

	sm.snapshots = append(sm.snapshots, snapshot)
}

// GetSnapshot retrieves a snapshot by version
func (sm *SnapshotManager) GetSnapshot(version uint64) (*VersionSnapshot, error) {
	for i := len(sm.snapshots) - 1; i >= 0; i-- {
		if sm.snapshots[i].Version <= version {
			return &sm.snapshots[i], nil
		}
	}
	return nil, fmt.Errorf("no snapshot found for version %d", version)
}

// GetSnapshotAtTime retrieves the snapshot closest to a specific time
func (sm *SnapshotManager) GetSnapshotAtTime(timestamp time.Time) (*VersionSnapshot, error) {
	var closest *VersionSnapshot
	var minDiff time.Duration = time.Duration(1<<63 - 1) // Max duration

	for i := range sm.snapshots {
		diff := timestamp.Sub(sm.snapshots[i].Timestamp)
		if diff >= 0 && diff < minDiff {
			closest = &sm.snapshots[i]
			minDiff = diff
		}
	}

	if closest == nil {
		return nil, fmt.Errorf("no snapshot found before time %v", timestamp)
	}

	return closest, nil
}

// ListSnapshots returns all available snapshots
func (sm *SnapshotManager) ListSnapshots() []VersionSnapshot {
	result := make([]VersionSnapshot, len(sm.snapshots))
	copy(result, sm.snapshots)
	return result
}

// PurgeOldSnapshots removes snapshots older than the specified duration
func (sm *SnapshotManager) PurgeOldSnapshots(maxAge time.Duration) int {
	cutoff := time.Now().Add(-maxAge)
	purged := 0

	for i := 0; i < len(sm.snapshots); {
		if sm.snapshots[i].Timestamp.Before(cutoff) {
			// Remove this snapshot
			copy(sm.snapshots[i:], sm.snapshots[i+1:])
			sm.snapshots = sm.snapshots[:len(sm.snapshots)-1]
			purged++
		} else {
			i++
		}
	}

	return purged
}

// GetStats returns statistics about the snapshot manager
func (sm *SnapshotManager) GetStats() map[string]interface{} {
	if len(sm.snapshots) == 0 {
		return map[string]interface{}{
			"count":           0,
			"max_snapshots":   sm.maxSnapshots,
			"oldest":          nil,
			"newest":          nil,
			"total_data_size": 0,
		}
	}

	oldest := sm.snapshots[0]
	newest := sm.snapshots[len(sm.snapshots)-1]

	var totalDataSize uint64
	for _, snapshot := range sm.snapshots {
		totalDataSize += snapshot.DataSize
	}

	return map[string]interface{}{
		"count":           len(sm.snapshots),
		"max_snapshots":   sm.maxSnapshots,
		"oldest":          oldest.Timestamp,
		"newest":          newest.Timestamp,
		"total_data_size": totalDataSize,
		"avg_data_size":   float64(totalDataSize) / float64(len(sm.snapshots)),
	}
}
