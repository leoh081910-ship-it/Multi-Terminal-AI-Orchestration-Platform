package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/yourusername/ai-orchestration-platform/internal/auth"
)

// TokenHandler handles token-related HTTP requests
type TokenHandler struct {
	tokenService *auth.TokenService
}

// NewTokenHandler creates a new token handler
func NewTokenHandler(tokenService *auth.TokenService) *TokenHandler {
	return &TokenHandler{
		tokenService: tokenService,
	}
}

// CreateTokenRequest represents a token creation request
type CreateTokenRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Scopes      []string `json:"scopes"`
	ExpiresIn   *int     `json:"expires_in"` // seconds
}

// TokenResponse represents a token response
type TokenResponse struct {
	ID          string     `json:"id"`
	Token       string     `json:"token,omitempty"` // Only included on creation
	Name        string     `json:"name"`
	Description string     `json:"description"`
	UserID      string     `json:"user_id"`
	Scopes      []string   `json:"scopes"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	Revoked     bool       `json:"revoked"`
}

// CreateToken handles POST /api/v1/auth/tokens
func (h *TokenHandler) CreateToken(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req CreateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	// Calculate expiration duration
	var expiresIn *time.Duration
	if req.ExpiresIn != nil {
		duration := time.Duration(*req.ExpiresIn) * time.Second
		expiresIn = &duration
	}

	// Generate token
	token, rawToken, err := h.tokenService.GenerateToken(
		r.Context(),
		userID,
		req.Name,
		req.Description,
		req.Scopes,
		expiresIn,
	)
	if err != nil {
		http.Error(w, "failed to create token", http.StatusInternalServerError)
		return
	}

	// Return response with raw token (only time it's visible)
	resp := TokenResponse{
		ID:          token.ID,
		Token:       rawToken,
		Name:        token.Name,
		Description: token.Description,
		UserID:      token.UserID,
		Scopes:      token.Scopes,
		CreatedAt:   token.CreatedAt,
		ExpiresAt:   token.ExpiresAt,
		Revoked:     token.Revoked,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// ListTokens handles GET /api/v1/auth/tokens
func (h *TokenHandler) ListTokens(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	// Get tokens
	tokens, err := h.tokenService.ListUserTokens(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to list tokens", http.StatusInternalServerError)
		return
	}

	// Convert to response
	resp := make([]TokenResponse, len(tokens))
	for i, token := range tokens {
		resp[i] = TokenResponse{
			ID:          token.ID,
			Name:        token.Name,
			Description: token.Description,
			UserID:      token.UserID,
			Scopes:      token.Scopes,
			CreatedAt:   token.CreatedAt,
			ExpiresAt:   token.ExpiresAt,
			LastUsedAt:  token.LastUsedAt,
			Revoked:     token.Revoked,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// RevokeToken handles POST /api/v1/auth/tokens/{id}/revoke
func (h *TokenHandler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	tokenID := chi.URLParam(r, "id")
	if tokenID == "" {
		http.Error(w, "token ID is required", http.StatusBadRequest)
		return
	}

	// Parse request
	var req struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Revoke token
	if err := h.tokenService.RevokeToken(r.Context(), tokenID, req.Reason); err != nil {
		http.Error(w, "failed to revoke token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteToken handles DELETE /api/v1/auth/tokens/{id}
func (h *TokenHandler) DeleteToken(w http.ResponseWriter, r *http.Request) {
	tokenID := chi.URLParam(r, "id")
	if tokenID == "" {
		http.Error(w, "token ID is required", http.StatusBadRequest)
		return
	}

	// Delete token
	if err := h.tokenService.DeleteToken(r.Context(), tokenID); err != nil {
		http.Error(w, "failed to delete token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RegisterTokenRoutes registers token routes
func RegisterTokenRoutes(r chi.Router, handler *TokenHandler, authMiddleware *auth.Middleware) {
	r.Route("/api/v1/auth/tokens", func(r chi.Router) {
		// All token routes require authentication
		r.Use(authMiddleware.Authenticate)

		r.Post("/", handler.CreateToken)
		r.Get("/", handler.ListTokens)
		r.Post("/{id}/revoke", handler.RevokeToken)
		r.Delete("/{id}", handler.DeleteToken)
	})
}
