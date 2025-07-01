"""
Index Manager for optimizing data queries
"""
import asyncio
import json
import pickle
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, List, Any, Optional, Set
import logging
import pandas as pd

logger = logging.getLogger(__name__)


class IndexEntry:
    """Represents an index entry"""
    
    def __init__(self, file_path: str, column: str, min_value: Any, max_value: Any, 
                 unique_values: Optional[Set] = None, null_count: int = 0):
        self.file_path = file_path
        self.column = column
        self.min_value = min_value
        self.max_value = max_value
        self.unique_values = unique_values
        self.null_count = null_count
        self.created_at = datetime.now(timezone.utc)
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization"""
        return {
            "file_path": self.file_path,
            "column": self.column,
            "min_value": self.min_value,
            "max_value": self.max_value,
            "unique_values": list(self.unique_values) if self.unique_values else None,
            "null_count": self.null_count,
            "created_at": self.created_at.isoformat()
        }
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "IndexEntry":
        """Create from dictionary"""
        entry = cls(
            file_path=data["file_path"],
            column=data["column"],
            min_value=data["min_value"],
            max_value=data["max_value"],
            unique_values=set(data["unique_values"]) if data["unique_values"] else None,
            null_count=data.get("null_count", 0)
        )
        entry.created_at = datetime.fromisoformat(data["created_at"])
        return entry


class IndexManager:
    """Manages indices for optimizing data queries"""
    
    def __init__(self, index_path: Path, index_columns: List[str]):
        self.index_path = index_path
        self.index_columns = index_columns
        self.indices: Dict[str, Dict[str, IndexEntry]] = {}  # {column: {file_path: IndexEntry}}
        self._lock = asyncio.Lock()
        
    async def initialize(self):
        """Initialize the index manager"""
        self.index_path.mkdir(parents=True, exist_ok=True)
        
        # Load existing indices
        await self._load_indices()
        
        logger.info(f"Index Manager initialized at {self.index_path}")
    
    async def _load_indices(self):
        """Load existing indices from disk"""
        index_file = self.index_path / "indices.json"
        
        if index_file.exists():
            try:
                with open(index_file, 'r') as f:
                    data = json.load(f)
                
                for column, file_indices in data.items():
                    self.indices[column] = {}
                    for file_path, index_data in file_indices.items():
                        self.indices[column][file_path] = IndexEntry.from_dict(index_data)
                
                logger.info(f"Loaded indices for {len(self.indices)} columns")
            
            except Exception as e:
                logger.error(f"Error loading indices: {e}")
                self.indices = {}
    
    async def _save_indices(self):
        """Save indices to disk"""
        index_file = self.index_path / "indices.json"
        
        try:
            data = {}
            for column, file_indices in self.indices.items():
                data[column] = {}
                for file_path, index_entry in file_indices.items():
                    data[column][file_path] = index_entry.to_dict()
            
            with open(index_file, 'w') as f:
                json.dump(data, f, indent=2, default=str)
        
        except Exception as e:
            logger.error(f"Error saving indices: {e}")
    
    async def update_indices_for_file(self, file_path: str):
        """Update indices for a specific file"""
        try:
            # Read the parquet file
            df = pd.read_parquet(file_path)
            
            await self.update_indices_for_data(df, file_path)
            
        except Exception as e:
            logger.error(f"Error updating indices for file {file_path}: {e}")
    
    async def update_indices_for_data(self, df: pd.DataFrame, file_path: str):
        """Update indices for given data and file path"""
        async with self._lock:
            for column in self.index_columns:
                if column not in df.columns:
                    continue
                
                try:
                    col_data = df[column]
                    
                    # Calculate statistics
                    min_value = col_data.min()
                    max_value = col_data.max()
                    null_count = col_data.isnull().sum()
                    
                    # For small columns, store unique values
                    unique_values = None
                    if len(col_data.unique()) <= 100:  # Threshold for storing unique values
                        unique_values = set(col_data.dropna().unique())
                    
                    # Create index entry
                    index_entry = IndexEntry(
                        file_path=file_path,
                        column=column,
                        min_value=min_value,
                        max_value=max_value,
                        unique_values=unique_values,
                        null_count=null_count
                    )
                    
                    # Store the index
                    if column not in self.indices:
                        self.indices[column] = {}
                    
                    self.indices[column][file_path] = index_entry
                    
                    logger.debug(f"Updated index for column {column} in file {file_path}")
                
                except Exception as e:
                    logger.error(f"Error creating index for column {column}: {e}")
            
            # Save indices to disk
            await self._save_indices()
    
    async def optimize_file_list(self, file_paths: List[str], 
                                query_filters: Dict[str, Any]) -> List[str]:
        """Optimize file list based on query filters using indices"""
        if not query_filters:
            return file_paths
        
        optimized_files = set(file_paths)
        
        for column, condition in query_filters.items():
            if column not in self.indices:
                continue
            
            column_indices = self.indices[column]
            matching_files = set()
            
            for file_path, index_entry in column_indices.items():
                if file_path not in optimized_files:
                    continue
                
                if self._file_matches_condition(index_entry, condition):
                    matching_files.add(file_path)
            
            # Intersect with current optimized files
            optimized_files &= matching_files
        
        # Sort by file modification time (newest first) for better cache utilization
        try:
            sorted_files = sorted(
                optimized_files,
                key=lambda f: Path(f).stat().st_mtime,
                reverse=True
            )
            return sorted_files
        except:
            return list(optimized_files)
    
    def _file_matches_condition(self, index_entry: IndexEntry, condition: Any) -> bool:
        """Check if a file matches a query condition based on its index"""
        try:
            if isinstance(condition, dict):
                # Complex condition
                if "$eq" in condition:
                    value = condition["$eq"]
                    if index_entry.unique_values:
                        return value in index_entry.unique_values
                    else:
                        return index_entry.min_value <= value <= index_entry.max_value
                
                elif "$gt" in condition:
                    return index_entry.max_value > condition["$gt"]
                
                elif "$lt" in condition:
                    return index_entry.min_value < condition["$lt"]
                
                elif "$gte" in condition:
                    return index_entry.max_value >= condition["$gte"]
                
                elif "$lte" in condition:
                    return index_entry.min_value <= condition["$lte"]
                
                elif "$in" in condition:
                    values = set(condition["$in"])
                    if index_entry.unique_values:
                        return bool(index_entry.unique_values & values)
                    else:
                        # Check if any value in the range might match
                        return any(
                            index_entry.min_value <= v <= index_entry.max_value 
                            for v in values
                        )
                
                elif "$like" in condition:
                    # For LIKE queries, we can't easily optimize with min/max
                    # So we include the file
                    return True
            
            else:
                # Simple equality
                if index_entry.unique_values:
                    return condition in index_entry.unique_values
                else:
                    return index_entry.min_value <= condition <= index_entry.max_value
        
        except Exception as e:
            logger.error(f"Error checking condition match: {e}")
            # If there's an error, include the file to be safe
            return True
        
        return True
    
    async def get_column_statistics(self, column: str) -> Dict[str, Any]:
        """Get statistics for a specific column across all files"""
        if column not in self.indices:
            return {}
        
        column_indices = self.indices[column]
        
        if not column_indices:
            return {}
        
        # Calculate overall statistics
        min_values = [entry.min_value for entry in column_indices.values() if entry.min_value is not None]
        max_values = [entry.max_value for entry in column_indices.values() if entry.max_value is not None]
        total_nulls = sum(entry.null_count for entry in column_indices.values())
        
        # Combine unique values if available
        all_unique_values = set()
        has_unique_values = True
        
        for entry in column_indices.values():
            if entry.unique_values is None:
                has_unique_values = False
                break
            all_unique_values.update(entry.unique_values)
        
        stats = {
            "column": column,
            "total_files": len(column_indices),
            "min_value": min(min_values) if min_values else None,
            "max_value": max(max_values) if max_values else None,
            "total_null_count": total_nulls
        }
        
        if has_unique_values and len(all_unique_values) <= 1000:
            stats["unique_values"] = sorted(list(all_unique_values))
            stats["unique_count"] = len(all_unique_values)
        
        return stats
    
    async def remove_file_indices(self, file_path: str):
        """Remove indices for a specific file"""
        async with self._lock:
            for column in self.indices:
                if file_path in self.indices[column]:
                    del self.indices[column][file_path]
            
            await self._save_indices()
    
    async def rebuild_indices(self, file_paths: List[str]):
        """Rebuild indices for given files"""
        logger.info(f"Rebuilding indices for {len(file_paths)} files")
        
        # Clear existing indices for these files
        async with self._lock:
            for column in self.indices:
                for file_path in file_paths:
                    if file_path in self.indices[column]:
                        del self.indices[column][file_path]
        
        # Rebuild indices
        for file_path in file_paths:
            await self.update_indices_for_file(file_path)
        
        logger.info("Index rebuild complete")
    
    async def get_status(self) -> Dict[str, Any]:
        """Get index manager status"""
        total_entries = sum(len(column_indices) for column_indices in self.indices.values())
        
        column_stats = {}
        for column in self.indices:
            column_stats[column] = {
                "total_files": len(self.indices[column]),
                "last_updated": max(
                    (entry.created_at for entry in self.indices[column].values()),
                    default=datetime.min.replace(tzinfo=timezone.utc)
                ).isoformat()
            }
        
        return {
            "indexed_columns": list(self.indices.keys()),
            "total_index_entries": total_entries,
            "column_statistics": column_stats
        }
    
    async def cleanup(self):
        """Cleanup index manager resources"""
        await self._save_indices()
        logger.info("Index Manager cleanup complete")
