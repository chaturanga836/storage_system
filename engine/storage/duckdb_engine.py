# engine/storage/duckdb_engine.py

import duckdb
import os
from typing import List, Dict, Any

DB_DIR = "data/duckdb"  # path to store .duckdb files

os.makedirs(DB_DIR, exist_ok=True)

def get_db_path(tenant_id: str) -> str:
    return os.path.join(DB_DIR, f"{tenant_id}.duckdb")

def init_db(tenant_id: str):
    db_path = get_db_path(tenant_id)
    with duckdb.connect(db_path) as conn:
        conn.execute("""
            CREATE TABLE IF NOT EXISTS example (
                id INTEGER,
                name TEXT,
                created_at TIMESTAMP DEFAULT now()
            );
        """)
    return db_path

def insert_example_data(tenant_id: str, data: List[Dict[str, Any]]):
    db_path = get_db_path(tenant_id)
    with duckdb.connect(db_path) as conn:
        for row in data:
            conn.execute("INSERT INTO example (id, name) VALUES (?, ?)", (row["id"], row["name"]))

def query_example_data(tenant_id: str) -> List[Dict[str, Any]]:
    db_path = get_db_path(tenant_id)
    with duckdb.connect(db_path) as conn:
        result = conn.execute("SELECT * FROM example").fetchall()
        return [{"id": row[0], "name": row[1], "created_at": row[2]} for row in result]
