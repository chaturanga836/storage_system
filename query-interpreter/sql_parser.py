"""
SQL Parser using SQLGlot for multi-dialect SQL support
"""
import logging
from typing import Dict, Any, List, Optional, Union
from dataclasses import dataclass
from enum import Enum
import sqlglot
from sqlglot import parse_one, transpile
from sqlglot.expressions import Expression, Select, Insert, Update, Delete, Create
import json

logger = logging.getLogger(__name__)


class QueryType(Enum):
    """Types of SQL queries"""
    SELECT = "select"
    INSERT = "insert" 
    UPDATE = "update"
    DELETE = "delete"
    CREATE = "create"
    DROP = "drop"
    ALTER = "alter"
    UNKNOWN = "unknown"


class SQLDialect(Enum):
    """Supported SQL dialects"""
    POSTGRES = "postgres"
    MYSQL = "mysql"
    SQLITE = "sqlite"
    BIGQUERY = "bigquery"
    SNOWFLAKE = "snowflake"
    SPARK = "spark"
    CLICKHOUSE = "clickhouse"
    GENERIC = "generic"


@dataclass
class ParsedQuery:
    """Parsed SQL query representation"""
    query_type: QueryType
    original_sql: str
    dialect: SQLDialect
    ast: Expression
    tables: List[str]
    columns: List[str]
    filters: Dict[str, Any]
    aggregations: List[Dict[str, Any]]
    joins: List[Dict[str, Any]]
    order_by: List[str]
    group_by: List[str]
    limit: Optional[int]
    offset: Optional[int]
    errors: List[str]
    warnings: List[str]


@dataclass
class InternalQuery:
    """Internal query representation for execution engine"""
    source_ids: List[str]
    filters: Dict[str, Any]
    projections: List[str]
    aggregations: List[Dict[str, Any]]
    joins: List[Dict[str, Any]]
    order_by: List[Dict[str, str]]
    group_by: List[str]
    limit: Optional[int]
    offset: Optional[int]
    query_type: str
    optimization_hints: Dict[str, Any]


