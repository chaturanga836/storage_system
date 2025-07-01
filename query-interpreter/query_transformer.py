"""
Query Transformer - Converts between different query representations
"""
import logging
from typing import Dict, Any, List, Optional
from dataclasses import dataclass, asdict
import json

from .sql_parser import ParsedQuery, InternalQuery, QueryType, SQLDialect

logger = logging.getLogger(__name__)


@dataclass
class QueryPlan:
    """Complete query execution plan"""
    query_id: str
    internal_query: InternalQuery
    optimization_plan: Optional[Dict[str, Any]] = None
    estimated_cost: Optional[Dict[str, Any]] = None
    execution_strategy: str = "default"
    parallelism: int = 1
    cache_enabled: bool = True
    timeout_ms: int = 30000


class QueryTransformer:
    """Transforms queries between different representations and optimizes them"""
    
    def __init__(self):
        self.transformation_rules = self._load_transformation_rules()
        self.optimization_patterns = self._load_optimization_patterns()
        
        logger.info("Query Transformer initialized")
    
    def transform_to_execution_plan(self, internal_query: InternalQuery, 
                                  query_id: Optional[str] = None) -> QueryPlan:
        """Transform internal query to complete execution plan"""
        
        query_id = query_id or f"query_{hash(str(internal_query))}"
        
        logger.info(f"Transforming query {query_id} to execution plan")
        
        # Apply transformation rules
        optimized_query = self._apply_transformation_rules(internal_query)
        
        # Determine execution strategy
        execution_strategy = self._determine_execution_strategy(optimized_query)
        
        # Calculate parallelism
        parallelism = self._calculate_optimal_parallelism(optimized_query)
        
        # Generate execution plan
        plan = QueryPlan(
            query_id=query_id,
            internal_query=optimized_query,
            execution_strategy=execution_strategy,
            parallelism=parallelism,
            cache_enabled=self._should_enable_cache(optimized_query),
            timeout_ms=self._calculate_timeout(optimized_query)
        )
        
        logger.info(f"Generated execution plan for {query_id}: {execution_strategy} strategy, "
                   f"parallelism={parallelism}")
        
        return plan
    
    def optimize_query(self, internal_query: InternalQuery) -> InternalQuery:
        """Apply query optimizations"""
        
        logger.info("Applying query optimizations")
        
        # Make a copy to avoid modifying original
        optimized = InternalQuery(
            source_ids=internal_query.source_ids.copy(),
            filters=internal_query.filters.copy(),
            projections=internal_query.projections.copy(),
            aggregations=internal_query.aggregations.copy(),
            joins=internal_query.joins.copy(),
            order_by=internal_query.order_by.copy(),
            group_by=internal_query.group_by.copy(),
            limit=internal_query.limit,
            offset=internal_query.offset,
            query_type=internal_query.query_type,
            optimization_hints=internal_query.optimization_hints.copy()
        )
        
        # Apply optimization patterns
        optimized = self._optimize_filters(optimized)
        optimized = self._optimize_projections(optimized)
        optimized = self._optimize_joins(optimized)
        optimized = self._optimize_aggregations(optimized)
        
        return optimized
    
    def _apply_transformation_rules(self, query: InternalQuery) -> InternalQuery:
        """Apply transformation rules to optimize query"""
        
        for rule_name, rule_func in self.transformation_rules.items():
            try:
                query = rule_func(query)
                logger.debug(f"Applied transformation rule: {rule_name}")
            except Exception as e:
                logger.warning(f"Failed to apply rule {rule_name}: {e}")
        
        return query
    
    def _optimize_filters(self, query: InternalQuery) -> InternalQuery:
        """Optimize filter conditions"""
        
        # Push filters down (apply as early as possible)
        # Combine similar filters
        # Remove redundant filters
        
        optimized_filters = {}
        
        for column, condition in query.filters.items():
            # Normalize filter conditions
            if isinstance(condition, dict):
                # Already in proper format
                optimized_filters[column] = condition
            else:
                # Convert simple equality to proper format
                optimized_filters[column] = {"eq": condition}
        
        query.filters = optimized_filters
        
        # Add optimization hint for filter pushdown
        if optimized_filters:
            query.optimization_hints["filter_pushdown"] = True
        
        return query
    
    def _optimize_projections(self, query: InternalQuery) -> InternalQuery:
        """Optimize column projections"""
        
        # Remove duplicate columns
        if query.projections:
            unique_projections = []
            seen = set()
            
            for col in query.projections:
                if col not in seen:
                    unique_projections.append(col)
                    seen.add(col)
            
            query.projections = unique_projections
        
        # Add optimization hint for column pruning
        if query.projections and "*" not in query.projections:
            query.optimization_hints["column_pruning"] = True
        
        return query
    
    def _optimize_joins(self, query: InternalQuery) -> InternalQuery:
        """Optimize JOIN operations"""
        
        if not query.joins:
            return query
        
        # Reorder joins for optimal execution
        # Convert to hash joins where appropriate
        # Detect star schema patterns
        
        optimized_joins = []
        
        for join in query.joins:
            # Add join optimization hints
            optimized_join = join.copy()
            
            # Suggest hash join for large tables
            if "condition" in optimized_join:
                optimized_join["optimization_hint"] = "hash_join_candidate"
            
            optimized_joins.append(optimized_join)
        
        query.joins = optimized_joins
        
        if len(optimized_joins) > 1:
            query.optimization_hints["join_reordering"] = True
        
        return query
    
    def _optimize_aggregations(self, query: InternalQuery) -> InternalQuery:
        """Optimize aggregation operations"""
        
        if not query.aggregations:
            return query
        
        # Combine compatible aggregations
        # Use pre-aggregation where possible
        # Optimize GROUP BY order
        
        # Sort GROUP BY columns for consistent execution
        if query.group_by:
            query.group_by = sorted(query.group_by)
            query.optimization_hints["group_by_optimization"] = True
        
        # Add aggregation optimization hints
        if len(query.aggregations) > 1:
            query.optimization_hints["multi_aggregation"] = True
        
        return query
    
    def _determine_execution_strategy(self, query: InternalQuery) -> str:
        """Determine optimal execution strategy"""
        
        # Analyze query characteristics
        num_sources = len(query.source_ids)
        has_joins = len(query.joins) > 0
        has_aggregations = len(query.aggregations) > 0
        has_complex_filters = any(
            isinstance(cond, dict) and len(cond) > 1 
            for cond in query.filters.values()
        )
        
        # Choose strategy based on characteristics
        if num_sources == 1 and not has_joins:
            if has_aggregations:
                return "single_source_aggregation"
            elif query.limit and query.limit < 1000:
                return "single_source_limit"
            else:
                return "single_source_scan"
        
        elif num_sources > 1 and has_joins:
            if has_complex_filters:
                return "multi_source_filtered_join"
            else:
                return "multi_source_join"
        
        elif num_sources > 1 and not has_joins:
            return "multi_source_union"
        
        else:
            return "default"
    
    def _calculate_optimal_parallelism(self, query: InternalQuery) -> int:
        """Calculate optimal parallelism level"""
        
        # Base parallelism on query characteristics
        base_parallelism = 1
        
        # More sources = more parallelism
        base_parallelism = min(len(query.source_ids), 8)
        
        # Reduce parallelism for small result sets
        if query.limit and query.limit < 100:
            base_parallelism = min(base_parallelism, 2)
        
        # Increase parallelism for aggregations
        if query.aggregations:
            base_parallelism = min(base_parallelism * 2, 16)
        
        # Reduce parallelism for complex joins
        if len(query.joins) > 2:
            base_parallelism = max(base_parallelism // 2, 1)
        
        return base_parallelism
    
    def _should_enable_cache(self, query: InternalQuery) -> bool:
        """Determine if caching should be enabled"""
        
        # Enable cache for:
        # - Aggregation queries
        # - Queries without filters (full table scans)
        # - Queries with simple filters
        
        if query.aggregations:
            return True
        
        if not query.filters:
            return True
        
        # Disable cache for time-sensitive queries
        time_columns = ["timestamp", "created_at", "updated_at", "date"]
        if any(col in time_columns for col in query.filters.keys()):
            return False
        
        return True
    
    def _calculate_timeout(self, query: InternalQuery) -> int:
        """Calculate appropriate timeout for query"""
        
        base_timeout = 30000  # 30 seconds
        
        # Increase timeout for complex queries
        complexity_score = (
            len(query.source_ids) * 1000 +
            len(query.joins) * 5000 +
            len(query.aggregations) * 2000 +
            len(query.filters) * 500
        )
        
        timeout = base_timeout + complexity_score
        
        # Cap at reasonable maximum
        return min(timeout, 300000)  # 5 minutes max
    
    def _load_transformation_rules(self) -> Dict[str, callable]:
        """Load query transformation rules"""
        
        return {
            "filter_normalization": self._normalize_filters,
            "projection_optimization": self._optimize_projection_order,
            "predicate_pushdown": self._push_down_predicates,
        }
    
    def _load_optimization_patterns(self) -> Dict[str, Any]:
        """Load optimization patterns"""
        
        return {
            "star_schema": {
                "pattern": "one_large_table_many_small",
                "optimization": "dimension_table_broadcast"
            },
            "time_series": {
                "pattern": "time_column_filter",
                "optimization": "partition_pruning"
            },
            "aggregation_heavy": {
                "pattern": "multiple_aggregations",
                "optimization": "pre_aggregation"
            }
        }
    
    def _normalize_filters(self, query: InternalQuery) -> InternalQuery:
        """Normalize filter conditions to standard format"""
        
        normalized_filters = {}
        
        for column, condition in query.filters.items():
            if isinstance(condition, dict):
                normalized_filters[column] = condition
            else:
                # Convert simple values to equality conditions
                normalized_filters[column] = {"eq": condition}
        
        query.filters = normalized_filters
        return query
    
    def _optimize_projection_order(self, query: InternalQuery) -> InternalQuery:
        """Optimize the order of projected columns"""
        
        if not query.projections or "*" in query.projections:
            return query
        
        # Put filtered columns first (they might be indexed)
        filter_columns = list(query.filters.keys())
        other_columns = [col for col in query.projections if col not in filter_columns]
        
        optimized_projections = filter_columns + other_columns
        query.projections = optimized_projections
        
        return query
    
    def _push_down_predicates(self, query: InternalQuery) -> InternalQuery:
        """Push filter predicates as close to data source as possible"""
        
        # This would be more complex in a real implementation
        # For now, just add optimization hint
        if query.filters:
            query.optimization_hints["predicate_pushdown"] = list(query.filters.keys())
        
        return query
    
    def to_dict(self, query_plan: QueryPlan) -> Dict[str, Any]:
        """Convert query plan to dictionary format"""
        return asdict(query_plan)
    
    def from_dict(self, data: Dict[str, Any]) -> QueryPlan:
        """Create query plan from dictionary"""
        
        internal_query_data = data["internal_query"]
        internal_query = InternalQuery(**internal_query_data)
        
        return QueryPlan(
            query_id=data["query_id"],
            internal_query=internal_query,
            optimization_plan=data.get("optimization_plan"),
            estimated_cost=data.get("estimated_cost"),
            execution_strategy=data.get("execution_strategy", "default"),
            parallelism=data.get("parallelism", 1),
            cache_enabled=data.get("cache_enabled", True),
            timeout_ms=data.get("timeout_ms", 30000)
        )
