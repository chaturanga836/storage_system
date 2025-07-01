"""
gRPC service implementation for Tenant Node
"""
import asyncio
import logging
import time
from typing import Dict, Any, List, AsyncIterator
import pandas as pd
import grpc
from grpc import aio

from .config import TenantConfig, SourceConfig
from .data_source import SourceManager

# Import generated gRPC files (these will be generated from proto)
# from .generated import storage_pb2, storage_pb2_grpc

logger = logging.getLogger(__name__)


class TenantNodeServicer:  # Extends storage_pb2_grpc.TenantNodeServicer
    """gRPC service implementation for tenant node operations"""
    
    def __init__(self, tenant_config: TenantConfig, source_manager: SourceManager):
        self.tenant_config = tenant_config
        self.source_manager = source_manager
        
    async def WriteData(self, request, context):
        """Handle data write requests"""
        try:
            # Convert gRPC request to pandas DataFrame
            records = []
            for record in request.records:
                row = {}
                for field_name, value in record.fields.items():
                    row[field_name] = self._extract_value(value)
                records.append(row)
            
            df = pd.DataFrame(records)
            
            # Get the source
            source = await self.source_manager.get_source(request.source_id)
            if not source:
                context.set_code(grpc.StatusCode.NOT_FOUND)
                context.set_details(f"Source not found: {request.source_id}")
                return self._create_write_response(False, "", "Source not found", 0)
            
            # Write data
            write_id = await source.write_data(
                df, 
                partition_by=list(request.partition_columns) if request.partition_columns else None
            )
            
            return self._create_write_response(True, write_id, "", len(records))
            
        except Exception as e:
            logger.error(f"Error in WriteData: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return self._create_write_response(False, "", str(e), 0)
    
    async def SearchData(self, request, context) -> AsyncIterator:
        """Handle data search requests with streaming response"""
        try:
            # Convert gRPC filters to internal format
            filters = self._convert_filters(request.filters)
            
            # Determine which sources to search
            source_ids = list(request.source_ids) if request.source_ids else await self.source_manager.list_sources()
            
            if not source_ids:
                return
            
            # Generate optimized execution plan if query optimizer is available
            execution_plan = None
            if hasattr(self.source_manager, 'query_optimizer') and self.source_manager.query_optimizer:
                try:
                    execution_plan = await self.source_manager.query_optimizer.optimize_query(
                        filters, [], source_ids, request.limit
                    )
                    logger.info(f"Using optimized execution plan: {execution_plan.plan_id}")
                except Exception as e:
                    logger.warning(f"Query optimization failed, using default execution: {e}")
            
            # Register query with auto-scaler if available
            query_id = f"search_{id(request)}"
            if hasattr(self.source_manager, 'auto_scaler') and self.source_manager.auto_scaler:
                self.source_manager.auto_scaler.register_query_start(query_id)
            
            start_time = time.time()
            
            try:
                # Search individual sources if specific sources requested
                if request.source_ids:
                    for source_id in source_ids:
                        source = await self.source_manager.get_source(source_id)
                        if not source:
                            continue
                        
                        try:
                            async for chunk in source.search_data(filters, request.limit, request.offset):
                                response = self._create_search_response(
                                    source_id, 
                                    chunk, 
                                    has_more=True,  # Simplified for now
                                    total_rows=len(chunk)
                                )
                                yield response
                        except Exception as e:
                            logger.error(f"Error searching source {source_id}: {e}")
                            response = self._create_search_response(
                                source_id, 
                                pd.DataFrame(), 
                                has_more=False, 
                                total_rows=0,
                                error=str(e)
                            )
                            yield response
                else:
                    # Search all sources in parallel
                    async for result in self.source_manager.search_all_sources(filters, request.limit, request.offset):
                        if result and "data" in result:
                            for chunk_data in result["data"]:
                                chunk_df = pd.DataFrame(chunk_data)
                                response = self._create_search_response(
                                    result["source_id"],
                                    chunk_df,
                                    has_more=False,  # Simplified
                                    total_rows=len(chunk_df)
                                )
                                yield response
                
                # Record execution stats for learning
                if execution_plan and hasattr(self.source_manager, 'query_optimizer'):
                    execution_time = (time.time() - start_time) * 1000  # Convert to ms
                    await self.source_manager.query_optimizer.record_execution_stats(
                        execution_plan.plan_id, execution_time, 0, True  # rows count would need to be tracked
                    )
                    
            finally:
                # Unregister query with auto-scaler
                if hasattr(self.source_manager, 'auto_scaler') and self.source_manager.auto_scaler:
                    self.source_manager.auto_scaler.register_query_complete(query_id)
            
        except Exception as e:
            logger.error(f"Error in SearchData: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
    
    async def AggregateData(self, request, context):
        """Handle data aggregation requests"""
        try:
            # Convert gRPC filters and aggregations
            filters = self._convert_filters(request.filters)
            aggregations = self._convert_aggregations(request.aggregations)
            
            # Generate optimized execution plan if query optimizer is available
            execution_plan = None
            source_ids = await self.source_manager.list_sources()
            
            if hasattr(self.source_manager, 'query_optimizer') and self.source_manager.query_optimizer:
                try:
                    execution_plan = await self.source_manager.query_optimizer.optimize_query(
                        filters, aggregations, source_ids, None
                    )
                    logger.info(f"Using optimized aggregation plan: {execution_plan.plan_id}")
                except Exception as e:
                    logger.warning(f"Aggregation optimization failed, using default execution: {e}")
            
            # Register query with auto-scaler if available
            query_id = f"aggregate_{id(request)}"
            if hasattr(self.source_manager, 'auto_scaler') and self.source_manager.auto_scaler:
                self.source_manager.auto_scaler.register_query_start(query_id)
            
            start_time = time.time()
            
            try:
                # Perform aggregation
                result = await self.source_manager.aggregate_all_sources(aggregations, filters)
                
                # Record execution stats for learning
                if execution_plan and hasattr(self.source_manager, 'query_optimizer'):
                    execution_time = (time.time() - start_time) * 1000  # Convert to ms
                    await self.source_manager.query_optimizer.record_execution_stats(
                        execution_plan.plan_id, execution_time, len(result.get("aggregated", {})), True
                    )
                
                return self._create_aggregate_response(result)
                
            finally:
                # Unregister query with auto-scaler
                if hasattr(self.source_manager, 'auto_scaler') and self.source_manager.auto_scaler:
                    self.source_manager.auto_scaler.register_query_complete(query_id)
            
        except Exception as e:
            logger.error(f"Error in AggregateData: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return self._create_aggregate_response({"error": str(e)})
    
    async def AddSource(self, request, context):
        """Add a new data source"""
        try:
            # Convert gRPC source config
            source_config = SourceConfig(
                source_id=request.config.source_id,
                name=request.config.name,
                connection_string=request.config.connection_string,
                data_path=request.config.data_path,
                schema_definition=dict(request.config.schema_definition),
                partition_columns=list(request.config.partition_columns),
                index_columns=list(request.config.index_columns),
                compression=request.config.compression or "snappy",
                max_file_size_mb=request.config.max_file_size_mb or 256,
                wal_enabled=request.config.wal_enabled
            )
            
            await self.source_manager.add_source(source_config)
            
            return self._create_simple_response(True)
            
        except Exception as e:
            logger.error(f"Error adding source: {e}")
            return self._create_simple_response(False, str(e))
    
    async def RemoveSource(self, request, context):
        """Remove a data source"""
        try:
            await self.source_manager.remove_source(request.source_id)
            return self._create_simple_response(True)
            
        except Exception as e:
            logger.error(f"Error removing source: {e}")
            return self._create_simple_response(False, str(e))
    
    async def ListSources(self, request, context):
        """List all data sources"""
        try:
            source_ids = await self.source_manager.list_sources()
            
            # Get detailed info for each source
            source_details = {}
            for source_id in source_ids:
                source = await self.source_manager.get_source(source_id)
                if source:
                    stats = await source.get_statistics()
                    source_details[source_id] = self._create_source_info(stats)
            
            return self._create_list_sources_response(source_ids, source_details)
            
        except Exception as e:
            logger.error(f"Error listing sources: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return self._create_list_sources_response([], {})
    
    async def GetSourceStats(self, request, context):
        """Get statistics for a specific source"""
        try:
            source = await self.source_manager.get_source(request.source_id)
            if not source:
                context.set_code(grpc.StatusCode.NOT_FOUND)
                context.set_details(f"Source not found: {request.source_id}")
                return self._create_source_stats_response(False, None, {}, "Source not found")
            
            stats = await source.get_statistics()
            source_info = self._create_source_info(stats)
            
            return self._create_source_stats_response(True, source_info, stats)
            
        except Exception as e:
            logger.error(f"Error getting source stats: {e}")
            return self._create_source_stats_response(False, None, {}, str(e))
    
    async def GetTenantStats(self, request, context):
        """Get comprehensive tenant statistics"""
        try:
            stats = await self.source_manager.get_tenant_statistics()
            return self._create_tenant_stats_response(stats)
            
        except Exception as e:
            logger.error(f"Error getting tenant stats: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return self._create_tenant_stats_response({"error": str(e)})
    
    async def HealthCheck(self, request, context):
        """Health check endpoint"""
        try:
            # Perform basic health checks
            source_count = len(await self.source_manager.list_sources())
            
            details = {
                "tenant_id": self.tenant_config.tenant_id,
                "source_count": str(source_count),
                "status": "healthy"
            }
            
            return self._create_health_response(True, "OK", details)
            
        except Exception as e:
            logger.error(f"Health check failed: {e}")
            return self._create_health_response(False, str(e), {})
    
    async def GetTenantStatus(self, request, context):
        """Get comprehensive tenant status including auto-scaling and compaction"""
        try:
            # Get basic tenant stats
            stats = await self.source_manager.get_tenant_statistics()
            
            # Add auto-scaler status
            auto_scaler_status = {}
            if hasattr(self.source_manager, 'auto_scaler') and self.source_manager.auto_scaler:
                auto_scaler_status = self.source_manager.auto_scaler.get_scaling_status()
            
            # Add compaction status
            compaction_status = {}
            if hasattr(self.source_manager, 'compaction_manager') and self.source_manager.compaction_manager:
                compaction_status = self.source_manager.compaction_manager.get_compaction_status()
            
            # Add query optimizer status
            optimizer_status = {}
            if hasattr(self.source_manager, 'query_optimizer') and self.source_manager.query_optimizer:
                optimizer_status = self.source_manager.query_optimizer.get_optimizer_stats()
            
            # Combine all status information
            comprehensive_status = {
                **stats,
                "auto_scaler": auto_scaler_status,
                "compaction": compaction_status,
                "query_optimizer": optimizer_status,
                "features": {
                    "auto_scaling_enabled": self.tenant_config.auto_scaling_enabled,
                    "auto_compaction_enabled": self.tenant_config.auto_compaction_enabled,
                    "query_optimization_enabled": self.tenant_config.query_optimization_enabled
                }
            }
            
            return self._create_status_response(comprehensive_status)
            
        except Exception as e:
            logger.error(f"Error getting tenant status: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return self._create_status_response({"error": str(e)})
    
    async def TriggerCompaction(self, request, context):
        """Manually trigger compaction for a specific source"""
        try:
            if not hasattr(self.source_manager, 'compaction_manager') or not self.source_manager.compaction_manager:
                context.set_code(grpc.StatusCode.UNIMPLEMENTED)
                context.set_details("Compaction manager not available")
                return self._create_simple_response(False, "Compaction not enabled")
            
            success = await self.source_manager.compaction_manager.trigger_manual_compaction(request.source_id)
            
            return self._create_simple_response(success, "" if success else "Compaction failed")
            
        except Exception as e:
            logger.error(f"Error triggering compaction: {e}")
            return self._create_simple_response(False, str(e))
    
    # Helper methods for converting between gRPC and internal formats
    
    def _extract_value(self, value):
        """Extract value from gRPC Value message"""
        # This would depend on the actual generated protobuf classes
        # For now, return a placeholder
        if hasattr(value, 'string_value') and value.string_value:
            return value.string_value
        elif hasattr(value, 'int_value'):
            return value.int_value
        elif hasattr(value, 'double_value'):
            return value.double_value
        elif hasattr(value, 'bool_value'):
            return value.bool_value
        elif hasattr(value, 'bytes_value'):
            return value.bytes_value
        return None
    
    def _convert_filters(self, grpc_filters) -> Dict[str, Any]:
        """Convert gRPC filters to internal format"""
        filters = {}
        
        for field_name, condition in grpc_filters.items():
            # Convert condition based on type
            if hasattr(condition, 'eq') and condition.eq:
                filters[field_name] = condition.eq
            elif hasattr(condition, 'gt') and condition.gt:
                filters[field_name] = {"$gt": condition.gt}
            elif hasattr(condition, 'lt') and condition.lt:
                filters[field_name] = {"$lt": condition.lt}
            elif hasattr(condition, 'in') and getattr(condition, 'in', None):
                filters[field_name] = {"$in": list(getattr(condition, 'in').values)}
            elif hasattr(condition, 'like') and condition.like:
                filters[field_name] = {"$like": condition.like}
            # Add more condition types as needed
        
        return filters
    
    def _convert_aggregations(self, grpc_aggregations) -> List[Dict[str, Any]]:
        """Convert gRPC aggregations to internal format"""
        aggregations = []
        
        for agg in grpc_aggregations:
            aggregations.append({
                "type": agg.type,
                "column": agg.column,
                "alias": agg.alias or f"{agg.type}_{agg.column}"
            })
        
        return aggregations
    
    # Helper methods for creating response objects
    # These would use the actual generated protobuf classes
    
    def _create_write_response(self, success: bool, write_id: str, error: str, rows: int):
        """Create WriteDataResponse"""
        # This would use the actual generated class
        # return storage_pb2.WriteDataResponse(
        #     success=success,
        #     write_id=write_id,
        #     error_message=error,
        #     rows_written=rows
        # )
        return {
            "success": success,
            "write_id": write_id,
            "error_message": error,
            "rows_written": rows
        }
    
    def _create_search_response(self, source_id: str, df: pd.DataFrame, 
                              has_more: bool, total_rows: int, error: str = ""):
        """Create SearchDataResponse"""
        records = []
        for _, row in df.iterrows():
            record_fields = {}
            for col, val in row.items():
                # Convert value to gRPC Value
                record_fields[col] = self._create_value(val)
            records.append({"fields": record_fields})
        
        return {
            "source_id": source_id,
            "records": records,
            "has_more": has_more,
            "total_rows": total_rows,
            "error_message": error
        }
    
    def _create_value(self, val):
        """Create gRPC Value from Python value"""
        if isinstance(val, str):
            return {"string_value": val}
        elif isinstance(val, int):
            return {"int_value": val}
        elif isinstance(val, float):
            return {"double_value": val}
        elif isinstance(val, bool):
            return {"bool_value": val}
        else:
            return {"string_value": str(val)}
    
    def _create_aggregate_response(self, result: Dict[str, Any]):
        """Create AggregateDataResponse"""
        return {
            "success": "error" not in result,
            "tenant_id": result.get("tenant_id", self.tenant_config.tenant_id),
            "source_results": result.get("sources", {}),
            "final_results": result.get("aggregated", {}),
            "error_message": result.get("error", "")
        }
    
    def _create_simple_response(self, success: bool, error: str = ""):
        """Create simple success/error response"""
        return {
            "success": success,
            "error_message": error
        }
    
    def _create_source_info(self, stats: Dict[str, Any]):
        """Create SourceInfo from statistics"""
        return {
            "source_id": stats.get("source_id", ""),
            "name": stats.get("source_id", ""),  # Using ID as name for now
            "total_files": stats.get("total_files", 0),
            "total_size_bytes": stats.get("total_size_bytes", 0),
            "total_rows": stats.get("total_rows", 0),
            "last_updated": stats.get("last_updated", "")
        }
    
    def _create_list_sources_response(self, source_ids: List[str], source_details: Dict):
        """Create ListSourcesResponse"""
        return {
            "source_ids": source_ids,
            "source_details": source_details
        }
    
    def _create_source_stats_response(self, success: bool, stats, detailed_stats: Dict, error: str = ""):
        """Create GetSourceStatsResponse"""
        return {
            "success": success,
            "stats": stats,
            "detailed_stats": detailed_stats,
            "error_message": error
        }
    
    def _create_tenant_stats_response(self, stats: Dict[str, Any]):
        """Create GetTenantStatsResponse"""
        source_stats = {}
        if "sources" in stats:
            for source_id, source_data in stats["sources"].items():
                if isinstance(source_data, dict) and "error" not in source_data:
                    source_stats[source_id] = self._create_source_info(source_data)
        
        return {
            "tenant_id": stats.get("tenant_id", self.tenant_config.tenant_id),
            "tenant_name": stats.get("tenant_name", self.tenant_config.tenant_name),
            "total_sources": stats.get("total_sources", 0),
            "total_files": stats.get("total_files", 0),
            "total_size_bytes": stats.get("total_size_bytes", 0),
            "total_rows": stats.get("total_rows", 0),
            "source_stats": source_stats
        }
    
    def _create_health_response(self, healthy: bool, status: str, details: Dict):
        """Create HealthCheckResponse"""
        return {
            "healthy": healthy,
            "status": status,
            "details": details
        }
    
    def _create_status_response(self, status_data: Dict[str, Any]):
        """Create comprehensive status response"""
        return {
            "status": status_data,
            "timestamp": time.time()
        }


class TenantNodeServer:
    """gRPC server for tenant node"""
    
    def __init__(self, tenant_config: TenantConfig, source_manager: SourceManager):
        self.tenant_config = tenant_config
        self.source_manager = source_manager
        self.server = None
        
    async def start(self):
        """Start the gRPC server"""
        self.server = aio.server()
        
        # Add servicer
        servicer = TenantNodeServicer(self.tenant_config, self.source_manager)
        # storage_pb2_grpc.add_TenantNodeServicer_to_server(servicer, self.server)
        
        # Add port
        listen_addr = f"[::]:{self.tenant_config.grpc_port}"
        self.server.add_insecure_port(listen_addr)
        
        # Start server
        await self.server.start()
        
        logger.info(f"Tenant Node gRPC server started on {listen_addr}")
        
        # Wait for termination
        await self.server.wait_for_termination()
    
    async def stop(self):
        """Stop the gRPC server"""
        if self.server:
            await self.server.stop(grace=10)
            logger.info("Tenant Node gRPC server stopped")
