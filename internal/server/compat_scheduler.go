package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/store"
)

const (
	compatAgentClaude = "Claude"
	compatAgentGemini = "Gemini"
	compatAgentCodex  = "Codex"
)

var compatAgents = []string{
	compatAgentClaude,
	compatAgentGemini,
	compatAgentCodex,
}

var compatTaskStatuses = []string{
	"backlog",
	"ready",
	"assigned",
	"in_progress",
	"review",
	"verified",
	"blocked",
	"done",
}

type compatSchedulerTask struct {
	ProjectID           string    `json:"project_id,omitempty"`
	TaskID              string    `json:"task_id"`
	Title               string    `json:"title"`
	OwnerAgent          string    `json:"owner_agent"`
	Status              string    `json:"status"`
	Type                string    `json:"type"`
	Priority            int       `json:"priority"`
	Description         string    `json:"description,omitempty"`
	DependsOn           []string  `json:"depends_on"`
	InputArtifacts      []string  `json:"input_artifacts"`
	OutputArtifacts     []string  `json:"output_artifacts"`
	AcceptanceCriteria  []string  `json:"acceptance_criteria"`
	BlockedReason       string    `json:"blocked_reason,omitempty"`
	ResultSummary       string    `json:"result_summary,omitempty"`
	NextAction          string    `json:"next_action,omitempty"`
	CurrentFocus        string    `json:"current_focus,omitempty"`
	DispatchMode        string    `json:"dispatch_mode"`
	AutoDispatchEnabled bool      `json:"auto_dispatch_enabled"`
	DispatchStatus      string    `json:"dispatch_status"`
	ExecutionRuntime    string    `json:"execution_runtime,omitempty"`
	ExecutionSessionID  string    `json:"execution_session_id,omitempty"`
	Command             string    `json:"command,omitempty"`
	WorkspacePath       string    `json:"workspace_path,omitempty"`
	ArtifactPath        string    `json:"artifact_path,omitempty"`
	LastDispatchAt      string    `json:"last_dispatch_at,omitempty"`
	DispatchAttempts    int       `json:"dispatch_attempts"`
	LastDispatchError   string    `json:"last_dispatch_error,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	// PRD-DA-001 coordination fields
	ParentTaskID       string `json:"parent_task_id,omitempty"`
	RootTaskID         string `json:"root_task_id,omitempty"`
	DerivedFromFailure string `json:"derived_from_failure,omitempty"`
	CoordinationStage  string `json:"coordination_stage,omitempty"`
	ReviewDecision     string `json:"review_decision,omitempty"`
	FailureCode        string `json:"failure_code,omitempty"`
	FailureSignature   string `json:"failure_signature,omitempty"`
	AutoRepairCount    int    `json:"auto_repair_count"`
	LastReviewTaskID   string `json:"last_review_task_id,omitempty"`

	// PR-OPS-003 execution heartbeat fields
	StartedAt       string `json:"started_at,omitempty"`
	LastHeartbeatAt string `json:"last_heartbeat_at,omitempty"`
	TimeoutAt       string `json:"timeout_at,omitempty"`
	Stalled         bool   `json:"stalled"`

	// PR-4: computed diagnostic fields for dashboard display
	NextAutomaticAction string `json:"next_automatic_action,omitempty"`
	StopLossStatus      string `json:"stop_loss_status,omitempty"`
	Recoverable         bool   `json:"recoverable"`
}

type compatTaskExecutionEvent struct {
	Event     string    `json:"event"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type compatTaskExecution struct {
	ExecutionID string                     `json:"execution_id"`
	TaskID      string                     `json:"task_id"`
	OwnerAgent  string                     `json:"owner_agent"`
	Runtime     string                     `json:"runtime"`
	SessionID   string                     `json:"session_id"`
	Status      string                     `json:"status"`
	Command     string                     `json:"command"`
	Workspace   string                     `json:"workspace,omitempty"`
	Artifacts   string                     `json:"artifacts,omitempty"`
	LogPath     string                     `json:"log_path,omitempty"`
	OutputTail  string                     `json:"output_tail,omitempty"`
	StartedAt   time.Time                  `json:"started_at"`
	UpdatedAt   time.Time                  `json:"updated_at"`
	CompletedAt *time.Time                 `json:"completed_at,omitempty"`
	Error       string                     `json:"error,omitempty"`
	Events      []compatTaskExecutionEvent `json:"events"`
}

type compatTaskExecutionList struct {
	Executions []compatTaskExecution `json:"executions"`
}

type compatRuntimeSummary struct {
	OwnerAgent        string     `json:"owner_agent"`
	Runtime           string     `json:"runtime"`
	Status            string     `json:"status"`
	CommandConfigured bool       `json:"command_configured"`
	ActiveSessions    int        `json:"active_sessions"`
	LastHeartbeatAt   *time.Time `json:"last_heartbeat_at,omitempty"`
	LastError         string     `json:"last_error,omitempty"`
}

type compatRuntimeSummaryList struct {
	Runtimes []compatRuntimeSummary `json:"runtimes"`
}

type compatBoardSummary struct {
	TotalTasks      int                   `json:"total_tasks"`
	CountsByStatus  map[string]int        `json:"counts_by_status"`
	CountsByAgent   map[string]int        `json:"counts_by_agent"`
	BlockedCount    int                   `json:"blocked_count"`
	RecentUpdates   []compatSchedulerTask `json:"recent_updates"`
	RecentDoneTasks []compatSchedulerTask `json:"recent_done_tasks"`
	CurrentFocus    []compatSchedulerTask `json:"current_focus"`
}

type compatAgentTaskSlice struct {
	Assigned   []compatSchedulerTask `json:"assigned"`
	InProgress []compatSchedulerTask `json:"in_progress"`
}

type compatAgentSummaryItem struct {
	OwnerAgent     string               `json:"owner_agent"`
	TotalTasks     int                  `json:"total_tasks"`
	CountsByStatus map[string]int       `json:"counts_by_status"`
	Tasks          compatAgentTaskSlice `json:"tasks"`
}

type compatAgentsSummary struct {
	Agents []compatAgentSummaryItem `json:"agents"`
}

func (s *Server) handleCompatListProjects(w http.ResponseWriter, r *http.Request) {
	registry := s.getProjectRegistry()
	if registry == nil {
		s.writeJSON(w, http.StatusOK, []compatProjectSummary{
			{
				ID:      compatDefaultProjectID,
				Name:    compatDefaultProjectID,
				Default: true,
			},
		})
		return
	}
	s.writeJSON(w, http.StatusOK, registry.summaries())
}

func (s *Server) handleCompatListSchedulerTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := s.compatProjectIDFromRequest(r)
	tasks, err := s.listCompatTasks(ctx, projectID)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to list compatibility scheduler tasks")
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list scheduler tasks"})
		return
	}

	statusFilter := r.URL.Query().Get("status")
	ownerFilter := r.URL.Query().Get("owner_agent")
	typeFilter := r.URL.Query().Get("type")

	filtered := make([]compatSchedulerTask, 0, len(tasks))
	for _, task := range tasks {
		if statusFilter != "" && task.Status != statusFilter {
			continue
		}
		if ownerFilter != "" && task.OwnerAgent != ownerFilter {
			continue
		}
		if typeFilter != "" && task.Type != typeFilter {
			continue
		}
		filtered = append(filtered, task)
	}

	s.writeJSON(w, http.StatusOK, filtered)
}

func (s *Server) handleCompatCreateSchedulerTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := s.compatProjectIDFromRequest(r)

	payload, err := decodeCompatRequestMap(r)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid request body"})
		return
	}

	payload["project_id"] = projectID
	card, err := buildCompatTaskCard(payload, nil, "", s.defaultCompatProjectID())
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"detail": err.Error()})
		return
	}

	if _, err := s.repo.CreateTask(ctx, card); err != nil {
		s.logger.Error().Err(err).Str("task_id", card.ID).Msg("failed to create compatibility scheduler task")
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create scheduler task"})
		return
	}

	task, err := s.repo.GetTaskByID(ctx, card.ID)
	if err != nil {
		s.logger.Error().Err(err).Str("task_id", card.ID).Msg("failed to reload compatibility scheduler task")
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load scheduler task"})
		return
	}

	s.writeJSON(w, http.StatusCreated, s.mapCompatTask(task))
}

func (s *Server) handleCompatUpdateSchedulerTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	projectID := s.compatProjectIDFromRequest(r)

	task, err := s.repo.GetTaskByID(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("task_id", id).Msg("failed to load compatibility scheduler task for update")
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update scheduler task"})
		return
	}
	if task == nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Task not found"})
		return
	}
	if !s.compatTaskBelongsToProject(task, projectID) {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Task not found"})
		return
	}

	updates, err := decodeCompatRequestMap(r)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid request body"})
		return
	}

	merged := mergeCompatPayload(decodeCompatPayload(task.CardJSON), updates)
	merged["project_id"] = projectID
	card, err := buildCompatTaskCard(merged, s.mapTaskView(task), id, s.defaultCompatProjectID())
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"detail": err.Error()})
		return
	}

	if err := s.repo.UpdateTask(ctx, id, card); err != nil {
		s.logger.Error().Err(err).Str("task_id", id).Msg("failed to update compatibility scheduler task")
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update scheduler task"})
		return
	}

	updatedTask, err := s.repo.GetTaskByID(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("task_id", id).Msg("failed to reload compatibility scheduler task")
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load scheduler task"})
		return
	}

	s.writeJSON(w, http.StatusOK, s.mapCompatTask(updatedTask))
}

func (s *Server) handleCompatDispatchSchedulerTask(w http.ResponseWriter, r *http.Request) {
	s.handleCompatForceDispatch(w, r, false)
}

func (s *Server) handleCompatRetrySchedulerTask(w http.ResponseWriter, r *http.Request) {
	s.handleCompatForceDispatch(w, r, true)
}

func (s *Server) handleCompatForceDispatch(w http.ResponseWriter, r *http.Request, isRetry bool) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	projectID := s.compatProjectIDFromRequest(r)

	updatedTask, err := s.dispatchCompatTask(ctx, id, isRetry, projectID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err == errCompatTaskNotFound {
			statusCode = http.StatusNotFound
		}
		s.logger.Error().Err(err).Str("task_id", id).Msg("failed to dispatch compatibility scheduler task")
		s.writeJSON(w, statusCode, map[string]string{"detail": compatDispatchErrorDetail(err)})
		return
	}

	s.writeJSON(w, http.StatusOK, s.mapCompatTask(updatedTask))
}

func (s *Server) handleCompatGetTaskExecution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	projectID := s.compatProjectIDFromRequest(r)

	task, err := s.repo.GetTaskByID(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("task_id", id).Msg("failed to load task execution")
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load task execution"})
		return
	}
	if task == nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Execution not found"})
		return
	}
	if !s.compatTaskBelongsToProject(task, projectID) {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Execution not found"})
		return
	}

	compatTask := s.mapCompatTask(task)
	execution, ok := s.buildCompatExecution(compatTask, true)
	if !ok {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Execution not found"})
		return
	}

	s.writeJSON(w, http.StatusOK, execution)
}

