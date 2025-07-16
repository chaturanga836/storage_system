package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"storage-engine/internal/api/ingestion"
	"storage-engine/internal/config"
	"storage-engine/internal/pb/storage"
	ingestionservice "storage-engine/internal/services/ingestion"
)

// IngestionServer implements storage.IngestionServiceServer
type IngestionServer struct {
	storage.UnimplementedIngestionServiceServer
	service *ingestionservice.Service
}

// Implement the required methods with correct signatures
func (s *IngestionServer) IngestRecord(ctx context.Context, req *storage.IngestRecordRequest) (*storage.IngestRecordResponse, error) {
	// TODO: Implement logic using s.service
	return &storage.IngestRecordResponse{}, nil
}

func (s *IngestionServer) IngestBatch(ctx context.Context, req *storage.IngestBatchRequest) (*storage.IngestBatchResponse, error) {
	// TODO: Implement logic using s.service
	return &storage.IngestBatchResponse{}, nil
}

func (s *IngestionServer) IngestStream(stream storage.IngestionService_IngestStreamServer) error {
	// TODO: Implement streaming logic
	return nil
}

func (s *IngestionServer) GetIngestionStatus(ctx context.Context, req *storage.IngestionStatusRequest) (*storage.IngestionStatusResponse, error) {
	// TODO: Implement logic using s.service
	return &storage.IngestionStatusResponse{}, nil
}

func (s *IngestionServer) HealthCheck(ctx context.Context, req *storage.HealthCheckRequest) (*storage.HealthCheckResponse, error) {
	// TODO: Implement health check logic
	return &storage.HealthCheckResponse{}, nil
}

func startHealthServer(port int) {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	addr := fmt.Sprintf(":%d", port)
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Printf("Health HTTP server stopped: %v", err)
		}
	}()
}

func main() {
	log.Println("🚀 Starting Ingestion Server...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Start HTTP health server (on same port as gRPC for integration test compatibility)
	startHealthServer(cfg.Ingestion.Port)

	// Create ingestion service
	ingestionService := ingestionservice.NewService(cfg)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Create and register the new IngestionServer
	ingestionServer := ingestion.NewIngestionServer(ingestionService)
	storage.RegisterIngestionServiceServer(grpcServer, ingestionServer)

	// Enable reflection for development
	reflection.Register(grpcServer)

	// Start listening
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Ingestion.GRPC_Port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", cfg.Ingestion.GRPC_Port, err)
	} else {
		log.Printf("✅ gRPC server listening on port %d", cfg.Ingestion.GRPC_Port)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("🛑 Shutting down Ingestion Server...")
		grpcServer.GracefulStop()
		cancel()
	}()

	log.Printf("📡 Ingestion Server listening on port %d", cfg.Ingestion.Port)
	log.Printf("✅ Health HTTP server listening on port %d", cfg.Ingestion.Port)

	// Start serving
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

	<-ctx.Done()
	log.Println("👋 Ingestion Server stopped")
}
