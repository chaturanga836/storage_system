package auth

import (
	"time"
	"errors"
	"github.com/chaturanga836/storage_system/go-control-plane/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("super-secret-key") // You should load from env/config in real apps

type Claims struct {
	Username      string `json:"username"`
	Role          string `json:"role"`
	IsSuperAdmin  bool   `json:"is_super_admin"`
	jwt.RegisteredClaims
}

func GenerateJWT(user models.User) (string, error) {
	expiration := time.Now().Add(2 * time.Hour)
	claims := &Claims{
		Username:     user.Username,
		Role:         user.Role,
		IsSuperAdmin: user.IsSuperAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiration),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func ParseJWT(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}
	return claims, nil
}
