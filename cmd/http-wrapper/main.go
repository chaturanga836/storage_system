package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"storage-engine/internal/config"
	"storage-engine/internal/services/ingestion"
)

// HTTPWrapper provides REST endpoints for the ingestion service
type HTTPWrapper struct {
	ingestionService *ingestion.Service
}

// NewHTTPWrapper creates a new HTTP wrapper for the ingestion service
func NewHTTPWrapper() (*HTTPWrapper, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %v", err)
	}

	// Create ingestion service
	ingestionSvc := ingestion.NewService(cfg)

	return &HTTPWrapper{ingestionService: ingestionSvc}, nil
}

// IngestRecordRequest represents the HTTP request for single record ingestion
type IngestRecordRequest struct {
	TenantID  string                 `json:"tenant_id"`
	RecordID  string                 `json:"record_id"`
	Data      map[string]interface{} `json:"data"`
	Timestamp string                 `json:"timestamp"`
	Metadata  map[string]string      `json:"metadata"`
}

// BatchIngestRequest represents the HTTP request for batch ingestion
type BatchIngestRequest struct {
	Records       []IngestRecordRequest `json:"records"`
	Transactional bool                  `json:"transactional"`
}

// setupRoutes configures the HTTP routes
func (h *HTTPWrapper) setupRoutes() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check
	r.GET("/health", h.healthCheck)

	// Ingestion endpoints
	r.POST("/api/v1/ingest/record", h.ingestRecord)
	r.POST("/api/v1/ingest/batch", h.ingestBatch)

	// Status endpoint
	r.GET("/api/v1/status", h.getStatus)

	return r
}

// healthCheck handles health check requests
func (h *HTTPWrapper) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "ingestion-http-wrapper",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
	})
}

// ingestRecord handles single record ingestion
func (h *HTTPWrapper) ingestRecord(c *gin.Context) {
	var req IngestRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Log the received request context
	log.Printf("📥 Received single record request: TenantID='%s', RecordID='%s', Data=%+v", req.TenantID, req.RecordID, req.Data)

	// Manual validation
	if req.TenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": "tenant_id is required",
		})
		return
	}
	if req.RecordID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": "record_id is required",
		})
		return
	}
	if req.Data == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": "data is required",
		})
		return
	}

	// Convert HTTP request to internal record format
	record, err := h.convertToInternalRecord(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to convert request",
			"details": err.Error(),
		})
		return
	}

	// Call ingestion service
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = h.ingestionService.IngestRecord(ctx, record)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Ingestion failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"record_id": req.RecordID,
		"tenant_id": req.TenantID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ingestBatch handles batch record ingestion
func (h *HTTPWrapper) ingestBatch(c *gin.Context) {
	var req BatchIngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Log the received batch request context
	log.Printf("📦 Received batch request: Transactional=%t, Records count=%d", req.Transactional, len(req.Records))
	for i, record := range req.Records {
		log.Printf("📦 Record %d: TenantID='%s', RecordID='%s', Data=%+v", i, record.TenantID, record.RecordID, record.Data)
	}

	// Manual validation
	log.Printf("🔍 Batch validation: received %d records", len(req.Records))
	if len(req.Records) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": "Records array cannot be empty",
		})
		return
	}

	// Validate each record manually
	for i, record := range req.Records {
		log.Printf("🔍 Validating record %d: TenantID='%s', RecordID='%s'", i, record.TenantID, record.RecordID)
		if record.TenantID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request format",
				"details": fmt.Sprintf("Record %d: tenant_id is required", i),
			})
			return
		}
		if record.RecordID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request format",
				"details": fmt.Sprintf("Record %d: record_id is required", i),
			})
			return
		}
		if record.Data == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request format",
				"details": fmt.Sprintf("Record %d: data is required", i),
			})
			return
		}
	}

	// Convert HTTP request to internal record format
	records, err := h.convertToInternalBatch(req)
	if err != nil {
		log.Printf("❌ Failed to convert batch: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to convert request",
			"details": err.Error(),
		})
		return
	}

	// Call ingestion service
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = h.ingestionService.IngestBatch(ctx, records)
	if err != nil {
		log.Printf("❌ Batch ingestion failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Batch ingestion failed",
			"details": err.Error(),
		})
		return
	}

	log.Printf("✅ Successfully ingested batch of %d records", len(req.Records))
	c.JSON(http.StatusOK, gin.H{
		"status":        "success",
		"records_count": len(req.Records),
		"transactional": req.Transactional,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	})
}

// getStatus handles status requests
func (h *HTTPWrapper) getStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "running",
		"service":   "ingestion-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"metrics": gin.H{
			"uptime": "active",
			"health": "good",
		},
	})
}

// Helper functions

// convertToInternalRecord converts HTTP request to internal record format
func (h *HTTPWrapper) convertToInternalRecord(req IngestRecordRequest) (map[string]interface{}, error) {
	// Parse timestamp if provided
	if req.Timestamp == "" {
		req.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	// Create internal record format that matches what the validation expects
	record := map[string]interface{}{
		"tenant_id": req.TenantID,
		"id":        req.RecordID,
		"timestamp": req.Timestamp,
		"data":      req.Data,
	}

	// Add metadata if provided
	if req.Metadata != nil {
		record["metadata"] = req.Metadata
	}

	return record, nil
}

// convertToInternalBatch converts HTTP batch request to internal format
func (h *HTTPWrapper) convertToInternalBatch(req BatchIngestRequest) ([]interface{}, error) {
	records := make([]interface{}, len(req.Records))

	for i, record := range req.Records {
		internalRecord, err := h.convertToInternalRecord(record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record %d: %v", i, err)
		}
		records[i] = internalRecord
	}

	return records, nil
}

func main() {
	// Create HTTP wrapper
	wrapper, err := NewHTTPWrapper()
	if err != nil {
		log.Fatalf("Failed to create HTTP wrapper: %v", err)
	}

	// Setup routes
	router := wrapper.setupRoutes()

	// Start HTTP server
	port := 8082
	log.Printf("🌐 HTTP REST API wrapper listening on port %d", port)
	log.Printf("📋 Test endpoints:")
	log.Printf("   GET  http://localhost:%d/health", port)
	log.Printf("   POST http://localhost:%d/api/v1/ingest/record", port)
	log.Printf("   POST http://localhost:%d/api/v1/ingest/batch", port)
	log.Printf("   GET  http://localhost:%d/api/v1/status", port)

	if err := router.Run(":" + strconv.Itoa(port)); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
