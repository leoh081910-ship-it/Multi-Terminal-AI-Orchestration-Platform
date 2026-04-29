// Package server provides the HTTP API server for the AI orchestration platform.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
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
	repo     *store.Repository
	logger   zerolog.Logger
	router   *chi.Mux
	projects *compatProjectRegistry

	projectsMu          sync.RWMutex
	projectConfigStore  *ProjectConfigStore
	projectQueueManager *ProjectQueueManager
	webDistDir          string

	// PRD-DA-001 coordination workers
	failureOrchestrator *FailureOrchestrator
	retryWorker         *RetryWorker
	reviewWorker        *ReviewWorker

	// PR-OPS-003 worker status tracking
	autoDispatcherActive  bool
	ttlCleanupActive      bool
	executionReaperActive bool
}

// New creates a new Server instance.
func New(repo *store.Repository, logger zerolog.Logger) *Server {
	s := &Server{
		repo:       repo,
		logger:     logger,
		router:     chi.NewRouter(),
		webDistDir: filepath.FromSlash("web/dist"),
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
	s.router.Use(s.cors)
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
		r.Route("/v1", func(r chi.Router) {
			// PR-OPS-003: System health endpoints
			r.Get("/system/health", s.handleSystemHealth)
			r.Get("/system/workers", s.handleSystemWorkers)

			r.Get("/projects", s.handleCompatListProjects)
			r.Post("/projects", s.handleCompatCreateProject)
			s.registerCompatProjectRoutes(r)
			r.Route("/projects/{projectID}", func(r chi.Router) {
				s.registerCompatProjectRoutes(r)
			})

			// PR-3: Recovery endpoint for stuck tasks
			r.Post("/recovery/stuck-tasks", s.handleRecoverStuckTasks)
		})

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

	s.registerStaticWebRoutes()
}

func (s *Server) registerCompatProjectRoutes(r chi.Router) {
	r.Route("/scheduler", func(r chi.Router) {
		r.Get("/tasks", s.handleCompatListSchedulerTasks)
		r.Post("/tasks", s.handleCompatCreateSchedulerTask)
		r.Patch("/tasks/{id}", s.handleCompatUpdateSchedulerTask)
		r.Post("/tasks/{id}/dispatch", s.handleCompatDispatchSchedulerTask)
		r.Post("/tasks/{id}/retry", s.handleCompatRetrySchedulerTask)
		r.Get("/tasks/{id}/execution", s.handleCompatGetTaskExecution)
		r.Get("/tasks/{id}/lineage", s.handleCompatGetTaskLineage)
		r.Post("/tasks/{id}/triage", s.handleCompatManualTriage)
		r.Get("/executions", s.handleCompatListExecutions)
		r.Get("/runtimes", s.handleCompatListRuntimes)
		r.Get("/failure-policies", s.handleCompatGetFailurePolicies)
		r.Get("/agents", s.handleCompatGetAgents)
	})
	r.Route("/board", func(r chi.Router) {
		r.Get("/summary", s.handleCompatBoardSummary)
	})
	r.Route("/agents", func(r chi.Router) {
		r.Get("/summary", s.handleCompatAgentsSummary)
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

	if engine.IsTerminal(task.State) {
		s.writeJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data: map[string]string{
				"id":     id,
				"status": task.State,
			},
		})
		return
	}

	if !canCancelTaskState(task.State) {
		s.writeJSON(w, http.StatusConflict, APIResponse{
			Success: false,
			Error:   "task state does not allow cancel",
		})
		return
	}

	if err := s.repo.UpdateTaskState(ctx, id, task.State, engine.StateFailed, "cancelled", &store.EventData{
		EventID:   uuid.NewString(),
		TaskID:    id,
		EventType: "state_transition",
		FromState: task.State,
		ToState:   engine.StateFailed,
		Timestamp: time.Now().UTC(),
		Reason:    "cancelled",
		Attempt:   task.RetryCount,
		Transport: task.Transport,
	}); err != nil {
		s.logger.Error().Err(err).Str("task_id", id).Str("state", task.State).Msg("failed to cancel task")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to cancel task",
		})
		return
	}

	s.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]string{
			"id":     id,
			"status": engine.StateFailed,
		},
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
		Attempt:   task.RetryCount,
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

