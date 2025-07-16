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

	ingestionapi "storage-engine/internal/api/ingestion"
	"storage-engine/internal/config"
	"storage-engine/internal/pb"
	ingestionservice "storage-engine/internal/services/ingestion"
)

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

	// Register ingestion handler
	ingestionHandler := ingestionapi.NewHandler(ingestionService)

	// Register with proto-generated service
	pb.RegisterIngestionServiceServer(grpcServer, ingestionHandler)

	// Enable reflection for development
	reflection.Register(grpcServer)

	// Start listening
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Ingestion.Port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", cfg.Ingestion.Port, err)
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

	// Start serving
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

	<-ctx.Done()
	log.Println("👋 Ingestion Server stopped")
}
