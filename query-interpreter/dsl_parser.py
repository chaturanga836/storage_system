"""
Custom Domain-Specific Language (DSL) Parser for storage queries
"""
import logging
import json
import re
from typing import Dict, Any, List, Optional, Union
from dataclasses import dataclass
from enum import Enum

from .sql_parser import InternalQuery

logger = logging.getLogger(__name__)


class DSLQueryType(Enum):
    """Types of DSL queries"""
    SEARCH = "search"
    AGGREGATE = "aggregate"
    INSERT = "insert"
    UPDATE = "update"
    DELETE = "delete"


@dataclass
class DSLQuery:
    """Parsed DSL query representation"""
    query_type: DSLQueryType
    original_query: str
    source: Optional[str]
    filters: Dict[str, Any]
    projections: List[str]
    aggregations: List[Dict[str, Any]]
    sort: List[Dict[str, str]]
    limit: Optional[int]
    offset: Optional[int]
    data: Optional[Dict[str, Any]]  # For INSERT/UPDATE
    errors: List[str]


class DSLParser:
    """Parser for custom query DSL syntax"""
    
    def __init__(self):
        self.operators = {
            "eq": "=",
            "ne": "!=", 
            "gt": ">",
            "gte": ">=",
            "lt": "<",
            "lte": "<=",
            "in": "IN",
            "nin": "NOT IN",
            "like": "LIKE",
            "regex": "REGEXP",
            "exists": "EXISTS",
            "null": "IS NULL",
            "notnull": "IS NOT NULL"
        }
        
        self.aggregation_functions = {
            "count", "sum", "avg", "min", "max", "first", "last"
        }
        
        logger.info("DSL Parser initialized")
    
    def parse_dsl(self, query_str: str) -> DSLQuery:
        """Parse DSL query string"""
        
        try:
            logger.info("Parsing DSL query")
            
            # Try to parse as JSON first
            if query_str.strip().startswith('{'):
                return self._parse_json_dsl(query_str)
            else:
                return self._parse_text_dsl(query_str)
                
        except Exception as e:
            logger.error(f"DSL parsing failed: {e}")
            return DSLQuery(
                query_type=DSLQueryType.SEARCH,
                original_query=query_str,
                source=None,
                filters={},
                projections=[],
                aggregations=[],
                sort=[],
                limit=None,
                offset=None,
                data=None,
                errors=[f"Parse error: {str(e)}"]
            )
    
    def _parse_json_dsl(self, query_str: str) -> DSLQuery:
        """Parse JSON-formatted DSL query"""
        
        try:
            query_obj = json.loads(query_str)
            
            # Determine query type
            if "aggregate" in query_obj:
                query_type = DSLQueryType.AGGREGATE
            elif "insert" in query_obj:
                query_type = DSLQueryType.INSERT
            elif "update" in query_obj:
                query_type = DSLQueryType.UPDATE
            elif "delete" in query_obj:
                query_type = DSLQueryType.DELETE
            else:
                query_type = DSLQueryType.SEARCH
            
            # Extract components
            source = query_obj.get("from") or query_obj.get("source")
            filters = self._parse_filters(query_obj.get("where", {}))
            projections = query_obj.get("select", [])
            aggregations = self._parse_aggregations(query_obj.get("aggregate", []))
            sort = self._parse_sort(query_obj.get("sort", []))
            limit = query_obj.get("limit")
            offset = query_obj.get("offset")
            data = query_obj.get("data") or query_obj.get("set")
            
            return DSLQuery(
                query_type=query_type,
                original_query=query_str,
                source=source,
                filters=filters,
                projections=projections,
                aggregations=aggregations,
                sort=sort,
                limit=limit,
                offset=offset,
                data=data,
                errors=[]
            )
            
        except json.JSONDecodeError as e:
            return DSLQuery(
                query_type=DSLQueryType.SEARCH,
                original_query=query_str,
                source=None,
                filters={},
                projections=[],
                aggregations=[],
                sort=[],
                limit=None,
                offset=None,
                data=None,
                errors=[f"Invalid JSON: {str(e)}"]
            )
    
    def _parse_text_dsl(self, query_str: str) -> DSLQuery:
        """Parse text-formatted DSL query"""
        
        # Simple text DSL format:
        # SEARCH FROM source WHERE conditions SELECT columns LIMIT n
        # AGGREGATE FROM source WHERE conditions GROUP BY columns AGGREGATE functions
        
        errors = []
        
        # Normalize query
        query = query_str.strip().upper()
        original_parts = query_str.strip().split()
        
        # Determine query type
        if query.startswith("SEARCH"):
            query_type = DSLQueryType.SEARCH
        elif query.startswith("AGGREGATE"):
            query_type = DSLQueryType.AGGREGATE
        elif query.startswith("INSERT"):
            query_type = DSLQueryType.INSERT
        elif query.startswith("UPDATE"):
            query_type = DSLQueryType.UPDATE
        elif query.startswith("DELETE"):
            query_type = DSLQueryType.DELETE
        else:
            query_type = DSLQueryType.SEARCH
            errors.append("Unknown query type, assuming SEARCH")
        
        # Parse components using regex
        source = self._extract_from_clause(query_str)
        filters = self._extract_where_clause(query_str)
        projections = self._extract_select_clause(query_str)
        aggregations = self._extract_aggregate_clause(query_str)
        sort = self._extract_sort_clause(query_str)
        limit = self._extract_limit_clause(query_str)
        offset = self._extract_offset_clause(query_str)
        
        return DSLQuery(
            query_type=query_type,
            original_query=query_str,
            source=source,
            filters=filters,
            projections=projections,
            aggregations=aggregations,
            sort=sort,
            limit=limit,
            offset=offset,
            data=None,
            errors=errors
        )
    
    def _parse_filters(self, where_obj: Dict[str, Any]) -> Dict[str, Any]:
        """Parse filter conditions from object"""
        filters = {}
        
        for field, condition in where_obj.items():
            if isinstance(condition, dict):
                # Complex condition: {"age": {"gt": 25, "lt": 65}}
                filters[field] = condition
            else:
                # Simple equality: {"name": "John"}
                filters[field] = {"eq": condition}
        
        return filters
    
    def _parse_aggregations(self, agg_list: List[Any]) -> List[Dict[str, Any]]:
        """Parse aggregation functions"""
        aggregations = []
        
        for agg in agg_list:
            if isinstance(agg, dict):
                aggregations.append(agg)
            elif isinstance(agg, str):
                # Parse string like "COUNT(*)" or "SUM(amount)"
                parsed = self._parse_agg_string(agg)
                if parsed:
                    aggregations.append(parsed)
        
        return aggregations
    
    def _parse_agg_string(self, agg_str: str) -> Optional[Dict[str, Any]]:
        """Parse aggregation string like 'COUNT(*)' or 'SUM(amount)'"""
        
        # Match pattern: FUNCTION(column)
        pattern = r'(\w+)\s*\(\s*([^)]+)\s*\)'
        match = re.match(pattern, agg_str.strip(), re.IGNORECASE)
        
        if match:
            func_name = match.group(1).lower()
            column = match.group(2).strip()
            
            if func_name in self.aggregation_functions:
                return {
                    "type": func_name,
                    "column": column,
                    "alias": f"{func_name}_{column}"
                }
        
        return None
    
    def _parse_sort(self, sort_list: List[Any]) -> List[Dict[str, str]]:
        """Parse sort conditions"""
        sort_conditions = []
        
        for sort_item in sort_list:
            if isinstance(sort_item, dict):
                sort_conditions.append(sort_item)
            elif isinstance(sort_item, str):
                # Parse "column ASC" or "column DESC"
                parts = sort_item.split()
                column = parts[0]
                direction = parts[1] if len(parts) > 1 else "ASC"
                sort_conditions.append({"column": column, "direction": direction.upper()})
        
        return sort_conditions
    
    def _extract_from_clause(self, query: str) -> Optional[str]:
        """Extract source from FROM clause"""
        pattern = r'\bFROM\s+(\w+)'
        match = re.search(pattern, query, re.IGNORECASE)
        return match.group(1) if match else None
    
    def _extract_where_clause(self, query: str) -> Dict[str, Any]:
        """Extract filters from WHERE clause"""
        filters = {}
        
        # Look for WHERE clause
        where_pattern = r'\bWHERE\s+(.+?)(?:\s+(?:SELECT|ORDER|LIMIT|$))'
        where_match = re.search(where_pattern, query, re.IGNORECASE)
        
        if where_match:
            where_clause = where_match.group(1)
            
            # Parse simple conditions: field = value, field > value, etc.
            condition_patterns = [
                (r'(\w+)\s*=\s*["\']([^"\']+)["\']', "eq"),
                (r'(\w+)\s*=\s*(\d+)', "eq"),
                (r'(\w+)\s*>\s*(\d+)', "gt"),
                (r'(\w+)\s*<\s*(\d+)', "lt"),
                (r'(\w+)\s*>=\s*(\d+)', "gte"),
                (r'(\w+)\s*<=\s*(\d+)', "lte"),
            ]
            
            for pattern, operator in condition_patterns:
                matches = re.findall(pattern, where_clause, re.IGNORECASE)
                for field, value in matches:
                    # Convert numeric values
                    if value.isdigit():
                        value = int(value)
                    elif value.replace('.', '').isdigit():
                        value = float(value)
                    
                    filters[field] = {operator: value}
        
        return filters
    
    def _extract_select_clause(self, query: str) -> List[str]:
        """Extract projections from SELECT clause"""
        pattern = r'\bSELECT\s+(.+?)(?:\s+(?:FROM|WHERE|ORDER|LIMIT|$))'
        match = re.search(pattern, query, re.IGNORECASE)
        
        if match:
            select_clause = match.group(1)
            # Split by comma and clean up
            columns = [col.strip() for col in select_clause.split(',')]
            return [col for col in columns if col]
        
        return []
    
    def _extract_aggregate_clause(self, query: str) -> List[Dict[str, Any]]:
        """Extract aggregations from AGGREGATE clause"""
        pattern = r'\bAGGREGATE\s+(.+?)(?:\s+(?:FROM|WHERE|ORDER|LIMIT|$))'
        match = re.search(pattern, query, re.IGNORECASE)
        
        if match:
            agg_clause = match.group(1)
            agg_funcs = [func.strip() for func in agg_clause.split(',')]
            
            aggregations = []
            for func in agg_funcs:
                parsed = self._parse_agg_string(func)
                if parsed:
                    aggregations.append(parsed)
            
            return aggregations
        
        return []
    
    def _extract_sort_clause(self, query: str) -> List[Dict[str, str]]:
        """Extract sort from ORDER BY clause"""
        pattern = r'\bORDER\s+BY\s+(.+?)(?:\s+(?:LIMIT|$))'
        match = re.search(pattern, query, re.IGNORECASE)
        
        if match:
            order_clause = match.group(1)
            sort_items = [item.strip() for item in order_clause.split(',')]
            
            sort_conditions = []
            for item in sort_items:
                parts = item.split()
                column = parts[0]
                direction = parts[1] if len(parts) > 1 else "ASC"
                sort_conditions.append({"column": column, "direction": direction.upper()})
            
            return sort_conditions
        
        return []
    
    def _extract_limit_clause(self, query: str) -> Optional[int]:
        """Extract LIMIT value"""
        pattern = r'\bLIMIT\s+(\d+)'
        match = re.search(pattern, query, re.IGNORECASE)
        return int(match.group(1)) if match else None
    
    def _extract_offset_clause(self, query: str) -> Optional[int]:
        """Extract OFFSET value"""
        pattern = r'\bOFFSET\s+(\d+)'
        match = re.search(pattern, query, re.IGNORECASE)
        return int(match.group(1)) if match else None
    
    def to_internal_query(self, dsl_query: DSLQuery) -> InternalQuery:
        """Convert DSL query to internal representation"""
        
        # Map DSL to internal format
        source_ids = [dsl_query.source] if dsl_query.source else []
        
        # Convert sort to internal format
        order_by = []
        for sort_item in dsl_query.sort:
            order_by.append({
                "column": sort_item["column"],
                "direction": sort_item["direction"]
            })
        
        return InternalQuery(
            source_ids=source_ids,
            filters=dsl_query.filters,
            projections=dsl_query.projections or ["*"],
            aggregations=dsl_query.aggregations,
            joins=[],  # DSL doesn't support joins yet
            order_by=order_by,
            group_by=[],  # Would need to extract from aggregations
            limit=dsl_query.limit,
            offset=dsl_query.offset,
            query_type=dsl_query.query_type.value,
            optimization_hints=self._generate_dsl_hints(dsl_query)
        )
    
    def _generate_dsl_hints(self, dsl_query: DSLQuery) -> Dict[str, Any]:
        """Generate optimization hints for DSL query"""
        hints = {}
        
        if dsl_query.filters:
            hints["has_filters"] = True
            hints["filter_columns"] = list(dsl_query.filters.keys())
        
        if dsl_query.aggregations:
            hints["has_aggregations"] = True
            hints["aggregation_types"] = [agg["type"] for agg in dsl_query.aggregations]
        
        if dsl_query.limit and dsl_query.limit < 100:
            hints["small_result_set"] = True
        
        return hints
    
    def format_dsl(self, dsl_query: DSLQuery) -> str:
        """Format DSL query for display"""
        
        parts = [dsl_query.query_type.value.upper()]
        
        if dsl_query.projections:
            parts.append(f"SELECT {', '.join(dsl_query.projections)}")
        
        if dsl_query.source:
            parts.append(f"FROM {dsl_query.source}")
        
        if dsl_query.filters:
            filter_strs = []
            for field, condition in dsl_query.filters.items():
                if isinstance(condition, dict):
                    for op, value in condition.items():
                        filter_strs.append(f"{field} {self.operators.get(op, op)} {value}")
                else:
                    filter_strs.append(f"{field} = {condition}")
            
            if filter_strs:
                parts.append(f"WHERE {' AND '.join(filter_strs)}")
        
        if dsl_query.sort:
            sort_strs = [f"{s['column']} {s['direction']}" for s in dsl_query.sort]
            parts.append(f"ORDER BY {', '.join(sort_strs)}")
        
        if dsl_query.limit:
            parts.append(f"LIMIT {dsl_query.limit}")
        
        if dsl_query.offset:
            parts.append(f"OFFSET {dsl_query.offset}")
        
        return " ".join(parts)
