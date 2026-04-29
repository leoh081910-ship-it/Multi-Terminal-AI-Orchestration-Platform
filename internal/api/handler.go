// Package api provides HTTP API and WebSocket handlers for the AI orchestration platform.
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/store"
	"github.com/rs/zerolog"
)

// Handler provides HTTP API handlers.
type Handler struct {
	repo   *store.Repository
	logger zerolog.Logger
	router chi.Router
}

// NewHandler creates a new API handler.
func NewHandler(repo *store.Repository, logger zerolog.Logger) *Handler {
	h := &Handler{
		repo:   repo,
		logger: logger,
		router: chi.NewRouter(),
	}
	h.setupRoutes()
	return h
}

// Router returns the HTTP router.
func (h *Handler) Router() chi.Router {
	return h.router
}

func (h *Handler) setupRoutes() {
	h.router.Use(middleware.RequestID)
	h.router.Use(middleware.RealIP)
	h.router.Use(middleware.Recoverer)
	h.router.Use(middleware.Timeout(30 * time.Second))
	h.router.Use(h.requestLogger)

	// Health check
	h.router.Get("/health", h.handleHealth)

	// API v1 routes
	h.router.Route("/api/v1", func(r chi.Router) {
		// Task routes
		r.Route("/tasks", func(r chi.Router) {
			r.Get("/", h.handleListTasks)
			r.Post("/", h.handleCreateTask)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", h.handleGetTask)
				r.Put("/", h.handleUpdateTask)
				r.Delete("/", h.handleDeleteTask)
				r.Post("/cancel", h.handleCancelTask)
				r.Post("/retry", h.handleRetryTask)
			})
		})

		// Wave routes
		r.Route("/waves", func(r chi.Router) {
			r.Get("/", h.handleListWaves)
			r.Route("/{dispatchRef}/{wave}", func(r chi.Router) {
				r.Get("/", h.handleGetWave)
				r.Put("/seal", h.handleSealWave)
			})
		})

		// Event routes
		r.Route("/events", func(r chi.Router) {
			r.Get("/", h.handleListEvents)
			r.Get("/stream", h.handleEventStream)
		})

		// Dispatch routes
		r.Route("/dispatches/{dispatchRef}", func(r chi.Router) {
			r.Get("/tasks", h.handleListTasksByDispatch)
			r.Post("/waves", h.handleCreateWave)
		})
	})
}

// Health response
type healthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		Version:   "1.0.0",
	}
	h.respondJSON(w, http.StatusOK, resp)
}

// Request/Response types
type createTaskRequest struct {
	ID                 string                 `json:"id"`
	DispatchRef        string                 `json:"dispatch_ref"`
	Source             string                 `json:"source"`
	SourceRef          string                 `json:"source_ref"`
	Type               string                 `json:"type"`
	Objective          string                 `json:"objective"`
	Context            map[string]interface{} `json:"context"`
	FilesToRead        []string               `json:"files_to_read"`
	FilesToModify      []string               `json:"files_to_modify"`
	AcceptanceCriteria []string               `json:"acceptance_criteria"`
	Relations          []TaskRelation         `json:"relations"`
	Wave               int                    `json:"wave"`
	Priority           int                    `json:"priority"`
	Transport          string                 `json:"transport"`
}

type TaskRelation struct {
	TaskID string `json:"task_id"`
	Type   string `json:"type"`
	Reason string `json:"reason,omitempty"`
}

type taskResponse struct {
	ID                 string          `json:"id"`
	DispatchRef        string          `json:"dispatch_ref"`
	State              string          `json:"state"`
	RetryCount         int             `json:"retry_count"`
	LoopIterationCount int             `json:"loop_iteration_count"`
	Transport          string          `json:"transport"`
	Wave               int             `json:"wave"`
	TopoRank           int             `json:"topo_rank"`
	WorkspacePath      string          `json:"workspace_path,omitempty"`
	ArtifactPath       string          `json:"artifact_path,omitempty"`
	LastErrorReason    string          `json:"last_error_reason,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	CardJSON           json.RawMessage `json:"card_json,omitempty"`
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (h *Handler) respondNotImplemented(w http.ResponseWriter, r *http.Request) {
	h.respondJSON(w, http.StatusNotImplemented, errorResponse{
		Error:   "not_implemented",
		Message: "API endpoint is not implemented on this handler",
	})
}

// Placeholder handlers - full implementation in separate files
func (h *Handler) handleListTasks(w http.ResponseWriter, r *http.Request) {
	h.respondNotImplemented(w, r)
}
func (h *Handler) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	h.respondNotImplemented(w, r)
}
func (h *Handler) handleGetTask(w http.ResponseWriter, r *http.Request) {
	h.respondNotImplemented(w, r)
}
func (h *Handler) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	h.respondNotImplemented(w, r)
}
func (h *Handler) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	h.respondNotImplemented(w, r)
}
func (h *Handler) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	h.respondNotImplemented(w, r)
}
func (h *Handler) handleRetryTask(w http.ResponseWriter, r *http.Request) {
	h.respondNotImplemented(w, r)
}
func (h *Handler) handleListWaves(w http.ResponseWriter, r *http.Request) {
	h.respondNotImplemented(w, r)
}
func (h *Handler) handleGetWave(w http.ResponseWriter, r *http.Request) {
	h.respondNotImplemented(w, r)
}
func (h *Handler) handleSealWave(w http.ResponseWriter, r *http.Request) {
	h.respondNotImplemented(w, r)
}
func (h *Handler) handleListEvents(w http.ResponseWriter, r *http.Request) {
	h.respondNotImplemented(w, r)
}
func (h *Handler) handleEventStream(w http.ResponseWriter, r *http.Request) {
	h.respondNotImplemented(w, r)
}
func (h *Handler) handleListTasksByDispatch(w http.ResponseWriter, r *http.Request) {
	h.respondNotImplemented(w, r)
}
func (h *Handler) handleCreateWave(w http.ResponseWriter, r *http.Request) {
	h.respondNotImplemented(w, r)
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		h.logger.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote", r.RemoteAddr).
			Dur("duration", time.Since(start)).
			Msg("HTTP request")
	})
}
