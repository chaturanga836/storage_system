package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Authenticator defines the interface for authentication operations
type Authenticator interface {
	// ValidateToken validates a JWT token and returns the claims
	ValidateToken(ctx context.Context, token string) (*Claims, error)
	
	// ValidateAPIKey validates an API key and returns the associated tenant info
	ValidateAPIKey(ctx context.Context, apiKey string) (*TenantInfo, error)
	
	// Authorize checks if the authenticated user has permission for the operation
	Authorize(ctx context.Context, claims *Claims, resource string, action string) error
}

// Claims represents the JWT token claims
type Claims struct {
	TenantID    string   `json:"tenant_id"`
	UserID      string   `json:"user_id"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// TenantInfo contains information about a tenant from API key validation
type TenantInfo struct {
	TenantID    string
	TenantName  string
	Permissions []string
	RateLimit   int64
}

// JWTAuthenticator implements the Authenticator interface using JWT tokens
type JWTAuthenticator struct {
	secretKey []byte
	issuer    string
}

// NewJWTAuthenticator creates a new JWT-based authenticator
func NewJWTAuthenticator(secretKey []byte, issuer string) *JWTAuthenticator {
	return &JWTAuthenticator{
		secretKey: secretKey,
		issuer:    issuer,
	}
}

// ValidateToken validates a JWT token and returns the claims
func (ja *JWTAuthenticator) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return ja.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	// Validate issuer
	if claims.Issuer != ja.issuer {
		return nil, fmt.Errorf("invalid issuer")
	}

	// Check if token is expired
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}

	return claims, nil
}

// ValidateAPIKey validates an API key (placeholder implementation)
func (ja *JWTAuthenticator) ValidateAPIKey(ctx context.Context, apiKey string) (*TenantInfo, error) {
	// TODO: Implement API key validation logic
	// This would typically involve:
	// 1. Hashing the API key
	// 2. Looking up the key in a database
	// 3. Retrieving associated tenant information
	// 4. Checking if the key is active/not expired
	
	return nil, fmt.Errorf("API key validation not implemented")
}

// Authorize checks if the authenticated user has permission for the operation
func (ja *JWTAuthenticator) Authorize(ctx context.Context, claims *Claims, resource string, action string) error {
	// Simple permission check based on claims
	requiredPermission := fmt.Sprintf("%s:%s", resource, action)
	
	for _, permission := range claims.Permissions {
		if permission == requiredPermission || permission == "*" {
			return nil
		}
	}
	
	return fmt.Errorf("insufficient permissions for %s on %s", action, resource)
}

// AuthMiddleware provides authentication middleware functionality
type AuthMiddleware struct {
	authenticator Authenticator
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authenticator Authenticator) *AuthMiddleware {
	return &AuthMiddleware{
		authenticator: authenticator,
	}
}

// ExtractAndValidateToken extracts and validates a token from the request context
func (am *AuthMiddleware) ExtractAndValidateToken(ctx context.Context, token string) (*Claims, error) {
	if token == "" {
		return nil, fmt.Errorf("missing authentication token")
	}

	// Remove "Bearer " prefix if present
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	return am.authenticator.ValidateToken(ctx, token)
}
