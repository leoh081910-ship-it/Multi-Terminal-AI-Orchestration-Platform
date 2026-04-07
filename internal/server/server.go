// Package server provides the HTTP API server for the AI orchestration platform.
package server

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/store"
	"github.com/rs/zerolog"
)

// APIResponse is the standard envelope for all API responses.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Server provides the HTTP API server.
type Server struct {
	repo   *store.Repository
	logger zerolog.Logger
	router *chi.Mux
}

// New creates a new Server instance.
func New(repo *store.Repository, logger zerolog.Logger) *Server {
	s := &Server{
		repo:   repo,
		logger: logger,
		router: chi.NewRouter(),
	}
	s.setupMiddleware()
	s.setupRoutes()
	return s
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) setupMiddleware() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(30 * time.Second))
	s.router.Use(s.requestLogger)
}

func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		s.logger.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", ww.Status()).
			Dur("duration", time.Since(start)).
			Msg("request")
	})
}

func (s *Server) setupRoutes() {
	s.router.Get("/health", s.handleHealth)

	s.router.Route("/api", func(r chi.Router) {
		// Task endpoints
		r.Route("/tasks", func(r chi.Router) {
			r.Get("/", s.handleListTasks)
			r.Get("/stats", s.handleTaskStats)
			r.Post("/", s.handleCreateTask)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", s.handleGetTask)
				r.Put("/", s.handleUpdateTask)
				r.Post("/cancel", s.handleCancelTask)
				r.Post("/retry", s.handleRetryTask)
			})
		})

		// Dispatch-scoped endpoints
		r.Route("/dispatches/{dispatchRef}", func(r chi.Router) {
			r.Get("/tasks", s.handleListTasksByDispatch)
			r.Post("/waves", s.handleCreateWave)
			r.Get("/waves/{wave}", s.handleGetWave)
			r.Put("/waves/{wave}/seal", s.handleSealWave)
		})
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"status": "ok"},
	})
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tasks, err := s.repo.ListAllTasks(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to list tasks")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to list tasks",
		})
		return
	}

	views := s.mapTaskViews(tasks)
	s.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    views,
	})
}

func (s *Server) handleTaskStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tasks, err := s.repo.ListAllTasks(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to list tasks for stats")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to load task stats",
		})
		return
	}

	byState := make(map[string]int, len(tasks))
	for _, task := range tasks {
		byState[task.State]++
	}

	sortedTasks := append([]*ent.Task(nil), tasks...)
	slices.SortStableFunc(sortedTasks, func(a, b *ent.Task) int {
		if cmp := b.CreatedAt.Compare(a.CreatedAt); cmp != 0 {
			return cmp
		}
		if a.ID == b.ID {
			return 0
		}
		if a.ID < b.ID {
			return 1
		}
		return -1
	})
	if len(sortedTasks) > 5 {
		sortedTasks = sortedTasks[:5]
	}

	recent := s.mapTaskViews(sortedTasks)
	s.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"total":   len(tasks),
			"byState": byState,
			"recent":  recent,
		},
	})
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var card store.TaskCard
	if err := json.NewDecoder(r.Body).Decode(&card); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "invalid request body",
		})
		return
	}

	if card.ID == "" || card.DispatchRef == "" || card.Transport == "" {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "missing required fields: id, dispatch_ref, transport",
		})
		return
	}

	id, err := s.repo.CreateTask(ctx, &card)
	if err != nil {
		s.logger.Error().Err(err).Str("task_id", card.ID).Msg("failed to create task")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to create task",
		})
		return
	}

	s.writeJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Data:    map[string]string{"id": id},
	})
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	task, err := s.repo.GetTaskByID(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("task_id", id).Msg("failed to get task")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to get task",
		})
		return
	}

	if task == nil {
		s.writeJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "task not found",
		})
		return
	}

	view := s.mapTaskView(task)
	s.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    view,
	})
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	var card store.TaskCard
	if err := json.NewDecoder(r.Body).Decode(&card); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "invalid request body",
		})
		return
	}

	if err := s.repo.UpdateTask(ctx, id, &card); err != nil {
		if ent.IsNotFound(err) {
			s.writeJSON(w, http.StatusNotFound, APIResponse{
				Success: false,
				Error:   "task not found",
			})
			return
		}
		s.logger.Error().Err(err).Str("task_id", id).Msg("failed to update task")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to update task",
		})
		return
	}

	s.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"id": id},
	})
}

func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	task, err := s.repo.GetTaskByID(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("task_id", id).Msg("failed to get task for cancel")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to cancel task",
		})
		return
	}

	if task == nil {
		s.writeJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "task not found",
		})
		return
	}

	s.writeJSON(w, http.StatusNotImplemented, APIResponse{
		Success: false,
		Error:   "cancel action is not supported by the current task state machine",
	})
}

