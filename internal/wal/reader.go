package wal

import (
	"fmt"
	"io"
)

// Reader provides a unified interface for reading WAL entries across multiple segments
type Reader struct {
	segments       []*Segment
	currentSegment *Segment
	segmentReader  *SegmentReader
	segmentIndex   int
	fromSeqID      uint64
	closed         bool
}

// NewReader creates a new WAL reader
func NewReader(segments []*Segment, startSegment *Segment, fromSeqID uint64) (*Reader, error) {
	// Find the index of the start segment
	startIndex := -1
	for i, segment := range segments {
		if segment == startSegment {
			startIndex = i
			break
		}
	}

	if startIndex == -1 {
		return nil, fmt.Errorf("start segment not found in segments list")
	}

	reader := &Reader{
		segments:     segments,
		segmentIndex: startIndex,
		fromSeqID:    fromSeqID,
	}

	// Initialize the first segment reader
	if err := reader.initCurrentSegmentReader(); err != nil {
		return nil, fmt.Errorf("failed to initialize segment reader: %w", err)
	}

	return reader, nil
}

// Next reads the next entry from the WAL
func (r *Reader) Next() (*Entry, error) {
	if r.closed {
		return nil, fmt.Errorf("reader is closed")
	}

	for {
		// Try to read from current segment
		if r.segmentReader != nil {
			entry, err := r.segmentReader.Next()
			if err == nil {
				return entry, nil
			}

			// If we hit EOF, move to next segment
			if err == io.EOF {
				if err := r.moveToNextSegment(); err != nil {
					return nil, err
				}
				continue
			}

			// Other errors
			return nil, fmt.Errorf("failed to read from segment: %w", err)
		}

		// No more segments
		return nil, io.EOF
	}
}

// Close closes the reader and releases resources
func (r *Reader) Close() error {
	if r.closed {
		return nil
	}

	r.closed = true

	if r.segmentReader != nil {
		return r.segmentReader.Close()
	}

	return nil
}

// Seek moves the reader to the specified sequence ID
func (r *Reader) Seek(seqID uint64) error {
	if r.closed {
		return fmt.Errorf("reader is closed")
	}

	// Find the segment containing the sequence ID
	targetSegmentIndex := -1
	for i, segment := range r.segments {
		if segment.Contains(seqID) {
			targetSegmentIndex = i
			break
		}
	}

	if targetSegmentIndex == -1 {
		return fmt.Errorf("sequence ID %d not found in any segment", seqID)
	}

	// Close current segment reader if it exists
	if r.segmentReader != nil {
		r.segmentReader.Close()
		r.segmentReader = nil
	}

	// Update state
	r.segmentIndex = targetSegmentIndex
	r.fromSeqID = seqID

	// Initialize new segment reader
	return r.initCurrentSegmentReader()
}

// GetPosition returns the current position (segment index and sequence ID)
func (r *Reader) GetPosition() (segmentIndex int, seqID uint64) {
	return r.segmentIndex, r.fromSeqID
}

// HasMore returns true if there are more entries to read
func (r *Reader) HasMore() bool {
	if r.closed {
		return false
	}

	// If we have a current segment reader and it's not at EOF, we have more
	if r.segmentReader != nil {
		return !r.segmentReader.eof
	}

	// Check if there are more segments after the current one
	return r.segmentIndex < len(r.segments)-1
}

// Count returns the estimated number of remaining entries
// This is an approximation and may not be exact
func (r *Reader) Count() int64 {
	if r.closed {
		return 0
	}

	count := int64(0)

	// Count entries in remaining segments
	for i := r.segmentIndex; i < len(r.segments); i++ {
		segment := r.segments[i]
		
		if i == r.segmentIndex {
			// For current segment, count from fromSeqID
			if segment.MaxSequenceID() >= r.fromSeqID {
				count += int64(segment.MaxSequenceID() - r.fromSeqID + 1)
			}
		} else {
			// For future segments, count all entries
			if segment.MaxSequenceID() > 0 {
				count += int64(segment.MaxSequenceID() - segment.MinSequenceID() + 1)
			}
		}
	}

	return count
}

// Private methods

func (r *Reader) initCurrentSegmentReader() error {
	if r.segmentIndex >= len(r.segments) {
		return nil // No more segments
	}

	r.currentSegment = r.segments[r.segmentIndex]
	
	segmentReader, err := r.currentSegment.NewReader(r.fromSeqID)
	if err != nil {
		return fmt.Errorf("failed to create segment reader: %w", err)
	}

	r.segmentReader = segmentReader
	return nil
}

func (r *Reader) moveToNextSegment() error {
	// Close current segment reader
	if r.segmentReader != nil {
		r.segmentReader.Close()
		r.segmentReader = nil
	}

	// Move to next segment
	r.segmentIndex++
	
	if r.segmentIndex >= len(r.segments) {
		return io.EOF // No more segments
	}

	// Reset fromSeqID to the beginning of the next segment
	nextSegment := r.segments[r.segmentIndex]
	r.fromSeqID = nextSegment.MinSequenceID()

	// Initialize reader for next segment
	return r.initCurrentSegmentReader()
}

// BatchReader provides batched reading of WAL entries
type BatchReader struct {
	reader    *Reader
	batchSize int
}

// NewBatchReader creates a new batch reader
func NewBatchReader(segments []*Segment, startSegment *Segment, fromSeqID uint64, batchSize int) (*BatchReader, error) {
	reader, err := NewReader(segments, startSegment, fromSeqID)
	if err != nil {
		return nil, err
	}

	return &BatchReader{
		reader:    reader,
		batchSize: batchSize,
	}, nil
}

// NextBatch reads the next batch of entries
func (br *BatchReader) NextBatch() ([]*Entry, error) {
	var entries []*Entry
	
	for len(entries) < br.batchSize {
		entry, err := br.reader.Next()
		if err != nil {
			if err == io.EOF {
				break // Return what we have
			}
			return nil, err
		}
		
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, io.EOF
	}

	return entries, nil
}

// Close closes the batch reader
func (br *BatchReader) Close() error {
	return br.reader.Close()
}

// Seek moves the batch reader to the specified sequence ID
func (br *BatchReader) Seek(seqID uint64) error {
	return br.reader.Seek(seqID)
}

// HasMore returns true if there are more entries to read
func (br *BatchReader) HasMore() bool {
	return br.reader.HasMore()
}
