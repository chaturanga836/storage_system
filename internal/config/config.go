package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// // KafkaConfig for Kafka messaging
type KafkaConfig struct {
	Brokers          []string `json:"brokers"`
	ClientID         string   `json:"client_id"`
	GroupID          string   `json:"group_id"`
	SecurityProtocol string   `json:"security_protocol"`
	SASLMechanism    string   `json:"sasl_mechanism"`
	SASLUsername     string   `json:"sasl_username"`
	SASLPassword     string   `json:"sasl_password"`
	EnableTLS        bool     `json:"enable_tls"`
	BatchSize        int      `json:"batch_size"`
	LingerMs         int      `json:"linger_ms"`
	CompressionType  string   `json:"compression_type"`
	RetryMax         int      `json:"retry_max"`
	RetryBackoffMs   int      `json:"retry_backoff_ms"`

	// Consumer-specific fields
	AutoCommitIntervalMs int `json:"auto_commit_interval_ms"`
	FetchTimeoutMs       int `json:"fetch_timeout_ms"`
	MaxRetries           int `json:"max_retries"`
	SessionTimeoutMs     int `json:"session_timeout_ms"`

	// Publisher-specific fields
	MaxRetryBackoffMs int `json:"max_retry_backoff_ms"`
}

// Config represents the complete system configuration
type Config struct {
	Ingestion     IngestionConfig     `json:"ingestion"`
	Query         QueryConfig         `json:"query"`
	DataProcessor DataProcessorConfig `json:"data_processor"`
	Storage       StorageConfig       `json:"storage"`
	WAL           WALConfig           `json:"wal"`
	Auth          AuthConfig          `json:"auth"`
	Catalog       CatalogConfig       `json:"catalog"`
	Kafka         KafkaConfig         `json:"kafka"`
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
	Port            int    `json:"port"`
	MaxConnections  int    `json:"max_connections"`
	QueryTimeout    string `json:"query_timeout"`
	CacheSize       int64  `json:"cache_size"`
	ParallelQueries int    `json:"parallel_queries"`
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

	// Fields expected by storage_manager.go
	BlockStorageType       string            `json:"block_storage_type"`
	LocalStoragePath       string            `json:"local_storage_path"`
	FlushIntervalMs        int               `json:"flush_interval_ms"`
	CompactionIntervalMs   int               `json:"compaction_interval_ms"`
	MemtableFlushThreshold int64             `json:"memtable_flush_threshold"`
	CatalogConfig          *CatalogConfig    `json:"catalog_config"`
	WALConfig              *WALConfig        `json:"wal_config"`
	CompactionConfig       *CompactionConfig `json:"compaction_config"`
	MVCCConfig             *MVCCConfig       `json:"mvcc_config"`
	MemtableConfig         *MemtableConfig   `json:"memtable_config"`
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

// CompactionConfig for compaction settings
type CompactionConfig struct {
	Strategy        string  `json:"strategy"`         // "size_tiered", "level", "time_window"
	MaxFiles        int     `json:"max_files"`        // Max files per level
	SizeRatio       float64 `json:"size_ratio"`       // Size ratio between levels
	MinFileSize     int64   `json:"min_file_size"`    // Minimum file size for compaction
	MaxFileSize     int64   `json:"max_file_size"`    // Maximum file size before splitting
	ParallelWorkers int     `json:"parallel_workers"` // Number of parallel compaction workers
}

// MVCCConfig for multi-version concurrency control
type MVCCConfig struct {
	MaxVersions     int    `json:"max_versions"`     // Maximum versions to keep
	GCInterval      string `json:"gc_interval"`      // Garbage collection interval
	RetentionPeriod string `json:"retention_period"` // How long to keep old versions
}

// MemtableConfig for in-memory table settings
type MemtableConfig struct {
	MaxSize         int64  `json:"max_size"`          // Maximum size in bytes
	FlushThreshold  int64  `json:"flush_threshold"`   // Flush when this size is reached
	IndexType       string `json:"index_type"`        // "skiplist", "btree", "hash"
	WriteBufferSize int    `json:"write_buffer_size"` // Write buffer size
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
			Port:            getEnvInt("QUERY_PORT", 8002),
			MaxConnections:  getEnvInt("QUERY_MAX_CONNECTIONS", 500),
			QueryTimeout:    getEnvString("QUERY_TIMEOUT", "30s"),
			CacheSize:       getEnvInt64("QUERY_CACHE_SIZE", 1024*1024*1024), // 1GB
			ParallelQueries: getEnvInt("QUERY_PARALLEL_QUERIES", 10),
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
		Kafka: KafkaConfig{
			Brokers:          getEnvStringSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			ClientID:         getEnvString("KAFKA_CLIENT_ID", "my-client"),
			GroupID:          getEnvString("KAFKA_GROUP_ID", "my-group"),
			SecurityProtocol: getEnvString("KAFKA_SECURITY_PROTOCOL", "PLAINTEXT"),
			SASLMechanism:    getEnvString("KAFKA_SASL_MECHANISM", "PLAIN"),
			SASLUsername:     getEnvString("KAFKA_SASL_USERNAME", ""),
			SASLPassword:     getEnvString("KAFKA_SASL_PASSWORD", ""),
			EnableTLS:        getEnvBool("KAFKA_ENABLE_TLS", false),
			BatchSize:        getEnvInt("KAFKA_BATCH_SIZE", 16384),
			LingerMs:         getEnvInt("KAFKA_LINGER_MS", 5),
			CompressionType:  getEnvString("KAFKA_COMPRESSION_TYPE", "none"),
			RetryMax:         getEnvInt("KAFKA_RETRY_MAX", 3),
			RetryBackoffMs:   getEnvInt("KAFKA_RETRY_BACKOFF_MS", 100),
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

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return split(value, ",")
	}
	return defaultValue
}

func split(s string, sep string) []string {
	var result []string
	for _, v := range strings.Split(s, sep) {
		if len(v) > 0 {
			result = append(result, v)
		}
	}
	return result
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
