"""
Cost-Based Optimizer (CBO) for query execution planning
"""
import asyncio
import logging
import time
from typing import Dict, Any, List, Optional, Tuple
from dataclasses import dataclass
from enum import Enum
from pathlib import Path
import json
import pandas as pd

logger = logging.getLogger(__name__)


class OperatorType(Enum):
    """Query operator types"""
    SCAN = "scan"
    FILTER = "filter"
    PROJECT = "project"
    AGGREGATE = "aggregate"
    JOIN = "join"
    SORT = "sort"
    LIMIT = "limit"


@dataclass
class QueryStats:
    """Statistics for query execution"""
    row_count: int
    file_count: int
    size_bytes: int
    selectivity: float = 1.0
    cardinality: int = 0
    distinct_values: int = 0


@dataclass
class CostEstimate:
    """Cost estimation for query operations"""
    cpu_cost: float
    io_cost: float
    memory_cost: float
    network_cost: float
    total_cost: float
    
    def __post_init__(self):
        if self.total_cost == 0:
            self.total_cost = self.cpu_cost + self.io_cost + self.memory_cost + self.network_cost


@dataclass
class ExecutionPlan:
    """Query execution plan"""
    plan_id: str
    operators: List[Dict[str, Any]]
    estimated_cost: CostEstimate
    estimated_time_ms: float
    parallelism_factor: int
    index_usage: List[str]
    file_pruning: Dict[str, Any]


