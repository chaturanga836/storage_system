def test_write_jsonl(tmp_path):
    from engine.writer import write_jsonl
    records = [{"id": 1, "timestamp": "2025-01-01T00:00:00Z"}]
    path = write_jsonl("testset", "t1", records)
    assert path.endswith(".jsonl")

def test_write_parquet(tmp_path):
    from engine.writer import write_parquet
    records = [{"id": 1, "timestamp": "2025-01-01T00:00:00Z"}]
    path = write_parquet("testset", "t2", records)
    assert path.endswith(".parquet")
