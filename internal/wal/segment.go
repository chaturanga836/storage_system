package wal

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sync"
)

// Segment represents a single WAL segment file
type Segment struct {
	mu       sync.RWMutex
	path     string
	file     *os.File
	writer   *bufio.Writer
	size     int64
	minSeqID uint64
	maxSeqID uint64
	closed   bool
}

// CreateSegment creates a new WAL segment
func CreateSegment(path string) (*Segment, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create segment file: %w", err)
	}

	return &Segment{
		path:   path,
		file:   file,
		writer: bufio.NewWriter(file),
	}, nil
}

// OpenSegment opens an existing WAL segment
func OpenSegment(path string) (*Segment, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open segment file: %w", err)
	}

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat segment file: %w", err)
	}

	segment := &Segment{
		path:   path,
		file:   file,
		writer: bufio.NewWriter(file),
		size:   stat.Size(),
	}

	// Read the segment to determine min/max sequence IDs
	if err := segment.scanSequenceIDs(); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to scan sequence IDs: %w", err)
	}

	return segment, nil
}

// Append appends an entry to the segment
func (s *Segment) Append(entry *Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("segment is closed")
	}

	// Serialize the entry
	data, err := entry.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	// Calculate checksum
	checksum := crc32.ChecksumIEEE(data)

	// Write length (4 bytes) + checksum (4 bytes) + data
	if err := binary.Write(s.writer, binary.LittleEndian, uint32(len(data))); err != nil {
		return fmt.Errorf("failed to write entry length: %w", err)
	}

	if err := binary.Write(s.writer, binary.LittleEndian, checksum); err != nil {
		return fmt.Errorf("failed to write checksum: %w", err)
	}

	if _, err := s.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write entry data: %w", err)
	}

	// Update size and sequence ID range
	recordSize := 8 + len(data) // 4 bytes length + 4 bytes checksum + data
	s.size += int64(recordSize)

	if s.minSeqID == 0 || entry.SequenceID < s.minSeqID {
		s.minSeqID = entry.SequenceID
	}
	if entry.SequenceID > s.maxSeqID {
		s.maxSeqID = entry.SequenceID
	}

	return nil
}

// Sync flushes the segment to disk
func (s *Segment) Sync() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("segment is closed")
	}

	if err := s.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush writer: %w", err)
	}

	if err := s.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}

// Close closes the segment
func (s *Segment) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true

	if err := s.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush writer: %w", err)
	}

	if err := s.file.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	return nil
}

// Size returns the current size of the segment
func (s *Segment) Size() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.size
}

// Path returns the file path of the segment
func (s *Segment) Path() string {
	return s.path
}

// Contains checks if the segment contains the given sequence ID
func (s *Segment) Contains(seqID uint64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.minSeqID <= seqID && seqID <= s.maxSeqID
}

// MinSequenceID returns the minimum sequence ID in the segment
func (s *Segment) MinSequenceID() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.minSeqID
}

// MaxSequenceID returns the maximum sequence ID in the segment
func (s *Segment) MaxSequenceID() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.maxSeqID
}

// NewReader creates a reader for this segment starting from the given sequence ID
func (s *Segment) NewReader(fromSeqID uint64) (*SegmentReader, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, fmt.Errorf("segment is closed")
	}

	file, err := os.Open(s.path)
	if err != nil {
		return nil, fmt.Errorf("failed to open segment for reading: %w", err)
	}

	return &SegmentReader{
		file:      file,
		reader:    bufio.NewReader(file),
		fromSeqID: fromSeqID,
	}, nil
}

// SegmentReader reads entries from a segment
type SegmentReader struct {
	file      *os.File
	reader    *bufio.Reader
	fromSeqID uint64
	eof       bool
}

// Next reads the next entry from the segment
func (sr *SegmentReader) Next() (*Entry, error) {
	if sr.eof {
		return nil, io.EOF
	}

	for {
		// Read entry length
		var length uint32
		if err := binary.Read(sr.reader, binary.LittleEndian, &length); err != nil {
			if err == io.EOF {
				sr.eof = true
			}
			return nil, err
		}

		// Read checksum
		var checksum uint32
		if err := binary.Read(sr.reader, binary.LittleEndian, &checksum); err != nil {
			return nil, fmt.Errorf("failed to read checksum: %w", err)
		}

		// Read entry data
		data := make([]byte, length)
		if _, err := io.ReadFull(sr.reader, data); err != nil {
			return nil, fmt.Errorf("failed to read entry data: %w", err)
		}

		// Verify checksum
		if crc32.ChecksumIEEE(data) != checksum {
			return nil, fmt.Errorf("checksum mismatch")
		}

		// Unmarshal entry
		entry, err := UnmarshalEntry(data)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal entry: %w", err)
		}

		// Skip entries before the requested sequence ID
		if entry.SequenceID < sr.fromSeqID {
			continue
		}

		return entry, nil
	}
}

// Close closes the segment reader
func (sr *SegmentReader) Close() error {
	return sr.file.Close()
}

// scanSequenceIDs scans the segment to determine min/max sequence IDs
func (s *Segment) scanSequenceIDs() error {
	if s.size == 0 {
		return nil // Empty segment
	}

	// Create a temporary reader
	file, err := os.Open(s.path)
	if err != nil {
		return fmt.Errorf("failed to open segment for scanning: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var minSeqID, maxSeqID uint64

	for {
		// Read entry length
		var length uint32
		if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read entry length: %w", err)
		}

		// Read checksum
		var checksum uint32
		if err := binary.Read(reader, binary.LittleEndian, &checksum); err != nil {
			return fmt.Errorf("failed to read checksum: %w", err)
		}

		// Read entry data
		data := make([]byte, length)
		if _, err := io.ReadFull(reader, data); err != nil {
			return fmt.Errorf("failed to read entry data: %w", err)
		}

		// Verify checksum
		if crc32.ChecksumIEEE(data) != checksum {
			return fmt.Errorf("checksum mismatch during scan")
		}

		// Unmarshal entry to get sequence ID
		entry, err := UnmarshalEntry(data)
		if err != nil {
			return fmt.Errorf("failed to unmarshal entry during scan: %w", err)
		}

		if minSeqID == 0 || entry.SequenceID < minSeqID {
			minSeqID = entry.SequenceID
		}
		if entry.SequenceID > maxSeqID {
			maxSeqID = entry.SequenceID
		}
	}

	s.minSeqID = minSeqID
	s.maxSeqID = maxSeqID
	return nil
}
