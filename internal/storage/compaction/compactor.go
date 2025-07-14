package compaction

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"storage-engine/internal/common"
	"storage-engine/internal/storage/block"
	"storage-engine/internal/storage/index"
	"storage-engine/internal/storage/parquet"
)

// CompactionLevel represents different levels in LSM tree
type CompactionLevel int

const (
	Level0 CompactionLevel = 0 // Memtable flushes
	Level1 CompactionLevel = 1 // First compaction level
	Level2 CompactionLevel = 2 // Second compaction level
	Level3 CompactionLevel = 3 // Third compaction level
	Level4 CompactionLevel = 4 // Fourth compaction level
	Level5 CompactionLevel = 5 // Fifth compaction level
	Level6 CompactionLevel = 6 // Highest level
)

// CompactionJob represents a single compaction operation
type CompactionJob struct {
	ID          string
	Level       CompactionLevel
	InputFiles  []string
	OutputFiles []string
	StartTime   time.Time
	EndTime     time.Time
	Status      string
	BytesRead   uint64
	BytesWritten uint64
	RecordsProcessed uint64
}

// SSTableInfo represents metadata about an SSTable file
type SSTableInfo struct {
	Path         string
	Level        CompactionLevel
	Size         uint64
	KeyRange     [2][]byte // [min_key, max_key]
	RecordCount  uint64
	CreationTime time.Time
	LastAccess   time.Time
}

// Compactor handles background compaction of SSTable files
type Compactor struct {
	mu            sync.RWMutex
	storage       block.StorageBackend
	indexManager  *index.IndexSerializer
	strategy      CompactionStrategy
	
	// SSTable tracking
	sstables      map[CompactionLevel][]*SSTableInfo
	compactionJobs map[string]*CompactionJob
	
	// Configuration
	maxLevelSize  map[CompactionLevel]uint64
	targetFileSize uint64
	compactionThreads int
	
	// Control channels
	stopCh        chan struct{}
	jobCh         chan *CompactionJob
	done          chan struct{}
	
	// Statistics
	totalCompactions uint64
	totalBytesCompacted uint64
	lastCompactionTime time.Time
}

// NewCompactor creates a new compactor instance
func NewCompactor(storage block.StorageBackend, indexManager *index.IndexSerializer, strategy CompactionStrategy) *Compactor {
	c := &Compactor{
		storage:      storage,
		indexManager: indexManager,
		strategy:     strategy,
		sstables:     make(map[CompactionLevel][]*SSTableInfo),
		compactionJobs: make(map[string]*CompactionJob),
		maxLevelSize: map[CompactionLevel]uint64{
			Level0: 10 * 1024 * 1024,    // 10 MB
			Level1: 100 * 1024 * 1024,   // 100 MB
			Level2: 1024 * 1024 * 1024,  // 1 GB
			Level3: 10 * 1024 * 1024 * 1024, // 10 GB
			Level4: 100 * 1024 * 1024 * 1024, // 100 GB
			Level5: 1000 * 1024 * 1024 * 1024, // 1 TB
			Level6: 10000 * 1024 * 1024 * 1024, // 10 TB
		},
		targetFileSize:    64 * 1024 * 1024, // 64 MB
		compactionThreads: 4,
		stopCh:           make(chan struct{}),
		jobCh:            make(chan *CompactionJob, 100),
		done:             make(chan struct{}),
	}
	
	// Initialize levels
	for level := Level0; level <= Level6; level++ {
		c.sstables[level] = make([]*SSTableInfo, 0)
	}
	
	return c
}

// Start begins the background compaction process
func (c *Compactor) Start(ctx context.Context) error {
	// Start worker goroutines
	for i := 0; i < c.compactionThreads; i++ {
		go c.compactionWorker(ctx)
	}
	
	// Start compaction scheduler
	go c.compactionScheduler(ctx)
	
	return nil
}

// Stop gracefully shuts down the compactor
func (c *Compactor) Stop() error {
	close(c.stopCh)
	<-c.done
	return nil
}

// AddSSTable registers a new SSTable file
func (c *Compactor) AddSSTable(path string, level CompactionLevel, size uint64, keyRange [2][]byte, recordCount uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	sstable := &SSTableInfo{
		Path:         path,
		Level:        level,
		Size:         size,
		KeyRange:     keyRange,
		RecordCount:  recordCount,
		CreationTime: time.Now(),
		LastAccess:   time.Now(),
	}
	
	c.sstables[level] = append(c.sstables[level], sstable)
	
	// Sort by key range for efficient lookups
	sort.Slice(c.sstables[level], func(i, j int) bool {
		return common.CompareBytes(c.sstables[level][i].KeyRange[0], c.sstables[level][j].KeyRange[0]) < 0
	})
	
	return nil
}

// RemoveSSTable removes an SSTable file from tracking
func (c *Compactor) RemoveSSTable(path string, level CompactionLevel) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	sstables := c.sstables[level]
	for i, sstable := range sstables {
		if sstable.Path == path {
			// Remove by swapping with last element
			sstables[i] = sstables[len(sstables)-1]
			c.sstables[level] = sstables[:len(sstables)-1]
			return nil
		}
	}
	
	return fmt.Errorf("sstable %s not found at level %d", path, level)
}

