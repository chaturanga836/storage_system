from fastapi import APIRouter, HTTPException
from pydantic import BaseModel
from typing import List, Dict
from engine.writer import write_jsonl, write_parquet
from engine.reader import read_jsonl, read_parquet, CACHE
from engine.catalog import list_partitions, get_catalog_stats

router = APIRouter()

class InsertPayload(BaseModel):
    dataset: str
    tenant_id: str
    records: List[Dict]
    format: str = "jsonl"  # or "parquet"

class QueryPayload(BaseModel):
    dataset: str
    tenant_id: str
    format: str = "jsonl"  # or "parquet"

@router.post("/insert")
def insert_data(payload: InsertPayload):
    if payload.format == "jsonl":
        path = write_jsonl(payload.dataset, payload.tenant_id, payload.records)
    elif payload.format == "parquet":
        path = write_parquet(payload.dataset, payload.tenant_id, payload.records)
    else:
        raise HTTPException(status_code=400, detail="Unsupported format")
    return {"status": "success", "path": path}

@router.post("/query")
def query_data(payload: QueryPayload):
    prefix = f"{payload.dataset}/{payload.tenant_id}/"
    files = sorted(list_partitions(prefix), reverse=True)
    if not files:
        raise HTTPException(status_code=404, detail="No data found")
    latest = files[0]
    if payload.format == "jsonl":
        data = read_jsonl(latest)
    elif payload.format == "parquet":
        data = read_parquet(latest)
    else:
        raise HTTPException(status_code=400, detail="Unsupported format")
    return {"file": latest, "records": data}

@router.get("/stats")
def get_stats():
    return get_catalog_stats()

@router.get("/cache")
def get_cache_info():
    return CACHE.stats()