func (s *Server) handleCompatListExecutions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := s.compatProjectIDFromRequest(r)
	tasks, err := s.listCompatTasks(ctx, projectID)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to list compatibility executions")
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list executions"})
		return
	}

	executions := make([]compatTaskExecution, 0, len(tasks))
	for _, task := range tasks {
		if execution, ok := s.buildCompatExecution(task, false); ok {
			executions = append(executions, execution)
		}
	}

	slices.SortStableFunc(executions, func(a, b compatTaskExecution) int {
		if cmp := b.UpdatedAt.Compare(a.UpdatedAt); cmp != 0 {
			return cmp
		}
		return strings.Compare(b.TaskID, a.TaskID)
	})

	s.writeJSON(w, http.StatusOK, compatTaskExecutionList{Executions: executions})
}

func (s *Server) handleCompatListRuntimes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := s.compatProjectIDFromRequest(r)
	tasks, err := s.listCompatTasks(ctx, projectID)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to list compatibility runtimes")
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list runtimes"})
		return
	}

	s.writeJSON(w, http.StatusOK, compatRuntimeSummaryList{Runtimes: s.buildCompatRuntimes(tasks, projectID)})
}

func (s *Server) handleCompatBoardSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := s.compatProjectIDFromRequest(r)
	tasks, err := s.listCompatTasks(ctx, projectID)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to build compatibility board summary")
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to build board summary"})
		return
	}

	countsByStatus := make(map[string]int, len(compatTaskStatuses))
	for _, status := range compatTaskStatuses {
		countsByStatus[status] = 0
	}

	countsByAgent := make(map[string]int, len(compatAgents))
	for _, agent := range compatAgents {
		countsByAgent[agent] = 0
	}

	blockedCount := 0
	recentDone := make([]compatSchedulerTask, 0, 5)
	currentFocus := make([]compatSchedulerTask, 0, 5)
	for _, task := range tasks {
		countsByStatus[task.Status]++
		countsByAgent[task.OwnerAgent]++
		if task.Status == "blocked" {
			blockedCount++
		}
		if task.Status == "done" && len(recentDone) < 5 {
			recentDone = append(recentDone, task)
		}
		if (task.Status == "assigned" || task.Status == "in_progress") && len(currentFocus) < 5 {
			currentFocus = append(currentFocus, task)
		}
	}

	recentUpdates := append([]compatSchedulerTask(nil), tasks...)
	if len(recentUpdates) > 5 {
		recentUpdates = recentUpdates[:5]
	}

	s.writeJSON(w, http.StatusOK, compatBoardSummary{
		TotalTasks:      len(tasks),
		CountsByStatus:  countsByStatus,
		CountsByAgent:   countsByAgent,
		BlockedCount:    blockedCount,
		RecentUpdates:   recentUpdates,
		RecentDoneTasks: recentDone,
		CurrentFocus:    currentFocus,
	})
}

func (s *Server) handleCompatAgentsSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := s.compatProjectIDFromRequest(r)
	tasks, err := s.listCompatTasks(ctx, projectID)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to build compatibility agents summary")
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to build agents summary"})
		return
	}

	items := make([]compatAgentSummaryItem, 0, len(compatAgents))
	for _, agent := range compatAgents {
		countsByStatus := make(map[string]int, len(compatTaskStatuses))
		for _, status := range compatTaskStatuses {
			countsByStatus[status] = 0
		}

		assigned := make([]compatSchedulerTask, 0)
		inProgress := make([]compatSchedulerTask, 0)
		totalTasks := 0
		for _, task := range tasks {
			if task.OwnerAgent != agent {
				continue
			}
			totalTasks++
			countsByStatus[task.Status]++
			if task.Status == "assigned" {
				assigned = append(assigned, task)
			}
			if task.Status == "in_progress" {
				inProgress = append(inProgress, task)
			}
		}

		items = append(items, compatAgentSummaryItem{
			OwnerAgent:     agent,
			TotalTasks:     totalTasks,
			CountsByStatus: countsByStatus,
			Tasks: compatAgentTaskSlice{
				Assigned:   assigned,
				InProgress: inProgress,
			},
		})
	}

	s.writeJSON(w, http.StatusOK, compatAgentsSummary{Agents: items})
}

func (s *Server) listCompatTasks(ctx context.Context, projectID string) ([]compatSchedulerTask, error) {
	entTasks, err := s.repo.ListAllTasks(ctx)
	if err != nil {
		return nil, err
	}

	tasks := make([]compatSchedulerTask, 0, len(entTasks))
	resolvedProjectID := normalizeCompatProjectID(firstCompatNonEmpty(projectID, s.defaultCompatProjectID()))
	for _, task := range entTasks {
		if normalizeCompatProjectID(firstCompatNonEmpty(task.ProjectID, s.defaultCompatProjectID())) != resolvedProjectID {
			continue
		}
		tasks = append(tasks, s.mapCompatTask(task))
	}

	slices.SortStableFunc(tasks, func(a, b compatSchedulerTask) int {
		if cmp := b.UpdatedAt.Compare(a.UpdatedAt); cmp != 0 {
			return cmp
		}
		return strings.Compare(b.TaskID, a.TaskID)
	})

	return tasks, nil
}

