package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"storage-engine/internal/api/query"
	"storage-engine/internal/config"
	"storage-engine/internal/services/query"
)

func main() {
	log.Println("🔍 Starting Query Server...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create query service
	queryService := query.NewService(cfg)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register query handler
	queryHandler := query.NewHandler(queryService)
	// TODO: Register with proto-generated service
	// pb.RegisterQueryServiceServer(grpcServer, queryHandler)

	// Enable reflection for development
	reflection.Register(grpcServer)

	// Start listening
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Query.Port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", cfg.Query.Port, err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("🛑 Shutting down Query Server...")
		grpcServer.GracefulStop()
		cancel()
	}()

	log.Printf("📡 Query Server listening on port %d", cfg.Query.Port)

	// Start serving
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

	<-ctx.Done()
	log.Println("👋 Query Server stopped")
}
