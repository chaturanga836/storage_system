package mvcc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionHistory_NewVersionHistory(t *testing.T) {
	capacity := 10
	vh := NewVersionHistory(capacity)

	require.NotNil(t, vh)
	assert.Equal(t, capacity, vh.capacity)
	assert.Equal(t, 0, vh.GetVersionCount())
}

func TestVersionHistory_AddVersion(t *testing.T) {
	vh := NewVersionHistory(3)

	// Add first version
	metadata1 := VersionMetadata{
		Version:   1,
		Timestamp: time.Now(),
		TxnID:     100,
		Deleted:   false,
		Size:      256,
		Checksum:  12345,
	}

	vh.AddVersion(metadata1)
	assert.Equal(t, 1, vh.GetVersionCount())

	// Add second version
	metadata2 := VersionMetadata{
		Version:   2,
		Timestamp: time.Now().Add(time.Second),
		TxnID:     101,
		Deleted:   false,
		Size:      512,
		Checksum:  23456,
	}

	vh.AddVersion(metadata2)
	assert.Equal(t, 2, vh.GetVersionCount())

	// Add third version
	metadata3 := VersionMetadata{
		Version:   3,
		Timestamp: time.Now().Add(2 * time.Second),
		TxnID:     102,
		Deleted:   false,
		Size:      128,
		Checksum:  34567,
	}

	vh.AddVersion(metadata3)
	assert.Equal(t, 3, vh.GetVersionCount())

	// Add fourth version - should remove the oldest
	metadata4 := VersionMetadata{
		Version:   4,
		Timestamp: time.Now().Add(3 * time.Second),
		TxnID:     103,
		Deleted:   false,
		Size:      64,
		Checksum:  45678,
	}

	vh.AddVersion(metadata4)
	assert.Equal(t, 3, vh.GetVersionCount()) // Still 3 due to capacity limit

	// Verify oldest version was removed
	_, err := vh.GetVersion(1)
	assert.Error(t, err)

	// Verify newest versions are still there
	version4, err := vh.GetVersion(4)
	require.NoError(t, err)
	assert.Equal(t, uint64(4), version4.Version)
}

func TestVersionHistory_GetVersion(t *testing.T) {
	vh := NewVersionHistory(5)

	metadata := VersionMetadata{
		Version:   42,
		Timestamp: time.Now(),
		TxnID:     200,
		Deleted:   false,
		Size:      1024,
		Checksum:  98765,
	}

	vh.AddVersion(metadata)

	// Get existing version
	result, err := vh.GetVersion(42)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), result.Version)
	assert.Equal(t, uint64(200), result.TxnID)
	assert.Equal(t, uint32(1024), result.Size)

	// Get non-existing version
	_, err = vh.GetVersion(999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version 999 not found")
}

func TestVersionHistory_GetLatestVersion(t *testing.T) {
	vh := NewVersionHistory(5)

	// Empty history
	_, err := vh.GetLatestVersion()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no versions available")

	// Add versions
	metadata1 := VersionMetadata{
		Version:   1,
		Timestamp: time.Now(),
		TxnID:     100,
	}
	vh.AddVersion(metadata1)

	metadata2 := VersionMetadata{
		Version:   2,
		Timestamp: time.Now().Add(time.Second),
		TxnID:     101,
	}
	vh.AddVersion(metadata2)

	// Get latest
	latest, err := vh.GetLatestVersion()
	require.NoError(t, err)
	assert.Equal(t, uint64(2), latest.Version)
	assert.Equal(t, uint64(101), latest.TxnID)
}

func TestVersionHistory_GetVersionsInRange(t *testing.T) {
	vh := NewVersionHistory(10)

	baseTime := time.Now()

	// Add versions with different timestamps
	metadata1 := VersionMetadata{
		Version:   1,
		Timestamp: baseTime,
		TxnID:     100,
	}
	vh.AddVersion(metadata1)

	metadata2 := VersionMetadata{
		Version:   2,
		Timestamp: baseTime.Add(5 * time.Second),
		TxnID:     101,
	}
	vh.AddVersion(metadata2)

	metadata3 := VersionMetadata{
		Version:   3,
		Timestamp: baseTime.Add(10 * time.Second),
		TxnID:     102,
	}
	vh.AddVersion(metadata3)

	metadata4 := VersionMetadata{
		Version:   4,
		Timestamp: baseTime.Add(15 * time.Second),
		TxnID:     103,
	}
	vh.AddVersion(metadata4)

	// Query range that includes versions 2 and 3
	start := baseTime.Add(2 * time.Second)
	end := baseTime.Add(12 * time.Second)

	versionsInRange := vh.GetVersionsInRange(start, end)
	assert.Len(t, versionsInRange, 2)

	assert.Equal(t, uint64(2), versionsInRange[0].Version)
	assert.Equal(t, uint64(3), versionsInRange[1].Version)
}

