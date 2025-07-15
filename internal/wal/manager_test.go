package wal

import (
	"context"
	"storage-engine/internal/common"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_NewManager(t *testing.T) {
	tempDir := t.TempDir()

	config := Config{
		DataDir:     tempDir,
		SegmentSize: 1024 * 1024, // 1MB
		SyncPolicy:  SyncAlways,
	}

	manager, err := NewManager(config)
	require.NoError(t, err)
	require.NotNil(t, manager)

	defer manager.Close()

	// Check that directory was created
	assert.DirExists(t, tempDir)
}

func TestManager_AppendEntry(t *testing.T) {
	tempDir := t.TempDir()

	config := Config{
		DataDir:     tempDir,
		SegmentSize: 1024 * 1024,
		SyncPolicy:  SyncAlways,
	}

	manager, err := NewManager(config)
	require.NoError(t, err)
	defer manager.Close()

	// Create test entry
	entry := &Entry{
		Type:     EntryTypeInsert,
		TenantID: common.TenantID("test-tenant"),
		RecordID: common.RecordID{
			TenantID: common.TenantID("test-tenant"),
			EntityID: common.EntityID("test-record"),
			Version:  1,
		},
		Data: map[string]interface{}{"key": "value"},
		Schema: common.SchemaID{
			TenantID: common.TenantID("test-tenant"),
			Name:     "test-schema",
			Version:  1,
		},
		Timestamp: common.Now(),
	}

	// Append the entry
	ctx := context.Background()
	err = manager.Append(ctx, entry)
	require.NoError(t, err)

	// Verify that sequence ID was assigned
	assert.True(t, entry.SequenceID > 0)
}

func TestManager_GetStats(t *testing.T) {
	tempDir := t.TempDir()

	config := Config{
		DataDir:     tempDir,
		SegmentSize: 1024 * 1024,
		SyncPolicy:  SyncAlways,
	}

	manager, err := NewManager(config)
	require.NoError(t, err)
	defer manager.Close()

	// Get initial stats
	stats := manager.GetStats()
	assert.Equal(t, 1, stats.TotalSegments) // Should have one initial segment
	assert.Equal(t, int64(0), stats.TotalEntries)

	// Add some entries
	entry := &Entry{
		Type:     EntryTypeInsert,
		TenantID: common.TenantID("test-tenant"),
		RecordID: common.RecordID{
			TenantID: common.TenantID("test-tenant"),
			EntityID: common.EntityID("test-record"),
			Version:  1,
		},
		Data: map[string]interface{}{"key": "value"},
		Schema: common.SchemaID{
			TenantID: common.TenantID("test-tenant"),
			Name:     "test-schema",
			Version:  1,
		},
		Timestamp: common.Now(),
	}

	ctx := context.Background()
	err = manager.Append(ctx, entry)
	require.NoError(t, err)

	// Get updated stats
	stats = manager.GetStats()
	assert.Equal(t, 1, stats.TotalSegments)
	assert.Equal(t, int64(1), stats.TotalEntries)
}
