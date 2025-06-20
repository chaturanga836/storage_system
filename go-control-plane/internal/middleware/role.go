package middleware

import (
	"log"
	"net/http"

	"github.com/chaturanga836/storage_system/go-control-plane/internal/auth"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/utils/contextkeys"
)

func RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// ✅ Use correct context key type
			claims, ok := r.Context().Value(contextkeys.UserClaimsKey).(*auth.Claims)
			if !ok {
				http.Error(w, "Missing claims", http.StatusUnauthorized)
				return
			}

			log.Printf("🔐 Role check: expected=%s, actual=%s", requiredRole, claims.Role)

			if claims.Role != requiredRole {
				http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
