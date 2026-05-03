package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/auth"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/backup"
)

// BackupHandler exposes HTTP endpoints for the backup service.
type BackupHandler struct {
	svc *backup.Service
}

// NewBackupHandler creates a backup HTTP handler.
func NewBackupHandler(svc *backup.Service) *BackupHandler {
	return &BackupHandler{svc: svc}
}

// RegisterBackupRoutes mounts backup endpoints under /api/v1/system/backups.
// All endpoints require an authenticated token.
func (s *Server) RegisterBackupRoutes(r chi.Router, h *BackupHandler, authMiddleware *auth.Middleware) {
	r.Route("/api/v1/system/backups", func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)

		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/stats", h.stats)
		r.Post("/{filename}/verify", h.verify)
		r.Post("/{filename}/restore", h.restore)
	})
}

// @Summary List all backups
// @Tags backup
// @Produce json
// @Security BearerAuth
// @Success 200 {object} APIResponse
// @Router /system/backups [get]
func (h *BackupHandler) list(w http.ResponseWriter, r *http.Request) {
	backups, err := h.svc.ListBackups()
	if err != nil {
		writeBackupError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeBackupJSON(w, http.StatusOK, APIResponse{Success: true, Data: backups})
}

// @Summary Create a manual backup
// @Tags backup
// @Produce json
// @Security BearerAuth
// @Success 201 {object} APIResponse
// @Router /system/backups [post]
func (h *BackupHandler) create(w http.ResponseWriter, r *http.Request) {
	info, err := h.svc.CreateBackup(r.Context())
	if err != nil {
		writeBackupError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeBackupJSON(w, http.StatusCreated, APIResponse{Success: true, Data: info})
}

// @Summary Get backup statistics
// @Tags backup
// @Produce json
// @Security BearerAuth
// @Success 200 {object} APIResponse
// @Router /system/backups/stats [get]
func (h *BackupHandler) stats(w http.ResponseWriter, r *http.Request) {
	writeBackupJSON(w, http.StatusOK, APIResponse{Success: true, Data: h.svc.Stats()})
}

// @Summary Verify a backup checksum
// @Tags backup
// @Produce json
// @Security BearerAuth
// @Param filename path string true "Backup filename"
// @Success 200 {object} APIResponse
// @Router /system/backups/{filename}/verify [post]
func (h *BackupHandler) verify(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	checksum, err := h.svc.VerifyBackup(filename)
	if err != nil {
		writeBackupError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeBackupJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"filename": filename, "checksum": checksum},
	})
}

// @Summary Restore from a backup
// @Tags backup
// @Produce json
// @Security BearerAuth
// @Param filename path string true "Backup filename"
// @Success 200 {object} APIResponse
// @Router /system/backups/{filename}/restore [post]
func (h *BackupHandler) restore(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if err := h.svc.RestoreBackup(filename); err != nil {
		writeBackupError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeBackupJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"restored_from": filename},
	})
}

func writeBackupJSON(w http.ResponseWriter, status int, v APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeBackupError(w http.ResponseWriter, status int, msg string) {
	writeBackupJSON(w, status, APIResponse{Success: false, Error: msg})
}
