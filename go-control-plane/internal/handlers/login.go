package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/chaturanga836/storage_system/go-control-plane/internal/auth"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/registry"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Username == "" || req.Password == "" {
		http.Error(w, "Invalid login payload", http.StatusBadRequest)
		return
	}

	user, ok := registry.GetUserByUsername(req.Username)
	if !ok || !auth.CheckPasswordHash(req.Password, user.Password) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateJWT(*user)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(LoginResponse{Token: token})
}
