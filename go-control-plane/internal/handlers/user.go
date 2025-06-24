package handlers

import (
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/models"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/utils"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/auth"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
	IsSuperAdmin bool   `json:"is_super_admin"`
}

func RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	users, err := utils.LoadUsers()
	if err != nil {
		http.Error(w, "Failed to load users", http.StatusInternalServerError)
		return
	}

	// Check for existing username
	for _, u := range users {
		if u.Username == req.Username {
			http.Error(w, "Username already exists", http.StatusConflict)
			return
		}
	}

	// Automatically assign super admin if this is the first user
	isSuper := len(users) == 0

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	newUser := models.User{
		Username:     req.Username,
		Password:     string(hashed),
		Role:         "user", // default role
		IsSuperAdmin: isSuper,
	}

	// Only allow role override if super admin already exists
	if isSuper {
		newUser.Role = "superadmin"
	}

	users = append(users, newUser)

	if err := utils.SaveUsers(users); err != nil {
		http.Error(w, "Failed to save user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User registered successfully"))
}

func Me(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*auth.Claims)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"username":       claims.Username,
		"role":           claims.Role,
		"is_super_admin": claims.IsSuperAdmin,
	})
}
