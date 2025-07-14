package mvcc

import (
	"encoding/binary"
	"fmt"
	"sort"
	"sync"
	"time"

	"storage-engine/internal/common"
)

// VersionedValue represents a single version of a key-value pair
type VersionedValue struct {
	Key       []byte
	Value     []byte
	Version   uint64    // Monotonically increasing version number
	Timestamp time.Time // Wall clock time when the version was created
	Deleted   bool      // True if this version represents a deletion
	TxnID     uint64    // Transaction ID that created this version
}

// VersionChain maintains multiple versions of a single key
type VersionChain struct {
	Key      []byte
	Versions []*VersionedValue // Sorted by version (newest first)
	mu       sync.RWMutex
}

// NewVersionChain creates a new version chain for a key
func NewVersionChain(key []byte) *VersionChain {
	return &VersionChain{
		Key:      make([]byte, len(key)),
		Versions: make([]*VersionedValue, 0, 4),
	}
}

// AddVersion adds a new version to the chain
func (vc *VersionChain) AddVersion(value []byte, version uint64, txnID uint64, deleted bool) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	
	versionedValue := &VersionedValue{
		Key:       make([]byte, len(vc.Key)),
		Value:     make([]byte, len(value)),
		Version:   version,
		Timestamp: time.Now(),
		Deleted:   deleted,
		TxnID:     txnID,
	}
	
	copy(versionedValue.Key, vc.Key)
	copy(versionedValue.Value, value)
	
	// Insert in sorted order (newest first)
	insertPos := 0
	for i, v := range vc.Versions {
		if version > v.Version {
			insertPos = i
			break
		}
		insertPos = i + 1
	}
	
	// Insert at position
	vc.Versions = append(vc.Versions, nil)
	copy(vc.Versions[insertPos+1:], vc.Versions[insertPos:])
	vc.Versions[insertPos] = versionedValue
}

// GetVersion retrieves a specific version by version number
func (vc *VersionChain) GetVersion(version uint64) (*VersionedValue, error) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	
	for _, v := range vc.Versions {
		if v.Version == version {
			return v, nil
		}
	}
	
	return nil, common.ErrVersionNotFound
}

// GetLatestVersion retrieves the most recent version
func (vc *VersionChain) GetLatestVersion() (*VersionedValue, error) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	
	if len(vc.Versions) == 0 {
		return nil, common.ErrKeyNotFound
	}
	
	return vc.Versions[0], nil
}

// GetVersionAtTime retrieves the version that was current at a specific time
func (vc *VersionChain) GetVersionAtTime(timestamp time.Time) (*VersionedValue, error) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	
	// Find the latest version that was created before or at the timestamp
	for _, v := range vc.Versions {
		if !v.Timestamp.After(timestamp) {
			return v, nil
		}
	}
	
	return nil, common.ErrVersionNotFound
}

// PruneVersions removes old versions, keeping only the specified number
func (vc *VersionChain) PruneVersions(keepCount int) int {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	
	if len(vc.Versions) <= keepCount {
		return 0
	}
	
	removedCount := len(vc.Versions) - keepCount
	vc.Versions = vc.Versions[:keepCount]
	
	return removedCount
}

// GetVersionCount returns the number of versions in the chain
func (vc *VersionChain) GetVersionCount() int {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return len(vc.Versions)
}

// VersionResolver handles version resolution and conflict detection
type VersionResolver struct {
	mu               sync.RWMutex
	versionChains    map[string]*VersionChain // Key hash -> VersionChain
	nextVersion      uint64                   // Global version counter
	versionMu        sync.Mutex               // Protects nextVersion
	
	// Configuration
	maxVersionsPerKey int
	pruneInterval     time.Duration
	
	// Garbage collection
	lastPruneTime     time.Time
	totalVersions     uint64
	prunedVersions    uint64
}

// NewVersionResolver creates a new version resolver
func NewVersionResolver() *VersionResolver {
	return &VersionResolver{
		versionChains:     make(map[string]*VersionChain),
		nextVersion:       1,
		maxVersionsPerKey: 10,
		pruneInterval:     time.Hour,
		lastPruneTime:     time.Now(),
	}
}