func (s *Server) mapCompatTask(task *ent.Task) compatSchedulerTask {
	view := s.mapTaskView(task)
	payload := decodeCompatPayload(view.CardJSON)

	title := firstCompatNonEmpty(
		readString(payload, "title"),
		readString(payload, "objective"),
		view.ID,
	)
	projectID := normalizeCompatProjectID(firstCompatNonEmpty(
		readString(payload, "project_id"),
		view.ProjectID,
		s.defaultCompatProjectID(),
	))
	ownerAgent := normalizeCompatAgent(firstCompatNonEmpty(
		readString(payload, "owner_agent"),
		readString(payload, "owner"),
		view.Transport,
	))
	status := firstCompatNonEmpty(
		readString(payload, "status"),
		mapCompatWorkflowStatus(view.State),
	)
	dispatchStatus := firstCompatNonEmpty(
		readString(payload, "dispatch_status"),
		mapCompatDispatchStatus(view.State),
	)
	taskType := firstCompatNonEmpty(
		readString(payload, "type"),
		"task",
	)
	description := firstCompatNonEmpty(
		readString(payload, "description"),
		readString(payload, "objective"),
	)
	dependsOn := readStringSlice(payload, "depends_on")
	if len(dependsOn) == 0 {
		dependsOn = readDependsOnRelations(payload)
	}
	dispatchMode := firstCompatNonEmpty(
		readString(payload, "dispatch_mode"),
		"manual",
	)
	autoDispatchEnabled, ok := readBool(payload, "auto_dispatch_enabled")
	if !ok {
		autoDispatchEnabled = dispatchMode == "auto"
	}

	executionRuntime := firstCompatNonEmpty(
		readString(payload, "execution_runtime"),
		readString(payload, "runtime"),
	)
	if executionRuntime == "" && dispatchStatus != "pending" {
		executionRuntime = inferRuntimeFromAgent(ownerAgent, dispatchStatus)
	}
	executionSessionID := firstCompatNonEmpty(
		readString(payload, "execution_session_id"),
		readString(payload, "session_id"),
	)
	lastDispatchAt := readString(payload, "last_dispatch_at")
	if lastDispatchAt == "" && dispatchStatus != "pending" {
		lastDispatchAt = view.UpdatedAt.UTC().Format(time.RFC3339)
	}
	dispatchAttempts, ok := readInt(payload, "dispatch_attempts")
	if !ok {
		dispatchAttempts = view.RetryCount
	}

	return compatSchedulerTask{
		ProjectID:           projectID,
		TaskID:              view.ID,
		Title:               title,
		OwnerAgent:          ownerAgent,
		Status:              status,
		Type:                taskType,
		Priority:            readIntDefault(payload, 3, "priority"),
		Description:         description,
		DependsOn:           compatStringSliceValue(dependsOn),
		InputArtifacts:      compatStringSliceField(payload, "input_artifacts"),
		OutputArtifacts:     compatStringSliceField(payload, "output_artifacts"),
		AcceptanceCriteria:  compatStringSliceField(payload, "acceptance_criteria"),
		BlockedReason:       firstCompatNonEmpty(readString(payload, "blocked_reason"), readString(payload, "block_reason")),
		ResultSummary:       readString(payload, "result_summary"),
		NextAction:          readString(payload, "next_action"),
		CurrentFocus:        firstCompatNonEmpty(readString(payload, "current_focus"), compatCurrentFocusFallback(title, status)),
		DispatchMode:        dispatchMode,
		AutoDispatchEnabled: autoDispatchEnabled,
		DispatchStatus:      dispatchStatus,
		ExecutionRuntime:    executionRuntime,
		ExecutionSessionID:  executionSessionID,
		Command:             readString(payload, "command"),
		WorkspacePath:       firstCompatNonEmpty(readString(payload, "workspace_path"), view.WorkspacePath),
		ArtifactPath:        firstCompatNonEmpty(readString(payload, "artifact_path"), view.ArtifactPath),
		LastDispatchAt:      lastDispatchAt,
		DispatchAttempts:    dispatchAttempts,
		LastDispatchError:   firstCompatNonEmpty(readString(payload, "last_dispatch_error"), view.LastErrorReason),
		CreatedAt:           view.CreatedAt.UTC(),
		UpdatedAt:           view.UpdatedAt.UTC(),

		// PRD-DA-001 coordination fields
		ParentTaskID:       readString(payload, "parent_task_id"),
		RootTaskID:         readString(payload, "root_task_id"),
		DerivedFromFailure: readString(payload, "derived_from_failure"),
		CoordinationStage:  readString(payload, "coordination_stage"),
		ReviewDecision:     readString(payload, "review_decision"),
		FailureCode:        readString(payload, "failure_code"),
		FailureSignature:   readString(payload, "failure_signature"),
		AutoRepairCount:    readIntDefault(payload, 0, "auto_repair_count"),
		LastReviewTaskID:   readString(payload, "last_review_task_id"),

		// PR-OPS-003 execution heartbeat fields
		StartedAt:       readString(payload, "started_at"),
		LastHeartbeatAt: readString(payload, "last_heartbeat_at"),
		TimeoutAt:       readString(payload, "timeout_at"),
		Stalled:         stalledFlag(dispatchStatus, readString(payload, "last_heartbeat_at"), view.UpdatedAt),

		// PR-4: computed diagnostic fields
		NextAutomaticAction: computeNextAutomaticAction(task, payload, dispatchStatus),
		StopLossStatus:      computeStopLossStatus(payload, task),
		Recoverable:         isTaskRecoverable(task, payload),
	}
}

