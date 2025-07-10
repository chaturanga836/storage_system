package block

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// Storage defines the interface for block storage operations
type Storage interface {
	// Reader operations
	Reader(ctx context.Context, path string) (io.ReadCloser, error)
	ReaderAt(ctx context.Context, path string) (io.ReaderAt, error)
	
	// Writer operations
	Writer(ctx context.Context, path string) (io.WriteCloser, error)
	
	// Metadata operations
	Stat(ctx context.Context, path string) (*Metadata, error)
	List(ctx context.Context, prefix string) ([]*Metadata, error)
	
	// Management operations
	Delete(ctx context.Context, path string) error
	Copy(ctx context.Context, src, dst string) error
	Move(ctx context.Context, src, dst string) error
	
	// Batch operations
	DeleteBatch(ctx context.Context, paths []string) error
	
	// Health and diagnostics
	Health(ctx context.Context) error
	Stats(ctx context.Context) (*Stats, error)
}

// Metadata represents file metadata
type Metadata struct {
	Path         string
	Size         int64
	ModTime      int64
	ETag         string
	ContentType  string
	StorageClass string
	Checksum     string
	CustomMetadata map[string]string
}

// Stats represents storage statistics
type Stats struct {
	TotalObjects int64
	TotalSize    int64
	AvailableSpace int64
	UsedSpace    int64
}

// Config holds configuration for block storage
type Config struct {
	Type     string            `yaml:"type" json:"type"` // local, s3, gcs, azure
	BaseDir  string            `yaml:"base_dir" json:"base_dir"`
	Options  map[string]string `yaml:"options" json:"options"`
}

// Factory creates storage instances based on configuration
type Factory struct{}

// NewFactory creates a new storage factory
func NewFactory() *Factory {
	return &Factory{}
}

// Create creates a new storage instance based on the configuration
func (f *Factory) Create(config Config) (Storage, error) {
	switch config.Type {
	case "local", "filesystem", "fs":
		return NewLocalFS(config)
	case "s3":
		return NewS3FS(config)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", config.Type)
	}
}

// StorageError represents storage-specific errors
type StorageError struct {
	Op   string
	Path string
	Err  error
}

func (e *StorageError) Error() string {
	return fmt.Sprintf("storage %s %s: %v", e.Op, e.Path, e.Err)
}

func (e *StorageError) Unwrap() error {
	return e.Err
}

// Common error variables
var (
	ErrNotFound      = &StorageError{Op: "stat", Err: fmt.Errorf("file not found")}
	ErrAlreadyExists = &StorageError{Op: "create", Err: fmt.Errorf("file already exists")}
	ErrPermission    = &StorageError{Op: "access", Err: fmt.Errorf("permission denied")}
	ErrInvalidPath   = &StorageError{Op: "validate", Err: fmt.Errorf("invalid path")}
)

// IsNotFound checks if an error indicates a file was not found
func IsNotFound(err error) bool {
	var storageErr *StorageError
	if errors.As(err, &storageErr) {
		return errors.Is(storageErr.Err, ErrNotFound.Err)
	}
	return false
}

// IsAlreadyExists checks if an error indicates a file already exists
func IsAlreadyExists(err error) bool {
	var storageErr *StorageError
	if errors.As(err, &storageErr) {
		return errors.Is(storageErr.Err, ErrAlreadyExists.Err)
	}
	return false
}