func canCancelTaskState(state string) bool {
	switch state {
	case engine.StateQueued,
		engine.StateRouted,
		engine.StateWorkspacePrepared,
		engine.StateRunning,
		engine.StatePatchReady,
		engine.StateVerified,
		engine.StateRetryWaiting,
		engine.StateVerifyFailed,
		engine.StateApplyFailed:
		return true
	default:
		return false
	}
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

// --- PRD-DA-001 Coordination Worker Management ---

// SetCoordinationWorkers sets the coordination background workers.
func (s *Server) SetCoordinationWorkers(orchestrator *FailureOrchestrator, retryW *RetryWorker, reviewW *ReviewWorker) {
	s.failureOrchestrator = orchestrator
	s.retryWorker = retryW
	s.reviewWorker = reviewW
}

// --- PRD-DA-001 API Handlers ---

// handleCompatGetTaskLineage returns the task lineage (parent/children chain).
// GET /api/v1/projects/{project}/scheduler/tasks/{id}/lineage
func (s *Server) handleCompatGetTaskLineage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	projectID := s.compatProjectIDFromRequest(r)

	task, err := s.repo.GetTaskByID(ctx, id)
	if err != nil || task == nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Task not found"})
		return
	}
	if !s.compatTaskBelongsToProject(task, projectID) {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Task not found"})
		return
	}

	allTasks, err := s.repo.ListAllTasks(ctx)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load tasks"})
		return
	}

	lineage := s.buildTaskLineage(task, allTasks)
	s.writeJSON(w, http.StatusOK, lineage)
}

type compatTaskLineage struct {
	RootTask    *compatSchedulerTask  `json:"root_task"`
	Ancestors   []compatSchedulerTask `json:"ancestors"`
	Descendants []compatSchedulerTask `json:"descendants"`
	Siblings    []compatSchedulerTask `json:"siblings"`
}

func (s *Server) buildTaskLineage(task *ent.Task, allTasks []*ent.Task) compatTaskLineage {
	payload := decodeCompatPayload(task.CardJSON)
	parentTaskID := readString(payload, "parent_task_id")
	rootTaskID := readString(payload, "root_task_id")
	if rootTaskID == "" {
		rootTaskID = task.ID
	}

	// Build a map for quick lookup
	taskMap := make(map[string]*ent.Task)
	taskPayloads := make(map[string]map[string]interface{})
	for _, t := range allTasks {
		taskMap[t.ID] = t
		taskPayloads[t.ID] = decodeCompatPayload(t.CardJSON)
	}

	// Find root task
	var root *compatSchedulerTask
	if rootTask, exists := taskMap[rootTaskID]; exists {
		mapped := s.mapCompatTask(rootTask)
		root = &mapped
	}

	// Recursively collect all ancestors (parent chain)
	ancestors := s.collectAncestors(task.ID, taskMap, taskPayloads)

	// Recursively collect all descendants (children chain)
	descendants := s.collectDescendants(task.ID, taskMap, taskPayloads)

	// Collect siblings (same parent, excluding self)
	var siblings []compatSchedulerTask
	if parentTaskID != "" {
		for _, t := range allTasks {
			tp := taskPayloads[t.ID]
			tParentID := readString(tp, "parent_task_id")
			if tParentID == parentTaskID && t.ID != task.ID {
				siblings = append(siblings, s.mapCompatTask(t))
			}
		}
	}

	if root == nil {
		mapped := s.mapCompatTask(task)
		root = &mapped
	}

	if ancestors == nil {
		ancestors = []compatSchedulerTask{}
	}
	if descendants == nil {
		descendants = []compatSchedulerTask{}
	}
	if siblings == nil {
		siblings = []compatSchedulerTask{}
	}

	return compatTaskLineage{
		RootTask:    root,
		Ancestors:   ancestors,
		Descendants: descendants,
		Siblings:    siblings,
	}
}

