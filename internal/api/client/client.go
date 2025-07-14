package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"storage-engine/internal/schema"
)

// Client provides a high-level API client for the storage system
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	timeout    time.Duration
	retryCount int
}

// ClientConfig holds client configuration
type ClientConfig struct {
	BaseURL    string
	APIKey     string
	Timeout    time.Duration
	RetryCount int
}

// DefaultClientConfig returns default client configuration
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		BaseURL:    "http://localhost:8080",
		Timeout:    30 * time.Second,
		RetryCount: 3,
	}
}

// NewClient creates a new storage system client
func NewClient(config *ClientConfig) *Client {
	if config == nil {
		config = DefaultClientConfig()
	}

	return &Client{
		baseURL: strings.TrimSuffix(config.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		apiKey:     config.APIKey,
		timeout:    config.Timeout,
		retryCount: config.RetryCount,
	}
}

// Table operations

// CreateTable creates a new table with the given schema
func (c *Client) CreateTable(ctx context.Context, tableSchema *schema.TableSchema) error {
	requestBody := &CreateTableRequest{
		Schema: tableSchema,
	}

	var response CreateTableResponse
	err := c.doRequest(ctx, "POST", "/api/v1/tables", requestBody, &response)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

// GetTable retrieves table information
func (c *Client) GetTable(ctx context.Context, tableName string) (*TableInfo, error) {
	path := fmt.Sprintf("/api/v1/tables/%s", url.PathEscape(tableName))
	
	var response GetTableResponse
	err := c.doRequest(ctx, "GET", path, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get table: %w", err)
	}

	return response.Table, nil
}

// ListTables lists all tables
func (c *Client) ListTables(ctx context.Context) ([]*TableInfo, error) {
	var response ListTablesResponse
	err := c.doRequest(ctx, "GET", "/api/v1/tables", nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	return response.Tables, nil
}

// DropTable drops a table
func (c *Client) DropTable(ctx context.Context, tableName string) error {
	path := fmt.Sprintf("/api/v1/tables/%s", url.PathEscape(tableName))
	
	var response DropTableResponse
	err := c.doRequest(ctx, "DELETE", path, nil, &response)
	if err != nil {
		return fmt.Errorf("failed to drop table: %w", err)
	}

	return nil
}

// Data operations

// InsertRecord inserts a single record into a table
func (c *Client) InsertRecord(ctx context.Context, tableName string, record map[string]interface{}) (*InsertResult, error) {
	return c.InsertRecords(ctx, tableName, []map[string]interface{}{record})
}

// InsertRecords inserts multiple records into a table
func (c *Client) InsertRecords(ctx context.Context, tableName string, records []map[string]interface{}) (*InsertResult, error) {
	path := fmt.Sprintf("/api/v1/tables/%s/records", url.PathEscape(tableName))
	
	requestBody := &InsertRecordsRequest{
		Records: records,
	}

	var response InsertRecordsResponse
	err := c.doRequest(ctx, "POST", path, requestBody, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to insert records: %w", err)
	}

	return response.Result, nil
}

// QueryRecords queries records from a table
func (c *Client) QueryRecords(ctx context.Context, request *QueryRequest) (*QueryResult, error) {
	path := fmt.Sprintf("/api/v1/tables/%s/query", url.PathEscape(request.TableName))
	
	var response QueryRecordsResponse
	err := c.doRequest(ctx, "POST", path, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}

	return response.Result, nil
}

// GetRecord retrieves a single record by ID
func (c *Client) GetRecord(ctx context.Context, tableName, recordID string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/api/v1/tables/%s/records/%s", url.PathEscape(tableName), url.PathEscape(recordID))
	
	var response GetRecordResponse
	err := c.doRequest(ctx, "GET", path, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}

	return response.Record, nil
}

// UpdateRecord updates a record
func (c *Client) UpdateRecord(ctx context.Context, tableName, recordID string, updates map[string]interface{}) error {
	path := fmt.Sprintf("/api/v1/tables/%s/records/%s", url.PathEscape(tableName), url.PathEscape(recordID))
	
	requestBody := &UpdateRecordRequest{
		Updates: updates,
	}

	var response UpdateRecordResponse
	err := c.doRequest(ctx, "PUT", path, requestBody, &response)
	if err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}

	return nil
}

// DeleteRecord deletes a record
func (c *Client) DeleteRecord(ctx context.Context, tableName, recordID string) error {
	path := fmt.Sprintf("/api/v1/tables/%s/records/%s", url.PathEscape(tableName), url.PathEscape(recordID))
	
	var response DeleteRecordResponse
	err := c.doRequest(ctx, "DELETE", path, nil, &response)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	return nil
}

// Schema operations

// GetSchema retrieves the schema for a table
func (c *Client) GetSchema(ctx context.Context, tableName string) (*schema.TableSchema, error) {
	path := fmt.Sprintf("/api/v1/schemas/%s", url.PathEscape(tableName))
	
	var response GetSchemaResponse
	err := c.doRequest(ctx, "GET", path, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	return response.Schema, nil
}

// UpdateSchema updates the schema for a table (schema evolution)
func (c *Client) UpdateSchema(ctx context.Context, tableName string, newSchema *schema.TableSchema) error {
	path := fmt.Sprintf("/api/v1/schemas/%s", url.PathEscape(tableName))
	
	requestBody := &UpdateSchemaRequest{
		Schema: newSchema,
	}

	var response UpdateSchemaResponse
	err := c.doRequest(ctx, "PUT", path, requestBody, &response)
	if err != nil {
		return fmt.Errorf("failed to update schema: %w", err)
	}

	return nil
}

// Index operations

// CreateIndex creates an index on a table
func (c *Client) CreateIndex(ctx context.Context, indexDef *IndexDefinition) error {
	requestBody := &CreateIndexRequest{
		Index: indexDef,
	}

	var response CreateIndexResponse
	err := c.doRequest(ctx, "POST", "/api/v1/indexes", requestBody, &response)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

// ListIndexes lists indexes for a table
func (c *Client) ListIndexes(ctx context.Context, tableName string) ([]*IndexInfo, error) {
	path := fmt.Sprintf("/api/v1/tables/%s/indexes", url.PathEscape(tableName))
	
	var response ListIndexesResponse
	err := c.doRequest(ctx, "GET", path, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}

	return response.Indexes, nil
}

// DropIndex drops an index
func (c *Client) DropIndex(ctx context.Context, indexName string) error {
	path := fmt.Sprintf("/api/v1/indexes/%s", url.PathEscape(indexName))
	
	var response DropIndexResponse
	err := c.doRequest(ctx, "DELETE", path, nil, &response)
	if err != nil {
		return fmt.Errorf("failed to drop index: %w", err)
	}

	return nil
}

// System operations

// GetStatus gets system status
func (c *Client) GetStatus(ctx context.Context) (*SystemStatus, error) {
	var response GetStatusResponse
	err := c.doRequest(ctx, "GET", "/api/v1/status", nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	return response.Status, nil
}

// GetMetrics gets system metrics
func (c *Client) GetMetrics(ctx context.Context) (*SystemMetrics, error) {
	var response GetMetricsResponse
	err := c.doRequest(ctx, "GET", "/api/v1/metrics", nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	return response.Metrics, nil
}

// TriggerCompaction manually triggers compaction
func (c *Client) TriggerCompaction(ctx context.Context, tableName string) error {
	path := fmt.Sprintf("/api/v1/tables/%s/compact", url.PathEscape(tableName))
	
	var response TriggerCompactionResponse
	err := c.doRequest(ctx, "POST", path, nil, &response)
	if err != nil {
		return fmt.Errorf("failed to trigger compaction: %w", err)
	}

	return nil
}

// Batch operations

// BatchInsert performs batch insert operations
func (c *Client) BatchInsert(ctx context.Context, operations []*BatchInsertOperation) (*BatchResult, error) {
	requestBody := &BatchInsertRequest{
		Operations: operations,
	}

	var response BatchInsertResponse
	err := c.doRequest(ctx, "POST", "/api/v1/batch/insert", requestBody, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to perform batch insert: %w", err)
	}

	return response.Result, nil
}

// Streaming operations

// StreamRecords streams records from a table (server-sent events)
func (c *Client) StreamRecords(ctx context.Context, tableName string, filters map[string]interface{}) (<-chan *StreamRecord, <-chan error) {
	recordChan := make(chan *StreamRecord, 100)
	errorChan := make(chan error, 1)

	go func() {
		defer close(recordChan)
		defer close(errorChan)

		path := fmt.Sprintf("/api/v1/tables/%s/stream", url.PathEscape(tableName))
		
		// Add filters as query parameters
		if len(filters) > 0 {
			params := url.Values{}
			for k, v := range filters {
				params.Add(k, fmt.Sprintf("%v", v))
			}
			path += "?" + params.Encode()
		}

		req, err := c.createRequest(ctx, "GET", path, nil)
		if err != nil {
			errorChan <- fmt.Errorf("failed to create stream request: %w", err)
			return
		}

		// Set headers for SSE
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			errorChan <- fmt.Errorf("failed to start stream: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errorChan <- fmt.Errorf("stream request failed with status: %d", resp.StatusCode)
			return
		}

		// Parse SSE stream
		decoder := json.NewDecoder(resp.Body)
		for {
			var record StreamRecord
			if err := decoder.Decode(&record); err != nil {
				if err == io.EOF {
					return
				}
				errorChan <- fmt.Errorf("failed to decode stream record: %w", err)
				return
			}

			select {
			case recordChan <- &record:
			case <-ctx.Done():
				return
			}
		}
	}()

	return recordChan, errorChan
}

// Low-level HTTP methods

// doRequest performs an HTTP request with retry logic
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var lastErr error

	for i := 0; i <= c.retryCount; i++ {
		req, err := c.createRequest(ctx, method, path, body)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if i < c.retryCount {
				time.Sleep(time.Duration(i+1) * time.Second)
				continue
			}
			break
		}

		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if result != nil {
				if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
					return fmt.Errorf("failed to decode response: %w", err)
				}
			}
			return nil
		}

		// Handle error response
		var errorResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return fmt.Errorf("request failed with status %d", resp.StatusCode)
		}

		return fmt.Errorf("API error: %s (code: %s)", errorResp.Message, errorResp.Code)
	}

	return fmt.Errorf("request failed after %d retries: %w", c.retryCount, lastErr)
}

// createRequest creates an HTTP request
func (c *Client) createRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "storage-system-client/1.0")

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	return req, nil
}