// GetNextVersion atomically gets the next version number
func (vr *VersionResolver) GetNextVersion() uint64 {
	vr.versionMu.Lock()
	defer vr.versionMu.Unlock()
	
	version := vr.nextVersion
	vr.nextVersion++
	return version
}

// Put stores a new version of a key-value pair
func (vr *VersionResolver) Put(key, value []byte, txnID uint64) (uint64, error) {
	keyHash := common.HashBytesToString(key)
	version := vr.GetNextVersion()
	
	vr.mu.Lock()
	defer vr.mu.Unlock()
	
	chain, exists := vr.versionChains[keyHash]
	if !exists {
		chain = NewVersionChain(key)
		vr.versionChains[keyHash] = chain
	}
	
	chain.AddVersion(value, version, txnID, false)
	vr.totalVersions++
	
	// Trigger pruning if needed
	if time.Since(vr.lastPruneTime) > vr.pruneInterval {
		go vr.pruneOldVersions()
	}
	
	return version, nil
}

// Delete marks a key as deleted with a new version
func (vr *VersionResolver) Delete(key []byte, txnID uint64) (uint64, error) {
	keyHash := common.HashBytesToString(key)
	version := vr.GetNextVersion()
	
	vr.mu.Lock()
	defer vr.mu.Unlock()
	
	chain, exists := vr.versionChains[keyHash]
	if !exists {
		chain = NewVersionChain(key)
		vr.versionChains[keyHash] = chain
	}
	
	chain.AddVersion(nil, version, txnID, true)
	vr.totalVersions++
	
	return version, nil
}

// Get retrieves the latest version of a key
func (vr *VersionResolver) Get(key []byte) (*VersionedValue, error) {
	keyHash := common.HashBytesToString(key)
	
	vr.mu.RLock()
	defer vr.mu.RUnlock()
	
	chain, exists := vr.versionChains[keyHash]
	if !exists {
		return nil, common.ErrKeyNotFound
	}
	
	latest, err := chain.GetLatestVersion()
	if err != nil {
		return nil, err
	}
	
	if latest.Deleted {
		return nil, common.ErrKeyNotFound
	}
	
	return latest, nil
}

// GetVersion retrieves a specific version of a key
func (vr *VersionResolver) GetVersion(key []byte, version uint64) (*VersionedValue, error) {
	keyHash := common.HashBytesToString(key)
	
	vr.mu.RLock()
	defer vr.mu.RUnlock()
	
	chain, exists := vr.versionChains[keyHash]
	if !exists {
		return nil, common.ErrKeyNotFound
	}
	
	versionedValue, err := chain.GetVersion(version)
	if err != nil {
		return nil, err
	}
	
	if versionedValue.Deleted {
		return nil, common.ErrKeyNotFound
	}
	
	return versionedValue, nil
}

// GetAtTime retrieves the version of a key that was current at a specific time
func (vr *VersionResolver) GetAtTime(key []byte, timestamp time.Time) (*VersionedValue, error) {
	keyHash := common.HashBytesToString(key)
	
	vr.mu.RLock()
	defer vr.mu.RUnlock()
	
	chain, exists := vr.versionChains[keyHash]
	if !exists {
		return nil, common.ErrKeyNotFound
	}
	
	versionedValue, err := chain.GetVersionAtTime(timestamp)
	if err != nil {
		return nil, err
	}
	
	if versionedValue.Deleted {
		return nil, common.ErrKeyNotFound
	}
	
	return versionedValue, nil
}

// ListVersions returns all versions of a key
func (vr *VersionResolver) ListVersions(key []byte) ([]*VersionedValue, error) {
	keyHash := common.HashBytesToString(key)
	
	vr.mu.RLock()
	defer vr.mu.RUnlock()
	
	chain, exists := vr.versionChains[keyHash]
	if !exists {
		return nil, common.ErrKeyNotFound
	}
	
	chain.mu.RLock()
	defer chain.mu.RUnlock()
	
	versions := make([]*VersionedValue, len(chain.Versions))
	copy(versions, chain.Versions)
	
	return versions, nil
}