// collectAncestors recursively collects all ancestor tasks (parent, grandparent, etc.)
func (s *Server) collectAncestors(taskID string, taskMap map[string]*ent.Task, taskPayloads map[string]map[string]interface{}) []compatSchedulerTask {
	var ancestors []compatSchedulerTask
	currentID := taskID

	for {
		tp, exists := taskPayloads[currentID]
		if !exists {
			break
		}
		parentID := readString(tp, "parent_task_id")
		if parentID == "" {
			break
		}
		if parentTask, exists := taskMap[parentID]; exists {
			ancestors = append(ancestors, s.mapCompatTask(parentTask))
			currentID = parentID
		} else {
			break
		}
	}

	return ancestors
}

// collectDescendants recursively collects all descendant tasks (children, grandchildren, etc.)
func (s *Server) collectDescendants(taskID string, taskMap map[string]*ent.Task, taskPayloads map[string]map[string]interface{}) []compatSchedulerTask {
	var descendants []compatSchedulerTask

	// First, find direct children
	var children []string
	for _, t := range taskMap {
		tp := taskPayloads[t.ID]
		tParentID := readString(tp, "parent_task_id")
		if tParentID == taskID {
			children = append(children, t.ID)
			descendants = append(descendants, s.mapCompatTask(t))
		}
	}

	// Recursively collect grandchildren
	for _, childID := range children {
		grandchildren := s.collectDescendants(childID, taskMap, taskPayloads)
		descendants = append(descendants, grandchildren...)
	}

	return descendants
}

// handleCompatManualTriage manually triggers triage for a task.
// POST /api/v1/projects/{project}/scheduler/tasks/{id}/triage
func (s *Server) handleCompatManualTriage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	projectID := s.compatProjectIDFromRequest(r)

	task, err := s.repo.GetTaskByID(ctx, id)
	if err != nil || task == nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Task not found"})
		return
	}
	if !s.compatTaskBelongsToProject(task, projectID) {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Task not found"})
		return
	}

	// Force task into triage state
	payload := decodeCompatPayload(task.CardJSON)
	reason := firstCompatNonEmpty(readString(payload, "last_dispatch_error"), readString(payload, "last_error_reason"), "manual_triage")

	if task.State == engine.StateRunning || task.State == engine.StateRetryWaiting {
		if err := s.transitionCompatTaskState(ctx, task, engine.StateTriage, "manual_triage", reason); err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to trigger triage"})
			return
		}
		syncCompatPayloadState(payload, engine.StateTriage)
		payload["coordination_stage"] = "triage"
		_ = s.persistCompatPayload(ctx, id, payload)
	}

	updatedTask, _ := s.repo.GetTaskByID(ctx, id)
	s.writeJSON(w, http.StatusOK, s.mapCompatTask(updatedTask))
}

// handleCompatGetFailurePolicies returns the current failure policy configuration.
// GET /api/v1/projects/{project}/scheduler/failure-policies
func (s *Server) handleCompatGetFailurePolicies(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"max_auto_repairs":  MaxAutoRepairs,
		"max_auto_retries":  MaxAutoRetries,
		"backoff_intervals": []string{"30s", "60s"},
		"failure_codes": []map[string]string{
			{"code": "artifact_missing", "description": "Artifact was not generated", "retryable": "true"},
			{"code": "artifact_spec_mismatch", "description": "Artifact path is incorrectly configured", "retryable": "true"},
			{"code": "git_no_changes", "description": "Nothing to commit, working tree clean", "retryable": "true"},
			{"code": "command_exit_nonzero", "description": "Command exited with non-zero status", "retryable": "true"},
			{"code": "workspace_write_failed", "description": "Failed to write to workspace", "retryable": "true"},
			{"code": "dependency_failed", "description": "Dependency task failed", "retryable": "false"},
			{"code": "non_retryable_failure", "description": "Unclassified non-retryable failure", "retryable": "false"},
		},
	})
}

