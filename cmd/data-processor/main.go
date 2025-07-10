package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"storage-engine/internal/config"
	"storage-engine/internal/services/data_processing"
)

func main() {
	log.Println("⚙️ Starting Data Processor...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create data processing service
	processor := data_processing.NewService(cfg)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("🛑 Shutting down Data Processor...")
		cancel()
	}()

	// Start background processes
	go func() {
		log.Println("🔄 Starting WAL replay process...")
		if err := processor.StartWALReplay(ctx); err != nil {
			log.Printf("❌ WAL replay error: %v", err)
		}
	}()

	go func() {
		log.Println("💾 Starting memtable flush process...")
		if err := processor.StartMemtableFlush(ctx); err != nil {
			log.Printf("❌ Memtable flush error: %v", err)
		}
	}()

	go func() {
		log.Println("🗜️ Starting compaction process...")
		if err := processor.StartCompaction(ctx); err != nil {
			log.Printf("❌ Compaction error: %v", err)
		}
	}()

	go func() {
		log.Println("📊 Starting index maintenance...")
		if err := processor.StartIndexMaintenance(ctx); err != nil {
			log.Printf("❌ Index maintenance error: %v", err)
		}
	}()

	log.Println("✅ Data Processor started successfully")

	// Keep the process running
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("👋 Data Processor stopped")
			return
		case <-ticker.C:
			// Periodic health check or metrics reporting
			log.Println("💓 Data Processor health check")
		}
	}
}
