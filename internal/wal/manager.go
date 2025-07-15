package wal

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Manager implements the Write-Ahead Log manager interface
type Manager struct {
	mu        sync.RWMutex
	config    Config
	dataDir   string
	segments  []*Segment
	current   *Segment
	nextSeqID uint64
	closed    bool
}

// NewManager creates a new WAL manager
func NewManager(config Config) (*Manager, error) {
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create WAL directory: %w", err)
	}

	manager := &Manager{
		config:    config,
		dataDir:   config.DataDir,
		nextSeqID: 1,
	}

	// Load existing segments
	if err := manager.loadSegments(); err != nil {
		return nil, fmt.Errorf("failed to load segments: %w", err)
	}

	// Create initial segment if none exist
	if len(manager.segments) == 0 {
		if err := manager.createNewSegment(); err != nil {
			return nil, fmt.Errorf("failed to create initial segment: %w", err)
		}
	}

	return manager, nil
}

// Append appends a new entry to the WAL
func (m *Manager) Append(ctx context.Context, entry *Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("WAL manager is closed")
	}

	// Check if we need to rotate to a new segment
	if m.current.Size() >= m.config.SegmentSize {
		if err := m.rotateSegment(); err != nil {
			return fmt.Errorf("failed to rotate segment: %w", err)
		}
	}

	// Set sequence ID
	entry.SequenceID = m.nextSeqID
	m.nextSeqID++

	// Append to current segment
	if err := m.current.Append(entry); err != nil {
		return fmt.Errorf("failed to append to segment: %w", err)
	}

	// Handle sync policy
	switch m.config.SyncPolicy {
	case SyncAlways:
		if err := m.current.Sync(); err != nil {
			return fmt.Errorf("failed to sync segment: %w", err)
		}
	case SyncBatch:
		// Sync will be handled by batch processing
	case SyncPeriodic:
		// Sync will be handled by periodic timer
	}

	return nil
}

// AppendBatch appends multiple entries atomically
func (m *Manager) AppendBatch(ctx context.Context, entries []*Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("WAL manager is closed")
	}

	// Calculate total size needed
	totalSize := int64(0)
	for _, entry := range entries {
		totalSize += int64(entry.EstimatedSize())
	}

	// Check if we need to rotate before batch
	if m.current.Size()+totalSize >= m.config.SegmentSize {
		if err := m.rotateSegment(); err != nil {
			return fmt.Errorf("failed to rotate segment: %w", err)
		}
	}

	// Append all entries
	for _, entry := range entries {
		entry.SequenceID = m.nextSeqID
		m.nextSeqID++

		if err := m.current.Append(entry); err != nil {
			return fmt.Errorf("failed to append entry to segment: %w", err)
		}
	}

	// Sync after batch if policy requires it
	if m.config.SyncPolicy == SyncBatch || m.config.SyncPolicy == SyncAlways {
		if err := m.current.Sync(); err != nil {
			return fmt.Errorf("failed to sync segment after batch: %w", err)
		}
	}

	return nil
}

// ReadFrom reads WAL entries starting from the given sequence ID
func (m *Manager) ReadFrom(ctx context.Context, fromSeqID uint64) (*Reader, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, fmt.Errorf("WAL manager is closed")
	}

	// Find the segment containing the starting sequence ID
	var startSegment *Segment
	for _, segment := range m.segments {
		if segment.Contains(fromSeqID) {
			startSegment = segment
			break
		}
	}

	if startSegment == nil {
		return nil, fmt.Errorf("sequence ID %d not found in any segment", fromSeqID)
	}

	return NewReader(m.segments, startSegment, fromSeqID)
}

// Checkpoint creates a checkpoint and removes old segments
func (m *Manager) Checkpoint(ctx context.Context, upToSeqID uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("WAL manager is closed")
	}

	// Find segments that can be removed (all entries <= upToSeqID)
	var toRemove []*Segment
	var toKeep []*Segment

	for _, segment := range m.segments {
		if segment.MaxSequenceID() <= upToSeqID && segment != m.current {
			toRemove = append(toRemove, segment)
		} else {
			toKeep = append(toKeep, segment)
		}
	}

	// Remove old segments
	for _, segment := range toRemove {
		if err := segment.Close(); err != nil {
			return fmt.Errorf("failed to close segment: %w", err)
		}
		if err := os.Remove(segment.Path()); err != nil {
			return fmt.Errorf("failed to remove segment file: %w", err)
		}
	}

	m.segments = toKeep
	return nil
}

// Replay replays WAL entries from the given sequence ID
func (m *Manager) Replay(ctx context.Context, fromSeqID uint64, handler func(*Entry) error) error {
	reader, err := m.ReadFrom(ctx, fromSeqID)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}
	defer reader.Close()

	for {
		entry, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read entry: %w", err)
		}

		if err := handler(entry); err != nil {
			return fmt.Errorf("handler failed for entry %d: %w", entry.SequenceID, err)
		}
	}

	return nil
}

// Close closes the WAL manager
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true

	// Close all segments
	for _, segment := range m.segments {
		if err := segment.Close(); err != nil {
			return fmt.Errorf("failed to close segment: %w", err)
		}
	}

	return nil
}

// GetStats returns WAL statistics
func (m *Manager) GetStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := Stats{
		SegmentCount: len(m.segments),
		NextSeqID:    m.nextSeqID,
	}

	totalSize := int64(0)
	for _, segment := range m.segments {
		totalSize += segment.Size()
	}
	stats.TotalSize = totalSize

	if len(m.segments) > 0 {
		stats.FirstSeqID = m.segments[0].MinSequenceID()
		stats.LastSeqID = m.segments[len(m.segments)-1].MaxSequenceID()
	}

	return stats
}

// Private methods

func (m *Manager) loadSegments() error {
	entries, err := os.ReadDir(m.dataDir)
	if err != nil {
		return fmt.Errorf("failed to read WAL directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".wal" {
			segmentPath := filepath.Join(m.dataDir, entry.Name())
			segment, err := OpenSegment(segmentPath)
			if err != nil {
				return fmt.Errorf("failed to open segment %s: %w", segmentPath, err)
			}
			m.segments = append(m.segments, segment)
		}
	}

	// Set current segment to the last one
	if len(m.segments) > 0 {
		m.current = m.segments[len(m.segments)-1]
		// Update next sequence ID based on the last entry
		m.nextSeqID = m.current.MaxSequenceID() + 1
	}

	return nil
}

func (m *Manager) createNewSegment() error {
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("wal-%d.wal", timestamp)
	segmentPath := filepath.Join(m.dataDir, filename)

	segment, err := CreateSegment(segmentPath)
	if err != nil {
		return fmt.Errorf("failed to create segment: %w", err)
	}

	m.segments = append(m.segments, segment)
	m.current = segment

	return nil
}

func (m *Manager) rotateSegment() error {
	// Close current segment
	if err := m.current.Close(); err != nil {
		return fmt.Errorf("failed to close current segment: %w", err)
	}

	// Create new segment
	if err := m.createNewSegment(); err != nil {
		return fmt.Errorf("failed to create new segment: %w", err)
	}

	// Clean up old segments if we exceed max count
	if len(m.segments) > m.config.MaxSegments {
		oldSegments := m.segments[:len(m.segments)-m.config.MaxSegments]
		for _, segment := range oldSegments {
			if err := os.Remove(segment.Path()); err != nil {
				return fmt.Errorf("failed to remove old segment: %w", err)
			}
		}
		m.segments = m.segments[len(m.segments)-m.config.MaxSegments:]
	}

	return nil
}
