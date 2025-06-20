package handlers

import (
	"fmt"
	"net/http"
	"log"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/auth"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/utils/contextkeys"
)

func ProtectedEndpoint(w http.ResponseWriter, r *http.Request) {
	log.Println("👀 ProtectedEndpoint handler reached")
	claims := r.Context().Value(contextkeys.UserClaimsKey).(*auth.Claims)
	message := fmt.Sprintf("Hello %s! You are authorized as '%s'", claims.Username, claims.Role)
	w.Write([]byte(message))
}

func AdminOnlyEndpoint(w http.ResponseWriter, r *http.Request) {
	log.Println("👀 AdminOnlyEndpoint handler reached")
	claims := r.Context().Value(contextkeys.UserClaimsKey).(*auth.Claims)
	w.Write([]byte(fmt.Sprintf("Welcome, Admin %s! You have full access.", claims.Username)))
}
