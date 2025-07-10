package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenManager handles token creation, validation, and management
type TokenManager struct {
	secretKey []byte
	issuer    string
	defaultTTL time.Duration
}

// NewTokenManager creates a new token manager
func NewTokenManager(secretKey []byte, issuer string, defaultTTL time.Duration) *TokenManager {
	return &TokenManager{
		secretKey:  secretKey,
		issuer:     issuer,
		defaultTTL: defaultTTL,
	}
}

// GenerateJWT creates a new JWT token for the given claims
func (tm *TokenManager) GenerateJWT(tenantID, userID string, permissions []string) (string, error) {
	now := time.Now()
	claims := &Claims{
		TenantID:    tenantID,
		UserID:      userID,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tm.issuer,
			Subject:   userID,
			Audience:  []string{tenantID},
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.defaultTTL)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(tm.secretKey)
}

// RefreshToken creates a new token from existing valid claims
func (tm *TokenManager) RefreshToken(existingClaims *Claims) (string, error) {
	// Create new claims with updated timestamps
	now := time.Now()
	newClaims := &Claims{
		TenantID:    existingClaims.TenantID,
		UserID:      existingClaims.UserID,
		Permissions: existingClaims.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tm.issuer,
			Subject:   existingClaims.UserID,
			Audience:  existingClaims.Audience,
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.defaultTTL)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	return token.SignedString(tm.secretKey)
}

// APIKeyManager handles API key generation and validation
type APIKeyManager struct {
	// In a real implementation, this would connect to a database
	// For now, we'll use in-memory storage
	keys map[string]*APIKeyInfo
}

// APIKeyInfo contains information about an API key
type APIKeyInfo struct {
	TenantID    string
	TenantName  string
	Permissions []string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
	IsActive    bool
	RateLimit   int64
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager() *APIKeyManager {
	return &APIKeyManager{
		keys: make(map[string]*APIKeyInfo),
	}
}

// GenerateAPIKey creates a new API key for a tenant
func (akm *APIKeyManager) GenerateAPIKey(tenantID, tenantName string, permissions []string, ttl *time.Duration) (string, error) {
	// Generate a random API key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}

	// Hash the key for storage
	hash := sha256.Sum256(keyBytes)
	hashedKey := hex.EncodeToString(hash[:])

	// Create key info
	keyInfo := &APIKeyInfo{
		TenantID:    tenantID,
		TenantName:  tenantName,
		Permissions: permissions,
		CreatedAt:   time.Now(),
		IsActive:    true,
		RateLimit:   1000, // Default rate limit
	}

	if ttl != nil {
		expiresAt := time.Now().Add(*ttl)
		keyInfo.ExpiresAt = &expiresAt
	}

	// Store the key info (in production, this would be in a database)
	akm.keys[hashedKey] = keyInfo

	// Return the unhashed key to the user (they won't see this again)
	return hex.EncodeToString(keyBytes), nil
}

// ValidateAPIKey validates an API key and returns tenant information
func (akm *APIKeyManager) ValidateAPIKey(apiKey string) (*TenantInfo, error) {
	// Hash the provided key
	keyBytes, err := hex.DecodeString(apiKey)
	if err != nil {
		return nil, fmt.Errorf("invalid API key format")
	}

	hash := sha256.Sum256(keyBytes)
	hashedKey := hex.EncodeToString(hash[:])

	// Look up the key
	keyInfo, exists := akm.keys[hashedKey]
	if !exists {
		return nil, fmt.Errorf("API key not found")
	}

	// Check if key is active
	if !keyInfo.IsActive {
		return nil, fmt.Errorf("API key is inactive")
	}

	// Check if key is expired
	if keyInfo.ExpiresAt != nil && keyInfo.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("API key has expired")
	}

	// Return tenant information
	return &TenantInfo{
		TenantID:    keyInfo.TenantID,
		TenantName:  keyInfo.TenantName,
		Permissions: keyInfo.Permissions,
		RateLimit:   keyInfo.RateLimit,
	}, nil
}

// RevokeAPIKey deactivates an API key
func (akm *APIKeyManager) RevokeAPIKey(apiKey string) error {
	keyBytes, err := hex.DecodeString(apiKey)
	if err != nil {
		return fmt.Errorf("invalid API key format")
	}

	hash := sha256.Sum256(keyBytes)
	hashedKey := hex.EncodeToString(hash[:])

	keyInfo, exists := akm.keys[hashedKey]
	if !exists {
		return fmt.Errorf("API key not found")
	}

	keyInfo.IsActive = false
	return nil
}
