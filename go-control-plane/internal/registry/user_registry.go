package registry

import (
	"log"

	"github.com/chaturanga836/storage_system/go-control-plane/internal/models"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/utils"
)

var users []models.User

func LoadUserRegistry() {
	loaded, err := utils.LoadUsers()
	if err != nil {
		log.Fatalf("❌ Failed to load users: %v", err)
	}
	users = loaded
	log.Printf("👥 Loaded %d users", len(users))
}

func GetUserByUsername(username string) (*models.User, bool) {
	for _, u := range users {
		if u.Username == username {
			return &u, true
		}
	}
	return nil, false
}
