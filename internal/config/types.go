package config

// Config holds the configuration for all services
type Config struct {
	// Service-specific configurations
	IngestionServer  IngestionServerConfig  `yaml:"ingestion_server" json:"ingestion_server"`
	QueryServer      QueryServerConfig      `yaml:"query_server" json:"query_server"`
	DataProcessor    DataProcessorConfig    `yaml:"data_processor" json:"data_processor"`
	
	// Shared configurations
	Auth             AuthConfig             `yaml:"auth" json:"auth"`
	Storage          StorageConfig          `yaml:"storage" json:"storage"`
	WAL              WALConfig              `yaml:"wal" json:"wal"`
	Database         DatabaseConfig         `yaml:"database" json:"database"`
	Monitoring       MonitoringConfig       `yaml:"monitoring" json:"monitoring"`
	Logging          LoggingConfig          `yaml:"logging" json:"logging"`
}

// IngestionServerConfig contains configuration for the ingestion server
type IngestionServerConfig struct {
	Host                string `yaml:"host" json:"host"`
	Port                int    `yaml:"port" json:"port"`
	MaxConcurrentConns  int    `yaml:"max_concurrent_conns" json:"max_concurrent_conns"`
	MaxMessageSize      int64  `yaml:"max_message_size" json:"max_message_size"`
	ReadTimeout         string `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout        string `yaml:"write_timeout" json:"write_timeout"`
	MemtableFlushSize   int64  `yaml:"memtable_flush_size" json:"memtable_flush_size"`
	MemtableFlushTime   string `yaml:"memtable_flush_time" json:"memtable_flush_time"`
	BatchSize           int    `yaml:"batch_size" json:"batch_size"`
}

// QueryServerConfig contains configuration for the query server
type QueryServerConfig struct {
	Host               string `yaml:"host" json:"host"`
	Port               int    `yaml:"port" json:"port"`
	MaxConcurrentConns int    `yaml:"max_concurrent_conns" json:"max_concurrent_conns"`
	QueryTimeout       string `yaml:"query_timeout" json:"query_timeout"`
	CacheSize          int64  `yaml:"cache_size" json:"cache_size"`
	MaxResultSize      int64  `yaml:"max_result_size" json:"max_result_size"`
}

// DataProcessorConfig contains configuration for the data processor
type DataProcessorConfig struct {
	WorkerCount        int    `yaml:"worker_count" json:"worker_count"`
	CompactionInterval string `yaml:"compaction_interval" json:"compaction_interval"`
	IndexUpdateBatch   int    `yaml:"index_update_batch" json:"index_update_batch"`
	ParquetBlockSize   int64  `yaml:"parquet_block_size" json:"parquet_block_size"`
}

// AuthConfig contains authentication and authorization settings
type AuthConfig struct {
	JWTSecret       string `yaml:"jwt_secret" json:"jwt_secret"`
	JWTIssuer       string `yaml:"jwt_issuer" json:"jwt_issuer"`
	JWTExpiration   string `yaml:"jwt_expiration" json:"jwt_expiration"`
	APIKeyTTL       string `yaml:"api_key_ttl" json:"api_key_ttl"`
	EnableRateLimit bool   `yaml:"enable_rate_limit" json:"enable_rate_limit"`
	RateLimitRPS    int    `yaml:"rate_limit_rps" json:"rate_limit_rps"`
}

// StorageConfig contains storage-related settings
type StorageConfig struct {
	DataDir           string `yaml:"data_dir" json:"data_dir"`
	IndexDir          string `yaml:"index_dir" json:"index_dir"`
	BackupDir         string `yaml:"backup_dir" json:"backup_dir"`
	CloudStorage      CloudStorageConfig `yaml:"cloud_storage" json:"cloud_storage"`
	CompressionType   string `yaml:"compression_type" json:"compression_type"`
	CompressionLevel  int    `yaml:"compression_level" json:"compression_level"`
	MaxFileSize       int64  `yaml:"max_file_size" json:"max_file_size"`
	RetentionPeriod   string `yaml:"retention_period" json:"retention_period"`
}

// CloudStorageConfig contains cloud storage settings
type CloudStorageConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	Provider   string `yaml:"provider" json:"provider"` // s3, gcs, azure
	Bucket     string `yaml:"bucket" json:"bucket"`
	Region     string `yaml:"region" json:"region"`
	AccessKey  string `yaml:"access_key" json:"access_key"`
	SecretKey  string `yaml:"secret_key" json:"secret_key"`
	Endpoint   string `yaml:"endpoint" json:"endpoint"`
}

// WALConfig contains Write-Ahead Log settings
type WALConfig struct {
	Dir             string `yaml:"dir" json:"dir"`
	SegmentSize     int64  `yaml:"segment_size" json:"segment_size"`
	MaxSegments     int    `yaml:"max_segments" json:"max_segments"`
	SyncPolicy      string `yaml:"sync_policy" json:"sync_policy"` // always, batch, periodic
	SyncInterval    string `yaml:"sync_interval" json:"sync_interval"`
	CompressionType string `yaml:"compression_type" json:"compression_type"`
}

// DatabaseConfig contains metadata database settings
type DatabaseConfig struct {
	Type     string `yaml:"type" json:"type"` // badger, postgres, etc.
	Path     string `yaml:"path" json:"path"`
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	Database string `yaml:"database" json:"database"`
	SSLMode  string `yaml:"ssl_mode" json:"ssl_mode"`
}

// MonitoringConfig contains monitoring and metrics settings
type MonitoringConfig struct {
	Enabled         bool   `yaml:"enabled" json:"enabled"`
	MetricsPort     int    `yaml:"metrics_port" json:"metrics_port"`
	HealthCheckPort int    `yaml:"health_check_port" json:"health_check_port"`
	ProfilerEnabled bool   `yaml:"profiler_enabled" json:"profiler_enabled"`
	ProfilerPort    int    `yaml:"profiler_port" json:"profiler_port"`
	TracingEnabled  bool   `yaml:"tracing_enabled" json:"tracing_enabled"`
	TracingEndpoint string `yaml:"tracing_endpoint" json:"tracing_endpoint"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level       string `yaml:"level" json:"level"` // debug, info, warn, error
	Format      string `yaml:"format" json:"format"` // json, text
	Output      string `yaml:"output" json:"output"` // stdout, file
	FilePath    string `yaml:"file_path" json:"file_path"`
	MaxSize     int    `yaml:"max_size" json:"max_size"`     // MB
	MaxBackups  int    `yaml:"max_backups" json:"max_backups"`
	MaxAge      int    `yaml:"max_age" json:"max_age"`       // days
	Compress    bool   `yaml:"compress" json:"compress"`
}
