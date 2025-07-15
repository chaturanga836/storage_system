package compaction

import (
	"sort"
	"time"
)

// CompactionStrategy defines how files are selected for compaction
type CompactionStrategy interface {
	SelectFilesForCompaction(level CompactionLevel, sstables []*SSTableInfo) []*SSTableInfo
	GetPriority(level CompactionLevel, sstables []*SSTableInfo) int
	GetName() string
}

// SizeTieredStrategy implements size-tiered compaction
type SizeTieredStrategy struct {
	minSSTableCount int
	maxSSTableCount int
	sizeRatio       float64
}

// NewSizeTieredStrategy creates a new size-tiered compaction strategy
func NewSizeTieredStrategy() *SizeTieredStrategy {
	return &SizeTieredStrategy{
		minSSTableCount: 4,
		maxSSTableCount: 32,
		sizeRatio:       1.2,
	}
}

func (s *SizeTieredStrategy) GetName() string {
	return "SizeTiered"
}

func (s *SizeTieredStrategy) SelectFilesForCompaction(level CompactionLevel, sstables []*SSTableInfo) []*SSTableInfo {
	if len(sstables) < s.minSSTableCount {
		return nil
	}

	// Sort by size (largest first for size-tiered)
	sorted := make([]*SSTableInfo, len(sstables))
	copy(sorted, sstables)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Size > sorted[j].Size
	})

	// Group SSTables by similar size
	var candidates []*SSTableInfo
	baseSize := sorted[0].Size

	for _, sstable := range sorted {
		ratio := float64(baseSize) / float64(sstable.Size)
		if ratio <= s.sizeRatio && len(candidates) < s.maxSSTableCount {
			candidates = append(candidates, sstable)
		}

		if len(candidates) >= s.minSSTableCount {
			break
		}
	}

	if len(candidates) >= s.minSSTableCount {
		return candidates
	}

	return nil
}

func (s *SizeTieredStrategy) GetPriority(level CompactionLevel, sstables []*SSTableInfo) int {
	if len(sstables) < s.minSSTableCount {
		return 0
	}

	// Higher priority for levels with more files
	priority := len(sstables) * 10

	// Boost priority for Level0 (memtable flushes)
	if level == Level0 {
		priority *= 2
	}

	return priority
}

// LeveledStrategy implements leveled compaction (similar to LevelDB/RocksDB)
type LeveledStrategy struct {
	targetFileSize   uint64
	maxBytesForLevel map[CompactionLevel]uint64
	overlappingRatio float64
}

// NewLeveledStrategy creates a new leveled compaction strategy
func NewLeveledStrategy() *LeveledStrategy {
	return &LeveledStrategy{
		targetFileSize: 64 * 1024 * 1024, // 64 MB
		maxBytesForLevel: map[CompactionLevel]uint64{
			Level0: 10 * 1024 * 1024,           // 10 MB
			Level1: 100 * 1024 * 1024,          // 100 MB
			Level2: 1024 * 1024 * 1024,         // 1 GB
			Level3: 10 * 1024 * 1024 * 1024,    // 10 GB
			Level4: 100 * 1024 * 1024 * 1024,   // 100 GB
			Level5: 1000 * 1024 * 1024 * 1024,  // 1 TB
			Level6: 10000 * 1024 * 1024 * 1024, // 10 TB
		},
		overlappingRatio: 0.1, // 10% overlap threshold
	}
}

func (l *LeveledStrategy) GetName() string {
	return "Leveled"
}

func (l *LeveledStrategy) SelectFilesForCompaction(level CompactionLevel, sstables []*SSTableInfo) []*SSTableInfo {
	if len(sstables) == 0 {
		return nil
	}

	if level == Level0 {
		// For Level0, compact all overlapping files
		return l.selectLevel0Files(sstables)
	}

	// For other levels, select files that exceed size threshold
	return l.selectLeveledFiles(level, sstables)
}

