"""
Configuration management for Tenant Node
"""
import os
from dataclasses import dataclass, field
from typing import Dict, List, Optional
from pathlib import Path


@dataclass
class SourceConfig:
    """Configuration for a data source"""
    source_id: str
    name: str
    connection_string: str
    data_path: str
    schema_definition: Dict
    partition_columns: List[str] = field(default_factory=list)
    index_columns: List[str] = field(default_factory=list)
    compression: str = "snappy"
    max_file_size_mb: int = 256
    wal_enabled: bool = True


@dataclass
class TenantConfig:
    """Configuration for the tenant node"""
    tenant_id: str
    tenant_name: str
    base_data_path: str
    sources: Dict[str, SourceConfig] = field(default_factory=dict)
    
    # Storage settings
    wal_retention_hours: int = 24
    index_refresh_interval_minutes: int = 5
    compaction_threshold_files: int = 10
    
    # Performance settings
    max_concurrent_searches: int = 10
    search_timeout_seconds: int = 30
    chunk_size: int = 10000
    
    # Auto-scaling settings
    auto_scaling_enabled: bool = True
    min_workers: int = 2
    max_workers: int = 50
    scale_up_threshold_cpu: float = 70.0
    scale_down_threshold_cpu: float = 30.0
    scale_up_threshold_memory: float = 80.0
    scale_down_threshold_memory: float = 40.0
    
    # Compaction settings
    auto_compaction_enabled: bool = True
    compaction_interval_minutes: int = 60
    min_file_size_mb: float = 50.0
    max_file_size_mb: float = 512.0
    target_file_size_mb: float = 256.0
    
    # Query optimization settings
    query_optimization_enabled: bool = True
    statistics_refresh_interval_minutes: int = 30
    cost_model_learning_enabled: bool = True
    
    # gRPC settings
    grpc_port: int = 50051
    grpc_max_workers: int = 10
    
    # REST API settings
    rest_port: int = 8000
    rest_host: str = "0.0.0.0"
    
    @classmethod
    def from_env(cls) -> "TenantConfig":
        """Create configuration from environment variables"""
        return cls(
            tenant_id=os.getenv("TENANT_ID", "default_tenant"),
            tenant_name=os.getenv("TENANT_NAME", "Default Tenant"),
            base_data_path=os.getenv("DATA_PATH", "./data"),
            grpc_port=int(os.getenv("GRPC_PORT", "50051")),
            rest_port=int(os.getenv("REST_PORT", "8000")),
            max_concurrent_searches=int(os.getenv("MAX_CONCURRENT_SEARCHES", "10")),
            auto_scaling_enabled=os.getenv("AUTO_SCALING_ENABLED", "true").lower() == "true",
            auto_compaction_enabled=os.getenv("AUTO_COMPACTION_ENABLED", "true").lower() == "true",
            query_optimization_enabled=os.getenv("QUERY_OPTIMIZATION_ENABLED", "true").lower() == "true",
            min_workers=int(os.getenv("MIN_WORKERS", "2")),
            max_workers=int(os.getenv("MAX_WORKERS", "50")),
        )
    
    def get_source_data_path(self, source_id: str) -> Path:
        """Get the data path for a specific source"""
        return Path(self.base_data_path) / "sources" / source_id
    
    def get_wal_path(self, source_id: str) -> Path:
        """Get the WAL path for a specific source"""
        return self.get_source_data_path(source_id) / "wal"
    
    def get_index_path(self, source_id: str) -> Path:
        """Get the index path for a specific source"""
        return self.get_source_data_path(source_id) / "index"
    
    def get_metadata_path(self, source_id: str) -> Path:
        """Get the metadata path for a specific source"""
        return self.get_source_data_path(source_id) / "metadata"
