package auth

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is a custom type for context keys
type contextKey string

const (
	// TokenContextKey is the context key for the authenticated token
	TokenContextKey contextKey = "auth_token"
	// UserIDContextKey is the context key for the authenticated user ID
	UserIDContextKey contextKey = "user_id"
)

// Middleware provides authentication middleware
type Middleware struct {
	tokenService *TokenService
}

// NewMiddleware creates a new authentication middleware
func NewMiddleware(tokenService *TokenService) *Middleware {
	return &Middleware{
		tokenService: tokenService,
	}
}

// Authenticate is the authentication middleware
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		// Check Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Validate token
		apiToken, err := m.tokenService.ValidateToken(r.Context(), token)
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Add token and user ID to context
		ctx := context.WithValue(r.Context(), TokenContextKey, apiToken)
		ctx = context.WithValue(ctx, UserIDContextKey, apiToken.UserID)

		// Call next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuthenticate is an optional authentication middleware
// It validates the token if present but doesn't require it
func (m *Middleware) OptionalAuthenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// No token, continue without authentication
			next.ServeHTTP(w, r)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			// Invalid format, continue without authentication
			next.ServeHTTP(w, r)
			return
		}

		token := parts[1]
		apiToken, err := m.tokenService.ValidateToken(r.Context(), token)
		if err != nil {
			// Invalid token, continue without authentication
			next.ServeHTTP(w, r)
			return
		}

		// Add token and user ID to context
		ctx := context.WithValue(r.Context(), TokenContextKey, apiToken)
		ctx = context.WithValue(ctx, UserIDContextKey, apiToken.UserID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetTokenFromContext retrieves the token from context
func GetTokenFromContext(ctx context.Context) (*APIToken, bool) {
	token, ok := ctx.Value(TokenContextKey).(*APIToken)
	return token, ok
}

// GetUserIDFromContext retrieves the user ID from context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDContextKey).(string)
	return userID, ok
}

// RequireScopes creates a middleware that requires specific scopes
func (m *Middleware) RequireScopes(scopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := GetTokenFromContext(r.Context())
			if !ok {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}

			// Check if token has all required scopes
			if !hasAllScopes(token.Scopes, scopes) {
				http.Error(w, "insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// hasAllScopes checks if tokenScopes contains all requiredScopes
func hasAllScopes(tokenScopes, requiredScopes []string) bool {
	scopeMap := make(map[string]bool)
	for _, scope := range tokenScopes {
		scopeMap[scope] = true
	}

	for _, required := range requiredScopes {
		if !scopeMap[required] {
			return false
		}
	}

	return true
}