func TestVersionHistory_PurgeOldVersions(t *testing.T) {
	vh := NewVersionHistory(10)

	baseTime := time.Now()

	// Add old versions
	metadata1 := VersionMetadata{
		Version:   1,
		Timestamp: baseTime.Add(-2 * time.Hour),
		TxnID:     100,
	}
	vh.AddVersion(metadata1)

	metadata2 := VersionMetadata{
		Version:   2,
		Timestamp: baseTime.Add(-1 * time.Hour),
		TxnID:     101,
	}
	vh.AddVersion(metadata2)

	// Add recent version
	metadata3 := VersionMetadata{
		Version:   3,
		Timestamp: baseTime.Add(-5 * time.Minute),
		TxnID:     102,
	}
	vh.AddVersion(metadata3)

	assert.Equal(t, 3, vh.GetVersionCount())

	// Purge versions older than 30 minutes
	purged := vh.PurgeOldVersions(30 * time.Minute)
	assert.Equal(t, 2, purged)
	assert.Equal(t, 1, vh.GetVersionCount())

	// Only recent version should remain
	latest, err := vh.GetLatestVersion()
	require.NoError(t, err)
	assert.Equal(t, uint64(3), latest.Version)
}

func TestVersionHistory_GetStats(t *testing.T) {
	vh := NewVersionHistory(5)

	// Empty history stats
	stats := vh.GetStats()
	assert.Equal(t, 0, stats["count"])
	assert.Equal(t, 5, stats["capacity"])
	assert.Nil(t, stats["oldest"])
	assert.Nil(t, stats["newest"])
	assert.Equal(t, 0, stats["total_size"])
	assert.Equal(t, 0, stats["deleted_count"])

	baseTime := time.Now()

	// Add some versions
	metadata1 := VersionMetadata{
		Version:   1,
		Timestamp: baseTime,
		TxnID:     100,
		Deleted:   false,
		Size:      100,
		Checksum:  111,
	}
	vh.AddVersion(metadata1)

	metadata2 := VersionMetadata{
		Version:   2,
		Timestamp: baseTime.Add(time.Second),
		TxnID:     101,
		Deleted:   true,
		Size:      200,
		Checksum:  222,
	}
	vh.AddVersion(metadata2)

	metadata3 := VersionMetadata{
		Version:   3,
		Timestamp: baseTime.Add(2 * time.Second),
		TxnID:     102,
		Deleted:   false,
		Size:      300,
		Checksum:  333,
	}
	vh.AddVersion(metadata3)

	// Get stats with data
	stats = vh.GetStats()
	assert.Equal(t, 3, stats["count"])
	assert.Equal(t, 5, stats["capacity"])
	assert.Equal(t, baseTime, stats["oldest"])
	assert.Equal(t, baseTime.Add(2*time.Second), stats["newest"])
	assert.Equal(t, uint64(600), stats["total_size"])
	assert.Equal(t, 1, stats["deleted_count"])
	assert.Equal(t, float64(200), stats["avg_size"])
}

func TestVersionHistory_Serialize(t *testing.T) {
	vh := NewVersionHistory(3)

	metadata := VersionMetadata{
		Version:     1,
		Timestamp:   time.Now(),
		TxnID:       100,
		Deleted:     false,
		Size:        256,
		Checksum:    12345,
		CompactedAt: time.Now().Add(time.Hour),
	}

	vh.AddVersion(metadata)

	data, err := vh.Serialize()
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Basic validation - should have header data
	assert.True(t, len(data) >= 8) // At least header size
}

func TestVersionChain_NewVersionChain(t *testing.T) {
	key := []byte("test-key")
	chain := NewVersionChain(key)

	require.NotNil(t, chain)
	assert.Equal(t, key, chain.Key)
	assert.Len(t, chain.Versions, 0)
}

func TestVersionedValue_Creation(t *testing.T) {
	key := []byte("test-key")
	value := []byte("test-value")

	versionedValue := &VersionedValue{
		Key:       key,
		Value:     value,
		Version:   1,
		Timestamp: time.Now(),
		Deleted:   false,
		TxnID:     42,
	}

	assert.Equal(t, key, versionedValue.Key)
	assert.Equal(t, value, versionedValue.Value)
	assert.Equal(t, uint64(1), versionedValue.Version)
	assert.Equal(t, uint64(42), versionedValue.TxnID)
	assert.False(t, versionedValue.Deleted)
}

func TestHashBytesToString(t *testing.T) {
	data := []byte("test data")
	hash := HashBytesToString(data)

	assert.NotEmpty(t, hash)
	assert.Equal(t, 64, len(hash)) // SHA256 produces 64 char hex string

	// Same input should produce same hash
	hash2 := HashBytesToString(data)
	assert.Equal(t, hash, hash2)

	// Different input should produce different hash
	hash3 := HashBytesToString([]byte("different data"))
	assert.NotEqual(t, hash, hash3)
}
