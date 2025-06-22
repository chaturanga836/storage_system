package handlers

import (
	"encoding/json"
	"log" // ← this line is missing
	"net/http"

	"github.com/gorilla/mux"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/duck"
)

func ListDuckDBTables(w http.ResponseWriter, r *http.Request) {
    log.Println("➡️ Start handler: ListDuckDBTables")

    tables, err := duck.ListTables()

    log.Println("✅ Reached after ListTables")

    if err != nil {
        log.Printf("❌ Error listing tables: %v", err)
        http.Error(w, "Failed to list tables: "+err.Error(), http.StatusInternalServerError)
        return
    }

    log.Println("📤 Sending response")

    json.NewEncoder(w).Encode(map[string]interface{}{
        "tables": tables,
    })
}


func GetTableRowCount(w http.ResponseWriter, r *http.Request) {
	table := mux.Vars(r)["name"]
	count, err := duck.RowCount(table)
	if err != nil {
		http.Error(w, "Failed to get row count: "+err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"table": table,
		"rows":  count,
	})
}
