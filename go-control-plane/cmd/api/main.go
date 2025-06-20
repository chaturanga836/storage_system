package main

import (
	"log"
	"net/http"

	"github.com/chaturanga836/storage_system/go-control-plane/internal/router"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/registry"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/duck"
	
)

func main() {
	err := duck.InitDuckDB("data/duck.db")
	if err != nil {
		log.Fatalf("DuckDB init failed: %v", err)
	}

	registry.LoadUserRegistry()
	r := router.SetupRoutes()

	log.Println("🚀 Starting Go Storage Control Plane on port 8081")

	// ✅ Wrap router with logging middleware
	if err := http.ListenAndServe(":8081", logRequest(r)); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// Logs every incoming HTTP request
func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📥 %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