class SQLParser:
    """Advanced SQL parser using SQLGlot"""
    
    def __init__(self, default_dialect: SQLDialect = SQLDialect.POSTGRES):
        self.default_dialect = default_dialect
        self.supported_dialects = {
            SQLDialect.POSTGRES: "postgres",
            SQLDialect.MYSQL: "mysql", 
            SQLDialect.SQLITE: "sqlite",
            SQLDialect.BIGQUERY: "bigquery",
            SQLDialect.SNOWFLAKE: "snowflake",
            SQLDialect.SPARK: "spark",
            SQLDialect.CLICKHOUSE: "clickhouse"
        }
        
        logger.info(f"SQL Parser initialized with default dialect: {default_dialect.value}")
    
    def parse_sql(self, sql: str, dialect: Optional[SQLDialect] = None) -> ParsedQuery:
        """Parse SQL string into structured representation"""
        
        try:
            # Use specified dialect or default
            dialect = dialect or self.default_dialect
            dialect_name = self.supported_dialects.get(dialect, "postgres")
            
            logger.info(f"Parsing SQL query with dialect: {dialect_name}")
            
            # Parse SQL using SQLGlot
            try:
                ast = parse_one(sql, dialect=dialect_name)
            except Exception as parse_error:
                logger.error(f"SQL parsing failed: {parse_error}")
                return ParsedQuery(
                    query_type=QueryType.UNKNOWN,
                    original_sql=sql,
                    dialect=dialect,
                    ast=None,
                    tables=[],
                    columns=[],
                    filters={},
                    aggregations=[],
                    joins=[],
                    order_by=[],
                    group_by=[],
                    limit=None,
                    offset=None,
                    errors=[f"Parse error: {str(parse_error)}"],
                    warnings=[]
                )
            
            # Determine query type
            query_type = self._determine_query_type(ast)
            
            # Extract query components
            tables = self._extract_tables(ast)
            columns = self._extract_columns(ast)
            filters = self._extract_filters(ast)
            aggregations = self._extract_aggregations(ast)
            joins = self._extract_joins(ast)
            order_by = self._extract_order_by(ast)
            group_by = self._extract_group_by(ast)
            limit = self._extract_limit(ast)
            offset = self._extract_offset(ast)
            
            # Validate query
            errors, warnings = self._validate_query(ast, tables, columns)
            
            parsed_query = ParsedQuery(
                query_type=query_type,
                original_sql=sql,
                dialect=dialect,
                ast=ast,
                tables=tables,
                columns=columns,
                filters=filters,
                aggregations=aggregations,
                joins=joins,
                order_by=order_by,
                group_by=group_by,
                limit=limit,
                offset=offset,
                errors=errors,
                warnings=warnings
            )
            
            logger.info(f"Successfully parsed {query_type.value} query with {len(tables)} tables")
            return parsed_query
            
        except Exception as e:
            logger.error(f"Unexpected error parsing SQL: {e}")
            return ParsedQuery(
                query_type=QueryType.UNKNOWN,
                original_sql=sql,
                dialect=dialect or self.default_dialect,
                ast=None,
                tables=[],
                columns=[],
                filters={},
                aggregations=[],
                joins=[],
                order_by=[],
                group_by=[],
                limit=None,
                offset=None,
                errors=[f"Unexpected error: {str(e)}"],
                warnings=[]
            )
    
    def _determine_query_type(self, ast: Expression) -> QueryType:
        """Determine the type of SQL query"""
        if isinstance(ast, Select):
            return QueryType.SELECT
        elif isinstance(ast, Insert):
            return QueryType.INSERT
        elif isinstance(ast, Update):
            return QueryType.UPDATE
        elif isinstance(ast, Delete):
            return QueryType.DELETE
        elif isinstance(ast, Create):
            return QueryType.CREATE
        else:
            return QueryType.UNKNOWN
    
    def _extract_tables(self, ast: Expression) -> List[str]:
        """Extract table names from the AST"""
        tables = []
        
        try:
            # Find all table references
            for table in ast.find_all(sqlglot.expressions.Table):
                table_name = table.name
                if table_name and table_name not in tables:
                    tables.append(table_name)
        except Exception as e:
            logger.warning(f"Error extracting tables: {e}")
        
        return tables
    
    def _extract_columns(self, ast: Expression) -> List[str]:
        """Extract column names from the AST"""
        columns = []
        
        try:
            # Find all column references
            for column in ast.find_all(sqlglot.expressions.Column):
                col_name = column.name
                if col_name and col_name not in columns and col_name != "*":
                    columns.append(col_name)
        except Exception as e:
            logger.warning(f"Error extracting columns: {e}")
        
        return columns
    
    def _extract_filters(self, ast: Expression) -> Dict[str, Any]:
        """Extract WHERE clause filters"""
        filters = {}
        
        try:
            if isinstance(ast, Select):
                where_clause = ast.find(sqlglot.expressions.Where)
                if where_clause:
                    # Convert WHERE clause to internal filter format
                    filters = self._parse_where_clause(where_clause)
        except Exception as e:
            logger.warning(f"Error extracting filters: {e}")
        
        return filters
    
    def _parse_where_clause(self, where_clause) -> Dict[str, Any]:
        """Parse WHERE clause into internal filter format"""
        filters = {}
        
        try:
            # This is a simplified parser - in reality, you'd want more sophisticated logic
            where_sql = str(where_clause)
            
            # Basic parsing for common patterns
            # In production, you'd use the AST more thoroughly
            if "=" in where_sql:
                # Simple equality filters
                parts = where_sql.replace("WHERE", "").strip().split("AND")
                for part in parts:
                    if "=" in part:
                        left, right = part.split("=", 1)
                        column = left.strip()
                        value = right.strip().strip("'\"")
                        filters[column] = {"eq": value}
            
        except Exception as e:
            logger.warning(f"Error parsing WHERE clause: {e}")
        
        return filters
    
    def _extract_aggregations(self, ast: Expression) -> List[Dict[str, Any]]:
        """Extract aggregation functions"""
        aggregations = []
        
        try:
            if isinstance(ast, Select):
                # Find aggregate functions
                for func in ast.find_all(sqlglot.expressions.AggFunc):
                    agg_type = func.this.__class__.__name__.lower()
                    
                    # Get the column being aggregated
                    args = func.expressions
                    column = str(args[0]) if args else "*"
                    
                    aggregations.append({
                        "type": agg_type,
                        "column": column,
                        "alias": func.alias or f"{agg_type}_{column}"
                    })
        except Exception as e:
            logger.warning(f"Error extracting aggregations: {e}")
        
        return aggregations
    
    def _extract_joins(self, ast: Expression) -> List[Dict[str, Any]]:
        """Extract JOIN information"""
        joins = []
        
        try:
            if isinstance(ast, Select):
                # Find JOIN clauses
                for join in ast.find_all(sqlglot.expressions.Join):
                    join_info = {
                        "type": join.kind or "INNER",
                        "table": str(join.this),
                        "condition": str(join.on) if join.on else None
                    }
                    joins.append(join_info)
        except Exception as e:
            logger.warning(f"Error extracting joins: {e}")
        
        return joins
    
    def _extract_order_by(self, ast: Expression) -> List[str]:
        """Extract ORDER BY columns"""
        order_by = []
        
        try:
            if isinstance(ast, Select):
                order_clause = ast.find(sqlglot.expressions.Order)
                if order_clause:
                    for expr in order_clause.expressions:
                        order_by.append(str(expr))
        except Exception as e:
            logger.warning(f"Error extracting ORDER BY: {e}")
        
        return order_by
    
    def _extract_group_by(self, ast: Expression) -> List[str]:
        """Extract GROUP BY columns"""
        group_by = []
        
        try:
            if isinstance(ast, Select):
                group_clause = ast.find(sqlglot.expressions.Group)
                if group_clause:
                    for expr in group_clause.expressions:
                        group_by.append(str(expr))
        except Exception as e:
            logger.warning(f"Error extracting GROUP BY: {e}")
        
        return group_by
    
    def _extract_limit(self, ast: Expression) -> Optional[int]:
        """Extract LIMIT value"""
        try:
            if isinstance(ast, Select):
                limit_clause = ast.find(sqlglot.expressions.Limit)
                if limit_clause:
                    return int(str(limit_clause.expression))
        except Exception as e:
            logger.warning(f"Error extracting LIMIT: {e}")
        
        return None
    
    def _extract_offset(self, ast: Expression) -> Optional[int]:
        """Extract OFFSET value"""
        try:
            if isinstance(ast, Select):
                offset_clause = ast.find(sqlglot.expressions.Offset)
                if offset_clause:
                    return int(str(offset_clause.expression))
        except Exception as e:
            logger.warning(f"Error extracting OFFSET: {e}")
        
        return None
    
    def _validate_query(self, ast: Expression, tables: List[str], columns: List[str]) -> tuple[List[str], List[str]]:
        """Validate parsed query and return errors/warnings"""
        errors = []
        warnings = []
        
        # Basic validation
        if not tables and isinstance(ast, (Select, Update, Delete)):
            errors.append("No tables found in query")
        
        if not columns and isinstance(ast, Select):
            warnings.append("No specific columns selected (using SELECT *)")
        
        # Check for potential issues
        if len(tables) > 5:
            warnings.append(f"Query involves many tables ({len(tables)}), consider optimization")
        
        return errors, warnings
    
    def transpile_sql(self, sql: str, from_dialect: SQLDialect, to_dialect: SQLDialect) -> str:
        """Transpile SQL from one dialect to another"""
        try:
            from_dialect_name = self.supported_dialects.get(from_dialect, "postgres")
            to_dialect_name = self.supported_dialects.get(to_dialect, "postgres")
            
            result = transpile(sql, read=from_dialect_name, write=to_dialect_name)
            return result[0] if result else sql
            
        except Exception as e:
            logger.error(f"SQL transpilation failed: {e}")
            return sql
    
    def format_sql(self, sql: str, dialect: Optional[SQLDialect] = None) -> str:
        """Format SQL query for better readability"""
        try:
            dialect_name = self.supported_dialects.get(dialect or self.default_dialect, "postgres")
            ast = parse_one(sql, dialect=dialect_name)
            return ast.sql(dialect=dialect_name, pretty=True)
        except Exception as e:
            logger.error(f"SQL formatting failed: {e}")
            return sql
    
    def to_internal_query(self, parsed_query: ParsedQuery) -> InternalQuery:
        """Convert parsed SQL to internal query representation"""
        
        # Map table names to source IDs (this would integrate with metadata catalog)
        source_ids = [self._map_table_to_source(table) for table in parsed_query.tables]
        
        # Convert filters to internal format
        internal_filters = self._convert_filters_to_internal(parsed_query.filters)
        
        # Convert ORDER BY to internal format
        order_by_internal = []
        for order_expr in parsed_query.order_by:
            # Parse "column ASC/DESC"
            parts = order_expr.split()
            column = parts[0]
            direction = parts[1] if len(parts) > 1 else "ASC"
            order_by_internal.append({"column": column, "direction": direction})
        
        return InternalQuery(
            source_ids=source_ids,
            filters=internal_filters,
            projections=parsed_query.columns or ["*"],
            aggregations=parsed_query.aggregations,
            joins=parsed_query.joins,
            order_by=order_by_internal,
            group_by=parsed_query.group_by,
            limit=parsed_query.limit,
            offset=parsed_query.offset,
            query_type=parsed_query.query_type.value,
            optimization_hints=self._generate_optimization_hints(parsed_query)
        )
    
    def _map_table_to_source(self, table_name: str) -> str:
        """Map SQL table name to internal source ID"""
        # This would integrate with the metadata catalog
        # For now, assume direct mapping
        return table_name
    
    def _convert_filters_to_internal(self, filters: Dict[str, Any]) -> Dict[str, Any]:
        """Convert SQL filters to internal filter format"""
        internal_filters = {}
        
        for column, condition in filters.items():
            if isinstance(condition, dict):
                if "eq" in condition:
                    internal_filters[column] = condition["eq"]
                else:
                    internal_filters[column] = condition
            else:
                internal_filters[column] = condition
        
        return internal_filters
    
    def _generate_optimization_hints(self, parsed_query: ParsedQuery) -> Dict[str, Any]:
        """Generate optimization hints for the query"""
        hints = {}
        
        # Suggest index usage
        if parsed_query.filters:
            hints["suggested_indexes"] = list(parsed_query.filters.keys())
        
        # Suggest partition pruning
        if any(col.endswith("_date") or col.endswith("_time") for col in parsed_query.filters.keys()):
            hints["partition_pruning_candidate"] = True
        
        # Suggest parallelization
        if len(parsed_query.tables) > 1:
            hints["parallel_join_candidate"] = True
        
        return hints
