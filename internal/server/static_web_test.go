package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

func TestStaticWebRoutesServeTriageIndex(t *testing.T) {
	dist := t.TempDir()
	indexPath := filepath.Join(dist, "index.html")
	if err := os.WriteFile(indexPath, []byte("<html><body>spa</body></html>"), 0644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	s := &Server{
		logger:     zerolog.Nop(),
		router:     chi.NewRouter(),
		webDistDir: dist,
	}
	s.registerStaticWebRoutes()

	req := httptest.NewRequest(http.MethodGet, "/triage", nil)
	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected /triage to serve SPA index, got status %d", res.Code)
	}
	if body := res.Body.String(); body != "<html><body>spa</body></html>" {
		t.Fatalf("unexpected /triage body: %q", body)
	}
}

func TestStaticWebRoutesHandleFaviconProbe(t *testing.T) {
	dist := t.TempDir()
	indexPath := filepath.Join(dist, "index.html")
	if err := os.WriteFile(indexPath, []byte("<html><body>spa</body></html>"), 0644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	s := &Server{
		logger:     zerolog.Nop(),
		router:     chi.NewRouter(),
		webDistDir: dist,
	}
	s.registerStaticWebRoutes()

	for _, path := range []string{"/favicon.ico", "/favicon.svg"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		res := httptest.NewRecorder()
		s.Handler().ServeHTTP(res, req)

		if res.Code != http.StatusNoContent {
			t.Fatalf("expected %s to return %d, got %d", path, http.StatusNoContent, res.Code)
		}
	}
}
