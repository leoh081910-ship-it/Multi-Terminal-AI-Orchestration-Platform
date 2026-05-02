package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TokenService handles API token operations
type TokenService struct {
	repo TokenRepository
}

// TokenRepository defines the interface for token storage
type TokenRepository interface {
	CreateToken(ctx context.Context, token *APIToken) error
	GetTokenByValue(ctx context.Context, tokenValue string) (*APIToken, error)
	GetTokensByUserID(ctx context.Context, userID string) ([]*APIToken, error)
	UpdateToken(ctx context.Context, token *APIToken) error
	DeleteToken(ctx context.Context, tokenID string) error
	RevokeToken(ctx context.Context, tokenID string, reason string) error
}

// APIToken represents an API token
type APIToken struct {
	ID            string
	Token         string
	Name          string
	Description   string
	UserID        string
	Scopes        []string
	CreatedAt     time.Time
	ExpiresAt     *time.Time
	LastUsedAt    *time.Time
	Revoked       bool
	RevokedAt     *time.Time
	RevokedReason string
}

// NewTokenService creates a new token service
func NewTokenService(repo TokenRepository) *TokenService {
	return &TokenService{
		repo: repo,
	}
}

// GenerateToken generates a new API token
func (s *TokenService) GenerateToken(ctx context.Context, userID, name, description string, scopes []string, expiresIn *time.Duration) (*APIToken, string, error) {
	// Generate random token
	rawToken, err := generateRandomToken(32)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Hash token for storage
	hashedToken := hashToken(rawToken)

	// Calculate expiration
	var expiresAt *time.Time
	if expiresIn != nil {
		exp := time.Now().Add(*expiresIn)
		expiresAt = &exp
	}

	token := &APIToken{
		ID:          uuid.New().String(),
		Token:       hashedToken,
		Name:        name,
		Description: description,
		UserID:      userID,
		Scopes:      scopes,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		Revoked:     false,
	}

	if err := s.repo.CreateToken(ctx, token); err != nil {
		return nil, "", fmt.Errorf("failed to create token: %w", err)
	}

	// Return token with raw value (only time it's visible)
	return token, rawToken, nil
}

// ValidateToken validates a token and returns the associated token info
func (s *TokenService) ValidateToken(ctx context.Context, rawToken string) (*APIToken, error) {
	hashedToken := hashToken(rawToken)

	token, err := s.repo.GetTokenByValue(ctx, hashedToken)
	if err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}

	// Check if revoked
	if token.Revoked {
		return nil, fmt.Errorf("token has been revoked")
	}

	// Check if expired
	if token.ExpiresAt != nil && time.Now().After(*token.ExpiresAt) {
		return nil, fmt.Errorf("token has expired")
	}

	// Update last used time
	now := time.Now()
	token.LastUsedAt = &now
	if err := s.repo.UpdateToken(ctx, token); err != nil {
		// Log error but don't fail validation
		fmt.Printf("failed to update last used time: %v\n", err)
	}

	return token, nil
}

// RevokeToken revokes a token
func (s *TokenService) RevokeToken(ctx context.Context, tokenID, reason string) error {
	return s.repo.RevokeToken(ctx, tokenID, reason)
}

// ListUserTokens lists all tokens for a user
func (s *TokenService) ListUserTokens(ctx context.Context, userID string) ([]*APIToken, error) {
	return s.repo.GetTokensByUserID(ctx, userID)
}

// DeleteToken deletes a token
func (s *TokenService) DeleteToken(ctx context.Context, tokenID string) error {
	return s.repo.DeleteToken(ctx, tokenID)
}

// generateRandomToken generates a cryptographically secure random token
func generateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// hashToken hashes a token using SHA-256
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(hash[:])
}
