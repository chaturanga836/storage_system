// Package pb contains the protobuf generated interfaces for the storage system
// This is a simplified interface to resolve the TODO sections until proper protoc generation
package pb

import (
	"context"
)

// IngestionServiceServer is the server interface for Ingestion service
// Based on proto/storage/ingestion.proto
type IngestionServiceServer interface {
	// Ingest a single record
	IngestRecord(context.Context, interface{}) (interface{}, error)

	// Ingest multiple records in a batch
	IngestBatch(context.Context, interface{}) (interface{}, error)

	// Stream records for high-throughput ingestion
	IngestStream(interface{}) error

	// Get ingestion status and metrics
	GetIngestionStatus(context.Context, interface{}) (interface{}, error)

	// Health check
	HealthCheck(context.Context, interface{}) (interface{}, error)
}

// QueryServiceServer is the server interface for Query service
// Based on proto/storage/query.proto
type QueryServiceServer interface {
	// Execute a simple query
	Query(context.Context, interface{}) (interface{}, error)

	// Execute a streaming query for large result sets
	QueryStream(interface{}, interface{}) error

	// Get record by ID
	GetRecord(context.Context, interface{}) (interface{}, error)

	// Get multiple records by IDs
	GetRecords(context.Context, interface{}) (interface{}, error)

	// Execute aggregation queries
	Aggregate(context.Context, interface{}) (interface{}, error)

	// Get query execution plan
	ExplainQuery(context.Context, interface{}) (interface{}, error)

	// Health check
	HealthCheck(context.Context, interface{}) (interface{}, error)
}

// Placeholder registration functions
// These will be replaced with proper protoc-generated functions later

func RegisterIngestionServiceServer(server interface{}, srv IngestionServiceServer) {
	// TODO: Implement proper gRPC registration when protoc is available
	// For now, this is a placeholder to resolve compilation errors
}

func RegisterQueryServiceServer(server interface{}, srv QueryServiceServer) {
	// TODO: Implement proper gRPC registration when protoc is available
	// For now, this is a placeholder to resolve compilation errors
}
