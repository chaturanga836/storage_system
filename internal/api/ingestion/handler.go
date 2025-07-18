package ingestion

import (
	"context"
	"fmt"
	"io"
	"log"

	"google.golang.org/protobuf/types/known/timestamppb"

	"storage-engine/internal/pb/storage"
	"storage-engine/internal/services/ingestion"
)

// IngestionServer implements storage.IngestionServiceServer
type IngestionServer struct {
	storage.UnimplementedIngestionServiceServer
	service *ingestion.Service
}

// NewIngestionServer creates a new ingestion server
func NewIngestionServer(service *ingestion.Service) *IngestionServer {
	return &IngestionServer{
		service: service,
	}
}

// IngestRecord handles single record ingestion
func (s *IngestionServer) IngestRecord(ctx context.Context, req *storage.IngestRecordRequest) (*storage.IngestRecordResponse, error) {
	log.Println("📥 Handling IngestRecord request")

	// 1. Validate request
	if req == nil || req.Record == nil {
		return nil, fmt.Errorf("missing record in request")
	}

	// 2. Call service logic (replace with your actual implementation)
	err := s.service.IngestRecord(ctx, req.Record)
	if err != nil {
		return nil, err
	}

	// 3. Convert result to protobuf response (replace with actual mapping)
	resp := &storage.IngestRecordResponse{
		Status:    &storage.Status{Code: 0, Message: "success"},
		RecordId:  "example-id",      // replace with result.RecordId
		Version:   1,                 // replace with result.Version
		Timestamp: timestamppb.Now(), // or set to current time
	}
	return resp, nil
}

// IngestBatch handles batch ingestion
func (s *IngestionServer) IngestBatch(ctx context.Context, req *storage.IngestBatchRequest) (*storage.IngestBatchResponse, error) {
	log.Println("📦 Handling IngestBatch request")

	// 1. Validate request
	if req == nil || len(req.Records) == 0 {
		return nil, fmt.Errorf("no records provided for batch ingestion")
	}

	// 2. Convert []*storage.DataRecord to []interface{} (map[string]interface{})
	var records []interface{}
	for _, r := range req.Records {
		if r == nil {
			continue
		}
		record := map[string]interface{}{
			"id":        r.Id,
			"timestamp": r.Timestamp,
			"data":      r.Data, // adjust if Data is a message or map
		}
		records = append(records, record)
	}

	// 3. Call service logic
	err := s.service.IngestBatch(ctx, records)
	if err != nil {
		return nil, err
	}

	// 4. Map results to protobuf response (replace with actual mapping)
	resp := &storage.IngestBatchResponse{
		Status:          &storage.Status{Code: 0, Message: "success"},
		SuccessfulCount: int32(len(req.Records)),   // replace with results.SuccessfulCount
		FailedCount:     0,                         // replace with results.FailedCount
		Results:         []*storage.IngestResult{}, // fill with actual results
		Timestamp:       nil,                       // set to current time or batch timestamp
	}
	return resp, nil
}

// IngestStream handles streaming ingestion
func (s *IngestionServer) IngestStream(stream storage.IngestionService_IngestStreamServer) error {
	log.Println("🌊 Handling IngestStream request")

	// Example streaming loop
	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break // client closed stream
			}
			return err
		}

		// Log the received message for debugging
		log.Printf("Received stream message: %+v", msg)

		// TODO: Map msg fields to record as per your proto definition
		//record := map[string]interface{}{}
		// Example: If msg has a field 'DataRecord', use its fields
		// if msg.DataRecord != nil {
		//     record["id"] = msg.DataRecord.Id
		//     record["timestamp"] = msg.DataRecord.Timestamp
		//     record["data"] = msg.DataRecord.Data
		// }

		// Call service logic
		//err = s.service.IngestRecord(stream.Context(), record)

		// Build gRPC response (replace with actual proto fields)
		resp := &storage.IngestStreamResponse{
			// Fill with actual fields from proto
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

// GetIngestionStatus returns ingestion metrics and status
func (s *IngestionServer) GetIngestionStatus(ctx context.Context, req *storage.IngestionStatusRequest) (*storage.IngestionStatusResponse, error) {
	log.Println("📊 Handling GetIngestionStatus request")

	// TODO: Call service to get metrics/status
	resp := &storage.IngestionStatusResponse{
		Status: &storage.Status{Code: 0, Message: "healthy"},
		Metrics: &storage.IngestionMetrics{
			TotalRecordsIngested: 1000, // example value
			RecordsPerSecond:     100,  // example value
			BytesPerSecond:       1024, // example value
			ActiveStreams:        1,    // example value
			PendingRecords:       0,
			FailedRecords:        0,
			AvgLatencyMs:         5.0,
			P99LatencyMs:         10.0,
			WalSizeBytes:         2048,
			MemtableCount:        1,
		},
		Timestamp: nil, // set to current time
	}
	return resp, nil
}

// HealthCheck handles health check requests
func (s *IngestionServer) HealthCheck(ctx context.Context, req *storage.HealthCheckRequest) (*storage.HealthCheckResponse, error) {
	log.Println("💓 Handling HealthCheck request")

	// TODO: Check dependencies, DB, etc.
	resp := &storage.HealthCheckResponse{
		Status:    storage.HealthCheckResponse_SERVING,
		Message:   "OK",
		Timestamp: timestamppb.Now(), // set to current time
	}
	return resp, nil
}
