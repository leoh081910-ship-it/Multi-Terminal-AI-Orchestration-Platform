package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticateMiddleware(t *testing.T) {
	repo := newMockTokenRepository()
	service := NewTokenService(repo)
	middleware := NewMiddleware(service)

	ctx := context.Background()
	userID := "user-123"

	_, rawToken, err := service.GenerateToken(ctx, userID, "Test", "", []string{"read"}, nil)
	require.NoError(t, err)

	handler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := GetTokenFromContext(r.Context())
		assert.True(t, ok)
		assert.NotNil(t, token)

		uid, ok := GetUserIDFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, userID, uid)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthenticateMissingToken(t *testing.T) {
	repo := newMockTokenRepository()
	service := NewTokenService(repo)
	middleware := NewMiddleware(service)

	handler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthenticateInvalidToken(t *testing.T) {
	repo := newMockTokenRepository()
	service := NewTokenService(repo)
	middleware := NewMiddleware(service)

	handler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOptionalAuthenticateWithToken(t *testing.T) {
	repo := newMockTokenRepository()
	service := NewTokenService(repo)
	middleware := NewMiddleware(service)

	ctx := context.Background()
	userID := "user-123"

	_, rawToken, err := service.GenerateToken(ctx, userID, "Test", "", []string{"read"}, nil)
	require.NoError(t, err)

	handler := middleware.OptionalAuthenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := GetTokenFromContext(r.Context())
		assert.True(t, ok)
		assert.NotNil(t, token)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOptionalAuthenticateWithoutToken(t *testing.T) {
	repo := newMockTokenRepository()
	service := NewTokenService(repo)
	middleware := NewMiddleware(service)

	handler := middleware.OptionalAuthenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := GetTokenFromContext(r.Context())
		assert.False(t, ok)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireScopes(t *testing.T) {
	repo := newMockTokenRepository()
	service := NewTokenService(repo)
	middleware := NewMiddleware(service)

	ctx := context.Background()
	userID := "user-123"

	_, rawToken, err := service.GenerateToken(ctx, userID, "Test", "", []string{"read", "write"}, nil)
	require.NoError(t, err)

	authHandler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scopeHandler := middleware.RequireScopes("read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		scopeHandler.ServeHTTP(w, r)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	w := httptest.NewRecorder()

	authHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireScopesInsufficientPermissions(t *testing.T) {
	repo := newMockTokenRepository()
	service := NewTokenService(repo)
	middleware := NewMiddleware(service)

	ctx := context.Background()
	userID := "user-123"

	_, rawToken, err := service.GenerateToken(ctx, userID, "Test", "", []string{"read"}, nil)
	require.NoError(t, err)

	authHandler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scopeHandler := middleware.RequireScopes("write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		}))
		scopeHandler.ServeHTTP(w, r)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	w := httptest.NewRecorder()

	authHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
