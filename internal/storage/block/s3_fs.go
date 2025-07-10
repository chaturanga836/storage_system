package block

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3FS implements the Storage interface for Amazon S3
type S3FS struct {
	client *s3.Client
	bucket string
	prefix string
}

// NewS3FS creates a new S3 filesystem storage
func NewS3FS(cfg Config) (*S3FS, error) {
	bucket := cfg.Options["bucket"]
	if bucket == "" {
		return nil, fmt.Errorf("bucket is required for S3 storage")
	}

	region := cfg.Options["region"]
	if region == "" {
		region = "us-east-1" // Default region
	}

	// Load AWS configuration
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg)

	// Optional prefix for all operations
	prefix := cfg.Options["prefix"]

	return &S3FS{
		client: client,
		bucket: bucket,
		prefix: prefix,
	}, nil
}

// Reader returns a reader for the specified path
func (s3fs *S3FS) Reader(ctx context.Context, path string) (io.ReadCloser, error) {
	key := s3fs.getKey(path)

	output, err := s3fs.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3fs.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isS3NotFound(err) {
			return nil, &StorageError{Op: "get", Path: path, Err: ErrNotFound.Err}
		}
		return nil, &StorageError{Op: "get", Path: path, Err: err}
	}

	return output.Body, nil
}

// ReaderAt returns a ReaderAt for the specified path
func (s3fs *S3FS) ReaderAt(ctx context.Context, path string) (io.ReaderAt, error) {
	// S3 doesn't natively support ReaderAt, so we'll implement a wrapper
	return &s3ReaderAt{
		s3fs: s3fs,
		key:  s3fs.getKey(path),
		ctx:  ctx,
	}, nil
}

// Writer returns a writer for the specified path
func (s3fs *S3FS) Writer(ctx context.Context, path string) (io.WriteCloser, error) {
	return &s3Writer{
		s3fs: s3fs,
		key:  s3fs.getKey(path),
		ctx:  ctx,
	}, nil
}

// Stat returns metadata for the specified path
func (s3fs *S3FS) Stat(ctx context.Context, path string) (*Metadata, error) {
	key := s3fs.getKey(path)

	output, err := s3fs.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s3fs.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isS3NotFound(err) {
			return nil, &StorageError{Op: "head", Path: path, Err: ErrNotFound.Err}
		}
		return nil, &StorageError{Op: "head", Path: path, Err: err}
	}

	metadata := &Metadata{
		Path:           path,
		Size:           aws.ToInt64(output.ContentLength),
		ModTime:        output.LastModified.Unix(),
		ETag:           aws.ToString(output.ETag),
		ContentType:    aws.ToString(output.ContentType),
		CustomMetadata: make(map[string]string),
	}

	// Copy S3 metadata
	for k, v := range output.Metadata {
		metadata.CustomMetadata[k] = v
	}

	if output.StorageClass != "" {
		metadata.StorageClass = string(output.StorageClass)
	}

	return metadata, nil
}

// List returns metadata for all files with the specified prefix
func (s3fs *S3FS) List(ctx context.Context, prefix string) ([]*Metadata, error) {
	key := s3fs.getKey(prefix)

	var results []*Metadata
	paginator := s3.NewListObjectsV2Paginator(s3fs.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s3fs.bucket),
		Prefix: aws.String(key),
	})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, &StorageError{Op: "list", Path: prefix, Err: err}
		}

		for _, object := range output.Contents {
			// Convert back to relative path
			relPath := s3fs.getRelativePath(aws.ToString(object.Key))

			metadata := &Metadata{
				Path:         relPath,
				Size:         aws.ToInt64(object.Size),
				ModTime:      object.LastModified.Unix(),
				ETag:         aws.ToString(object.ETag),
				CustomMetadata: make(map[string]string),
			}

			if object.StorageClass != "" {
				metadata.StorageClass = string(object.StorageClass)
			}

			results = append(results, metadata)
		}
	}

	return results, nil
}

// Delete removes the file at the specified path
func (s3fs *S3FS) Delete(ctx context.Context, path string) error {
	key := s3fs.getKey(path)

	_, err := s3fs.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s3fs.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return &StorageError{Op: "delete", Path: path, Err: err}
	}

	return nil
}

