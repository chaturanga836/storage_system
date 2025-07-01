"""
Data source management for handling multiple connection sources
Each source manages parquet files, WAL, indexing, and metadata
"""
import asyncio
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, List, Optional, Any, AsyncIterator
import json
import logging

import pandas as pd
import pyarrow as pa
import pyarrow.parquet as pq

# Try to import polars, fall back to pandas if not available
try:
    import polars as pl
    HAS_POLARS = True
except ImportError:
    HAS_POLARS = False
    pl = None

# Try to import watchdog, it's optional
try:
    from watchdog.observers import Observer
    from watchdog.events import FileSystemEventHandler
    HAS_WATCHDOG = True
except ImportError:
    HAS_WATCHDOG = False
    Observer = None
    FileSystemEventHandler = None

from .config import SourceConfig, TenantConfig
from .wal import WALManager
from .index import IndexManager
from .metadata import MetadataManager

logger = logging.getLogger(__name__)


class DataSource:
    """Manages a single data source with parquet files, WAL, indexing"""
    
    def __init__(self, config: SourceConfig, tenant_config: TenantConfig):
        self.config = config
        self.tenant_config = tenant_config
        self.source_id = config.source_id
        
        # Initialize paths
        self.data_path = tenant_config.get_source_data_path(config.source_id)
        self.parquet_path = self.data_path / "parquet"
        
        # Initialize managers
        self.wal_manager = WALManager(
            tenant_config.get_wal_path(config.source_id),
            retention_hours=tenant_config.wal_retention_hours
        )
        
        self.index_manager = IndexManager(
            tenant_config.get_index_path(config.source_id),
            config.index_columns
        )
        
        self.metadata_manager = MetadataManager(
            tenant_config.get_metadata_path(config.source_id)
        )
        
        self._ensure_directories()
        self._file_observer = None
        
    def _ensure_directories(self):
        """Ensure all required directories exist"""
        self.data_path.mkdir(parents=True, exist_ok=True)
        self.parquet_path.mkdir(parents=True, exist_ok=True)
        
    async def initialize(self):
        """Initialize the data source"""
        logger.info(f"Initializing data source: {self.source_id}")
        
        await self.wal_manager.initialize()
        await self.index_manager.initialize()
        await self.metadata_manager.initialize()
        
        # Start file system monitoring
        self._start_file_monitoring()
        
        logger.info(f"Data source initialized: {self.source_id}")
    
    def _start_file_monitoring(self):
        """Start monitoring file system changes"""
        if not HAS_WATCHDOG:
            logger.warning("Watchdog not available, file monitoring disabled")
            return
            
        class ChangeHandler(FileSystemEventHandler):
            def __init__(self, data_source):
                self.data_source = data_source
                
            def on_modified(self, event):
                if not event.is_directory and event.src_path.endswith('.parquet'):
                    asyncio.create_task(
                        self.data_source._on_file_changed(event.src_path)
                    )
        
        self._file_observer = Observer()
        self._file_observer.schedule(
            ChangeHandler(self),
            str(self.parquet_path),
            recursive=True
        )
        self._file_observer.start()
    
    async def _on_file_changed(self, file_path: str):
        """Handle file system changes"""
        logger.debug(f"File changed: {file_path}")
        # Update metadata and indices as needed
        await self.metadata_manager.update_file_metadata(file_path)
        await self.index_manager.update_indices_for_file(file_path)
    
    async def write_data(self, data: pd.DataFrame, partition_by: Optional[List[str]] = None) -> str:
        """Write data to the source with WAL logging"""
        write_id = str(uuid.uuid4())
        
        try:
            # Log to WAL first
            await self.wal_manager.log_write_operation(write_id, data.to_dict('records'))
            
            # Determine partition strategy
            partition_cols = partition_by or self.config.partition_columns
            
            # Write to parquet
            file_path = await self._write_parquet_file(data, write_id, partition_cols)
            
            # Update metadata
            await self.metadata_manager.register_file(
                file_path,
                {
                    'write_id': write_id,
                    'row_count': len(data),
                    'columns': list(data.columns),
                    'partitions': partition_cols,
                    'created_at': datetime.now(timezone.utc).isoformat(),
                    'file_size': Path(file_path).stat().st_size
                }
            )
            
            # Update indices
            await self.index_manager.update_indices_for_data(data, file_path)
            
            # Mark WAL operation as complete
            await self.wal_manager.mark_operation_complete(write_id)
            
            logger.info(f"Data written successfully: {write_id}, file: {file_path}")
            return write_id
            
        except Exception as e:
            # Mark WAL operation as failed
            await self.wal_manager.mark_operation_failed(write_id, str(e))
            logger.error(f"Failed to write data: {write_id}, error: {e}")
            raise
    
    async def _write_parquet_file(self, data: pd.DataFrame, write_id: str, partition_cols: List[str]) -> str:
        """Write data to parquet file"""
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        file_name = f"{timestamp}_{write_id}.parquet"
        
        if partition_cols:
            # Partitioned write
            table = pa.Table.from_pandas(data)
            file_path = self.parquet_path / "partitioned" / file_name
            file_path.parent.mkdir(parents=True, exist_ok=True)
            
            pq.write_table(
                table,
                str(file_path),
                compression=self.config.compression,
                partition_cols=partition_cols
            )
        else:
            # Single file write
            file_path = self.parquet_path / file_name
            data.to_parquet(
                str(file_path),
                compression=self.config.compression,
                index=False
            )
        
        return str(file_path)
    
    async def search_data(self, query_filter: Dict[str, Any], 
                         limit: Optional[int] = None,
                         offset: int = 0) -> AsyncIterator[pd.DataFrame]:
        """Search data with filtering and pagination"""
        logger.info(f"Searching data in source {self.source_id}")
        
        # Get relevant files from metadata
        relevant_files = await self.metadata_manager.get_files_for_query(query_filter)
        
        if not relevant_files:
            return
        
        # Use index to optimize query
        optimized_files = await self.index_manager.optimize_file_list(
            relevant_files, query_filter
        )
        
        processed_rows = 0
        returned_rows = 0
        
        for file_path in optimized_files:
            if limit and returned_rows >= limit:
                break
                
            try:
                # Read parquet file - use polars if available, otherwise pandas
                if HAS_POLARS:
                    df = pl.read_parquet(file_path)
                    # Apply filters
                    filtered_df = await self._apply_filters_polars(df, query_filter)
                    
                    if filtered_df.is_empty():
                        continue
                    
                    # Handle pagination
                    if offset > processed_rows + len(filtered_df):
                        processed_rows += len(filtered_df)
                        continue
                    
                    # Calculate slice for this chunk
                    start_idx = max(0, offset - processed_rows)
                    end_idx = len(filtered_df)
                    
                    if limit:
                        remaining = limit - returned_rows
                        end_idx = min(end_idx, start_idx + remaining)
                    
                    chunk = filtered_df.slice(start_idx, end_idx - start_idx)
                    
                    if not chunk.is_empty():
                        yield chunk.to_pandas()
                        returned_rows += len(chunk)
                    
                    processed_rows += len(filtered_df)
                    
                else:
                    # Use pandas fallback
                    df = pd.read_parquet(file_path)
                    # Apply filters
                    filtered_df = await self._apply_filters_pandas(df, query_filter)
                    
                    if filtered_df.empty:
                        continue
                    
                    # Handle pagination
                    if offset > processed_rows + len(filtered_df):
                        processed_rows += len(filtered_df)
                        continue
                    
                    # Calculate slice for this chunk
                    start_idx = max(0, offset - processed_rows)
                    end_idx = len(filtered_df)
                    
                    if limit:
                        remaining = limit - returned_rows
                        end_idx = min(end_idx, start_idx + remaining)
                    
                    chunk = filtered_df.iloc[start_idx:end_idx]
                    
                    if not chunk.empty:
                        yield chunk
                        returned_rows += len(chunk)
                    
                    processed_rows += len(filtered_df)
                
            except Exception as e:
                logger.error(f"Error reading file {file_path}: {e}")
                continue
    
    async def _apply_filters_polars(self, df, filters: Dict[str, Any]):
        """Apply query filters to polars dataframe"""
        if not HAS_POLARS:
            raise ValueError("Polars not available")
            
        result = df
        
        for column, condition in filters.items():
            if column not in df.columns:
                continue
                
            if isinstance(condition, dict):
                # Complex condition
                if "$eq" in condition:
                    result = result.filter(pl.col(column) == condition["$eq"])
                elif "$gt" in condition:
                    result = result.filter(pl.col(column) > condition["$gt"])
                elif "$lt" in condition:
                    result = result.filter(pl.col(column) < condition["$lt"])
                elif "$in" in condition:
                    result = result.filter(pl.col(column).is_in(condition["$in"]))
                elif "$like" in condition:
                    result = result.filter(pl.col(column).str.contains(condition["$like"]))
            else:
                # Simple equality
                result = result.filter(pl.col(column) == condition)
        
        return result
    
    async def _apply_filters_pandas(self, df: pd.DataFrame, filters: Dict[str, Any]) -> pd.DataFrame:
        """Apply query filters to pandas dataframe"""
        result = df.copy()
        
        for column, condition in filters.items():
            if column not in df.columns:
                continue
                
            if isinstance(condition, dict):
                # Complex condition
                if "$eq" in condition:
                    result = result[result[column] == condition["$eq"]]
                elif "$gt" in condition:
                    result = result[result[column] > condition["$gt"]]
                elif "$lt" in condition:
                    result = result[result[column] < condition["$lt"]]
                elif "$in" in condition:
                    result = result[result[column].isin(condition["$in"])]
                elif "$like" in condition:
                    result = result[result[column].str.contains(condition["$like"], na=False)]
            else:
                # Simple equality
                result = result[result[column] == condition]
        
        return result
    
    async def aggregate_data(self, aggregations: List[Dict[str, Any]], 
                           filters: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Perform aggregations on the data"""
        logger.info(f"Performing aggregations in source {self.source_id}")
        
        results = {}
        
        # Get relevant files
        relevant_files = await self.metadata_manager.get_files_for_query(filters or {})
        
        for agg_config in aggregations:
            agg_type = agg_config.get("type")
            column = agg_config.get("column")
            alias = agg_config.get("alias", f"{agg_type}_{column}")
            
            if agg_type == "count":
                results[alias] = await self._count_aggregate(relevant_files, filters)
            elif agg_type == "sum":
                results[alias] = await self._sum_aggregate(relevant_files, column, filters)
            elif agg_type == "avg":
                results[alias] = await self._avg_aggregate(relevant_files, column, filters)
            elif agg_type == "min":
                results[alias] = await self._min_aggregate(relevant_files, column, filters)
            elif agg_type == "max":
                results[alias] = await self._max_aggregate(relevant_files, column, filters)
        
        return results
    
    async def _count_aggregate(self, files: List[str], filters: Optional[Dict[str, Any]]) -> int:
        """Count aggregation"""
        total_count = 0
        
        for file_path in files:
            try:
                df = pd.read_parquet(file_path)
                if filters:
                    df = await self._apply_filters_pandas(df, filters)
                total_count += len(df)
            except Exception as e:
                logger.error(f"Error in count aggregation for {file_path}: {e}")
        
        return total_count
    
    async def _sum_aggregate(self, files: List[str], column: str, 
                           filters: Optional[Dict[str, Any]]) -> float:
        """Sum aggregation"""
        total_sum = 0.0
        
        for file_path in files:
            try:
                df = pd.read_parquet(file_path)
                if filters:
                    df = await self._apply_filters_pandas(df, filters)
                if column in df.columns:
                    total_sum += df[column].sum()
            except Exception as e:
                logger.error(f"Error in sum aggregation for {file_path}: {e}")
        
        return total_sum
    
    async def _avg_aggregate(self, files: List[str], column: str,
                           filters: Optional[Dict[str, Any]]) -> float:
        """Average aggregation"""
        total_sum = 0.0
        total_count = 0
        
        for file_path in files:
            try:
                df = pd.read_parquet(file_path)
                if filters:
                    df = await self._apply_filters_pandas(df, filters)
                if column in df.columns and len(df) > 0:
                    total_sum += df[column].sum()
                    total_count += len(df)
            except Exception as e:
                logger.error(f"Error in avg aggregation for {file_path}: {e}")
        
        return total_sum / total_count if total_count > 0 else 0.0
    
    async def _min_aggregate(self, files: List[str], column: str,
                           filters: Optional[Dict[str, Any]]) -> Any:
        """Min aggregation"""
        min_value = None
        
        for file_path in files:
            try:
                df = pd.read_parquet(file_path)
                if filters:
                    df = await self._apply_filters_pandas(df, filters)
                if column in df.columns and len(df) > 0:
                    file_min = df[column].min()
                    if min_value is None or file_min < min_value:
                        min_value = file_min
            except Exception as e:
                logger.error(f"Error in min aggregation for {file_path}: {e}")
        
        return min_value
    
    async def _max_aggregate(self, files: List[str], column: str,
                           filters: Optional[Dict[str, Any]]) -> Any:
        """Max aggregation"""
        max_value = None
        
        for file_path in files:
            try:
                df = pd.read_parquet(file_path)
                if filters:
                    df = await self._apply_filters_pandas(df, filters)
                if column in df.columns and len(df) > 0:
                    file_max = df[column].max()
                    if max_value is None or file_max > max_value:
                        max_value = file_max
            except Exception as e:
                logger.error(f"Error in max aggregation for {file_path}: {e}")
        
        return max_value
    
    async def get_schema(self) -> Dict[str, Any]:
        """Get the schema information for this source"""
        return await self.metadata_manager.get_schema_info()
    
    async def get_statistics(self) -> Dict[str, Any]:
        """Get statistics about this data source"""
        return {
            "source_id": self.source_id,
            "total_files": await self.metadata_manager.get_file_count(),
            "total_size_bytes": await self.metadata_manager.get_total_size(),
            "total_rows": await self.metadata_manager.get_total_rows(),
            "last_updated": await self.metadata_manager.get_last_updated(),
            "wal_status": await self.wal_manager.get_status(),
            "index_status": await self.index_manager.get_status()
        }
    
    async def cleanup(self):
        """Cleanup resources"""
        if self._file_observer and HAS_WATCHDOG:
            self._file_observer.stop()
            self._file_observer.join()
        
        await self.wal_manager.cleanup()
        await self.index_manager.cleanup()
        await self.metadata_manager.cleanup()


class SourceManager:
    """Manages multiple data sources for a tenant"""
    
    def __init__(self, tenant_config: TenantConfig):
        self.tenant_config = tenant_config
        self.sources: Dict[str, DataSource] = {}
        
    async def initialize(self):
        """Initialize all configured sources"""
        logger.info(f"Initializing source manager for tenant: {self.tenant_config.tenant_id}")
        
        for source_id, source_config in self.tenant_config.sources.items():
            await self.add_source(source_config)
        
        logger.info(f"Source manager initialized with {len(self.sources)} sources")
    
    async def add_source(self, source_config: SourceConfig):
        """Add a new data source"""
        logger.info(f"Adding data source: {source_config.source_id}")
        
        source = DataSource(source_config, self.tenant_config)
        await source.initialize()
        
        self.sources[source_config.source_id] = source
        
        logger.info(f"Data source added: {source_config.source_id}")
    
    async def remove_source(self, source_id: str):
        """Remove a data source"""
        if source_id in self.sources:
            await self.sources[source_id].cleanup()
            del self.sources[source_id]
            logger.info(f"Data source removed: {source_id}")
    
    async def get_source(self, source_id: str) -> Optional[DataSource]:
        """Get a specific data source"""
        return self.sources.get(source_id)
    
    async def list_sources(self) -> List[str]:
        """List all available source IDs"""
        return list(self.sources.keys())
    
    async def search_all_sources(self, query_filter: Dict[str, Any],
                                limit: Optional[int] = None,
                                offset: int = 0) -> AsyncIterator[Dict[str, Any]]:
        """Search across all sources in parallel"""
        logger.info(f"Searching across {len(self.sources)} sources")
        
        # Create search tasks for all sources
        search_tasks = []
        for source_id, source in self.sources.items():
            task = asyncio.create_task(
                self._search_source_with_metadata(source_id, source, query_filter, limit, offset)
            )
            search_tasks.append(task)
        
        # Wait for all searches to complete
        results = await asyncio.gather(*search_tasks, return_exceptions=True)
        
        # Yield results as they come
        for result in results:
            if isinstance(result, Exception):
                logger.error(f"Search error: {result}")
                continue
            
            if result:
                yield result
    
    async def _search_source_with_metadata(self, source_id: str, source: DataSource,
                                         query_filter: Dict[str, Any],
                                         limit: Optional[int],
                                         offset: int) -> Optional[Dict[str, Any]]:
        """Search a single source and return results with metadata"""
        try:
            results = []
            async for chunk in source.search_data(query_filter, limit, offset):
                results.append(chunk.to_dict('records'))
            
            if results:
                return {
                    "source_id": source_id,
                    "data": results,
                    "total_chunks": len(results),
                    "total_rows": sum(len(chunk) for chunk in results)
                }
        
        except Exception as e:
            logger.error(f"Error searching source {source_id}: {e}")
            return None
    
    async def aggregate_all_sources(self, aggregations: List[Dict[str, Any]],
                                  filters: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Perform aggregations across all sources"""
        logger.info(f"Performing aggregations across {len(self.sources)} sources")
        
        # Create aggregation tasks for all sources
        agg_tasks = []
        for source_id, source in self.sources.items():
            task = asyncio.create_task(
                self._aggregate_source_with_metadata(source_id, source, aggregations, filters)
            )
            agg_tasks.append(task)
        
        # Wait for all aggregations to complete
        results = await asyncio.gather(*agg_tasks, return_exceptions=True)
        
        # Combine results
        combined_results = {
            "tenant_id": self.tenant_config.tenant_id,
            "sources": {},
            "aggregated": {}
        }
        
        for result in results:
            if isinstance(result, Exception):
                logger.error(f"Aggregation error: {result}")
                continue
            
            if result:
                source_id = result["source_id"]
                combined_results["sources"][source_id] = result["aggregations"]
        
        # Calculate final aggregations
        combined_results["aggregated"] = self._combine_aggregations(
            aggregations, 
            [r["aggregations"] for r in results if not isinstance(r, Exception) and r]
        )
        
        return combined_results
    
    async def _aggregate_source_with_metadata(self, source_id: str, source: DataSource,
                                            aggregations: List[Dict[str, Any]],
                                            filters: Optional[Dict[str, Any]]) -> Dict[str, Any]:
        """Aggregate a single source and return results with metadata"""
        try:
            agg_results = await source.aggregate_data(aggregations, filters)
            return {
                "source_id": source_id,
                "aggregations": agg_results
            }
        except Exception as e:
            logger.error(f"Error aggregating source {source_id}: {e}")
            return {"source_id": source_id, "aggregations": {}, "error": str(e)}
    
    def _combine_aggregations(self, agg_configs: List[Dict[str, Any]], 
                            source_results: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Combine aggregation results from multiple sources"""
        combined = {}
        
        for agg_config in agg_configs:
            agg_type = agg_config.get("type")
            alias = agg_config.get("alias", f"{agg_type}_{agg_config.get('column', '')}")
            
            if agg_type == "count":
                combined[alias] = sum(result.get(alias, 0) for result in source_results)
            elif agg_type == "sum":
                combined[alias] = sum(result.get(alias, 0) for result in source_results)
            elif agg_type == "avg":
                # Weighted average would require more complex calculation
                # For now, simple average of averages
                values = [result.get(alias, 0) for result in source_results if result.get(alias, 0) > 0]
                combined[alias] = sum(values) / len(values) if values else 0
            elif agg_type == "min":
                values = [result.get(alias) for result in source_results if result.get(alias) is not None]
                combined[alias] = min(values) if values else None
            elif agg_type == "max":
                values = [result.get(alias) for result in source_results if result.get(alias) is not None]
                combined[alias] = max(values) if values else None
        
        return combined
    
    async def get_tenant_statistics(self) -> Dict[str, Any]:
        """Get comprehensive statistics for the tenant"""
        stats = {
            "tenant_id": self.tenant_config.tenant_id,
            "tenant_name": self.tenant_config.tenant_name,
            "total_sources": len(self.sources),
            "sources": {}
        }
        
        # Get statistics from each source
        for source_id, source in self.sources.items():
            try:
                source_stats = await source.get_statistics()
                stats["sources"][source_id] = source_stats
            except Exception as e:
                logger.error(f"Error getting statistics for source {source_id}: {e}")
                stats["sources"][source_id] = {"error": str(e)}
        
        # Calculate totals
        stats["total_files"] = sum(
            s.get("total_files", 0) for s in stats["sources"].values() 
            if isinstance(s, dict) and "error" not in s
        )
        stats["total_size_bytes"] = sum(
            s.get("total_size_bytes", 0) for s in stats["sources"].values()
            if isinstance(s, dict) and "error" not in s
        )
        stats["total_rows"] = sum(
            s.get("total_rows", 0) for s in stats["sources"].values()
            if isinstance(s, dict) and "error" not in s
        )
        
        return stats
    
    async def cleanup(self):
        """Cleanup all sources"""
        logger.info("Cleaning up source manager")
        
        cleanup_tasks = [source.cleanup() for source in self.sources.values()]
        await asyncio.gather(*cleanup_tasks, return_exceptions=True)
        
        self.sources.clear()
        
        logger.info("Source manager cleanup complete")
