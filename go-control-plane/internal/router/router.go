package router

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/handlers"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/middleware"
)

func SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/health", handlers.HealthCheck).Methods("GET")
	r.HandleFunc("/auth/login", handlers.Login).Methods("POST")

	r.Handle("/protected", middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.ProtectedEndpoint))).Methods("GET")

	// ✅ Correct middleware nesting: JWT inside, role outside
	r.Handle("/admin-only",
		middleware.JWTAuthMiddleware(
			middleware.RequireRole("admin")(http.HandlerFunc(handlers.AdminOnlyEndpoint)),
		),
	).Methods("GET")

	r.HandleFunc("/auth/register", handlers.RegisterUser).Methods("POST")

	r.HandleFunc("/tenants/register", handlers.RegisterTenant).Methods("POST")
	r.HandleFunc("/tenants/assign", handlers.AssignNode).Methods("POST")

	r.Handle("/monitor",
		middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.Monitor)),
	).Methods("GET")

	r.Handle("/duckdb/tables",
		middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.ListDuckDBTables)),
	).Methods("GET")

	r.Handle("/duckdb/table/{name}/count",
		middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.GetTableRowCount)),
	).Methods("GET")

	r.Handle("/auth/me", middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.Me))).Methods("GET")

	// ✅ Add fallback OPTIONS handler to support CORS preflight globally
	r.PathPrefix("/").Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	return r
}