// Copy copies a file from src to dst
func (s3fs *S3FS) Copy(ctx context.Context, src, dst string) error {
	srcKey := s3fs.getKey(src)
	dstKey := s3fs.getKey(dst)

	_, err := s3fs.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(s3fs.bucket),
		Key:        aws.String(dstKey),
		CopySource: aws.String(s3fs.bucket + "/" + srcKey),
	})
	if err != nil {
		if isS3NotFound(err) {
			return &StorageError{Op: "copy", Path: src, Err: ErrNotFound.Err}
		}
		return &StorageError{Op: "copy", Path: src, Err: err}
	}

	return nil
}

// Move moves a file from src to dst
func (s3fs *S3FS) Move(ctx context.Context, src, dst string) error {
	// Copy then delete
	if err := s3fs.Copy(ctx, src, dst); err != nil {
		return err
	}

	return s3fs.Delete(ctx, src)
}

// DeleteBatch removes multiple files
func (s3fs *S3FS) DeleteBatch(ctx context.Context, paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	// Convert paths to delete objects
	var objects []types.ObjectIdentifier
	for _, path := range paths {
		objects = append(objects, types.ObjectIdentifier{
			Key: aws.String(s3fs.getKey(path)),
		})
	}

	// S3 supports batch delete of up to 1000 objects
	const batchSize = 1000
	for i := 0; i < len(objects); i += batchSize {
		end := i + batchSize
		if end > len(objects) {
			end = len(objects)
		}

		batch := objects[i:end]
		_, err := s3fs.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(s3fs.bucket),
			Delete: &types.Delete{
				Objects: batch,
			},
		})
		if err != nil {
			return &StorageError{Op: "delete_batch", Path: "batch", Err: err}
		}
	}

	return nil
}

// Health checks the health of the storage
func (s3fs *S3FS) Health(ctx context.Context) error {
	// Try to list objects to test connectivity
	_, err := s3fs.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(s3fs.bucket),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		return fmt.Errorf("S3 health check failed: %w", err)
	}

	return nil
}

// Stats returns storage statistics
func (s3fs *S3FS) Stats(ctx context.Context) (*Stats, error) {
	// Note: This is a basic implementation that counts all objects
	// For large buckets, this could be expensive
	var totalObjects int64
	var totalSize int64

	prefix := s3fs.prefix
	paginator := s3.NewListObjectsV2Paginator(s3fs.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s3fs.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, &StorageError{Op: "stats", Path: "bucket", Err: err}
		}

		for _, object := range output.Contents {
			totalObjects++
			totalSize += aws.ToInt64(object.Size)
		}
	}

	return &Stats{
		TotalObjects: totalObjects,
		TotalSize:    totalSize,
		UsedSpace:    totalSize,
		// AvailableSpace is not applicable for S3
	}, nil
}

// Helper methods

func (s3fs *S3FS) getKey(path string) string {
	if s3fs.prefix == "" {
		return path
	}
	return s3fs.prefix + "/" + path
}

func (s3fs *S3FS) getRelativePath(key string) string {
	if s3fs.prefix == "" {
		return key
	}
	return strings.TrimPrefix(key, s3fs.prefix+"/")
}

func isS3NotFound(err error) bool {
	// Check for S3-specific not found errors
	// This is a simplified check; in production you might want more sophisticated error handling
	return strings.Contains(err.Error(), "NoSuchKey") || strings.Contains(err.Error(), "NotFound")
}

// s3ReaderAt implements io.ReaderAt for S3 objects
type s3ReaderAt struct {
	s3fs *S3FS
	key  string
	ctx  context.Context
}

func (s3ra *s3ReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	rangeHeader := fmt.Sprintf("bytes=%d-%d", off, off+int64(len(p))-1)

	output, err := s3ra.s3fs.client.GetObject(s3ra.ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3ra.s3fs.bucket),
		Key:    aws.String(s3ra.key),
		Range:  aws.String(rangeHeader),
	})
	if err != nil {
		return 0, err
	}
	defer output.Body.Close()

	return io.ReadFull(output.Body, p)
}

// s3Writer implements io.WriteCloser for S3 objects
type s3Writer struct {
	s3fs   *S3FS
	key    string
	ctx    context.Context
	buffer []byte
}

func (s3w *s3Writer) Write(p []byte) (n int, err error) {
	s3w.buffer = append(s3w.buffer, p...)
	return len(p), nil
}

func (s3w *s3Writer) Close() error {
	// Upload the entire buffer to S3
	_, err := s3w.s3fs.client.PutObject(s3w.ctx, &s3.PutObjectInput{
		Bucket: aws.String(s3w.s3fs.bucket),
		Key:    aws.String(s3w.key),
		Body:   strings.NewReader(string(s3w.buffer)),
	})
	return err
}