// HasConflict checks if there are conflicting versions for concurrent transactions
func (vr *VersionResolver) HasConflict(key []byte, baseTxnID uint64, currentTxnID uint64) (bool, error) {
	keyHash := common.HashBytesToString(key)
	
	vr.mu.RLock()
	defer vr.mu.RUnlock()
	
	chain, exists := vr.versionChains[keyHash]
	if !exists {
		return false, nil
	}
	
	chain.mu.RLock()
	defer chain.mu.RUnlock()
	
	// Check if there are any versions created by other transactions
	// between the base transaction and current transaction
	for _, version := range chain.Versions {
		if version.TxnID != baseTxnID && version.TxnID != currentTxnID {
			// Found a version created by a different transaction
			return true, nil
		}
	}
	
	return false, nil
}

// pruneOldVersions removes old versions to free memory
func (vr *VersionResolver) pruneOldVersions() {
	vr.mu.Lock()
	defer vr.mu.Unlock()
	
	totalPruned := 0
	
	for _, chain := range vr.versionChains {
		pruned := chain.PruneVersions(vr.maxVersionsPerKey)
		totalPruned += pruned
	}
	
	vr.prunedVersions += uint64(totalPruned)
	vr.lastPruneTime = time.Now()
}

// GetStats returns statistics about the version resolver
func (vr *VersionResolver) GetStats() map[string]interface{} {
	vr.mu.RLock()
	defer vr.mu.RUnlock()
	
	totalChains := len(vr.versionChains)
	totalVersionsCount := uint64(0)
	
	for _, chain := range vr.versionChains {
		totalVersionsCount += uint64(chain.GetVersionCount())
	}
	
	return map[string]interface{}{
		"total_chains":          totalChains,
		"total_versions":        totalVersionsCount,
		"next_version":          vr.nextVersion,
		"pruned_versions":       vr.prunedVersions,
		"last_prune_time":       vr.lastPruneTime,
		"max_versions_per_key":  vr.maxVersionsPerKey,
		"avg_versions_per_key":  float64(totalVersionsCount) / float64(totalChains),
	}
}

// Serialize converts version data to bytes for persistence
func (vr *VersionResolver) Serialize() ([]byte, error) {
	vr.mu.RLock()
	defer vr.mu.RUnlock()
	
	// Calculate total size needed
	totalSize := 8 + 8 // header: chain count + next version
	
	for keyHash, chain := range vr.versionChains {
		totalSize += 4 + len(keyHash) // key hash length + key hash
		totalSize += 4                // version count
		
		chain.mu.RLock()
		for _, version := range chain.Versions {
			totalSize += 4 + len(version.Key)   // key length + key
			totalSize += 4 + len(version.Value) // value length + value
			totalSize += 8                      // version number
			totalSize += 8                      // timestamp
			totalSize += 1                      // deleted flag
			totalSize += 8                      // transaction ID
		}
		chain.mu.RUnlock()
	}
	
	data := make([]byte, totalSize)
	offset := 0
	
	// Write header
	binary.LittleEndian.PutUint64(data[offset:], uint64(len(vr.versionChains)))
	offset += 8
	binary.LittleEndian.PutUint64(data[offset:], vr.nextVersion)
	offset += 8
	
	// Write version chains
	for keyHash, chain := range vr.versionChains {
		// Key hash
		binary.LittleEndian.PutUint32(data[offset:], uint32(len(keyHash)))
		offset += 4
		copy(data[offset:], keyHash)
		offset += len(keyHash)
		
		chain.mu.RLock()
		// Version count
		binary.LittleEndian.PutUint32(data[offset:], uint32(len(chain.Versions)))
		offset += 4
		
		// Versions
		for _, version := range chain.Versions {
			// Key
			binary.LittleEndian.PutUint32(data[offset:], uint32(len(version.Key)))
			offset += 4
			copy(data[offset:], version.Key)
			offset += len(version.Key)
			
			// Value
			binary.LittleEndian.PutUint32(data[offset:], uint32(len(version.Value)))
			offset += 4
			copy(data[offset:], version.Value)
			offset += len(version.Value)
			
			// Version number
			binary.LittleEndian.PutUint64(data[offset:], version.Version)
			offset += 8
			
			// Timestamp
			binary.LittleEndian.PutUint64(data[offset:], uint64(version.Timestamp.Unix()))
			offset += 8
			
			// Deleted flag
			if version.Deleted {
				data[offset] = 1
			} else {
				data[offset] = 0
			}
			offset += 1
			
			// Transaction ID
			binary.LittleEndian.PutUint64(data[offset:], version.TxnID)
			offset += 8
		}
		chain.mu.RUnlock()
	}
	
	return data, nil
}

