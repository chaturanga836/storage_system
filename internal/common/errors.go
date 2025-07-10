package common

import (
	"fmt"
)

// ErrorCode represents different types of errors in the system
type ErrorCode int

const (
	// General errors
	ErrInternal ErrorCode = iota + 1000
	ErrInvalidInput
	ErrNotFound
	ErrAlreadyExists
	ErrTimeout
	ErrUnavailable

	// Authentication errors
	ErrUnauthorized ErrorCode = iota + 2000
	ErrForbidden
	ErrInvalidToken
	ErrTokenExpired

	// Storage errors
	ErrStorageCorrupted ErrorCode = iota + 3000
	ErrStorageFull
	ErrStorageUnavailable
	ErrInvalidChecksum

	// WAL errors
	ErrWALCorrupted ErrorCode = iota + 4000
	ErrWALSegmentNotFound
	ErrWALReplayFailed

	// Schema errors
	ErrSchemaNotFound ErrorCode = iota + 5000
	ErrSchemaInvalid
	ErrSchemaVersionMismatch
	ErrSchemaEvolutionFailed

	// Index errors
	ErrIndexCorrupted ErrorCode = iota + 6000
	ErrIndexNotFound
	ErrIndexBuildFailed

	// Query errors
	ErrQueryInvalid ErrorCode = iota + 7000
	ErrQueryTimeout
	ErrQueryTooComplex
)

// StorageError represents an error in the storage system
type StorageError struct {
	Code    ErrorCode
	Message string
	Cause   error
	Context map[string]interface{}
}

// Error implements the error interface
func (e *StorageError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *StorageError) Unwrap() error {
	return e.Cause
}

// NewError creates a new StorageError
func NewError(code ErrorCode, message string) *StorageError {
	return &StorageError{
		Code:    code,
		Message: message,
		Context: make(map[string]interface{}),
	}
}

// NewErrorWithCause creates a new StorageError with an underlying cause
func NewErrorWithCause(code ErrorCode, message string, cause error) *StorageError {
	return &StorageError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// WithContext adds context to the error
func (e *StorageError) WithContext(key string, value interface{}) *StorageError {
	e.Context[key] = value
	return e
}

// IsErrorCode checks if an error has a specific error code
func IsErrorCode(err error, code ErrorCode) bool {
	if storageErr, ok := err.(*StorageError); ok {
		return storageErr.Code == code
	}
	return false
}

// Common error constructors
func ErrInternalError(message string) *StorageError {
	return NewError(ErrInternal, message)
}

func ErrInvalidInputError(message string) *StorageError {
	return NewError(ErrInvalidInput, message)
}

func ErrNotFoundError(message string) *StorageError {
	return NewError(ErrNotFound, message)
}

func ErrUnauthorizedError(message string) *StorageError {
	return NewError(ErrUnauthorized, message)
}

func ErrStorageCorruptedError(message string) *StorageError {
	return NewError(ErrStorageCorrupted, message)
}

func ErrSchemaNotFoundError(schemaID string) *StorageError {
	return NewError(ErrSchemaNotFound, fmt.Sprintf("schema not found: %s", schemaID))
}