func (s *Server) buildCompatExecution(task compatSchedulerTask, includeOutput bool) (compatTaskExecution, bool) {
	if task.DispatchStatus == "pending" {
		return compatTaskExecution{}, false
	}

	startedAt := parseCompatTime(task.LastDispatchAt, task.UpdatedAt)
	updatedAt := task.UpdatedAt
	var completedAt *time.Time
	if task.DispatchStatus == "completed" || task.DispatchStatus == "failed" {
		completedAt = &updatedAt
	}

	eventTime := startedAt
	if task.DispatchStatus == "completed" || task.DispatchStatus == "failed" {
		eventTime = updatedAt
	}

	eventMessage := task.CurrentFocus
	if eventMessage == "" {
		eventMessage = defaultCompatExecutionMessage(task)
	}

	logPath := ""
	outputTail := ""
	executionManager := s.compatExecutionForProject(task.ProjectID)
	if executionManager != nil {
		logPath = executionManager.logPath(task.TaskID)
		if includeOutput {
			outputTail = executionManager.readLogTail(task.TaskID, 48*1024)
		}
	}

	return compatTaskExecution{
		ExecutionID: "EX-" + task.TaskID,
		TaskID:      task.TaskID,
		OwnerAgent:  task.OwnerAgent,
		Runtime:     firstCompatNonEmpty(task.ExecutionRuntime, inferRuntimeFromAgent(task.OwnerAgent, task.DispatchStatus)),
		SessionID:   firstCompatNonEmpty(task.ExecutionSessionID, "SE-"+task.TaskID),
		Status:      task.DispatchStatus,
		Command:     task.Command,
		Workspace:   task.WorkspacePath,
		Artifacts:   task.ArtifactPath,
		LogPath:     logPath,
		OutputTail:  outputTail,
		StartedAt:   startedAt,
		UpdatedAt:   updatedAt,
		CompletedAt: completedAt,
		Error:       task.LastDispatchError,
		Events: []compatTaskExecutionEvent{
			{
				Event:     task.DispatchStatus,
				Status:    task.DispatchStatus,
				Message:   eventMessage,
				Timestamp: eventTime,
			},
		},
	}, true
}

func (s *Server) buildCompatRuntimes(tasks []compatSchedulerTask, projectID string) []compatRuntimeSummary {
	runtimes := make([]compatRuntimeSummary, 0, len(compatAgents))
	projectExecution := s.compatExecutionForProject(projectID)
	for _, agent := range compatAgents {
		activeSessions := 0
		var lastHeartbeat *time.Time
		lastError := ""

		for _, task := range tasks {
			if task.OwnerAgent != agent {
				continue
			}
			if task.DispatchStatus == "running" {
				activeSessions++
				ts := task.UpdatedAt.UTC()
				if lastHeartbeat == nil || ts.After(*lastHeartbeat) {
					lastHeartbeat = &ts
				}
			}
			if task.LastDispatchError != "" && lastError == "" {
				lastError = task.LastDispatchError
			}
		}

		commandConfigured := projectExecution != nil && projectExecution.commandConfigured(agent)
		status := "unavailable"
		if commandConfigured || activeSessions > 0 {
			status = "available"
		}

		runtimes = append(runtimes, compatRuntimeSummary{
			OwnerAgent:        agent,
			Runtime:           inferRuntimeFromAgent(agent, ""),
			Status:            status,
			CommandConfigured: commandConfigured,
			ActiveSessions:    activeSessions,
			LastHeartbeatAt:   lastHeartbeat,
			LastError:         lastError,
		})
	}

	return runtimes
}

func decodeCompatPayload(cardJSON string) map[string]interface{} {
	if strings.TrimSpace(cardJSON) == "" {
		return nil
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(cardJSON), &payload); err != nil {
		return nil
	}
	return payload
}

func decodeCompatRequestMap(r *http.Request) (map[string]interface{}, error) {
	defer r.Body.Close()

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}
	return payload, nil
}

func mergeCompatPayload(base map[string]interface{}, updates map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{}, len(base)+len(updates))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range updates {
		merged[key] = value
	}
	return merged
}