// GetSSTablesForLevel returns all SSTables at a given level
func (c *Compactor) GetSSTablesForLevel(level CompactionLevel) []*SSTableInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Return copy to avoid race conditions
	result := make([]*SSTableInfo, len(c.sstables[level]))
	copy(result, c.sstables[level])
	return result
}

// GetLevelSize returns the total size of all SSTables at a level
func (c *Compactor) GetLevelSize(level CompactionLevel) uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	var totalSize uint64
	for _, sstable := range c.sstables[level] {
		totalSize += sstable.Size
	}
	return totalSize
}

// ShouldCompact determines if a level needs compaction
func (c *Compactor) ShouldCompact(level CompactionLevel) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	currentSize := c.GetLevelSize(level)
	maxSize := c.maxLevelSize[level]
	
	return currentSize > maxSize
}

// ScheduleCompaction adds a compaction job to the queue
func (c *Compactor) ScheduleCompaction(level CompactionLevel, inputFiles []string) error {
	jobID := fmt.Sprintf("compact_%d_%d", level, time.Now().Unix())
	
	job := &CompactionJob{
		ID:         jobID,
		Level:      level,
		InputFiles: inputFiles,
		StartTime:  time.Now(),
		Status:     "scheduled",
	}
	
	c.mu.Lock()
	c.compactionJobs[jobID] = job
	c.mu.Unlock()
	
	select {
	case c.jobCh <- job:
		return nil
	default:
		return fmt.Errorf("compaction queue is full")
	}
}

// compactionScheduler continuously monitors for compaction opportunities
func (c *Compactor) compactionScheduler(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.checkAndScheduleCompactions()
		}
	}
}

// checkAndScheduleCompactions identifies levels that need compaction
func (c *Compactor) checkAndScheduleCompactions() {
	for level := Level0; level <= Level5; level++ { // Don't compact Level6
		if c.ShouldCompact(level) {
			inputFiles := c.strategy.SelectFilesForCompaction(level, c.GetSSTablesForLevel(level))
			if len(inputFiles) > 0 {
				var filePaths []string
				for _, info := range inputFiles {
					filePaths = append(filePaths, info.Path)
				}
				
				err := c.ScheduleCompaction(level, filePaths)
				if err != nil {
					// Log error but continue
					fmt.Printf("Failed to schedule compaction for level %d: %v\n", level, err)
				}
			}
		}
	}
}

// compactionWorker processes compaction jobs
func (c *Compactor) compactionWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case job := <-c.jobCh:
			err := c.executeCompaction(ctx, job)
			if err != nil {
				job.Status = "failed"
				fmt.Printf("Compaction job %s failed: %v\n", job.ID, err)
			} else {
				job.Status = "completed"
				c.totalCompactions++
				c.lastCompactionTime = time.Now()
			}
			job.EndTime = time.Now()
		}
	}
}

// executeCompaction performs the actual compaction work
func (c *Compactor) executeCompaction(ctx context.Context, job *CompactionJob) error {
	job.Status = "running"
	
	// Read input files
	var inputRecords []*parquet.Record
	for _, filePath := range job.InputFiles {
		records, err := c.readSSTableFile(ctx, filePath)
		if err != nil {
			return fmt.Errorf("failed to read input file %s: %w", filePath, err)
		}
		inputRecords = append(inputRecords, records...)
		job.BytesRead += uint64(len(records)) * 100 // Rough estimate
	}
	
	// Sort and deduplicate records
	sortedRecords := c.sortAndDeduplicateRecords(inputRecords)
	job.RecordsProcessed = uint64(len(sortedRecords))
	
	// Write output files
	outputLevel := job.Level + 1
	if outputLevel > Level6 {
		outputLevel = Level6
	}
	
	outputFiles, err := c.writeCompactedFiles(ctx, sortedRecords, outputLevel)
	if err != nil {
		return fmt.Errorf("failed to write compacted files: %w", err)
	}
	
	job.OutputFiles = outputFiles
	for _, file := range outputFiles {
		job.BytesWritten += c.estimateFileSize(file)
	}
	
	// Update SSTable tracking
	err = c.updateSSTableTracking(job)
	if err != nil {
		return fmt.Errorf("failed to update sstable tracking: %w", err)
	}
	
	// Clean up input files
	for _, filePath := range job.InputFiles {
		err := c.storage.DeleteBlock(ctx, filePath)
		if err != nil {
			fmt.Printf("Warning: failed to delete input file %s: %v\n", filePath, err)
		}
	}
	
	return nil
}

// readSSTableFile reads records from an SSTable file
func (c *Compactor) readSSTableFile(ctx context.Context, filePath string) ([]*parquet.Record, error) {
	// Read the file data
	data, err := c.storage.ReadBlock(ctx, filePath)
	if err != nil {
		return nil, err
	}
	
	// Create a parquet reader
	reader := parquet.NewReader(data)
	
	// Read all records
	var records []*parquet.Record
	for {
		record, err := reader.ReadRecord()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
		records = append(records, record)
	}
	
	return records, nil
}

