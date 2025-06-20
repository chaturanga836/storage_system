package duck

import (
	"database/sql"
	_ "github.com/marcboeker/go-duckdb"
	"fmt"
)

const dbPath = "data/engine.db" // or wherever your main DuckDB file lives

func getConnection() (*sql.DB, error) {
	return sql.Open("duckdb", dbPath)
}

func ListTables() ([]string, error) {
	db, err := getConnection()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, nil
}

func RowCount(table string) (int64, error) {
	db, err := getConnection()
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var count int64
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	err = db.QueryRow(query).Scan(&count)
	return count, err
}
