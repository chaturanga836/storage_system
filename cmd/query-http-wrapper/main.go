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
	"storage-engine/internal/services/query"
)

// HTTPWrapper provides REST endpoints for the query service
type HTTPWrapper struct {
	queryService *query.Service
}

// NewHTTPWrapper creates a new HTTP wrapper for the query service
func NewHTTPWrapper() (*HTTPWrapper, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %v", err)
	}

	// Create query service
	querySvc := query.NewService(cfg)

	return &HTTPWrapper{queryService: querySvc}, nil
}

// QueryRequest represents the HTTP request for querying records
type QueryRequest struct {
	TenantID   string            `json:"tenant_id"`
	Filters    []QueryFilter     `json:"filters,omitempty"`
	Projection []string          `json:"projection,omitempty"`
	OrderBy    []OrderBy         `json:"order_by,omitempty"`
	Limit      int               `json:"limit,omitempty"`
	Offset     int               `json:"offset,omitempty"`
	TimeRange  *TimeRange        `json:"time_range,omitempty"`
	Options    map[string]string `json:"options,omitempty"`
}

// QueryFilter represents a filter condition
type QueryFilter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, gt, lt, gte, lte, in, contains
	Value    interface{} `json:"value"`
}

// OrderBy represents ordering specification
type OrderBy struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // asc, desc
}

// TimeRange represents a time range filter
type TimeRange struct {
	Start string `json:"start"` // RFC3339 format
	End   string `json:"end"`   // RFC3339 format
}

// GetRecordRequest represents the HTTP request for getting a record by ID
type GetRecordRequest struct {
	TenantID string `json:"tenant_id"`
	RecordID string `json:"record_id"`
}

// AggregateRequest represents the HTTP request for aggregation queries
type AggregateRequest struct {
	TenantID    string            `json:"tenant_id"`
	Aggregation string            `json:"aggregation"` // count, sum, avg, min, max
	Field       string            `json:"field,omitempty"`
	GroupBy     []string          `json:"group_by,omitempty"`
	Filters     []QueryFilter     `json:"filters,omitempty"`
	TimeRange   *TimeRange        `json:"time_range,omitempty"`
	Options     map[string]string `json:"options,omitempty"`
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

	// Query endpoints
	r.POST("/api/v1/query", h.executeQuery)
	r.GET("/api/v1/record/:tenant_id/:record_id", h.getRecord)
	r.POST("/api/v1/query/aggregate", h.executeAggregate)
	r.POST("/api/v1/query/explain", h.explainQuery)

	// Status endpoint
	r.GET("/api/v1/status", h.getStatus)

	return r
}

// healthCheck handles health check requests
func (h *HTTPWrapper) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "query-http-wrapper",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
	})
}

// executeQuery handles query execution requests
func (h *HTTPWrapper) executeQuery(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Log the received request context
	log.Printf("🔍 Received query request: TenantID='%s', Filters=%+v", req.TenantID, req.Filters)

	// Manual validation
	if req.TenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": "tenant_id is required",
		})
		return
	}

	// Convert HTTP request to internal query format
	query, err := h.convertToInternalQuery(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to convert request",
			"details": err.Error(),
		})
		return
	}

	// Call query service
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := h.queryService.ExecuteQuery(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Query execution failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"results":   result,
		"tenant_id": req.TenantID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// getRecord handles record retrieval by ID
func (h *HTTPWrapper) getRecord(c *gin.Context) {
	tenantID := c.Param("tenant_id")
	recordID := c.Param("record_id")

	log.Printf("📄 Received get record request: TenantID='%s', RecordID='%s'", tenantID, recordID)

	if tenantID == "" || recordID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid parameters",
			"details": "tenant_id and record_id are required",
		})
		return
	}

	// Call query service
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := h.queryService.GetRecord(ctx, map[string]interface{}{
		"tenant_id": tenantID,
		"record_id": recordID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Record retrieval failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"record":    result,
		"tenant_id": tenantID,
		"record_id": recordID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// executeAggregate handles aggregation query requests
func (h *HTTPWrapper) executeAggregate(c *gin.Context) {
	var req AggregateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	log.Printf("📊 Received aggregate request: TenantID='%s', Aggregation='%s'", req.TenantID, req.Aggregation)

	// Manual validation
	if req.TenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": "tenant_id is required",
		})
		return
	}
	if req.Aggregation == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": "aggregation is required",
		})
		return
	}

	// Convert HTTP request to internal aggregate format
	aggQuery, err := h.convertToInternalAggregate(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to convert request",
			"details": err.Error(),
		})
		return
	}

	// Call query service
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := h.queryService.ExecuteAggregate(ctx, aggQuery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Aggregation failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "success",
		"results":     result,
		"aggregation": req.Aggregation,
		"tenant_id":   req.TenantID,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	})
}

