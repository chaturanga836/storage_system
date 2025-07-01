"""
Query Interpreter Service - Main entry point
"""
import asyncio
import logging
from pathlib import Path
from typing import Dict, Any, Optional
from fastapi import FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
import uvicorn
import json

from .sql_parser import SQLParser, SQLDialect, ParsedQuery
from .dsl_parser import DSLParser, DSLQuery
from .query_transformer import QueryTransformer, QueryPlan

logger = logging.getLogger(__name__)

# FastAPI app
app = FastAPI(
    title="Query Interpreter Service",
    description="SQL and DSL query parsing and transformation service",
    version="1.0.0"
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Request/Response models
class SQLParseRequest(BaseModel):
    sql: str
    dialect: Optional[str] = "postgres"

class DSLParseRequest(BaseModel):
    query: str

class TransformRequest(BaseModel):
    internal_query: Dict[str, Any]
    query_id: Optional[str] = None

class SQLTranspileRequest(BaseModel):
    sql: str
    from_dialect: str = "postgres"
    to_dialect: str = "postgres"

class ParseResponse(BaseModel):
    success: bool
    result: Optional[Dict[str, Any]] = None
    errors: Optional[List[str]] = None
    warnings: Optional[List[str]] = None

class TransformResponse(BaseModel):
    success: bool
    execution_plan: Optional[Dict[str, Any]] = None
    errors: Optional[List[str]] = None


class QueryInterpreterService:
    """Main query interpreter service"""
    
    def __init__(self, config_path: str = "config.json"):
        self.config_path = config_path
        self.sql_parser = SQLParser()
        self.dsl_parser = DSLParser()
        self.transformer = QueryTransformer()
        
        # Load configuration
        self.config = self._load_config()
        
        logger.info("Query Interpreter Service initialized")
    
    def _load_config(self) -> Dict[str, Any]:
        """Load service configuration"""
        try:
            if Path(self.config_path).exists():
                with open(self.config_path, 'r') as f:
                    return json.load(f)
        except Exception as e:
            logger.warning(f"Could not load config: {e}")
        
        # Default configuration
        return {
            "service": {
                "name": "query-interpreter",
                "version": "1.0.0",
                "port": 8085
            },
            "sql": {
                "default_dialect": "postgres",
                "supported_dialects": ["postgres", "mysql", "sqlite", "bigquery"]
            },
            "optimization": {
                "enable_query_caching": True,
                "max_cache_size": 1000,
                "default_timeout_ms": 30000
            }
        }
    
    async def parse_sql(self, sql: str, dialect: Optional[str] = None) -> ParsedQuery:
        """Parse SQL query"""
        try:
            sql_dialect = SQLDialect(dialect) if dialect else SQLDialect.POSTGRES
            return self.sql_parser.parse_sql(sql, sql_dialect)
        except ValueError:
            # Unknown dialect, use default
            return self.sql_parser.parse_sql(sql, SQLDialect.POSTGRES)
    
    async def parse_dsl(self, query: str) -> DSLQuery:
        """Parse DSL query"""
        return self.dsl_parser.parse_dsl(query)
    
    async def transform_to_execution_plan(self, internal_query_dict: Dict[str, Any], 
                                        query_id: Optional[str] = None) -> QueryPlan:
        """Transform internal query to execution plan"""
        
        # Convert dict to InternalQuery object
        from .sql_parser import InternalQuery
        internal_query = InternalQuery(**internal_query_dict)
        
        return self.transformer.transform_to_execution_plan(internal_query, query_id)
    
    async def transpile_sql(self, sql: str, from_dialect: str, to_dialect: str) -> str:
        """Transpile SQL between dialects"""
        try:
            from_d = SQLDialect(from_dialect)
            to_d = SQLDialect(to_dialect)
            return self.sql_parser.transpile_sql(sql, from_d, to_d)
        except ValueError as e:
            logger.error(f"Invalid dialect: {e}")
            return sql


# Global service instance
service = QueryInterpreterService()


@app.post("/sql/parse", response_model=ParseResponse)
async def parse_sql(request: SQLParseRequest):
    """Parse SQL query endpoint"""
    try:
        parsed_query = await service.parse_sql(request.sql, request.dialect)
        
        # Convert to serializable format
        result = {
            "query_type": parsed_query.query_type.value,
            "tables": parsed_query.tables,
            "columns": parsed_query.columns,
            "filters": parsed_query.filters,
            "aggregations": parsed_query.aggregations,
            "joins": parsed_query.joins,
            "order_by": parsed_query.order_by,
            "group_by": parsed_query.group_by,
            "limit": parsed_query.limit,
            "offset": parsed_query.offset,
            "dialect": parsed_query.dialect.value
        }
        
        # Convert to internal query format
        internal_query = service.sql_parser.to_internal_query(parsed_query)
        result["internal_query"] = {
            "source_ids": internal_query.source_ids,
            "filters": internal_query.filters,
            "projections": internal_query.projections,
            "aggregations": internal_query.aggregations,
            "joins": internal_query.joins,
            "order_by": internal_query.order_by,
            "group_by": internal_query.group_by,
            "limit": internal_query.limit,
            "offset": internal_query.offset,
            "query_type": internal_query.query_type,
            "optimization_hints": internal_query.optimization_hints
        }
        
        return ParseResponse(
            success=len(parsed_query.errors) == 0,
            result=result,
            errors=parsed_query.errors,
            warnings=parsed_query.warnings
        )
        
    except Exception as e:
        logger.error(f"SQL parsing failed: {e}")
        return ParseResponse(
            success=False,
            errors=[f"Parsing failed: {str(e)}"]
        )


@app.post("/dsl/parse", response_model=ParseResponse)
async def parse_dsl(request: DSLParseRequest):
    """Parse DSL query endpoint"""
    try:
        parsed_query = await service.parse_dsl(request.query)
        
        # Convert to serializable format
        result = {
            "query_type": parsed_query.query_type.value,
            "source": parsed_query.source,
            "filters": parsed_query.filters,
            "projections": parsed_query.projections,
            "aggregations": parsed_query.aggregations,
            "sort": parsed_query.sort,
            "limit": parsed_query.limit,
            "offset": parsed_query.offset,
            "data": parsed_query.data
        }
        
        # Convert to internal query format
        internal_query = service.dsl_parser.to_internal_query(parsed_query)
        result["internal_query"] = {
            "source_ids": internal_query.source_ids,
            "filters": internal_query.filters,
            "projections": internal_query.projections,
            "aggregations": internal_query.aggregations,
            "joins": internal_query.joins,
            "order_by": internal_query.order_by,
            "group_by": internal_query.group_by,
            "limit": internal_query.limit,
            "offset": internal_query.offset,
            "query_type": internal_query.query_type,
            "optimization_hints": internal_query.optimization_hints
        }
        
        return ParseResponse(
            success=len(parsed_query.errors) == 0,
            result=result,
            errors=parsed_query.errors
        )
        
    except Exception as e:
        logger.error(f"DSL parsing failed: {e}")
        return ParseResponse(
            success=False,
            errors=[f"Parsing failed: {str(e)}"]
        )


@app.post("/transform", response_model=TransformResponse)
async def transform_query(request: TransformRequest):
    """Transform internal query to execution plan"""
    try:
        execution_plan = await service.transform_to_execution_plan(
            request.internal_query, 
            request.query_id
        )
        
        # Convert to serializable format
        plan_dict = service.transformer.to_dict(execution_plan)
        
        return TransformResponse(
            success=True,
            execution_plan=plan_dict
        )
        
    except Exception as e:
        logger.error(f"Query transformation failed: {e}")
        return TransformResponse(
            success=False,
            errors=[f"Transformation failed: {str(e)}"]
        )


@app.post("/sql/transpile")
async def transpile_sql(request: SQLTranspileRequest):
    """Transpile SQL between dialects"""
    try:
        result = await service.transpile_sql(
            request.sql, 
            request.from_dialect, 
            request.to_dialect
        )
        
        return {
            "success": True,
            "sql": result,
            "from_dialect": request.from_dialect,
            "to_dialect": request.to_dialect
        }
        
    except Exception as e:
        logger.error(f"SQL transpilation failed: {e}")
        return {
            "success": False,
            "error": str(e),
            "sql": request.sql  # Return original on failure
        }


@app.post("/sql/format")
async def format_sql(request: SQLParseRequest):
    """Format SQL for readability"""
    try:
        formatted = service.sql_parser.format_sql(request.sql, SQLDialect(request.dialect))
        
        return {
            "success": True,
            "formatted_sql": formatted,
            "dialect": request.dialect
        }
        
    except Exception as e:
        logger.error(f"SQL formatting failed: {e}")
        return {
            "success": False,
            "error": str(e),
            "formatted_sql": request.sql
        }


@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "service": "query-interpreter",
        "version": service.config["service"]["version"]
    }


