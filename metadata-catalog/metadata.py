"""
Metadata Manager for tracking file information and schemas
"""
import asyncio
import json
import os
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, List, Any, Optional
import logging

logger = logging.getLogger(__name__)


class FileMetadata:
    """Represents metadata for a single file"""
    
    def __init__(self, file_path: str, metadata: Dict[str, Any]):
        self.file_path = file_path
        self.write_id = metadata.get("write_id")
        self.row_count = metadata.get("row_count", 0)
        self.columns = metadata.get("columns", [])
        self.partitions = metadata.get("partitions", [])
        self.created_at = datetime.fromisoformat(metadata["created_at"]) if "created_at" in metadata else datetime.now(timezone.utc)
        self.file_size = metadata.get("file_size", 0)
        self.schema_hash = metadata.get("schema_hash")
        self.last_accessed = metadata.get("last_accessed")
        self.custom_metadata = metadata.get("custom_metadata", {})
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization"""
        return {
            "file_path": self.file_path,
            "write_id": self.write_id,
            "row_count": self.row_count,
            "columns": self.columns,
            "partitions": self.partitions,
            "created_at": self.created_at.isoformat(),
            "file_size": self.file_size,
            "schema_hash": self.schema_hash,
            "last_accessed": self.last_accessed,
            "custom_metadata": self.custom_metadata
        }
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "FileMetadata":
        """Create from dictionary"""
        return cls(data["file_path"], data)
    
    def update_access_time(self):
        """Update the last accessed time"""
        self.last_accessed = datetime.now(timezone.utc).isoformat()


class MetadataManager:
    """Manages metadata for files and schemas"""
    
    def __init__(self, metadata_path: Path):
        self.metadata_path = metadata_path
        self.files_metadata: Dict[str, FileMetadata] = {}
        self.schema_registry: Dict[str, Dict[str, Any]] = {}
        self._lock = asyncio.Lock()
        
    async def initialize(self):
        """Initialize the metadata manager"""
        self.metadata_path.mkdir(parents=True, exist_ok=True)
        
        # Load existing metadata
        await self._load_metadata()
        await self._load_schema_registry()
        
        logger.info(f"Metadata Manager initialized at {self.metadata_path}")
    
    async def _load_metadata(self):
        """Load existing file metadata from disk"""
        metadata_file = self.metadata_path / "files.json"
        
        if metadata_file.exists():
            try:
                with open(metadata_file, 'r') as f:
                    data = json.load(f)
                
                for file_path, metadata in data.items():
                    self.files_metadata[file_path] = FileMetadata.from_dict(metadata)
                
                logger.info(f"Loaded metadata for {len(self.files_metadata)} files")
            
            except Exception as e:
                logger.error(f"Error loading file metadata: {e}")
                self.files_metadata = {}
    
    async def _save_metadata(self):
        """Save file metadata to disk"""
        metadata_file = self.metadata_path / "files.json"
        
        try:
            data = {}
            for file_path, metadata in self.files_metadata.items():
                data[file_path] = metadata.to_dict()
            
            with open(metadata_file, 'w') as f:
                json.dump(data, f, indent=2, default=str)
        
        except Exception as e:
            logger.error(f"Error saving file metadata: {e}")
    
    async def _load_schema_registry(self):
        """Load schema registry from disk"""
        schema_file = self.metadata_path / "schemas.json"
        
        if schema_file.exists():
            try:
                with open(schema_file, 'r') as f:
                    self.schema_registry = json.load(f)
                
                logger.info(f"Loaded {len(self.schema_registry)} schemas")
            
            except Exception as e:
                logger.error(f"Error loading schema registry: {e}")
                self.schema_registry = {}
    
    async def _save_schema_registry(self):
        """Save schema registry to disk"""
        schema_file = self.metadata_path / "schemas.json"
        
        try:
            with open(schema_file, 'w') as f:
                json.dump(self.schema_registry, f, indent=2, default=str)
        
        except Exception as e:
            logger.error(f"Error saving schema registry: {e}")
    
    async def register_file(self, file_path: str, metadata: Dict[str, Any]):
        """Register a new file with its metadata"""
        async with self._lock:
            file_metadata = FileMetadata(file_path, metadata)
            self.files_metadata[file_path] = file_metadata
            
            # Update schema registry if needed
            if file_metadata.columns:
                await self._update_schema_registry(file_metadata)
            
            await self._save_metadata()
            
            logger.debug(f"Registered file metadata: {file_path}")
    
    async def _update_schema_registry(self, file_metadata: FileMetadata):
        """Update the schema registry with file information"""
        # Create a simple schema hash based on column names
        schema_key = "_".join(sorted(file_metadata.columns))
        
        if schema_key not in self.schema_registry:
            self.schema_registry[schema_key] = {
                "columns": file_metadata.columns,
                "files": [],
                "first_seen": file_metadata.created_at.isoformat(),
                "last_updated": file_metadata.created_at.isoformat()
            }
        
        # Add file to schema if not already present
        if file_metadata.file_path not in self.schema_registry[schema_key]["files"]:
            self.schema_registry[schema_key]["files"].append(file_metadata.file_path)
            self.schema_registry[schema_key]["last_updated"] = datetime.now(timezone.utc).isoformat()
        
        await self._save_schema_registry()
    
    async def update_file_metadata(self, file_path: str, updates: Optional[Dict[str, Any]] = None):
        """Update metadata for an existing file"""
        async with self._lock:
            if file_path not in self.files_metadata:
                # File not in metadata, try to create basic metadata
                if os.path.exists(file_path):
                    stat = os.stat(file_path)
                    basic_metadata = {
                        "file_size": stat.st_size,
                        "created_at": datetime.fromtimestamp(stat.st_mtime, tz=timezone.utc).isoformat()
                    }
                    if updates:
                        basic_metadata.update(updates)
                    
                    await self.register_file(file_path, basic_metadata)
                return
            
            # Update existing metadata
            file_metadata = self.files_metadata[file_path]
            
            if updates:
                for key, value in updates.items():
                    if hasattr(file_metadata, key):
                        setattr(file_metadata, key, value)
                    else:
                        file_metadata.custom_metadata[key] = value
            
            # Update file size if file exists
            if os.path.exists(file_path):
                file_metadata.file_size = os.path.getsize(file_path)
            
            file_metadata.update_access_time()
            
            await self._save_metadata()
    
    async def get_files_for_query(self, query_filter: Dict[str, Any]) -> List[str]:
        """Get list of files that might contain data matching the query"""
        # For now, return all files. In a more sophisticated implementation,
        # we could filter based on partition information, date ranges, etc.
        
        relevant_files = []
        
        for file_path, metadata in self.files_metadata.items():
            # Check if file still exists
            if not os.path.exists(file_path):
                continue
            
            # Basic filtering based on metadata
            if self._file_matches_metadata_filter(metadata, query_filter):
                relevant_files.append(file_path)
        
        # Sort by creation time (newest first) for better query performance
        relevant_files.sort(
            key=lambda f: self.files_metadata[f].created_at,
            reverse=True
        )
        
        return relevant_files
    
    def _file_matches_metadata_filter(self, metadata: FileMetadata, query_filter: Dict[str, Any]) -> bool:
        """Check if a file matches basic metadata filters"""
        # Check if required columns exist
        query_columns = set(query_filter.keys())
        file_columns = set(metadata.columns)
        
        # If query requires columns that don't exist in the file, skip it
        if query_columns and not query_columns.issubset(file_columns):
            return False
        
        # Additional metadata-based filtering could be added here
        # For example, date range filtering based on partition information
        
        return True
    
    async def get_file_metadata(self, file_path: str) -> Optional[FileMetadata]:
        """Get metadata for a specific file"""
        return self.files_metadata.get(file_path)
    
    async def remove_file_metadata(self, file_path: str):
        """Remove metadata for a specific file"""
        async with self._lock:
            if file_path in self.files_metadata:
                del self.files_metadata[file_path]
                await self._save_metadata()
                
                # Update schema registry
                await self._cleanup_schema_registry()
    
    async def _cleanup_schema_registry(self):
        """Remove files that no longer exist from schema registry"""
        for schema_key, schema_info in self.schema_registry.items():
            existing_files = [
                f for f in schema_info["files"] 
                if f in self.files_metadata
            ]
            schema_info["files"] = existing_files
        
        # Remove empty schemas
        self.schema_registry = {
            k: v for k, v in self.schema_registry.items() 
            if v["files"]
        }
        
        await self._save_schema_registry()
    
    async def get_schema_info(self) -> Dict[str, Any]:
        """Get schema information for all files"""
        return {
            "schemas": self.schema_registry,
            "total_schemas": len(self.schema_registry),
            "total_files": len(self.files_metadata)
        }
    
    async def get_file_count(self) -> int:
        """Get total number of tracked files"""
        return len(self.files_metadata)
    
    async def get_total_size(self) -> int:
        """Get total size of all tracked files"""
        return sum(metadata.file_size for metadata in self.files_metadata.values())
    
    async def get_total_rows(self) -> int:
        """Get total number of rows across all files"""
        return sum(metadata.row_count for metadata in self.files_metadata.values())
    
    async def get_last_updated(self) -> Optional[str]:
        """Get the timestamp of the most recently updated file"""
        if not self.files_metadata:
            return None
        
        latest = max(
            metadata.created_at for metadata in self.files_metadata.values()
        )
        return latest.isoformat()
    
    async def get_files_by_date_range(self, start_date: datetime, end_date: datetime) -> List[str]:
        """Get files created within a specific date range"""
        matching_files = []
        
        for file_path, metadata in self.files_metadata.items():
            if start_date <= metadata.created_at <= end_date:
                matching_files.append(file_path)
        
        return matching_files
    
    async def get_files_by_size_range(self, min_size: int, max_size: int) -> List[str]:
        """Get files within a specific size range"""
        matching_files = []
        
        for file_path, metadata in self.files_metadata.items():
            if min_size <= metadata.file_size <= max_size:
                matching_files.append(file_path)
        
        return matching_files
    
    async def get_statistics(self) -> Dict[str, Any]:
        """Get comprehensive metadata statistics"""
        if not self.files_metadata:
            return {
                "total_files": 0,
                "total_size_bytes": 0,
                "total_rows": 0,
                "schemas": {}
            }
        
        file_sizes = [m.file_size for m in self.files_metadata.values()]
        row_counts = [m.row_count for m in self.files_metadata.values()]
        
        return {
            "total_files": len(self.files_metadata),
            "total_size_bytes": sum(file_sizes),
            "total_rows": sum(row_counts),
            "average_file_size": sum(file_sizes) / len(file_sizes),
            "average_rows_per_file": sum(row_counts) / len(row_counts),
            "largest_file_size": max(file_sizes),
            "smallest_file_size": min(file_sizes),
            "schemas": len(self.schema_registry),
            "oldest_file": min(m.created_at for m in self.files_metadata.values()).isoformat(),
            "newest_file": max(m.created_at for m in self.files_metadata.values()).isoformat()
        }
    
    async def cleanup(self):
        """Cleanup metadata manager resources"""
        await self._save_metadata()
        await self._save_schema_registry()
        logger.info("Metadata Manager cleanup complete")