func (s *Server) handleRetryTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	task, err := s.repo.GetTaskByID(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("task_id", id).Msg("failed to get task for retry")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to retry task",
		})
		return
	}

	if task == nil {
		s.writeJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "task not found",
		})
		return
	}

	if !engine.CanRetry(task.State) {
		s.writeJSON(w, http.StatusConflict, APIResponse{
			Success: false,
			Error:   "task state does not allow retry",
		})
		return
	}

	if err := s.repo.UpdateTaskState(ctx, id, task.State, engine.StateRetryWaiting, "manual_retry", &store.EventData{
		EventID:   uuid.NewString(),
		TaskID:    id,
		EventType: "state_transition",
		FromState: task.State,
		ToState:   engine.StateRetryWaiting,
		Timestamp: time.Now().UTC(),
		Reason:    "manual_retry",
		Attempt:   task.RetryCount + 1,
		Transport: task.Transport,
	}); err != nil {
		s.logger.Error().Err(err).Str("task_id", id).Str("state", task.State).Msg("failed to retry task")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to retry task",
		})
		return
	}

	s.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]string{
			"id":     id,
			"status": engine.StateRetryWaiting,
		},
	})
}

func (s *Server) handleListTasksByDispatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dispatchRef := chi.URLParam(r, "dispatchRef")

	tasks, err := s.repo.ListTasksByDispatchRef(ctx, dispatchRef)
	if err != nil {
		s.logger.Error().Err(err).Str("dispatch_ref", dispatchRef).Msg("failed to list tasks")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to list tasks",
		})
		return
	}

	views := s.mapTaskViews(tasks)
	s.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    views,
	})
}

func (s *Server) handleCreateWave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dispatchRef := chi.URLParam(r, "dispatchRef")

	var req struct {
		Wave int `json:"wave"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "invalid request body: expected {\"wave\": <int>}",
		})
		return
	}

	if err := s.repo.UpsertWave(ctx, dispatchRef, req.Wave); err != nil {
		s.logger.Error().Err(err).
			Str("dispatch_ref", dispatchRef).
			Int("wave", req.Wave).
			Msg("failed to upsert wave")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to create wave",
		})
		return
	}

	s.writeJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Data:    map[string]string{"dispatch_ref": dispatchRef},
	})
}

func (s *Server) handleGetWave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dispatchRef := chi.URLParam(r, "dispatchRef")
	waveNum, ok := s.parseWaveParam(w, r)
	if !ok {
		return
	}

	wave, err := s.repo.GetWave(ctx, dispatchRef, waveNum)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get wave")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to get wave",
		})
		return
	}

	if wave == nil {
		s.writeJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "wave not found",
		})
		return
	}

	s.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    wave,
	})
}

func (s *Server) handleSealWave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dispatchRef := chi.URLParam(r, "dispatchRef")
	waveNum, ok := s.parseWaveParam(w, r)
	if !ok {
		return
	}

	if err := s.repo.SealWave(ctx, dispatchRef, waveNum); err != nil {
		s.logger.Error().Err(err).Msg("failed to seal wave")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to seal wave",
		})
		return
	}

	s.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"status": "sealed"},
	})
}

func (s *Server) mapTaskView(task *ent.Task) *store.TaskView {
	view, err := store.BuildTaskView(task)
	if err != nil {
		s.logger.Warn().Err(err).Str("task_id", task.ID).Msg("failed to rebuild task from card_json; using structured fallback")
	}
	if view != nil {
		return view
	}
	return &store.TaskView{
		ID:                 task.ID,
		DispatchRef:        task.DispatchRef,
		State:              task.State,
		RetryCount:         task.RetryCount,
		LoopIterationCount: task.LoopIterationCount,
		Transport:          task.Transport,
		Wave:               task.Wave,
		TopoRank:           task.TopoRank,
		WorkspacePath:      task.WorkspacePath,
		ArtifactPath:       task.ArtifactPath,
		LastErrorReason:    task.LastErrorReason,
		CreatedAt:          task.CreatedAt,
		UpdatedAt:          task.UpdatedAt,
		TerminalAt:         task.TerminalAt,
		CardJSON:           task.CardJSON,
	}
}

func (s *Server) mapTaskViews(tasks []*ent.Task) []*store.TaskView {
	views := make([]*store.TaskView, 0, len(tasks))
	for _, task := range tasks {
		views = append(views, s.mapTaskView(task))
	}
	return views
}

// parseWaveParam extracts and validates the wave URL parameter.
func (s *Server) parseWaveParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "wave")
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "wave must be a non-negative integer",
		})
		return 0, false
	}
	return n, true
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.logger.Error().Err(err).Msg("failed to encode JSON response")
	}
}