// sortAndDeduplicateRecords sorts records by key and removes duplicates
func (c *Compactor) sortAndDeduplicateRecords(records []*parquet.Record) []*parquet.Record {
	// Sort by key
	sort.Slice(records, func(i, j int) bool {
		return common.CompareBytes(records[i].Key, records[j].Key) < 0
	})
	
	// Deduplicate (keep latest version)
	if len(records) == 0 {
		return records
	}
	
	var deduplicated []*parquet.Record
	deduplicated = append(deduplicated, records[0])
	
	for i := 1; i < len(records); i++ {
		if common.CompareBytes(records[i].Key, records[i-1].Key) != 0 {
			deduplicated = append(deduplicated, records[i])
		} else {
			// Keep the newer record (assuming higher timestamp means newer)
			if records[i].Timestamp > records[i-1].Timestamp {
				deduplicated[len(deduplicated)-1] = records[i]
			}
		}
	}
	
	return deduplicated
}

// writeCompactedFiles writes sorted records to new SSTable files
func (c *Compactor) writeCompactedFiles(ctx context.Context, records []*parquet.Record, level CompactionLevel) ([]string, error) {
	var outputFiles []string
	
	if len(records) == 0 {
		return outputFiles, nil
	}
	
	// Split records into appropriately sized files
	recordsPerFile := int(c.targetFileSize / 1024) // Rough estimate
	if recordsPerFile < 100 {
		recordsPerFile = 100
	}
	
	for i := 0; i < len(records); i += recordsPerFile {
		end := i + recordsPerFile
		if end > len(records) {
			end = len(records)
		}
		
		fileRecords := records[i:end]
		fileName := fmt.Sprintf("level_%d_%d_%d.parquet", level, time.Now().Unix(), i)
		filePath := fmt.Sprintf("sstables/%s", fileName)
		
		// Create parquet writer
		writer := parquet.NewWriter()
		
		// Write records
		for _, record := range fileRecords {
			err := writer.WriteRecord(record)
			if err != nil {
				return nil, fmt.Errorf("failed to write record: %w", err)
			}
		}
		
		// Get the written data
		data, err := writer.Finalize()
		if err != nil {
			return nil, fmt.Errorf("failed to finalize parquet file: %w", err)
		}
		
		// Write to storage
		err = c.storage.WriteBlock(ctx, filePath, data)
		if err != nil {
			return nil, fmt.Errorf("failed to write file %s: %w", filePath, err)
		}
		
		outputFiles = append(outputFiles, filePath)
		
		// Add to SSTable tracking
		keyRange := [2][]byte{fileRecords[0].Key, fileRecords[len(fileRecords)-1].Key}
		err = c.AddSSTable(filePath, level, uint64(len(data)), keyRange, uint64(len(fileRecords)))
		if err != nil {
			return nil, fmt.Errorf("failed to add sstable to tracking: %w", err)
		}
	}
	
	return outputFiles, nil
}

// updateSSTableTracking updates the SSTable metadata after compaction
func (c *Compactor) updateSSTableTracking(job *CompactionJob) error {
	// Remove input files from tracking
	for _, filePath := range job.InputFiles {
		err := c.RemoveSSTable(filePath, job.Level)
		if err != nil {
			return fmt.Errorf("failed to remove input sstable %s: %w", filePath, err)
		}
	}
	
	// Output files are already added in writeCompactedFiles
	return nil
}

// estimateFileSize estimates the size of a file
func (c *Compactor) estimateFileSize(filePath string) uint64 {
	// This is a rough estimate - in a real implementation,
	// we would track actual file sizes
	return c.targetFileSize
}

// GetCompactionStats returns statistics about compaction operations
func (c *Compactor) GetCompactionStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total_compactions":     c.totalCompactions,
		"total_bytes_compacted": c.totalBytesCompacted,
		"last_compaction_time":  c.lastCompactionTime,
		"active_jobs":           len(c.compactionJobs),
		"queue_size":            len(c.jobCh),
	}
	
	// Add level statistics
	levelStats := make(map[string]interface{})
	for level := Level0; level <= Level6; level++ {
		levelStats[fmt.Sprintf("level_%d_files", level)] = len(c.sstables[level])
		levelStats[fmt.Sprintf("level_%d_size", level)] = c.GetLevelSize(level)
		levelStats[fmt.Sprintf("level_%d_max_size", level)] = c.maxLevelSize[level]
	}
	stats["levels"] = levelStats
	
	return stats
}

// GetActiveJobs returns information about currently running compaction jobs
func (c *Compactor) GetActiveJobs() []*CompactionJob {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	var activeJobs []*CompactionJob
	for _, job := range c.compactionJobs {
		if job.Status == "running" || job.Status == "scheduled" {
			activeJobs = append(activeJobs, job)
		}
	}
	
	return activeJobs
}
