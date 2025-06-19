def test_register_and_fetch_partition():
    from engine.catalog import register_partition, get_partition_meta
    register_partition("foo/bar/file.jsonl", 100, 10, "ts1", "ts2")
    meta = get_partition_meta("foo/bar/file.jsonl")
    assert meta["rows"] == 10
