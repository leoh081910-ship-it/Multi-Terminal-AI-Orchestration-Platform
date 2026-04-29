package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) SetWebDistDir(dir string) {
	if strings.TrimSpace(dir) == "" {
		return
	}
	s.webDistDir = filepath.FromSlash(dir)
}

func (s *Server) registerStaticWebRoutes() {
	dist := filepath.Clean(s.webDistDir)
	if dist == "." || dist == "" {
		return
	}
	if _, err := os.Stat(dist); err != nil {
		s.logger.Warn().Err(err).Str("dist_dir", dist).Msg("web dist directory unavailable; skipping static board routes")
		return
	}

	indexPath := filepath.Join(dist, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		s.logger.Warn().Err(err).Str("index", indexPath).Msg("web index.html missing; skipping static board routes")
		return
	}

	assetsDir := filepath.Join(dist, "assets")
	if _, err := os.Stat(assetsDir); err == nil {
		assetsFS := http.StripPrefix("/assets/", http.FileServer(http.Dir(assetsDir)))
		s.router.Handle("/assets/*", assetsFS)
	}

	s.router.Get("/board", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, indexPath)
	})
	s.router.Get("/board/*", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, indexPath)
	})
}
