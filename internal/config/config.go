package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// Config represents the complete system configuration
type Config struct {
	Ingestion     IngestionConfig     `json:"ingestion"`
	Query         QueryConfig         `json:"query"`
	DataProcessor DataProcessorConfig `json:"data_processor"`
	Storage       StorageConfig       `json:"storage"`
	WAL           WALConfig          `json:"wal"`
	Auth          AuthConfig         `json:"auth"`
	Catalog       CatalogConfig      `json:"catalog"`
}

// IngestionConfig for the ingestion server
type IngestionConfig struct {
	Port           int    `json:"port"`
	MaxConnections int    `json:"max_connections"`
	BatchSize      int    `json:"batch_size"`
	FlushInterval  string `json:"flush_interval"`
}

// QueryConfig for the query server
type QueryConfig struct {
	Port              int    `json:"port"`
	MaxConnections    int    `json:"max_connections"`
	QueryTimeout      string `json:"query_timeout"`
	CacheSize         int64  `json:"cache_size"`
	ParallelQueries   int    `json:"parallel_queries"`
}

// DataProcessorConfig for background processing
type DataProcessorConfig struct {
	CompactionInterval string `json:"compaction_interval"`
	FlushInterval      string `json:"flush_interval"`
	IndexInterval      string `json:"index_interval"`
	WorkerCount        int    `json:"worker_count"`
}

// StorageConfig for storage backends
type StorageConfig struct {
	DataPath         string        `json:"data_path"`
	Backend          string        `json:"backend"` // "local" or "s3"
	LocalFS          LocalFSConfig `json:"local_fs"`
	S3               S3Config      `json:"s3"`
	ParquetBlockSize int64         `json:"parquet_block_size"`
	CompressionType  string        `json:"compression_type"`
}

// LocalFSConfig for local file system storage
type LocalFSConfig struct {
	BasePath string `json:"base_path"`
}

// S3Config for S3 storage backend
type S3Config struct {
	Bucket          string `json:"bucket"`
	Region          string `json:"region"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Endpoint        string `json:"endpoint"`
}

// WALConfig for Write-Ahead Log
type WALConfig struct {
	Path            string `json:"path"`
	SegmentSize     int64  `json:"segment_size"`
	RetentionPeriod string `json:"retention_period"`
	SyncInterval    string `json:"sync_interval"`
}

// AuthConfig for authentication
type AuthConfig struct {
	Enabled     bool   `json:"enabled"`
	JWTSecret   string `json:"jwt_secret"`
	TokenExpiry string `json:"token_expiry"`
}

// CatalogConfig for metadata catalog
type CatalogConfig struct {
	Backend string `json:"backend"` // "badger" or "external"
	Path    string `json:"path"`
}

// Load loads configuration from environment variables and config file
func Load() (*Config, error) {
	cfg := &Config{
		Ingestion: IngestionConfig{
			Port:           getEnvInt("INGESTION_PORT", 8001),
			MaxConnections: getEnvInt("INGESTION_MAX_CONNECTIONS", 1000),
			BatchSize:      getEnvInt("INGESTION_BATCH_SIZE", 1000),
			FlushInterval:  getEnvString("INGESTION_FLUSH_INTERVAL", "5s"),
		},
		Query: QueryConfig{
			Port:              getEnvInt("QUERY_PORT", 8002),
			MaxConnections:    getEnvInt("QUERY_MAX_CONNECTIONS", 500),
			QueryTimeout:      getEnvString("QUERY_TIMEOUT", "30s"),
			CacheSize:         getEnvInt64("QUERY_CACHE_SIZE", 1024*1024*1024), // 1GB
			ParallelQueries:   getEnvInt("QUERY_PARALLEL_QUERIES", 10),
		},
		DataProcessor: DataProcessorConfig{
			CompactionInterval: getEnvString("COMPACTION_INTERVAL", "1h"),
			FlushInterval:      getEnvString("FLUSH_INTERVAL", "10s"),
			IndexInterval:      getEnvString("INDEX_INTERVAL", "30s"),
			WorkerCount:        getEnvInt("WORKER_COUNT", 4),
		},
		Storage: StorageConfig{
			DataPath:         getEnvString("STORAGE_DATA_PATH", "./data"),
			Backend:          getEnvString("STORAGE_BACKEND", "local"),
			ParquetBlockSize: getEnvInt64("PARQUET_BLOCK_SIZE", 64*1024*1024), // 64MB
			CompressionType:  getEnvString("COMPRESSION_TYPE", "snappy"),
			LocalFS: LocalFSConfig{
				BasePath: getEnvString("LOCAL_FS_BASE_PATH", "./data"),
			},
			S3: S3Config{
				Bucket:          getEnvString("S3_BUCKET", ""),
				Region:          getEnvString("S3_REGION", "us-east-1"),
				AccessKeyID:     getEnvString("S3_ACCESS_KEY_ID", ""),
				SecretAccessKey: getEnvString("S3_SECRET_ACCESS_KEY", ""),
				Endpoint:        getEnvString("S3_ENDPOINT", ""),
			},
		},
		WAL: WALConfig{
			Path:            getEnvString("WAL_PATH", "./wal"),
			SegmentSize:     getEnvInt64("WAL_SEGMENT_SIZE", 256*1024*1024), // 256MB
			RetentionPeriod: getEnvString("WAL_RETENTION_PERIOD", "7d"),
			SyncInterval:    getEnvString("WAL_SYNC_INTERVAL", "1s"),
		},
		Auth: AuthConfig{
			Enabled:     getEnvBool("AUTH_ENABLED", true),
			JWTSecret:   getEnvString("JWT_SECRET", "your-secret-key"),
			TokenExpiry: getEnvString("TOKEN_EXPIRY", "24h"),
		},
		Catalog: CatalogConfig{
			Backend: getEnvString("CATALOG_BACKEND", "badger"),
			Path:    getEnvString("CATALOG_PATH", "./catalog"),
		},
	}

	return cfg, nil
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// String returns a pretty-printed JSON representation of the config
func (c *Config) String() string {
	data, _ := json.MarshalIndent(c, "", "  ")
	return string(data)
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Ingestion.Port <= 0 || c.Ingestion.Port > 65535 {
		return fmt.Errorf("invalid ingestion port: %d", c.Ingestion.Port)
	}
	
	if c.Query.Port <= 0 || c.Query.Port > 65535 {
		return fmt.Errorf("invalid query port: %d", c.Query.Port)
	}

	if c.Storage.Backend != "local" && c.Storage.Backend != "s3" {
		return fmt.Errorf("invalid storage backend: %s", c.Storage.Backend)
	}

	return nil
}
