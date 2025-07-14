package block

import (
	"context"
	"io"
)

// StorageBackend defines the interface for block storage operations
type StorageBackend interface {
	// Read reads data from the storage backend
	Read(ctx context.Context, path string) ([]byte, error)

	// Write writes data to the storage backend
	Write(ctx context.Context, path string, data []byte) error

	// ReadBlock reads a block of data from the storage backend (alias for Read)
	ReadBlock(ctx context.Context, path string) ([]byte, error)

	// WriteBlock writes a block of data to the storage backend (alias for Write)
	WriteBlock(ctx context.Context, path string, data []byte) error

	// Delete deletes data from the storage backend
	Delete(ctx context.Context, path string) error

	// Exists checks if a path exists in the storage backend
	Exists(ctx context.Context, path string) (bool, error)

	// List lists all paths matching a prefix
	List(ctx context.Context, prefix string) ([]string, error)

	// Size returns the size of data at the given path
	Size(ctx context.Context, path string) (int64, error)

	// OpenReader opens a reader for the given path
	OpenReader(ctx context.Context, path string) (io.ReadCloser, error)

	// OpenWriter opens a writer for the given path
	OpenWriter(ctx context.Context, path string) (io.WriteCloser, error)

	// Copy copies data from one path to another
	Copy(ctx context.Context, srcPath, dstPath string) error

	// Health checks the health of the storage backend
	Health(ctx context.Context) error

	// Close closes the storage backend
	Close() error
}

// LocalStorageBackend is a simple local filesystem implementation
type LocalStorageBackend struct {
	basePath string
}

// NewLocalStorageBackend creates a new local storage backend
func NewLocalStorageBackend(basePath string) *LocalStorageBackend {
	return &LocalStorageBackend{
		basePath: basePath,
	}
}

// Implementation methods would go here...
// For now, we'll just provide the interface
