"""
REST API for Tenant Node using FastAPI
"""
import asyncio
import logging
from typing import Dict, List, Any, Optional
from datetime import datetime
import pandas as pd
from fastapi import FastAPI, HTTPException, Query, Body
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
import json
import uvicorn

from config import TenantConfig, SourceConfig
from data_source import SourceManager

logger = logging.getLogger(__name__)


# Pydantic models for request/response
class DataRecord(BaseModel):
    fields: Dict[str, Any]


class WriteDataRequest(BaseModel):
    source_id: str
    records: List[DataRecord]
    partition_columns: Optional[List[str]] = None
    metadata: Optional[Dict[str, str]] = None


class WriteDataResponse(BaseModel):
    success: bool
    write_id: Optional[str] = None
    error_message: Optional[str] = None
    rows_written: int = 0


class FilterCondition(BaseModel):
    eq: Optional[str] = None
    ne: Optional[str] = None
    gt: Optional[str] = None
    lt: Optional[str] = None
    gte: Optional[str] = None
    lte: Optional[str] = None
    in_list: Optional[List[str]] = None
    not_in: Optional[List[str]] = None
    like: Optional[str] = None
    regex: Optional[str] = None


class SearchDataRequest(BaseModel):
    source_ids: Optional[List[str]] = None
    filters: Optional[Dict[str, FilterCondition]] = None
    limit: Optional[int] = None
    offset: int = 0
    select_columns: Optional[List[str]] = None


class SearchDataResponse(BaseModel):
    source_id: str
    records: List[Dict[str, Any]]
    total_rows: int
    has_more: bool
    error_message: Optional[str] = None


class AggregationSpec(BaseModel):
    type: str  # count, sum, avg, min, max
    column: Optional[str] = None  # Not needed for count
    alias: Optional[str] = None


class AggregateDataRequest(BaseModel):
    source_ids: Optional[List[str]] = None
    filters: Optional[Dict[str, FilterCondition]] = None
    aggregations: List[AggregationSpec]


class AggregateDataResponse(BaseModel):
    success: bool
    tenant_id: str
    source_results: Dict[str, Dict[str, float]]
    final_results: Dict[str, float]
    error_message: Optional[str] = None


class SourceConfigRequest(BaseModel):
    source_id: str
    name: str
    connection_string: str
    data_path: str
    schema_definition: Dict[str, str]
    partition_columns: Optional[List[str]] = None
    index_columns: Optional[List[str]] = None
    compression: str = "snappy"
    max_file_size_mb: int = 256
    wal_enabled: bool = True


class SimpleResponse(BaseModel):
    success: bool
    error_message: Optional[str] = None


class SourceInfo(BaseModel):
    source_id: str
    name: str
    total_files: int
    total_size_bytes: int
    total_rows: int
    last_updated: Optional[str] = None


class ListSourcesResponse(BaseModel):
    source_ids: List[str]
    source_details: Dict[str, SourceInfo]


class SourceStatsResponse(BaseModel):
    success: bool
    stats: Optional[SourceInfo] = None
    detailed_stats: Dict[str, Any]
    error_message: Optional[str] = None


class TenantStatsResponse(BaseModel):
    tenant_id: str
    tenant_name: str
    total_sources: int
    total_files: int
    total_size_bytes: int
    total_rows: int
    source_stats: Dict[str, SourceInfo]


class HealthCheckResponse(BaseModel):
    healthy: bool
    status: str
    details: Dict[str, str]


