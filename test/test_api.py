import sys
import os
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

def test_insert_and_query():
    from fastapi.testclient import TestClient
    from main import app
    client = TestClient(app)

    payload = {
        "dataset": "sample",
        "tenant_id": "t1",
        "format": "jsonl",
        "records": [{"id": 123, "timestamp": "2025-01-01T00:00:00Z"}]
    }

    insert_res = client.post("/insert", json=payload)
    assert insert_res.status_code == 200

    query_res = client.post("/query", json={
        "dataset": "sample",
        "tenant_id": "t1",
        "format": "jsonl"
    })
    assert query_res.status_code == 200
    assert query_res.json()["records"][0]["id"] == 123
