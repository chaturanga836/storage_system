package main

import (
	"log"
	"net/http"

	"github.com/rs/cors"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/duck"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/registry"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/router"
)

func main() {
	// ✅ Initialize DuckDB
	err := duck.InitDuckDB("data/duck.db")
	if err != nil {
		log.Fatalf("DuckDB init failed: %v", err)
	}

	// ✅ Load users into in-memory registry
	registry.LoadUserRegistry()

	// ✅ Setup routes
	r := router.SetupRoutes()

	// ✅ Enable CORS for frontend (localhost:3000)
	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}).Handler(logRequest(r))

	log.Println("🚀 Starting Go Storage Control Plane on port 8081")

	// ✅ Wrap: CORS → Logging → Router
	// handler := c.Handler(logRequest(r))
	if err := http.ListenAndServe(":8081", handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// ✅ Middleware: Log requests, handle OPTIONS preflight
func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📥 %s %s", r.Method, r.URL.Path)

		// ✅ Allow preflight to succeed
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