// explainQuery handles query plan explanation requests
func (h *HTTPWrapper) explainQuery(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	log.Printf("📋 Received explain query request: TenantID='%s'", req.TenantID)

	// For now, return a mock explanation
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"plan": gin.H{
			"query_type":     "scan",
			"estimated_cost": 100,
			"steps": []gin.H{
				{"step": 1, "operation": "index_scan", "table": "records"},
				{"step": 2, "operation": "filter", "condition": "tenant_id = " + req.TenantID},
				{"step": 3, "operation": "project", "fields": req.Projection},
			},
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// getStatus handles status requests
func (h *HTTPWrapper) getStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "running",
		"service":   "query-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"metrics": gin.H{
			"uptime":       "active",
			"health":       "good",
			"cache_status": "enabled",
			"query_count":  42, // Mock data
			"avg_latency":  "150ms",
		},
	})
}

// Helper functions

// convertToInternalQuery converts HTTP request to internal query format
func (h *HTTPWrapper) convertToInternalQuery(req QueryRequest) (map[string]interface{}, error) {
	query := map[string]interface{}{
		"tenant_id": req.TenantID,
		"filters":   req.Filters,
	}

	if req.Projection != nil {
		query["projection"] = req.Projection
	}
	if req.OrderBy != nil {
		query["order_by"] = req.OrderBy
	}
	if req.Limit > 0 {
		query["limit"] = req.Limit
	}
	if req.Offset > 0 {
		query["offset"] = req.Offset
	}
	if req.TimeRange != nil {
		query["time_range"] = req.TimeRange
	}
	if req.Options != nil {
		query["options"] = req.Options
	}

	return query, nil
}

// convertToInternalAggregate converts HTTP request to internal aggregate format
func (h *HTTPWrapper) convertToInternalAggregate(req AggregateRequest) (map[string]interface{}, error) {
	aggQuery := map[string]interface{}{
		"tenant_id":   req.TenantID,
		"aggregation": req.Aggregation,
	}

	if req.Field != "" {
		aggQuery["field"] = req.Field
	}
	if req.GroupBy != nil {
		aggQuery["group_by"] = req.GroupBy
	}
	if req.Filters != nil {
		aggQuery["filters"] = req.Filters
	}
	if req.TimeRange != nil {
		aggQuery["time_range"] = req.TimeRange
	}
	if req.Options != nil {
		aggQuery["options"] = req.Options
	}

	return aggQuery, nil
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
	port := 8083
	log.Printf("🌐 Query HTTP REST API wrapper listening on port %d", port)
	log.Printf("📋 Test endpoints:")
	log.Printf("   GET  http://localhost:%d/health", port)
	log.Printf("   POST http://localhost:%d/api/v1/query", port)
	log.Printf("   GET  http://localhost:%d/api/v1/record/{tenant_id}/{record_id}", port)
	log.Printf("   POST http://localhost:%d/api/v1/query/aggregate", port)
	log.Printf("   POST http://localhost:%d/api/v1/query/explain", port)
	log.Printf("   GET  http://localhost:%d/api/v1/status", port)

	if err := router.Run(":" + strconv.Itoa(port)); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