func (l *LeveledStrategy) selectLevel0Files(sstables []*SSTableInfo) []*SSTableInfo {
	if len(sstables) < 4 {
		return nil
	}

	// Sort by creation time (oldest first)
	sorted := make([]*SSTableInfo, len(sstables))
	copy(sorted, sstables)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CreationTime.Before(sorted[j].CreationTime)
	})

	// Select files that have overlapping key ranges
	var candidates []*SSTableInfo

	for _, sstable := range sorted {
		if len(candidates) == 0 {
			candidates = append(candidates, sstable)
			continue
		}

		// Check if this file overlaps with any selected file
		overlaps := false
		for _, candidate := range candidates {
			if l.keyRangesOverlap(sstable.KeyRange, candidate.KeyRange) {
				overlaps = true
				break
			}
		}

		if overlaps || len(candidates) < 4 {
			candidates = append(candidates, sstable)
		}

		// Limit to avoid too large compactions
		if len(candidates) >= 8 {
			break
		}
	}

	if len(candidates) >= 4 {
		return candidates
	}

	return nil
}

func (l *LeveledStrategy) selectLeveledFiles(level CompactionLevel, sstables []*SSTableInfo) []*SSTableInfo {
	// Calculate total size at this level
	var totalSize uint64
	for _, sstable := range sstables {
		totalSize += sstable.Size
	}

	maxSize := l.maxBytesForLevel[level]
	if totalSize <= maxSize {
		return nil
	}

	// Sort by access time (least recently used first)
	sorted := make([]*SSTableInfo, len(sstables))
	copy(sorted, sstables)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LastAccess.Before(sorted[j].LastAccess)
	})

	// Select oldest files until we reach target size reduction
	var candidates []*SSTableInfo
	var candidateSize uint64
	targetReduction := totalSize - maxSize

	for _, sstable := range sorted {
		candidates = append(candidates, sstable)
		candidateSize += sstable.Size

		if candidateSize >= targetReduction {
			break
		}
	}

	return candidates
}

func (l *LeveledStrategy) keyRangesOverlap(range1, range2 [2][]byte) bool {
	// Check if range1.max < range2.min or range2.max < range1.min
	// If either is true, they don't overlap

	if len(range1[1]) > 0 && len(range2[0]) > 0 {
		if compareBytes(range1[1], range2[0]) < 0 {
			return false
		}
	}

	if len(range2[1]) > 0 && len(range1[0]) > 0 {
		if compareBytes(range2[1], range1[0]) < 0 {
			return false
		}
	}

	return true
}

func (l *LeveledStrategy) GetPriority(level CompactionLevel, sstables []*SSTableInfo) int {
	var totalSize uint64
	for _, sstable := range sstables {
		totalSize += sstable.Size
	}

	maxSize := l.maxBytesForLevel[level]
	if totalSize <= maxSize {
		return 0
	}

	// Priority based on how much over the limit we are
	overageRatio := float64(totalSize) / float64(maxSize)
	priority := int(overageRatio * 100)

	// Higher priority for lower levels
	levelMultiplier := int(Level6 - level + 1)
	priority *= levelMultiplier

	return priority
}

// TimeWindowStrategy implements time-based compaction
type TimeWindowStrategy struct {
	windowSize        time.Duration
	maxFilesPerWindow int
	compactionDelay   time.Duration
}

// NewTimeWindowStrategy creates a new time-window compaction strategy
func NewTimeWindowStrategy(windowSize time.Duration) *TimeWindowStrategy {
	return &TimeWindowStrategy{
		windowSize:        windowSize,
		maxFilesPerWindow: 10,
		compactionDelay:   time.Hour, // Wait 1 hour before compacting
	}
}

func (t *TimeWindowStrategy) GetName() string {
	return "TimeWindow"
}

