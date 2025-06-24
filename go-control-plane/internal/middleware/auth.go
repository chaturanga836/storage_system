package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/chaturanga836/storage_system/go-control-plane/internal/auth"
)

func JWTAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		// ✅ Log the incoming Authorization header
		log.Printf("🪪 Incoming Auth Header: %s", authHeader)

		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Missing or invalid token", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.ParseJWT(tokenStr)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// ✅ Log the validated user
		log.Printf("✅ JWT validated for user: %s with role: %s", claims.Username, claims.Role)

		ctx := context.WithValue(r.Context(), "claims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
