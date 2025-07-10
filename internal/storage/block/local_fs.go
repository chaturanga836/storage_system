package block

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalFS implements the Storage interface for local filesystem
type LocalFS struct {
	baseDir string
}

// NewLocalFS creates a new local filesystem storage
func NewLocalFS(config Config) (*LocalFS, error) {
	baseDir := config.BaseDir
	if baseDir == "" {
		return nil, fmt.Errorf("base_dir is required for local filesystem storage")
	}

	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &LocalFS{
		baseDir: baseDir,
	}, nil
}

// Reader returns a reader for the specified path
func (lfs *LocalFS) Reader(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := lfs.getFullPath(path)
	
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &StorageError{Op: "open", Path: path, Err: ErrNotFound.Err}
		}
		return nil, &StorageError{Op: "open", Path: path, Err: err}
	}

	return file, nil
}

// ReaderAt returns a ReaderAt for the specified path
func (lfs *LocalFS) ReaderAt(ctx context.Context, path string) (io.ReaderAt, error) {
	fullPath := lfs.getFullPath(path)
	
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &StorageError{Op: "open", Path: path, Err: ErrNotFound.Err}
		}
		return nil, &StorageError{Op: "open", Path: path, Err: err}
	}

	return file, nil
}

// Writer returns a writer for the specified path
func (lfs *LocalFS) Writer(ctx context.Context, path string) (io.WriteCloser, error) {
	fullPath := lfs.getFullPath(path)
	
	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, &StorageError{Op: "mkdir", Path: path, Err: err}
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return nil, &StorageError{Op: "create", Path: path, Err: err}
	}

	return file, nil
}

// Stat returns metadata for the specified path
func (lfs *LocalFS) Stat(ctx context.Context, path string) (*Metadata, error) {
	fullPath := lfs.getFullPath(path)
	
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &StorageError{Op: "stat", Path: path, Err: ErrNotFound.Err}
		}
		return nil, &StorageError{Op: "stat", Path: path, Err: err}
	}

	return &Metadata{
		Path:    path,
		Size:    info.Size(),
		ModTime: info.ModTime().Unix(),
		CustomMetadata: make(map[string]string),
	}, nil
}

// List returns metadata for all files with the specified prefix
func (lfs *LocalFS) List(ctx context.Context, prefix string) ([]*Metadata, error) {
	fullPrefix := lfs.getFullPath(prefix)
	
	var results []*Metadata
	
	err := filepath.Walk(fullPrefix, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() {
			// Convert back to relative path
			relPath, err := filepath.Rel(lfs.baseDir, path)
			if err != nil {
				return err
			}
			
			// Normalize path separators
			relPath = filepath.ToSlash(relPath)
			
			results = append(results, &Metadata{
				Path:    relPath,
				Size:    info.Size(),
				ModTime: info.ModTime().Unix(),
				CustomMetadata: make(map[string]string),
			})
		}
		
		return nil
	})
	
	if err != nil {
		if os.IsNotExist(err) {
			return []*Metadata{}, nil // Return empty slice for non-existent prefix
		}
		return nil, &StorageError{Op: "list", Path: prefix, Err: err}
	}

	return results, nil
}

// Delete removes the file at the specified path
func (lfs *LocalFS) Delete(ctx context.Context, path string) error {
	fullPath := lfs.getFullPath(path)
	
	err := os.Remove(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &StorageError{Op: "delete", Path: path, Err: ErrNotFound.Err}
		}
		return &StorageError{Op: "delete", Path: path, Err: err}
	}

	return nil
}

// Copy copies a file from src to dst
func (lfs *LocalFS) Copy(ctx context.Context, src, dst string) error {
	srcPath := lfs.getFullPath(src)
	dstPath := lfs.getFullPath(dst)
	
	// Ensure destination directory exists
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return &StorageError{Op: "mkdir", Path: dst, Err: err}
	}

	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &StorageError{Op: "copy", Path: src, Err: ErrNotFound.Err}
		}
		return &StorageError{Op: "copy", Path: src, Err: err}
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return &StorageError{Op: "copy", Path: dst, Err: err}
	}
	defer dstFile.Close()

	// Copy data
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return &StorageError{Op: "copy", Path: dst, Err: err}
	}

	return nil
}

// Move moves a file from src to dst
func (lfs *LocalFS) Move(ctx context.Context, src, dst string) error {
	srcPath := lfs.getFullPath(src)
	dstPath := lfs.getFullPath(dst)
	
	// Ensure destination directory exists
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return &StorageError{Op: "mkdir", Path: dst, Err: err}
	}

	err := os.Rename(srcPath, dstPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &StorageError{Op: "move", Path: src, Err: ErrNotFound.Err}
		}
		return &StorageError{Op: "move", Path: src, Err: err}
	}

	return nil
}

// DeleteBatch removes multiple files
func (lfs *LocalFS) DeleteBatch(ctx context.Context, paths []string) error {
	var errors []error
	
	for _, path := range paths {
		if err := lfs.Delete(ctx, path); err != nil {
			// Continue with other deletions, collect errors
			errors = append(errors, err)
		}
	}
	
	if len(errors) > 0 {
		// Return first error for now
		return errors[0]
	}
	
	return nil
}

// Health checks the health of the storage
func (lfs *LocalFS) Health(ctx context.Context) error {
	// Check if base directory is accessible
	info, err := os.Stat(lfs.baseDir)
	if err != nil {
		return fmt.Errorf("base directory not accessible: %w", err)
	}
	
	if !info.IsDir() {
		return fmt.Errorf("base path is not a directory")
	}
	
	// Try to create a temporary file to test write permissions
	tempFile := filepath.Join(lfs.baseDir, ".health_check_temp")
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("cannot write to storage: %w", err)
	}
	file.Close()
	os.Remove(tempFile)
	
	return nil
}

// Stats returns storage statistics
func (lfs *LocalFS) Stats(ctx context.Context) (*Stats, error) {
	var totalObjects int64
	var totalSize int64
	
	err := filepath.Walk(lfs.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() {
			totalObjects++
			totalSize += info.Size()
		}
		
		return nil
	})
	
	if err != nil {
		return nil, &StorageError{Op: "stats", Path: lfs.baseDir, Err: err}
	}
	
	// Get available space
	// Note: This is a simplified implementation
	// In production, you might want to use syscalls to get actual disk space
	
	return &Stats{
		TotalObjects: totalObjects,
		TotalSize:    totalSize,
		UsedSpace:    totalSize,
		// AvailableSpace would require platform-specific syscalls
	}, nil
}

// getFullPath converts a relative path to a full path within the base directory
func (lfs *LocalFS) getFullPath(path string) string {
	// Clean the path to prevent directory traversal attacks
	cleanPath := filepath.Clean(path)
	
	// Remove leading slash if present
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	
	// Join with base directory
	return filepath.Join(lfs.baseDir, cleanPath)
}
