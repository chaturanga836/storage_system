package handlers

import (
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/models"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/utils"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	users, _ := utils.LoadUsers()
	for _, u := range users {
		if u.Username == req.Username {
			http.Error(w, "Username already exists", http.StatusConflict)
			return
		}
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), 12)

	users = append(users, models.User{
		Username: req.Username,
		Password: string(hashed),
		Role:     req.Role,
	})

	if err := utils.SaveUsers(users); err != nil {
		http.Error(w, "Failed to save user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User registered successfully"))
}
