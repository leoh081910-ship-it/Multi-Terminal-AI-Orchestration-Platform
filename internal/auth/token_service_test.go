package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTokenRepository struct {
	tokens map[string]*APIToken
}

func newMockTokenRepository() *mockTokenRepository {
	return &mockTokenRepository{
		tokens: make(map[string]*APIToken),
	}
}

func (m *mockTokenRepository) CreateToken(ctx context.Context, token *APIToken) error {
	m.tokens[token.ID] = token
	return nil
}

func (m *mockTokenRepository) GetTokenByValue(ctx context.Context, tokenValue string) (*APIToken, error) {
	for _, t := range m.tokens {
		if t.Token == tokenValue {
			return t, nil
		}
	}
	return nil, assert.AnError
}

func (m *mockTokenRepository) GetTokensByUserID(ctx context.Context, userID string) ([]*APIToken, error) {
	var result []*APIToken
	for _, t := range m.tokens {
		if t.UserID == userID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockTokenRepository) UpdateToken(ctx context.Context, token *APIToken) error {
	m.tokens[token.ID] = token
	return nil
}

func (m *mockTokenRepository) DeleteToken(ctx context.Context, tokenID string) error {
	delete(m.tokens, tokenID)
	return nil
}

func (m *mockTokenRepository) RevokeToken(ctx context.Context, tokenID string, reason string) error {
	if token, ok := m.tokens[tokenID]; ok {
		token.Revoked = true
		now := time.Now()
		token.RevokedAt = &now
		token.RevokedReason = reason
	}
	return nil
}

func TestGenerateToken(t *testing.T) {
	repo := newMockTokenRepository()
	service := NewTokenService(repo)

	ctx := context.Background()
	userID := "user-123"
	name := "Test Token"
	description := "Test Description"
	scopes := []string{"read", "write"}
	expiresIn := 24 * time.Hour

	token, rawToken, err := service.GenerateToken(ctx, userID, name, description, scopes, &expiresIn)

	require.NoError(t, err)
	assert.NotEmpty(t, token.ID)
	assert.NotEmpty(t, token.Token)
	assert.NotEmpty(t, rawToken)
	assert.Equal(t, name, token.Name)
	assert.Equal(t, description, token.Description)
	assert.Equal(t, userID, token.UserID)
	assert.Equal(t, scopes, token.Scopes)
	assert.NotNil(t, token.ExpiresAt)
	assert.False(t, token.Revoked)
}

func TestValidateToken(t *testing.T) {
	repo := newMockTokenRepository()
	service := NewTokenService(repo)

	ctx := context.Background()
	userID := "user-123"

	token, rawToken, err := service.GenerateToken(ctx, userID, "Test", "", []string{"read"}, nil)
	require.NoError(t, err)

	validatedToken, err := service.ValidateToken(ctx, rawToken)
	require.NoError(t, err)
	assert.Equal(t, token.ID, validatedToken.ID)
	assert.Equal(t, token.UserID, validatedToken.UserID)
	assert.NotNil(t, validatedToken.LastUsedAt)
}

func TestValidateExpiredToken(t *testing.T) {
	repo := newMockTokenRepository()
	service := NewTokenService(repo)

	ctx := context.Background()
	userID := "user-123"
	expiresIn := -1 * time.Hour

	_, rawToken, err := service.GenerateToken(ctx, userID, "Expired", "", []string{"read"}, &expiresIn)
	require.NoError(t, err)

	_, err = service.ValidateToken(ctx, rawToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestValidateRevokedToken(t *testing.T) {
	repo := newMockTokenRepository()
	service := NewTokenService(repo)

	ctx := context.Background()
	userID := "user-123"

	token, rawToken, err := service.GenerateToken(ctx, userID, "Test", "", []string{"read"}, nil)
	require.NoError(t, err)

	err = service.RevokeToken(ctx, token.ID, "test revocation")
	require.NoError(t, err)

	_, err = service.ValidateToken(ctx, rawToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "revoked")
}

func TestListUserTokens(t *testing.T) {
	repo := newMockTokenRepository()
	service := NewTokenService(repo)

	ctx := context.Background()
	userID := "user-123"

	_, _, err := service.GenerateToken(ctx, userID, "Token 1", "", []string{"read"}, nil)
	require.NoError(t, err)

	_, _, err = service.GenerateToken(ctx, userID, "Token 2", "", []string{"write"}, nil)
	require.NoError(t, err)

	tokens, err := service.ListUserTokens(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, tokens, 2)
}
