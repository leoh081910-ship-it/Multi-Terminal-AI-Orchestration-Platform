package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/ai-orchestration-platform/ent"
	"github.com/yourusername/ai-orchestration-platform/ent/apitoken"
)

// EntTokenRepository implements TokenRepository using ent
type EntTokenRepository struct {
	client *ent.Client
}

// NewEntTokenRepository creates a new ent-based token repository
func NewEntTokenRepository(client *ent.Client) *EntTokenRepository {
	return &EntTokenRepository{
		client: client,
	}
}

// CreateToken creates a new token
func (r *EntTokenRepository) CreateToken(ctx context.Context, token *APIToken) error {
	builder := r.client.APIToken.Create().
		SetID(token.ID).
		SetToken(token.Token).
		SetName(token.Name).
		SetUserID(token.UserID).
		SetCreatedAt(token.CreatedAt).
		SetRevoked(token.Revoked)

	if token.Description != "" {
		builder.SetDescription(token.Description)
	}

	if len(token.Scopes) > 0 {
		builder.SetScopes(token.Scopes)
	}

	if token.ExpiresAt != nil {
		builder.SetExpiresAt(*token.ExpiresAt)
	}

	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create token: %w", err)
	}

	return nil
}

// GetTokenByValue retrieves a token by its value
func (r *EntTokenRepository) GetTokenByValue(ctx context.Context, tokenValue string) (*APIToken, error) {
	t, err := r.client.APIToken.Query().
		Where(apitoken.TokenEQ(tokenValue)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	return entTokenToAPIToken(t), nil
}

// GetTokensByUserID retrieves all tokens for a user
func (r *EntTokenRepository) GetTokensByUserID(ctx context.Context, userID string) ([]*APIToken, error) {
	tokens, err := r.client.APIToken.Query().
		Where(apitoken.UserIDEQ(userID)).
		Order(ent.Desc(apitoken.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokens: %w", err)
	}

	result := make([]*APIToken, len(tokens))
	for i, t := range tokens {
		result[i] = entTokenToAPIToken(t)
	}

	return result, nil
}

// UpdateToken updates a token
func (r *EntTokenRepository) UpdateToken(ctx context.Context, token *APIToken) error {
	builder := r.client.APIToken.UpdateOneID(token.ID)

	if token.LastUsedAt != nil {
		builder.SetLastUsedAt(*token.LastUsedAt)
	}

	if token.Revoked {
		builder.SetRevoked(true)
		if token.RevokedAt != nil {
			builder.SetRevokedAt(*token.RevokedAt)
		}
		if token.RevokedReason != "" {
			builder.SetRevokedReason(token.RevokedReason)
		}
	}

	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update token: %w", err)
	}

	return nil
}

// DeleteToken deletes a token
func (r *EntTokenRepository) DeleteToken(ctx context.Context, tokenID string) error {
	err := r.client.APIToken.DeleteOneID(tokenID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	return nil
}

// RevokeToken revokes a token
func (r *EntTokenRepository) RevokeToken(ctx context.Context, tokenID string, reason string) error {
	now := time.Now()
	_, err := r.client.APIToken.UpdateOneID(tokenID).
		SetRevoked(true).
		SetRevokedAt(now).
		SetRevokedReason(reason).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	return nil
}

// entTokenToAPIToken converts ent.APIToken to auth.APIToken
func entTokenToAPIToken(t *ent.APIToken) *APIToken {
	token := &APIToken{
		ID:        t.ID,
		Token:     t.Token,
		Name:      t.Name,
		UserID:    t.UserID,
		CreatedAt: t.CreatedAt,
		Revoked:   t.Revoked,
	}

	if desc, ok := t.Description(); ok {
		token.Description = desc
	}

	if scopes, ok := t.Scopes(); ok {
		token.Scopes = scopes
	}

	if exp, ok := t.ExpiresAt(); ok {
		token.ExpiresAt = &exp
	}

	if lastUsed, ok := t.LastUsedAt(); ok {
		token.LastUsedAt = &lastUsed
	}

	if revokedAt, ok := t.RevokedAt(); ok {
		token.RevokedAt = &revokedAt
	}

	if reason, ok := t.RevokedReason(); ok {
		token.RevokedReason = reason
	}

	return token
}