func buildCompatTaskCard(payload map[string]interface{}, existing *store.TaskView, forcedTaskID string, defaultProjectID string) (*store.TaskCard, error) {
	normalized := mergeCompatPayload(payload, map[string]interface{}{})

	taskID := firstCompatNonEmpty(
		forcedTaskID,
		readString(normalized, "task_id"),
		readString(normalized, "id"),
		compatExistingString(existing, func(view *store.TaskView) string { return view.ID }),
	)
	if taskID == "" {
		taskID = "TS-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))[:8]
	}

	title := firstCompatNonEmpty(readString(normalized, "title"), compatExistingString(existing, compatExistingTitle))
	if title == "" {
		return nil, errCompatFieldRequired("title")
	}

	taskType := firstCompatNonEmpty(readString(normalized, "type"), compatExistingPayloadString(existing, "type"), "task")
	ownerAgent := firstCompatNonEmpty(readString(normalized, "owner_agent"), compatExistingPayloadString(existing, "owner_agent"))
	if ownerAgent == "" {
		return nil, errCompatFieldRequired("owner_agent")
	}
	ownerAgent = normalizeCompatAgent(ownerAgent)

	status := firstCompatNonEmpty(readString(normalized, "status"), compatExistingPayloadString(existing, "status"), "backlog")
	dispatchMode := firstCompatNonEmpty(readString(normalized, "dispatch_mode"), compatExistingPayloadString(existing, "dispatch_mode"), "auto")
	autoDispatchEnabled, ok := readBool(normalized, "auto_dispatch_enabled")
	if !ok {
		autoDispatchEnabled = dispatchMode == "auto"
	}
	dispatchStatus := firstCompatNonEmpty(
		readString(normalized, "dispatch_status"),
		compatExistingPayloadString(existing, "dispatch_status"),
		compatDefaultDispatchStatus(status),
	)
	priority := readIntDefault(normalized, compatExistingInt(existing, 3, "priority"), "priority")
	if priority <= 0 {
		priority = 3
	}
	projectID := normalizeCompatProjectID(firstCompatNonEmpty(
		readString(normalized, "project_id"),
		compatExistingPayloadString(existing, "project_id"),
		defaultProjectID,
	))
	if projectID == "" {
		projectID = compatDefaultProjectID
	}

	dispatchRef := firstCompatNonEmpty(
		readString(normalized, "dispatch_ref"),
		compatExistingString(existing, func(view *store.TaskView) string { return view.DispatchRef }),
		projectID,
	)
	transport := firstCompatNonEmpty(
		readString(normalized, "transport"),
		compatExistingString(existing, func(view *store.TaskView) string { return view.Transport }),
		"cli",
	)
	retryCount := readIntDefault(normalized, compatExistingInt(existing, 0, "retry_count"), "retry_count")
	if retryCount < 0 {
		retryCount = 0
	}
	dispatchAttempts := readIntDefault(normalized, 0, "dispatch_attempts")
	if dispatchAttempts < 0 {
		dispatchAttempts = 0
	}
	loopIterationCount := readIntDefault(normalized, compatExistingInt(existing, 0, "loop_iteration_count"), "loop_iteration_count")
	wave := readIntDefault(normalized, compatExistingIntValue(existing, func(view *store.TaskView) int { return view.Wave }, 1), "wave")
	if wave == 0 {
		wave = 1
	}
	topoRank := readIntDefault(normalized, compatExistingIntValue(existing, func(view *store.TaskView) int { return view.TopoRank }, 0), "topo_rank")
	workspacePath := firstCompatNonEmpty(readString(normalized, "workspace_path"), compatExistingString(existing, func(view *store.TaskView) string { return view.WorkspacePath }))
	artifactPath := firstCompatNonEmpty(readString(normalized, "artifact_path"), compatExistingString(existing, func(view *store.TaskView) string { return view.ArtifactPath }))
	lastDispatchError := compatExistingString(existing, func(view *store.TaskView) string { return view.LastErrorReason })
	if value, ok := readOptionalString(normalized, "last_dispatch_error"); ok {
		lastDispatchError = value
	}
	if value, ok := readOptionalString(normalized, "last_error_reason"); ok {
		lastDispatchError = value
	}

	normalized["task_id"] = taskID
	normalized["id"] = taskID
	normalized["title"] = title
	normalized["project_id"] = projectID
	normalized["type"] = taskType
	normalized["owner_agent"] = ownerAgent
	normalized["priority"] = priority
	normalized["status"] = status
	normalized["depends_on"] = compatStringSliceField(normalized, "depends_on")
	normalized["input_artifacts"] = compatStringSliceField(normalized, "input_artifacts")
	normalized["output_artifacts"] = compatStringSliceField(normalized, "output_artifacts")
	normalized["files_to_modify"] = compatFilesToModify(normalized)
	normalized["acceptance_criteria"] = compatStringSliceField(normalized, "acceptance_criteria")
	normalized["dispatch_mode"] = dispatchMode
	normalized["auto_dispatch_enabled"] = autoDispatchEnabled
	normalized["dispatch_status"] = dispatchStatus
	normalized["dispatch_attempts"] = dispatchAttempts
	normalized["dispatch_ref"] = dispatchRef
	normalized["transport"] = transport
	normalized["retry_count"] = retryCount
	normalized["loop_iteration_count"] = loopIterationCount
	normalized["wave"] = wave
	normalized["topo_rank"] = topoRank
	normalized["workspace_path"] = workspacePath
	normalized["artifact_path"] = artifactPath
	normalized["last_error_reason"] = lastDispatchError

	internalState := compatWorkflowState(status, dispatchStatus)
	normalized["state"] = internalState

	cardJSONBytes, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}

	return &store.TaskCard{
		ProjectID:          projectID,
		ID:                 taskID,
		DispatchRef:        dispatchRef,
		State:              internalState,
		RetryCount:         retryCount,
		LoopIterationCount: loopIterationCount,
		Transport:          transport,
		Wave:               wave,
		TopoRank:           topoRank,
		WorkspacePath:      workspacePath,
		ArtifactPath:       artifactPath,
		LastErrorReason:    lastDispatchError,
		CardJSON:           string(cardJSONBytes),
	}, nil
}

func readString(payload map[string]interface{}, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func readOptionalString(payload map[string]interface{}, key string) (string, bool) {
	if payload == nil {
		return "", false
	}
	value, ok := payload[key]
	if !ok {
		return "", false
	}
	if value == nil {
		return "", true
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v), true
	default:
		return "", true
	}
}