@app.get("/status")
async def get_status():
    """Get service status and configuration"""
    return {
        "service": service.config["service"],
        "sql_config": service.config["sql"],
        "optimization": service.config["optimization"],
        "dialects_supported": list(service.sql_parser.supported_dialects.keys()),
        "features": {
            "sql_parsing": True,
            "dsl_parsing": True,
            "query_transformation": True,
            "sql_transpilation": True,
            "sql_formatting": True
        }
    }


@app.get("/")
async def root():
    """Root endpoint with service information"""
    return {
        "service": "Query Interpreter",
        "version": "1.0.0",
        "description": "SQL and DSL query parsing and transformation service",
        "endpoints": {
            "POST /sql/parse": "Parse SQL queries",
            "POST /dsl/parse": "Parse DSL queries", 
            "POST /transform": "Transform to execution plan",
            "POST /sql/transpile": "Transpile SQL between dialects",
            "POST /sql/format": "Format SQL queries",
            "GET /health": "Health check",
            "GET /status": "Service status"
        }
    }


class QueryInterpreterServer:
    """Query Interpreter gRPC/HTTP server"""
    
    def __init__(self, host: str = "0.0.0.0", port: int = 8085):
        self.host = host
        self.port = port
    
    async def start(self):
        """Start the query interpreter service"""
        logger.info(f"Starting Query Interpreter Service on {self.host}:{self.port}")
        
        config = uvicorn.Config(
            app=app,
            host=self.host,
            port=self.port,
            log_level="info"
        )
        
        server = uvicorn.Server(config)
        await server.serve()


if __name__ == "__main__":
    import sys
    
    # Configure logging
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    
    # Start service
    server = QueryInterpreterServer()
    
    try:
        asyncio.run(server.start())
    except KeyboardInterrupt:
        logger.info("Query Interpreter service stopped")
        sys.exit(0)