// handleCompatGetAgents returns agent configuration including system roles.
// GET /api/v1/projects/{project}/scheduler/agents
func (s *Server) handleCompatGetAgents(w http.ResponseWriter, r *http.Request) {
	agents := []map[string]interface{}{
		{"agent_id": "Claude", "agent_role": "executor", "capabilities": []string{"can_execute", "can_triage"}, "specialties": []string{"spec", "no-op review", "analysis"}, "enabled": true},
		{"agent_id": "Gemini", "agent_role": "executor", "capabilities": []string{"can_execute"}, "specialties": []string{"UI", "frontend fix", "content remediation"}, "enabled": true},
		{"agent_id": "Codex", "agent_role": "executor", "capabilities": []string{"can_execute", "can_retry"}, "specialties": []string{"code fix", "artifact path fix", "debug"}, "enabled": true},
		{"agent_id": "Coordinator", "agent_role": "system", "capabilities": []string{"can_triage"}, "specialties": []string{"failure triage", "auto-dispatch"}, "enabled": true},
		{"agent_id": "Reviewer", "agent_role": "system", "capabilities": []string{"can_review"}, "specialties": []string{"code review", "rework decision"}, "enabled": true},
	}
	s.writeJSON(w, http.StatusOK, map[string]interface{}{"agents": agents})
}