func readStringSlice(payload map[string]interface{}, key string) []string {
	if payload == nil {
		return nil
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return nil
	}

	switch items := value.(type) {
	case []string:
		result := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item != "" {
				result = append(result, item)
			}
		}
		return result
	case []interface{}:
		result := make([]string, 0, len(items))
		for _, item := range items {
			text, ok := item.(string)
			if !ok {
				continue
			}
			text = strings.TrimSpace(text)
			if text != "" {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}

func compatStringSliceField(payload map[string]interface{}, key string) []string {
	return compatStringSliceValue(readStringSlice(payload, key))
}

func compatStringSliceValue(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return values
}

func readDependsOnRelations(payload map[string]interface{}) []string {
	if payload == nil {
		return nil
	}
	value, ok := payload["relations"]
	if !ok || value == nil {
		return nil
	}

	items, ok := value.([]interface{})
	if !ok {
		return nil
	}

	result := make([]string, 0, len(items))
	for _, item := range items {
		relation, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if readString(relation, "type") != "depends_on" {
			continue
		}
		taskID := readString(relation, "task_id")
		if taskID != "" {
			result = append(result, taskID)
		}
	}
	return result
}

func readBool(payload map[string]interface{}, key string) (bool, bool) {
	if payload == nil {
		return false, false
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return false, false
	}
	result, ok := value.(bool)
	return result, ok
}

func readInt(payload map[string]interface{}, key string) (int, bool) {
	if payload == nil {
		return 0, false
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return 0, false
	}

	switch v := value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	default:
		return 0, false
	}
}

func readIntDefault(payload map[string]interface{}, fallback int, key string) int {
	if value, ok := readInt(payload, key); ok {
		return value
	}
	return fallback
}

func compatWorkflowState(status, dispatchStatus string) string {
	switch status {
	case "assigned":
		if dispatchStatus == "pending" {
			return engine.StateRouted
		}
		return engine.StateRouted
	case "in_progress":
		if dispatchStatus == "dispatched" {
			return engine.StateWorkspacePrepared
		}
		return engine.StateRunning
	case "review":
		return engine.StatePatchReady
	case "verified":
		return engine.StateVerified
	case "done":
		return engine.StateDone
	case "blocked":
		if dispatchStatus == "failed" {
			return engine.StateVerifyFailed
		}
		return engine.StateFailed
	case "ready":
		if dispatchStatus == "failed" {
			return engine.StateRetryWaiting
		}
		if dispatchStatus == "queued" {
			return engine.StateRouted
		}
		return engine.StateQueued
	case "backlog":
		return engine.StateQueued
	case "triage":
		return engine.StateTriage
	case "review_pending":
		return engine.StateReviewPending
	}

	switch dispatchStatus {
	case "queued":
		return engine.StateRouted
	case "dispatched":
		return engine.StateWorkspacePrepared
	case "running":
		return engine.StateRunning
	case "completed":
		return engine.StateDone
	case "failed":
		return engine.StateVerifyFailed
	case "triage":
		return engine.StateTriage
	case "review_pending":
		return engine.StateReviewPending
	default:
		return engine.StateQueued
	}
}

func compatDefaultDispatchStatus(status string) string {
	switch status {
	case "assigned":
		return "queued"
	case "in_progress":
		return "running"
	case "review":
		return "completed"
	case "verified":
		return "completed"
	case "done":
		return "completed"
	case "blocked":
		return "failed"
	default:
		return "pending"
	}
}

func compatExistingPayload(existing *store.TaskView) map[string]interface{} {
	if existing == nil {
		return nil
	}
	return decodeCompatPayload(existing.CardJSON)
}

func compatExistingPayloadString(existing *store.TaskView, key string) string {
	return readString(compatExistingPayload(existing), key)
}

func compatExistingString(existing *store.TaskView, getter func(*store.TaskView) string) string {
	if existing == nil {
		return ""
	}
	return getter(existing)
}

func compatExistingInt(existing *store.TaskView, fallback int, key string) int {
	if existing == nil {
		return fallback
	}
	if value, ok := readInt(compatExistingPayload(existing), key); ok {
		return value
	}
	return fallback
}

func compatExistingIntValue(existing *store.TaskView, getter func(*store.TaskView) int, fallback int) int {
	if existing == nil {
		return fallback
	}
	return getter(existing)
}

func compatExistingTitle(existing *store.TaskView) string {
	if existing == nil {
		return ""
	}
	return firstCompatNonEmpty(
		readString(compatExistingPayload(existing), "title"),
		readString(compatExistingPayload(existing), "objective"),
		existing.ID,
	)
}

func errCompatFieldRequired(field string) error {
	return &compatValidationError{field: field}
}

type compatValidationError struct {
	field string
}

func (e *compatValidationError) Error() string {
	return e.field + " is required"
}

func mapCompatWorkflowStatus(state string) string {
	switch state {
	case engine.StateQueued, engine.StateRetryWaiting:
		return "ready"
	case engine.StateRouted:
		return "assigned"
	case engine.StateWorkspacePrepared, engine.StateRunning:
		return "in_progress"
	case engine.StatePatchReady:
		return "review"
	case engine.StateVerified:
		return "verified"
	case engine.StateMerged, engine.StateDone:
		return "done"
	case engine.StateVerifyFailed, engine.StateApplyFailed, engine.StateFailed:
		return "blocked"
	case engine.StateTriage:
		return "triage"
	case engine.StateReviewPending:
		return "review_pending"
	default:
		return "backlog"
	}
}

func mapCompatDispatchStatus(state string) string {
	switch state {
	case engine.StateQueued:
		return "pending"
	case engine.StateRouted:
		return "queued"
	case engine.StateWorkspacePrepared:
		return "dispatched"
	case engine.StateRunning:
		return "running"
	case engine.StatePatchReady, engine.StateVerified, engine.StateMerged, engine.StateDone:
		return "completed"
	case engine.StateRetryWaiting, engine.StateVerifyFailed, engine.StateApplyFailed, engine.StateFailed:
		return "failed"
	case engine.StateTriage:
		return "triage"
	case engine.StateReviewPending:
		return "review_pending"
	default:
		return "pending"
	}
}

func normalizeCompatAgent(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "claude", "claude-code", "claude_code", "cli":
		return compatAgentClaude
	case "gemini":
		return compatAgentGemini
	case "codex":
		return compatAgentCodex
	default:
		return compatAgentClaude
	}
}

func inferRuntimeFromAgent(agent, dispatchStatus string) string {
	switch normalizeCompatAgent(agent) {
	case compatAgentGemini:
		return "gemini"
	case compatAgentCodex:
		return "codex"
	default:
		if dispatchStatus == "" {
			return "claude"
		}
		return "claude"
	}
}

func compatCurrentFocusFallback(title, status string) string {
	if title == "" {
		return ""
	}
	if status == "assigned" || status == "in_progress" || status == "review" {
		return title
	}
	return ""
}