func (t *TimeWindowStrategy) SelectFilesForCompaction(level CompactionLevel, sstables []*SSTableInfo) []*SSTableInfo {
	if len(sstables) < 2 {
		return nil
	}

	now := time.Now()

	// Group files by time window
	windows := make(map[int64][]*SSTableInfo)
	for _, sstable := range sstables {
		// Only consider files older than compaction delay
		if now.Sub(sstable.CreationTime) < t.compactionDelay {
			continue
		}

		windowStart := sstable.CreationTime.Truncate(t.windowSize).Unix()
		windows[windowStart] = append(windows[windowStart], sstable)
	}

	// Find the window with the most files
	var bestWindow []*SSTableInfo
	maxFiles := 0

	for _, window := range windows {
		if len(window) > maxFiles && len(window) >= 2 {
			bestWindow = window
			maxFiles = len(window)
		}
	}

	if len(bestWindow) >= 2 {
		return bestWindow
	}

	return nil
}

func (t *TimeWindowStrategy) GetPriority(level CompactionLevel, sstables []*SSTableInfo) int {
	now := time.Now()
	oldFileCount := 0

	for _, sstable := range sstables {
		if now.Sub(sstable.CreationTime) > t.compactionDelay {
			oldFileCount++
		}
	}

	// Priority based on number of old files
	priority := oldFileCount * 5

	// Higher priority for files that are much older
	for _, sstable := range sstables {
		age := now.Sub(sstable.CreationTime)
		if age > t.compactionDelay*2 {
			priority += 10
		}
	}

	return priority
}

// AdaptiveStrategy combines multiple strategies based on workload patterns
type AdaptiveStrategy struct {
	strategies      []CompactionStrategy
	currentStrategy int
	lastSwitchTime  time.Time
	switchInterval  time.Duration

	// Workload metrics
	readHeavy       bool
	writeHeavy      bool
	compactionRatio float64
}

// NewAdaptiveStrategy creates a new adaptive compaction strategy
func NewAdaptiveStrategy() *AdaptiveStrategy {
	return &AdaptiveStrategy{
		strategies: []CompactionStrategy{
			NewLeveledStrategy(),
			NewSizeTieredStrategy(),
			NewTimeWindowStrategy(24 * time.Hour),
		},
		currentStrategy: 0,
		switchInterval:  time.Hour,
		lastSwitchTime:  time.Now(),
	}
}

func (a *AdaptiveStrategy) GetName() string {
	return "Adaptive"
}

func (a *AdaptiveStrategy) SelectFilesForCompaction(level CompactionLevel, sstables []*SSTableInfo) []*SSTableInfo {
	// Switch strategies periodically based on workload
	if time.Since(a.lastSwitchTime) > a.switchInterval {
		a.selectBestStrategy(level, sstables)
		a.lastSwitchTime = time.Now()
	}

	return a.strategies[a.currentStrategy].SelectFilesForCompaction(level, sstables)
}

func (a *AdaptiveStrategy) GetPriority(level CompactionLevel, sstables []*SSTableInfo) int {
	return a.strategies[a.currentStrategy].GetPriority(level, sstables)
}

func (a *AdaptiveStrategy) selectBestStrategy(level CompactionLevel, sstables []*SSTableInfo) {
	// Simple heuristics for strategy selection
	fileCount := len(sstables)

	if fileCount > 20 {
		// Many files: use size-tiered for better write performance
		a.currentStrategy = 1
	} else if fileCount < 5 {
		// Few files: use leveled for better read performance
		a.currentStrategy = 0
	} else {
		// Medium number: use time-window for predictable compaction
		a.currentStrategy = 2
	}
}

// Helper function for byte comparison
func compareBytes(a, b []byte) int {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	for i := 0; i < minLen; i++ {
		if a[i] < b[i] {
			return -1
		} else if a[i] > b[i] {
			return 1
		}
	}

	if len(a) < len(b) {
		return -1
	} else if len(a) > len(b) {
		return 1
	}

	return 0
}

// GetStrategyByName returns a strategy instance by name
func GetStrategyByName(name string) CompactionStrategy {
	switch name {
	case "SizeTiered":
		return NewSizeTieredStrategy()
	case "Leveled":
		return NewLeveledStrategy()
	case "TimeWindow":
		return NewTimeWindowStrategy(24 * time.Hour)
	case "Adaptive":
		return NewAdaptiveStrategy()
	default:
		return NewLeveledStrategy() // Default to leveled
	}
}