// Request/Response types

// Table operations
type CreateTableRequest struct {
	Schema *schema.TableSchema `json:"schema"`
}

type CreateTableResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type GetTableResponse struct {
	Table *TableInfo `json:"table"`
}

type ListTablesResponse struct {
	Tables []*TableInfo `json:"tables"`
}

type DropTableResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// Data operations
type InsertRecordsRequest struct {
	Records []map[string]interface{} `json:"records"`
}

type InsertRecordsResponse struct {
	Result *InsertResult `json:"result"`
}

type QueryRecordsResponse struct {
	Result *QueryResult `json:"result"`
}

type GetRecordResponse struct {
	Record map[string]interface{} `json:"record"`
}

type UpdateRecordRequest struct {
	Updates map[string]interface{} `json:"updates"`
}

type UpdateRecordResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type DeleteRecordResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// Schema operations
type GetSchemaResponse struct {
	Schema *schema.TableSchema `json:"schema"`
}

type UpdateSchemaRequest struct {
	Schema *schema.TableSchema `json:"schema"`
}

type UpdateSchemaResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// Index operations
type CreateIndexRequest struct {
	Index *IndexDefinition `json:"index"`
}

type CreateIndexResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type ListIndexesResponse struct {
	Indexes []*IndexInfo `json:"indexes"`
}

type DropIndexResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// System operations
type GetStatusResponse struct {
	Status *SystemStatus `json:"status"`
}

type GetMetricsResponse struct {
	Metrics *SystemMetrics `json:"metrics"`
}

type TriggerCompactionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// Batch operations
type BatchInsertRequest struct {
	Operations []*BatchInsertOperation `json:"operations"`
}

type BatchInsertResponse struct {
	Result *BatchResult `json:"result"`
}

// Common types

type TableInfo struct {
	Name        string                `json:"name"`
	Schema      *schema.TableSchema   `json:"schema"`
	RecordCount int64                 `json:"record_count"`
	Size        int64                 `json:"size"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
	Indexes     []*IndexInfo          `json:"indexes,omitempty"`
}

type InsertResult struct {
	InsertedCount int      `json:"inserted_count"`
	RecordIDs     []string `json:"record_ids"`
	Errors        []string `json:"errors,omitempty"`
}

type QueryRequest struct {
	TableName   string                 `json:"table_name"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
	Projections []string               `json:"projections,omitempty"`
	OrderBy     []OrderByClause        `json:"order_by,omitempty"`
	Limit       int                    `json:"limit,omitempty"`
	Offset      int                    `json:"offset,omitempty"`
}

type OrderByClause struct {
	Field string `json:"field"`
	Desc  bool   `json:"desc,omitempty"`
}

type QueryResult struct {
	Records []map[string]interface{} `json:"records"`
	Total   int                      `json:"total"`
	HasMore bool                     `json:"has_more"`
}

type IndexDefinition struct {
	Name      string   `json:"name"`
	TableName string   `json:"table_name"`
	Columns   []string `json:"columns"`
	Unique    bool     `json:"unique,omitempty"`
	Type      string   `json:"type,omitempty"` // btree, hash, etc.
}

type IndexInfo struct {
	Name      string    `json:"name"`
	TableName string    `json:"table_name"`
	Columns   []string  `json:"columns"`
	Unique    bool      `json:"unique"`
	Type      string    `json:"type"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

type SystemStatus struct {
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

type SystemMetrics struct {
	IngestedRecords   int64   `json:"ingested_records"`
	StoredRecords     int64   `json:"stored_records"`
	TablesCount       int     `json:"tables_count"`
	IndexesCount      int     `json:"indexes_count"`
	StorageUsed       int64   `json:"storage_used"`
	MemoryUsed        int64   `json:"memory_used"`
	QueryLatencyP50   float64 `json:"query_latency_p50"`
	QueryLatencyP95   float64 `json:"query_latency_p95"`
	QueryLatencyP99   float64 `json:"query_latency_p99"`
}

type BatchInsertOperation struct {
	TableName string                 `json:"table_name"`
	Records   []map[string]interface{} `json:"records"`
}

type BatchResult struct {
	SuccessCount int      `json:"success_count"`
	ErrorCount   int      `json:"error_count"`
	Errors       []string `json:"errors,omitempty"`
}

type StreamRecord struct {
	TableName string                 `json:"table_name"`
	RecordID  string                 `json:"record_id"`
	Data      map[string]interface{} `json:"data"`
	Operation string                 `json:"operation"` // insert, update, delete
	Timestamp time.Time              `json:"timestamp"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