func defaultCompatExecutionMessage(task compatSchedulerTask) string {
	switch task.DispatchStatus {
	case "running":
		return "Task is running"
	case "failed":
		return firstCompatNonEmpty(task.LastDispatchError, "Task execution failed")
	case "completed":
		return "Task execution completed"
	case "queued":
		return "Task is queued"
	case "dispatched":
		return "Task dispatched"
	default:
		return "Task execution state recorded"
	}
}

func parseCompatTime(raw string, fallback time.Time) time.Time {
	if raw == "" {
		return fallback.UTC()
	}
	ts, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return fallback.UTC()
	}
	return ts.UTC()
}

func firstCompatNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (s *Server) defaultCompatProjectID() string {
	registry := s.getProjectRegistry()
	if registry == nil {
		return compatDefaultProjectID
	}
	return registry.defaultProject()
}

func (s *Server) compatProjectIDFromRequest(r *http.Request) string {
	return normalizeCompatProjectID(firstCompatNonEmpty(
		chi.URLParam(r, "projectID"),
		r.URL.Query().Get("project"),
		s.defaultCompatProjectID(),
	))
}

func (s *Server) compatTaskBelongsToProject(task *ent.Task, projectID string) bool {
	if task == nil {
		return false
	}
	return normalizeCompatProjectID(firstCompatNonEmpty(task.ProjectID, s.defaultCompatProjectID())) ==
		normalizeCompatProjectID(firstCompatNonEmpty(projectID, s.defaultCompatProjectID()))
}

func (s *Server) compatExecutionForProject(projectID string) *compatExecutionManager {
	registry := s.getProjectRegistry()
	if registry == nil {
		return nil
	}
	entry, err := registry.resolve(projectID)
	if err != nil {
		return nil
	}
	return entry.execution
}

func compatProjectIDFromCardJSON(cardJSON, defaultProjectID string) string {
	return normalizeCompatProjectID(firstCompatNonEmpty(
		readString(decodeCompatPayload(cardJSON), "project_id"),
		defaultProjectID,
	))
}

// stalledFlag determines if a running task appears stalled based on heartbeat timing.
// PR-OPS-003: running tasks without recent heartbeat are considered stalled.
func stalledFlag(dispatchStatus, lastHeartbeat string, updatedAt time.Time) bool {
	if dispatchStatus != "running" {
		return false
	}
	// If no heartbeat recorded, use updated_at as proxy
	lastActivity := updatedAt
	if lastHeartbeat != "" {
		parsed, err := time.Parse(time.RFC3339, lastHeartbeat)
		if err == nil {
			lastActivity = parsed
		}
	}
	// Stale threshold: 5 minutes without activity while running
	return time.Since(lastActivity) > 5*time.Minute
}

// computeNextAutomaticAction returns a human-readable description of what the
// platform will automatically do next for this task.
// PR-4: Used by the dashboard to show "what happens next" for stuck tasks.
func computeNextAutomaticAction(task *ent.Task, payload map[string]interface{}, dispatchStatus string) string {
	if payload == nil {
		return ""
	}
	taskType := readString(payload, "type")

	switch task.State {
	case engine.StateQueued:
		if readBoolDefault(payload, true, "auto_dispatch_enabled") {
			return "auto_dispatcher will pick up when agent is available"
		}
		return "waiting for manual dispatch"
	case engine.StateRunning:
		return "execution in progress - heartbeat monitoring active"
	case engine.StateTriage:
		return "failure_orchestrator will classify and create remediation"
	case engine.StateRetryWaiting:
		if IsSystemTaskType(taskType) {
			return "system task - will be auto-dispatched when agent available"
		}
		return "retry_worker will re-dispatch after backoff (30-60s)"
	case engine.StateReviewPending:
		return "review_worker will create review task"
	case engine.StateVerified:
		return "merge_queue will merge and complete"
	case engine.StateFailed:
		return "recoverable via POST /api/v1/recovery/stuck-tasks"
	case engine.StateDone:
		return ""
	case engine.StatePatchReady:
		return "transitioning to verified"
	default:
		return ""
	}
}

// computeStopLossStatus returns a human-readable stop-loss status string.
// PR-4: Shows repair budget and retry budget usage.
func computeStopLossStatus(payload map[string]interface{}, task *ent.Task) string {
	if payload == nil {
		return ""
	}
	repairCount := readIntDefault(payload, 0, "auto_repair_count")
	retryCount := task.RetryCount

	parts := make([]string, 0, 2)
	if repairCount > 0 {
		parts = append(parts, fmt.Sprintf("repair %d/%d", repairCount, MaxAutoRepairs))
	}
	if retryCount > 0 {
		parts = append(parts, fmt.Sprintf("retry %d/%d", retryCount, MaxAutoRetries))
	}

	if repairCount >= MaxAutoRepairs {
		parts = append([]string{"REPAIR_LIMIT_HIT"}, parts...)
	}
	if retryCount >= MaxAutoRetries {
		parts = append([]string{"RETRY_LIMIT_HIT"}, parts...)
	}

	if len(parts) == 0 {
		return "budget OK"
	}
	return strings.Join(parts, " | ")
}

// isTaskRecoverable determines if a stuck task can be recovered.
// PR-4: Tasks stuck due to platform defects are recoverable.
func isTaskRecoverable(task *ent.Task, payload map[string]interface{}) bool {
	switch task.State {
	case engine.StateFailed, engine.StateRetryWaiting, engine.StateReviewPending:
		return true
	default:
		return false
	}
}

// readBoolDefault reads a bool from payload with a default fallback.
func readBoolDefault(payload map[string]interface{}, fallback bool, key string) bool {
	if val, ok := readBool(payload, key); ok {
		return val
	}
	return fallback
}