// handleRecoverStuckTasks handles PR-3: recovery of tasks stuck due to platform defects.
// POST /api/v1/recovery/stuck-tasks
// Body: {"task_ids": ["TS-xxx", ...], "reason": "platform_defect"}
func (s *Server) handleRecoverStuckTasks(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskIDs []string `json:"task_ids"`
		Reason  string   `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid request body"})
		return
	}
	if len(req.TaskIDs) == 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "task_ids required"})
		return
	}
	if req.Reason == "" {
		req.Reason = "platform_defect"
	}

	ctx := r.Context()
	recovered := make([]map[string]string, 0, len(req.TaskIDs))

	for _, taskID := range req.TaskIDs {
		result := s.recoverStuckTask(ctx, taskID, req.Reason)
		recovered = append(recovered, result)
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"recovered": recovered,
		"count":     len(recovered),
	})
}

// recoverStuckTask attempts to recover a single stuck task.
// PR-3: For tasks stuck in blocked/retry_waiting/review_pending due to platform defects:
// 1. Retires old failed sub-tasks (marks as done so they don't count against repair budget)
// 2. Resets the root task to retry_waiting so the retry worker can pick it up
func (s *Server) recoverStuckTask(ctx context.Context, taskID, reason string) map[string]string {
	result := map[string]string{"task_id": taskID}

	task, err := s.repo.GetTaskByID(ctx, taskID)
	if err != nil || task == nil {
		result["status"] = "not_found"
		result["error"] = "task not found"
		return result
	}

	payload := decodeCompatPayload(task.CardJSON)

	switch task.State {
	case engine.StateFailed:
		// Blocked (failed) — retire bad children, reset parent
		retiredCount := s.retireFailedChildren(ctx, taskID)
		// Reset repair budget
		payload["auto_repair_count"] = 0
		payload["coordination_stage"] = "recovery"
		payload["failure_code"] = ""
		payload["failure_signature"] = ""
		payload["last_dispatch_error"] = ""
		payload["last_error_reason"] = ""

		// Transition failed → retry_waiting
		if err := s.transitionCompatTaskState(ctx, task, engine.StateRetryWaiting, "recovery_"+reason, ""); err != nil {
			result["status"] = "transition_failed"
			result["error"] = err.Error()
			return result
		}
		syncCompatPayloadState(payload, engine.StateRetryWaiting)
		payload["dispatch_status"] = "failed"
		payload["status"] = "ready"
		payload["execution_session_id"] = nil
		_ = s.persistCompatPayload(ctx, taskID, payload)

		result["status"] = "recovered"
		result["new_state"] = engine.StateRetryWaiting
		result["children_retired"] = fmt.Sprintf("%d", retiredCount)

	case engine.StateVerifyFailed, engine.StateApplyFailed:
		retiredCount := s.retireFailedChildren(ctx, taskID)
		noopMergeFailure := isNoopMergeFailure(task, payload)
		payload["auto_repair_count"] = 0
		payload["failure_code"] = ""
		payload["failure_signature"] = ""
		payload["last_dispatch_error"] = ""
		payload["last_error_reason"] = ""
		payload["execution_session_id"] = nil

		if noopMergeFailure {
			if err := s.transitionCompatTaskState(ctx, task, engine.StateVerified, "recovery_"+reason, "recover noop merge failure"); err != nil {
				result["status"] = "transition_failed"
				result["error"] = err.Error()
				return result
			}
			syncCompatPayloadState(payload, engine.StateVerified)
			payload["coordination_stage"] = "recovery_approved"
			payload["dispatch_status"] = "completed"
			payload["status"] = "verified"
			payload["review_decision"] = firstCompatNonEmpty(readString(payload, "review_decision"), "approved")
			_ = s.persistCompatPayload(ctx, taskID, payload)

			result["status"] = "recovered"
			result["new_state"] = engine.StateVerified
			result["children_retired"] = fmt.Sprintf("%d", retiredCount)
			break
		}

		if err := s.transitionCompatTaskState(ctx, task, engine.StateRetryWaiting, "recovery_"+reason, "recover verification/apply failure"); err != nil {
			result["status"] = "transition_failed"
			result["error"] = err.Error()
			return result
		}
		syncCompatPayloadState(payload, engine.StateRetryWaiting)
		payload["coordination_stage"] = "recovery"
		payload["dispatch_status"] = "failed"
		payload["status"] = "ready"
		_ = s.persistCompatPayload(ctx, taskID, payload)

		result["status"] = "recovered"
		result["new_state"] = engine.StateRetryWaiting
		result["children_retired"] = fmt.Sprintf("%d", retiredCount)

	case engine.StateRetryWaiting:
		// Already in retry_waiting but may be stuck by failed children
		retiredCount := s.retireFailedChildren(ctx, taskID)
		payload["auto_repair_count"] = 0
		_ = s.persistCompatPayload(ctx, taskID, payload)

		result["status"] = "recovered"
		result["new_state"] = engine.StateRetryWaiting
		result["children_retired"] = fmt.Sprintf("%d", retiredCount)

	case engine.StateReviewPending:
		// Stuck in review — auto-approve
		if err := s.transitionCompatTaskState(ctx, task, engine.StateVerified, "recovery_"+reason, ""); err != nil {
			result["status"] = "transition_failed"
			result["error"] = err.Error()
			return result
		}
		syncCompatPayloadState(payload, engine.StateVerified)
		payload["review_decision"] = "approved"
		payload["coordination_stage"] = "recovery_approved"
		payload["dispatch_status"] = "completed"
		payload["status"] = "verified"
		payload["execution_session_id"] = nil
		_ = s.persistCompatPayload(ctx, taskID, payload)

		result["status"] = "recovered"
		result["new_state"] = engine.StateVerified

	default:
		result["status"] = "not_recoverable"
		result["current_state"] = task.State
	}

	return result
}

func isNoopMergeFailure(task *ent.Task, payload map[string]interface{}) bool {
	lastError := strings.ToLower(strings.TrimSpace(firstCompatNonEmpty(
		task.LastErrorReason,
		readString(payload, "last_dispatch_error"),
		readString(payload, "last_error_reason"),
	)))
	if lastError == "" {
		return false
	}

	return strings.Contains(lastError, "nothing to commit") ||
		strings.Contains(lastError, "working tree clean")
}

// retireFailedChildren marks failed child tasks as done so they no longer
// count against the parent's repair budget.
func (s *Server) retireFailedChildren(ctx context.Context, parentTaskID string) int {
	allTasks, err := s.repo.ListAllTasks(ctx)
	if err != nil {
		return 0
	}

	retired := 0
	for _, t := range allTasks {
		if t.ID == parentTaskID {
			continue
		}
		payload := decodeCompatPayload(t.CardJSON)
		parentID := readString(payload, "parent_task_id")
		if parentID != parentTaskID {
			continue
		}
		// Only retire tasks in failure states
		if t.State == engine.StateFailed || t.State == engine.StateVerifyFailed ||
			t.State == engine.StateApplyFailed || t.State == engine.StateTriage {
			// Transition to done (via verified for valid state path)
			if err := s.transitionCompatTaskState(ctx, t, engine.StateDone, "retired_by_recovery", ""); err != nil {
				s.logger.Warn().Err(err).Str("task_id", t.ID).Msg("failed to retire child task during recovery")
				continue
			}
			payload["dispatch_status"] = "completed"
			payload["status"] = "done"
			payload["coordination_stage"] = "retired"
			_ = s.persistCompatPayload(ctx, t.ID, payload)
			retired++
		}
	}

	return retired
}