// Deserialize restores version data from bytes
func (vr *VersionResolver) Deserialize(data []byte) error {
	if len(data) < 16 {
		return fmt.Errorf("invalid version data: too short")
	}
	
	vr.mu.Lock()
	defer vr.mu.Unlock()
	
	offset := 0
	
	// Read header
	chainCount := binary.LittleEndian.Uint64(data[offset:])
	offset += 8
	vr.nextVersion = binary.LittleEndian.Uint64(data[offset:])
	offset += 8
	
	// Read version chains
	vr.versionChains = make(map[string]*VersionChain)
	
	for i := uint64(0); i < chainCount; i++ {
		if offset+4 > len(data) {
			return fmt.Errorf("invalid version data: truncated key hash length")
		}
		
		// Key hash
		keyHashLen := binary.LittleEndian.Uint32(data[offset:])
		offset += 4
		
		if offset+int(keyHashLen) > len(data) {
			return fmt.Errorf("invalid version data: truncated key hash")
		}
		
		keyHash := string(data[offset : offset+int(keyHashLen)])
		offset += int(keyHashLen)
		
		if offset+4 > len(data) {
			return fmt.Errorf("invalid version data: truncated version count")
		}
		
		// Version count
		versionCount := binary.LittleEndian.Uint32(data[offset:])
		offset += 4
		
		// Create chain
		chain := &VersionChain{
			Versions: make([]*VersionedValue, 0, versionCount),
		}
		
		// Read versions
		for j := uint32(0); j < versionCount; j++ {
			if offset+4 > len(data) {
				return fmt.Errorf("invalid version data: truncated key length")
			}
			
			// Key
			keyLen := binary.LittleEndian.Uint32(data[offset:])
			offset += 4
			
			if offset+int(keyLen) > len(data) {
				return fmt.Errorf("invalid version data: truncated key")
			}
			
			key := make([]byte, keyLen)
			copy(key, data[offset:offset+int(keyLen)])
			offset += int(keyLen)
			
			if j == 0 {
				// Set chain key from first version
				chain.Key = make([]byte, len(key))
				copy(chain.Key, key)
			}
			
			if offset+4 > len(data) {
				return fmt.Errorf("invalid version data: truncated value length")
			}
			
			// Value
			valueLen := binary.LittleEndian.Uint32(data[offset:])
			offset += 4
			
			if offset+int(valueLen) > len(data) {
				return fmt.Errorf("invalid version data: truncated value")
			}
			
			value := make([]byte, valueLen)
			copy(value, data[offset:offset+int(valueLen)])
			offset += int(valueLen)
			
			if offset+17 > len(data) {
				return fmt.Errorf("invalid version data: truncated version metadata")
			}
			
			// Version metadata
			version := binary.LittleEndian.Uint64(data[offset:])
			offset += 8
			
			timestamp := time.Unix(int64(binary.LittleEndian.Uint64(data[offset:])), 0)
			offset += 8
			
			deleted := data[offset] == 1
			offset += 1
			
			txnID := binary.LittleEndian.Uint64(data[offset:])
			offset += 8
			
			versionedValue := &VersionedValue{
				Key:       key,
				Value:     value,
				Version:   version,
				Timestamp: timestamp,
				Deleted:   deleted,
				TxnID:     txnID,
			}
			
			chain.Versions = append(chain.Versions, versionedValue)
		}
		
		// Sort versions (newest first)
		sort.Slice(chain.Versions, func(i, j int) bool {
			return chain.Versions[i].Version > chain.Versions[j].Version
		})
		
		vr.versionChains[keyHash] = chain
	}
	
	return nil
}

// Clear removes all version data
func (vr *VersionResolver) Clear() {
	vr.mu.Lock()
	defer vr.mu.Unlock()
	
	vr.versionChains = make(map[string]*VersionChain)
	vr.nextVersion = 1
	vr.totalVersions = 0
	vr.prunedVersions = 0
}
