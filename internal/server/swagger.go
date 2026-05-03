package server

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func (s *Server) registerSwaggerRoutes() {
	s.router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
	// Serve the generated spec directly
	s.router.Get("/swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "docs/swagger.json")
	})
}
