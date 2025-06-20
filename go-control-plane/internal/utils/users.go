package utils

import (
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/chaturanga836/storage_system/go-control-plane/internal/models"
)

var userFile = "data/users.json"
var mu sync.Mutex

func LoadUsers() ([]models.User, error) {
	data, err := os.ReadFile(userFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []models.User{}, nil
		}
		return nil, err
	}
	var users []models.User
	err = json.Unmarshal(data, &users)
	return users, err
}

func SaveUsers(users []models.User) error {
	mu.Lock()
	defer mu.Unlock()

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(userFile, data, 0644)
}
