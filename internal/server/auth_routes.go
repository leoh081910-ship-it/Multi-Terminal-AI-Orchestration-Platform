package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/auth"
)

// RegisterAuthRoutes registers authentication routes
func (s *Server) RegisterAuthRoutes(r chi.Router, tokenHandler *TokenHandler, authMiddleware *auth.Middleware) {
	r.Route("/auth/tokens", func(r chi.Router) {
		// All token routes require authentication
		r.Use(authMiddleware.Authenticate)

		r.Post("/", tokenHandler.CreateToken)
		r.Get("/", tokenHandler.ListTokens)
		r.Post("/{id}/revoke", tokenHandler.RevokeToken)
		r.Delete("/{id}", tokenHandler.DeleteToken)
	})
}
