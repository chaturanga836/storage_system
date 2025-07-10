package common

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// GenerateID generates a unique identifier
func GenerateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// ParseDuration parses a duration string with support for common units
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// FormatBytes formats bytes into human readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// SanitizePath sanitizes a file path by removing dangerous characters
func SanitizePath(path string) string {
	// Remove dangerous characters and normalize
	sanitized := strings.ReplaceAll(path, "..", "")
	sanitized = strings.ReplaceAll(sanitized, "//", "/")
	return sanitized
}

// Min returns the minimum of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max returns the maximum of two integers
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MinInt64 returns the minimum of two int64 values
func MinInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// MaxInt64 returns the maximum of two int64 values
func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// Retry executes a function with exponential backoff retry logic
func Retry(attempts int, delay time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		if i < attempts-1 {
			time.Sleep(delay)
			delay *= 2 // Exponential backoff
		}
	}
	return err
}

// Contains checks if a slice contains a specific string
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RemoveDuplicates removes duplicate strings from a slice
func RemoveDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

// BatchProcess processes items in batches
func BatchProcess[T any](items []T, batchSize int, processFn func([]T) error) error {
	for i := 0; i < len(items); i += batchSize {
		end := Min(i+batchSize, len(items))
		batch := items[i:end]
		if err := processFn(batch); err != nil {
			return err
		}
	}
	return nil
}