class TenantNodeAPI:
    """REST API for tenant node operations"""
    
    def __init__(self, tenant_config: TenantConfig, source_manager: SourceManager):
        self.tenant_config = tenant_config
        self.source_manager = source_manager
        self.app = FastAPI(
            title="Tenant Node API",
            description="REST API for hybrid columnar and JSON storage system tenant node",
            version="1.0.0"
        )
        
        self._setup_routes()
    
    def _setup_routes(self):
        """Setup FastAPI routes"""
        
        @self.app.post("/data/write", response_model=WriteDataResponse)
        async def write_data(request: WriteDataRequest):
            """Write data to a source"""
            try:
                # Convert request to DataFrame
                records = [record.fields for record in request.records]
                df = pd.DataFrame(records)
                
                # Get the source
                source = await self.source_manager.get_source(request.source_id)
                if not source:
                    raise HTTPException(status_code=404, detail=f"Source not found: {request.source_id}")
                
                # Write data
                write_id = await source.write_data(df, partition_by=request.partition_columns)
                
                return WriteDataResponse(
                    success=True,
                    write_id=write_id,
                    rows_written=len(records)
                )
                
            except Exception as e:
                logger.error(f"Error writing data: {e}")
                return WriteDataResponse(
                    success=False,
                    error_message=str(e)
                )
        
        @self.app.post("/data/search")
        async def search_data(request: SearchDataRequest):
            """Search data across sources"""
            try:
                # Convert filters
                filters = self._convert_rest_filters(request.filters) if request.filters else {}
                
                # Determine sources to search
                source_ids = request.source_ids if request.source_ids else await self.source_manager.list_sources()
                
                results = []
                
                if request.source_ids:
                    # Search specific sources
                    for source_id in source_ids:
                        source = await self.source_manager.get_source(source_id)
                        if not source:
                            continue
                        
                        try:
                            source_results = []
                            async for chunk in source.search_data(filters, request.limit, request.offset):
                                source_results.extend(chunk.to_dict('records'))
                            
                            if source_results:
                                results.append(SearchDataResponse(
                                    source_id=source_id,
                                    records=source_results,
                                    total_rows=len(source_results),
                                    has_more=False  # Simplified
                                ))
                        
                        except Exception as e:
                            logger.error(f"Error searching source {source_id}: {e}")
                            results.append(SearchDataResponse(
                                source_id=source_id,
                                records=[],
                                total_rows=0,
                                has_more=False,
                                error_message=str(e)
                            ))
                else:
                    # Search all sources
                    async for result in self.source_manager.search_all_sources(filters, request.limit, request.offset):
                        if result and "data" in result:
                            all_records = []
                            for chunk_data in result["data"]:
                                all_records.extend(chunk_data)
                            
                            results.append(SearchDataResponse(
                                source_id=result["source_id"],
                                records=all_records,
                                total_rows=len(all_records),
                                has_more=False
                            ))
                
                return results
                
            except Exception as e:
                logger.error(f"Error searching data: {e}")
                raise HTTPException(status_code=500, detail=str(e))
        
        @self.app.post("/data/search/stream")
        async def search_data_stream(request: SearchDataRequest):
            """Search data with streaming response"""
            try:
                # Convert filters
                filters = self._convert_rest_filters(request.filters) if request.filters else {}
                
                async def generate_results():
                    # Search all sources
                    async for result in self.source_manager.search_all_sources(filters, request.limit, request.offset):
                        if result and "data" in result:
                            for chunk_data in result["data"]:
                                response = SearchDataResponse(
                                    source_id=result["source_id"],
                                    records=chunk_data,
                                    total_rows=len(chunk_data),
                                    has_more=False
                                )
                                yield f"data: {response.model_dump_json()}\n\n"
                
                return StreamingResponse(
                    generate_results(),
                    media_type="text/plain",
                    headers={"Content-Type": "text/event-stream"}
                )
                
            except Exception as e:
                logger.error(f"Error in streaming search: {e}")
                raise HTTPException(status_code=500, detail=str(e))
        
        @self.app.post("/data/aggregate", response_model=AggregateDataResponse)
        async def aggregate_data(request: AggregateDataRequest):
            """Perform data aggregations"""
            try:
                # Convert filters and aggregations
                filters = self._convert_rest_filters(request.filters) if request.filters else {}
                aggregations = self._convert_rest_aggregations(request.aggregations)
                
                # Perform aggregation
                result = await self.source_manager.aggregate_all_sources(aggregations, filters)
                
                return AggregateDataResponse(
                    success="error" not in result,
                    tenant_id=result.get("tenant_id", self.tenant_config.tenant_id),
                    source_results=result.get("sources", {}),
                    final_results=result.get("aggregated", {}),
                    error_message=result.get("error", "")
                )
                
            except Exception as e:
                logger.error(f"Error aggregating data: {e}")
                return AggregateDataResponse(
                    success=False,
                    tenant_id=self.tenant_config.tenant_id,
                    source_results={},
                    final_results={},
                    error_message=str(e)
                )
        
        @self.app.post("/sources/add", response_model=SimpleResponse)
        async def add_source(config: SourceConfigRequest):
            """Add a new data source"""
            try:
                source_config = SourceConfig(
                    source_id=config.source_id,
                    name=config.name,
                    connection_string=config.connection_string,
                    data_path=config.data_path,
                    schema_definition=config.schema_definition,
                    partition_columns=config.partition_columns or [],
                    index_columns=config.index_columns or [],
                    compression=config.compression,
                    max_file_size_mb=config.max_file_size_mb,
                    wal_enabled=config.wal_enabled
                )
                
                await self.source_manager.add_source(source_config)
                
                return SimpleResponse(success=True)
                
            except Exception as e:
                logger.error(f"Error adding source: {e}")
                return SimpleResponse(success=False, error_message=str(e))
        
        @self.app.delete("/sources/{source_id}", response_model=SimpleResponse)
        async def remove_source(source_id: str):
            """Remove a data source"""
            try:
                await self.source_manager.remove_source(source_id)
                return SimpleResponse(success=True)
                
            except Exception as e:
                logger.error(f"Error removing source: {e}")
                return SimpleResponse(success=False, error_message=str(e))
        
        @self.app.get("/sources", response_model=ListSourcesResponse)
        async def list_sources():
            """List all data sources"""
            try:
                source_ids = await self.source_manager.list_sources()
                
                # Get detailed info for each source
                source_details = {}
                for source_id in source_ids:
                    source = await self.source_manager.get_source(source_id)
                    if source:
                        stats = await source.get_statistics()
                        source_details[source_id] = SourceInfo(
                            source_id=stats["source_id"],
                            name=stats["source_id"],  # Using ID as name for now
                            total_files=stats["total_files"],
                            total_size_bytes=stats["total_size_bytes"],
                            total_rows=stats["total_rows"],
                            last_updated=stats["last_updated"]
                        )
                
                return ListSourcesResponse(
                    source_ids=source_ids,
                    source_details=source_details
                )
                
            except Exception as e:
                logger.error(f"Error listing sources: {e}")
                return ListSourcesResponse(source_ids=[], source_details={})
        
        @self.app.get("/sources/{source_id}/stats", response_model=SourceStatsResponse)
        async def get_source_stats(source_id: str):
            """Get statistics for a specific source"""
            try:
                source = await self.source_manager.get_source(source_id)
                if not source:
                    raise HTTPException(status_code=404, detail=f"Source not found: {source_id}")
                
                stats = await source.get_statistics()
                source_info = SourceInfo(
                    source_id=stats["source_id"],
                    name=stats["source_id"],
                    total_files=stats["total_files"],
                    total_size_bytes=stats["total_size_bytes"],
                    total_rows=stats["total_rows"],
                    last_updated=stats["last_updated"]
                )
                
                return SourceStatsResponse(
                    success=True,
                    stats=source_info,
                    detailed_stats=stats
                )
                
            except HTTPException:
                raise
            except Exception as e:
                logger.error(f"Error getting source stats: {e}")
                return SourceStatsResponse(
                    success=False,
                    detailed_stats={},
                    error_message=str(e)
                )
        
        @self.app.get("/tenant/stats", response_model=TenantStatsResponse)
        async def get_tenant_stats():
            """Get comprehensive tenant statistics"""
            try:
                stats = await self.source_manager.get_tenant_statistics()
                
                source_stats = {}
                if "sources" in stats:
                    for source_id, source_data in stats["sources"].items():
                        if isinstance(source_data, dict) and "error" not in source_data:
                            source_stats[source_id] = SourceInfo(
                                source_id=source_data["source_id"],
                                name=source_data["source_id"],
                                total_files=source_data["total_files"],
                                total_size_bytes=source_data["total_size_bytes"],
                                total_rows=source_data["total_rows"],
                                last_updated=source_data["last_updated"]
                            )
                
                return TenantStatsResponse(
                    tenant_id=stats["tenant_id"],
                    tenant_name=stats["tenant_name"],
                    total_sources=stats["total_sources"],
                    total_files=stats["total_files"],
                    total_size_bytes=stats["total_size_bytes"],
                    total_rows=stats["total_rows"],
                    source_stats=source_stats
                )
                
            except Exception as e:
                logger.error(f"Error getting tenant stats: {e}")
                raise HTTPException(status_code=500, detail=str(e))
        
        @self.app.get("/health", response_model=HealthCheckResponse)
        async def health_check():
            """Health check endpoint"""
            try:
                source_count = len(await self.source_manager.list_sources())
                
                details = {
                    "tenant_id": self.tenant_config.tenant_id,
                    "source_count": str(source_count),
                    "timestamp": datetime.now().isoformat()
                }
                
                return HealthCheckResponse(
                    healthy=True,
                    status="OK",
                    details=details
                )
                
            except Exception as e:
                logger.error(f"Health check failed: {e}")
                return HealthCheckResponse(
                    healthy=False,
                    status=str(e),
                    details={}
                )
    
    def _convert_rest_filters(self, filters: Dict[str, FilterCondition]) -> Dict[str, Any]:
        """Convert REST API filters to internal format"""
        converted = {}
        
        for field_name, condition in filters.items():
            if condition.eq is not None:
                converted[field_name] = condition.eq
            elif condition.ne is not None:
                converted[field_name] = {"$ne": condition.ne}
            elif condition.gt is not None:
                converted[field_name] = {"$gt": condition.gt}
            elif condition.lt is not None:
                converted[field_name] = {"$lt": condition.lt}
            elif condition.gte is not None:
                converted[field_name] = {"$gte": condition.gte}
            elif condition.lte is not None:
                converted[field_name] = {"$lte": condition.lte}
            elif condition.in_list is not None:
                converted[field_name] = {"$in": condition.in_list}
            elif condition.not_in is not None:
                converted[field_name] = {"$not_in": condition.not_in}
            elif condition.like is not None:
                converted[field_name] = {"$like": condition.like}
            elif condition.regex is not None:
                converted[field_name] = {"$regex": condition.regex}
        
        return converted
    
    def _convert_rest_aggregations(self, aggregations: List[AggregationSpec]) -> List[Dict[str, Any]]:
        """Convert REST API aggregations to internal format"""
        converted = []
        
        for agg in aggregations:
            converted.append({
                "type": agg.type,
                "column": agg.column,
                "alias": agg.alias or f"{agg.type}_{agg.column}" if agg.column else agg.type
            })
        
        return converted
    
    async def start(self):
        """Start the REST API server"""
        config = uvicorn.Config(
            self.app,
            host=self.tenant_config.rest_host,
            port=self.tenant_config.rest_port,
            log_level="info"
        )
        
        server = uvicorn.Server(config)
        
        logger.info(f"Tenant Node REST API starting on {self.tenant_config.rest_host}:{self.tenant_config.rest_port}")
        
        await server.serve()
    
    def get_app(self):
        """Get the FastAPI application"""
        return self.app