class QueryOptimizer:
    """Cost-based query optimizer for efficient execution planning"""
    
    def __init__(self, tenant_config):
        self.tenant_config = tenant_config
        
        # Statistics cache
        self.table_stats = {}
        self.column_stats = {}
        self.file_stats = {}
        
        # Cost model parameters
        self.cost_factors = {
            "cpu_per_row": 0.001,  # CPU cost per row processed
            "io_per_mb": 0.1,      # I/O cost per MB read
            "memory_per_mb": 0.01, # Memory cost per MB used
            "network_per_mb": 0.05, # Network cost per MB transferred
            "index_benefit": 0.8,   # Cost reduction from using index
            "partition_pruning": 0.7 # Cost reduction from partition pruning
        }
        
        # Query history for learning
        self.query_history = []
        self.execution_stats = {}
        
        logger.info("Cost-based optimizer initialized")
    
    async def optimize_query(self, query_filter: Dict[str, Any], 
                           aggregations: List[Dict[str, Any]],
                           source_ids: List[str],
                           limit: Optional[int] = None) -> ExecutionPlan:
        """Generate optimized execution plan for a query"""
        
        try:
            logger.info("Optimizing query execution plan")
            
            # Collect statistics
            await self._collect_statistics(source_ids)
            
            # Generate multiple execution plans
            plans = await self._generate_execution_plans(
                query_filter, aggregations, source_ids, limit
            )
            
            # Select the best plan
            best_plan = self._select_best_plan(plans)
            
            logger.info(f"Selected plan {best_plan.plan_id} with estimated cost {best_plan.estimated_cost.total_cost:.2f}")
            
            return best_plan
            
        except Exception as e:
            logger.error(f"Error optimizing query: {e}")
            # Return default plan as fallback
            return await self._create_default_plan(query_filter, aggregations, source_ids, limit)
    
    async def _collect_statistics(self, source_ids: List[str]):
        """Collect and update statistics for cost estimation"""
        
        for source_id in source_ids:
            if source_id not in self.table_stats:
                await self._collect_table_statistics(source_id)
    
    async def _collect_table_statistics(self, source_id: str):
        """Collect statistics for a specific source/table"""
        try:
            source_path = Path(self.tenant_config.base_data_path) / "sources" / source_id
            parquet_path = source_path / "parquet"
            
            if not parquet_path.exists():
                return
            
            # Scan parquet files to collect statistics
            total_rows = 0
            total_size = 0
            file_count = 0
            column_stats = {}
            
            for parquet_file in parquet_path.rglob("*.parquet"):
                try:
                    # Read parquet metadata
                    import pyarrow.parquet as pq
                    parquet_metadata = pq.read_metadata(str(parquet_file))
                    
                    file_rows = parquet_metadata.num_rows
                    file_size = parquet_file.stat().st_size
                    
                    total_rows += file_rows
                    total_size += file_size
                    file_count += 1
                    
                    # Sample file for column statistics
                    if file_count <= 3:  # Sample first 3 files
                        df_sample = pd.read_parquet(str(parquet_file))
                        await self._update_column_statistics(source_id, df_sample, column_stats)
                    
                except Exception as e:
                    logger.warning(f"Error reading parquet metadata from {parquet_file}: {e}")
                    continue
            
            # Store table statistics
            self.table_stats[source_id] = QueryStats(
                row_count=total_rows,
                file_count=file_count,
                size_bytes=total_size
            )
            
            self.column_stats[source_id] = column_stats
            
            logger.debug(f"Collected statistics for {source_id}: {total_rows} rows, {file_count} files, {total_size/1024/1024:.1f} MB")
            
        except Exception as e:
            logger.error(f"Error collecting statistics for {source_id}: {e}")
    
    async def _update_column_statistics(self, source_id: str, df_sample: pd.DataFrame, column_stats: Dict):
        """Update column-level statistics from a sample"""
        
        for column in df_sample.columns:
            if column not in column_stats:
                column_stats[column] = {
                    "distinct_values": set(),
                    "null_count": 0,
                    "min_value": None,
                    "max_value": None,
                    "data_type": str(df_sample[column].dtype)
                }
            
            col_stats = column_stats[column]
            col_data = df_sample[column].dropna()
            
            # Update distinct values (keep only a sample to avoid memory issues)
            if len(col_stats["distinct_values"]) < 1000:
                col_stats["distinct_values"].update(col_data.unique()[:100])
            
            # Update null count
            col_stats["null_count"] += df_sample[column].isnull().sum()
            
            # Update min/max
            if not col_data.empty:
                if pd.api.types.is_numeric_dtype(col_data):
                    col_min, col_max = col_data.min(), col_data.max()
                    col_stats["min_value"] = min(col_stats["min_value"] or col_min, col_min)
                    col_stats["max_value"] = max(col_stats["max_value"] or col_max, col_max)
    
    async def _generate_execution_plans(self, query_filter: Dict[str, Any],
                                      aggregations: List[Dict[str, Any]],
                                      source_ids: List[str],
                                      limit: Optional[int]) -> List[ExecutionPlan]:
        """Generate multiple execution plan alternatives"""
        
        plans = []
        
        # Plan 1: Sequential scan with filters
        plan1 = await self._create_sequential_plan(query_filter, aggregations, source_ids, limit)
        plans.append(plan1)
        
        # Plan 2: Index-based plan (if applicable)
        plan2 = await self._create_index_plan(query_filter, aggregations, source_ids, limit)
        if plan2:
            plans.append(plan2)
        
        # Plan 3: Parallel scan plan
        plan3 = await self._create_parallel_plan(query_filter, aggregations, source_ids, limit)
        plans.append(plan3)
        
        # Plan 4: Partition-pruned plan
        plan4 = await self._create_partition_pruned_plan(query_filter, aggregations, source_ids, limit)
        if plan4:
            plans.append(plan4)
        
        return plans
    
    async def _create_sequential_plan(self, query_filter: Dict[str, Any],
                                    aggregations: List[Dict[str, Any]],
                                    source_ids: List[str],
                                    limit: Optional[int]) -> ExecutionPlan:
        """Create a sequential execution plan"""
        
        operators = []
        total_cost = CostEstimate(0, 0, 0, 0, 0)
        
        for source_id in source_ids:
            stats = self.table_stats.get(source_id, QueryStats(0, 0, 0))
            
            # Scan operator
            scan_cost = self._estimate_scan_cost(stats)
            operators.append({
                "type": OperatorType.SCAN.value,
                "source_id": source_id,
                "cost": scan_cost,
                "rows_out": stats.row_count
            })
            total_cost.io_cost += scan_cost.io_cost
            total_cost.cpu_cost += scan_cost.cpu_cost
            
            # Filter operator
            if query_filter:
                filter_selectivity = self._estimate_filter_selectivity(query_filter, source_id)
                filter_cost = self._estimate_filter_cost(stats, filter_selectivity)
                operators.append({
                    "type": OperatorType.FILTER.value,
                    "filter": query_filter,
                    "selectivity": filter_selectivity,
                    "cost": filter_cost,
                    "rows_out": int(stats.row_count * filter_selectivity)
                })
                total_cost.cpu_cost += filter_cost.cpu_cost
        
        # Aggregate operator
        if aggregations:
            agg_cost = self._estimate_aggregation_cost(operators, aggregations)
            operators.append({
                "type": OperatorType.AGGREGATE.value,
                "aggregations": aggregations,
                "cost": agg_cost,
                "rows_out": 1  # Aggregation typically produces few rows
            })
            total_cost.cpu_cost += agg_cost.cpu_cost
            total_cost.memory_cost += agg_cost.memory_cost
        
        # Limit operator
        if limit:
            operators.append({
                "type": OperatorType.LIMIT.value,
                "limit": limit,
                "cost": CostEstimate(0, 0, 0, 0, 0),
                "rows_out": min(limit, operators[-1]["rows_out"])
            })
        
        total_cost.total_cost = (total_cost.cpu_cost + total_cost.io_cost + 
                               total_cost.memory_cost + total_cost.network_cost)
        
        return ExecutionPlan(
            plan_id="sequential_plan",
            operators=operators,
            estimated_cost=total_cost,
            estimated_time_ms=total_cost.total_cost * 100,  # Rough time estimate
            parallelism_factor=1,
            index_usage=[],
            file_pruning={}
        )
    
    async def _create_parallel_plan(self, query_filter: Dict[str, Any],
                                  aggregations: List[Dict[str, Any]],
                                  source_ids: List[str],
                                  limit: Optional[int]) -> ExecutionPlan:
        """Create a parallel execution plan"""
        
        # Start with sequential plan
        sequential_plan = await self._create_sequential_plan(query_filter, aggregations, source_ids, limit)
        
        # Calculate optimal parallelism
        parallelism = min(len(source_ids), self.tenant_config.max_concurrent_searches)
        
        # Adjust costs for parallelism
        parallel_cost = CostEstimate(
            cpu_cost=sequential_plan.estimated_cost.cpu_cost / parallelism,
            io_cost=sequential_plan.estimated_cost.io_cost / parallelism,
            memory_cost=sequential_plan.estimated_cost.memory_cost * parallelism,  # More memory needed
            network_cost=sequential_plan.estimated_cost.network_cost,
            total_cost=0
        )
        parallel_cost.total_cost = (parallel_cost.cpu_cost + parallel_cost.io_cost + 
                                  parallel_cost.memory_cost + parallel_cost.network_cost)
        
        return ExecutionPlan(
            plan_id="parallel_plan",
            operators=sequential_plan.operators,
            estimated_cost=parallel_cost,
            estimated_time_ms=parallel_cost.total_cost * 100 / parallelism,
            parallelism_factor=parallelism,
            index_usage=[],
            file_pruning={}
        )
    
    async def _create_index_plan(self, query_filter: Dict[str, Any],
                               aggregations: List[Dict[str, Any]],
                               source_ids: List[str],
                               limit: Optional[int]) -> Optional[ExecutionPlan]:
        """Create an index-based execution plan"""
        
        # Check if any filters can benefit from indices
        index_usage = []
        can_use_index = False
        
        for source_id in source_ids:
            for filter_column in query_filter.keys():
                # Check if we have an index for this column
                # This would integrate with the actual index manager
                if self._has_index(source_id, filter_column):
                    index_usage.append(f"{source_id}.{filter_column}")
                    can_use_index = True
        
        if not can_use_index:
            return None
        
        # Create index-optimized plan
        sequential_plan = await self._create_sequential_plan(query_filter, aggregations, source_ids, limit)
        
        # Apply index benefit
        index_cost = CostEstimate(
            cpu_cost=sequential_plan.estimated_cost.cpu_cost * self.cost_factors["index_benefit"],
            io_cost=sequential_plan.estimated_cost.io_cost * self.cost_factors["index_benefit"],
            memory_cost=sequential_plan.estimated_cost.memory_cost,
            network_cost=sequential_plan.estimated_cost.network_cost,
            total_cost=0
        )
        index_cost.total_cost = (index_cost.cpu_cost + index_cost.io_cost + 
                               index_cost.memory_cost + index_cost.network_cost)
        
        return ExecutionPlan(
            plan_id="index_plan",
            operators=sequential_plan.operators,
            estimated_cost=index_cost,
            estimated_time_ms=index_cost.total_cost * 100,
            parallelism_factor=1,
            index_usage=index_usage,
            file_pruning={}
        )
    
    async def _create_partition_pruned_plan(self, query_filter: Dict[str, Any],
                                          aggregations: List[Dict[str, Any]],
                                          source_ids: List[str],
                                          limit: Optional[int]) -> Optional[ExecutionPlan]:
        """Create a partition-pruned execution plan"""
        
        # Check if filters can benefit from partition pruning
        pruning_info = {}
        can_prune = False
        
        for source_id in source_ids:
            prunable_partitions = self._estimate_partition_pruning(source_id, query_filter)
            if prunable_partitions["pruned_ratio"] > 0:
                pruning_info[source_id] = prunable_partitions
                can_prune = True
        
        if not can_prune:
            return None
        
        # Create partition-pruned plan
        sequential_plan = await self._create_sequential_plan(query_filter, aggregations, source_ids, limit)
        
        # Calculate average pruning benefit
        avg_pruning = sum(info["pruned_ratio"] for info in pruning_info.values()) / len(pruning_info)
        pruning_benefit = 1.0 - (avg_pruning * self.cost_factors["partition_pruning"])
        
        pruned_cost = CostEstimate(
            cpu_cost=sequential_plan.estimated_cost.cpu_cost * pruning_benefit,
            io_cost=sequential_plan.estimated_cost.io_cost * pruning_benefit,
            memory_cost=sequential_plan.estimated_cost.memory_cost * pruning_benefit,
            network_cost=sequential_plan.estimated_cost.network_cost,
            total_cost=0
        )
        pruned_cost.total_cost = (pruned_cost.cpu_cost + pruned_cost.io_cost + 
                                pruned_cost.memory_cost + pruned_cost.network_cost)
        
        return ExecutionPlan(
            plan_id="partition_pruned_plan",
            operators=sequential_plan.operators,
            estimated_cost=pruned_cost,
            estimated_time_ms=pruned_cost.total_cost * 100,
            parallelism_factor=1,
            index_usage=[],
            file_pruning=pruning_info
        )
    
    def _estimate_scan_cost(self, stats: QueryStats) -> CostEstimate:
        """Estimate cost of scanning a table"""
        size_mb = stats.size_bytes / (1024 * 1024)
        
        return CostEstimate(
            cpu_cost=stats.row_count * self.cost_factors["cpu_per_row"],
            io_cost=size_mb * self.cost_factors["io_per_mb"],
            memory_cost=min(size_mb, 100) * self.cost_factors["memory_per_mb"],  # Cap memory usage
            network_cost=0,
            total_cost=0
        )
    
    def _estimate_filter_cost(self, stats: QueryStats, selectivity: float) -> CostEstimate:
        """Estimate cost of applying filters"""
        return CostEstimate(
            cpu_cost=stats.row_count * selectivity * self.cost_factors["cpu_per_row"] * 0.5,
            io_cost=0,
            memory_cost=0,
            network_cost=0,
            total_cost=0
        )
    
    def _estimate_aggregation_cost(self, previous_operators: List[Dict], 
                                 aggregations: List[Dict[str, Any]]) -> CostEstimate:
        """Estimate cost of aggregation operations"""
        # Get estimated input rows from previous operators
        input_rows = previous_operators[-1]["rows_out"] if previous_operators else 1000
        
        # Cost depends on aggregation type and number of groups
        agg_complexity = len(aggregations)
        
        return CostEstimate(
            cpu_cost=input_rows * agg_complexity * self.cost_factors["cpu_per_row"] * 2,
            io_cost=0,
            memory_cost=input_rows * 0.001,  # Memory for grouping
            network_cost=0,
            total_cost=0
        )
    
    def _estimate_filter_selectivity(self, query_filter: Dict[str, Any], source_id: str) -> float:
        """Estimate filter selectivity based on statistics"""
        selectivity = 1.0
        
        col_stats = self.column_stats.get(source_id, {})
        
        for column, condition in query_filter.items():
            if column not in col_stats:
                selectivity *= 0.5  # Default selectivity
                continue
            
            col_info = col_stats[column]
            distinct_count = len(col_info["distinct_values"])
            
            if isinstance(condition, dict):
                if "$eq" in condition:
                    selectivity *= 1.0 / max(distinct_count, 1)
                elif "$gt" in condition or "$lt" in condition:
                    selectivity *= 0.3  # Assume 30% for range queries
                elif "$in" in condition:
                    in_values = len(condition["$in"])
                    selectivity *= min(in_values / max(distinct_count, 1), 1.0)
                elif "$like" in condition:
                    selectivity *= 0.2  # Assume 20% for LIKE queries
            else:
                # Simple equality
                selectivity *= 1.0 / max(distinct_count, 1)
        
        return max(selectivity, 0.001)  # Minimum selectivity
    
    def _has_index(self, source_id: str, column: str) -> bool:
        """Check if an index exists for the given column"""
        # This would integrate with the actual index manager
        # For now, assume common columns have indices
        common_indexed_columns = ["id", "timestamp", "created_at", "updated_at", "user_id"]
        return column.lower() in common_indexed_columns
    
    def _estimate_partition_pruning(self, source_id: str, query_filter: Dict[str, Any]) -> Dict[str, Any]:
        """Estimate benefit of partition pruning"""
        # This would integrate with actual partition metadata
        # For now, provide estimates based on common partition columns
        
        partition_columns = ["date", "timestamp", "year", "month", "day"]
        pruned_ratio = 0.0
        
        for column, condition in query_filter.items():
            if column.lower() in partition_columns:
                if isinstance(condition, dict):
                    if "$eq" in condition:
                        pruned_ratio = 0.9  # Can prune 90% of partitions
                    elif "$gt" in condition or "$lt" in condition:
                        pruned_ratio = 0.5  # Can prune 50% of partitions
                else:
                    pruned_ratio = 0.9
                break
        
        return {
            "pruned_ratio": pruned_ratio,
            "estimated_partitions_scanned": int((1.0 - pruned_ratio) * 10),  # Assume 10 partitions
            "total_partitions": 10
        }
    
    def _select_best_plan(self, plans: List[ExecutionPlan]) -> ExecutionPlan:
        """Select the best execution plan based on cost"""
        if not plans:
            raise ValueError("No execution plans generated")
        
        # Sort by total cost
        plans.sort(key=lambda p: p.estimated_cost.total_cost)
        
        best_plan = plans[0]
        
        logger.info(f"Plan comparison:")
        for plan in plans:
            logger.info(f"  {plan.plan_id}: cost={plan.estimated_cost.total_cost:.2f}, "
                       f"time={plan.estimated_time_ms:.1f}ms, parallelism={plan.parallelism_factor}")
        
        return best_plan
    
    async def _create_default_plan(self, query_filter: Dict[str, Any],
                                 aggregations: List[Dict[str, Any]],
                                 source_ids: List[str],
                                 limit: Optional[int]) -> ExecutionPlan:
        """Create a simple default plan as fallback"""
        return ExecutionPlan(
            plan_id="default_plan",
            operators=[{
                "type": OperatorType.SCAN.value,
                "source_ids": source_ids,
                "cost": CostEstimate(10, 10, 5, 0, 25),
                "rows_out": 1000
            }],
            estimated_cost=CostEstimate(10, 10, 5, 0, 25),
            estimated_time_ms=1000,
            parallelism_factor=1,
            index_usage=[],
            file_pruning={}
        )
    
    async def record_execution_stats(self, plan_id: str, actual_time_ms: float, 
                                   actual_rows: int, success: bool):
        """Record actual execution statistics for learning"""
        if plan_id not in self.execution_stats:
            self.execution_stats[plan_id] = []
        
        self.execution_stats[plan_id].append({
            "timestamp": time.time(),
            "actual_time_ms": actual_time_ms,
            "actual_rows": actual_rows,
            "success": success
        })
        
        # Keep only recent statistics
        if len(self.execution_stats[plan_id]) > 100:
            self.execution_stats[plan_id] = self.execution_stats[plan_id][-100:]
        
        # Update cost factors based on actual performance
        await self._update_cost_model(plan_id, actual_time_ms)
    
    async def _update_cost_model(self, plan_id: str, actual_time_ms: float):
        """Update cost model based on actual execution results"""
        # Simple learning: adjust cost factors based on prediction accuracy
        # In a real implementation, this would use machine learning
        
        stats = self.execution_stats.get(plan_id, [])
        if len(stats) >= 5:  # Need enough data points
            recent_stats = stats[-5:]
            avg_actual_time = sum(s["actual_time_ms"] for s in recent_stats) / len(recent_stats)
            
            # This is a simplified adjustment
            if avg_actual_time > 0:
                adjustment_factor = actual_time_ms / avg_actual_time
                if 0.5 < adjustment_factor < 2.0:  # Reasonable range
                    # Slightly adjust cost factors
                    self.cost_factors["cpu_per_row"] *= (1.0 + (adjustment_factor - 1.0) * 0.1)
                    self.cost_factors["io_per_mb"] *= (1.0 + (adjustment_factor - 1.0) * 0.1)
    
    def get_optimizer_stats(self) -> Dict[str, Any]:
        """Get optimizer statistics and performance"""
        return {
            "tables_analyzed": len(self.table_stats),
            "cost_factors": self.cost_factors,
            "execution_history": {
                plan_id: len(stats) for plan_id, stats in self.execution_stats.items()
            },
            "recent_plans": len(self.query_history)
        }
