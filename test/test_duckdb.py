# test/test_duckdb.py

import duckdb

def test_duckdb_connection():
    con = duckdb.connect(':memory:')
    result = con.execute("SELECT 42").fetchone()
    assert result[0] == 42
