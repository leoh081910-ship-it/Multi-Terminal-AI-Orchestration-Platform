// Package engine provides the core orchestration engine for the AI orchestration platform.
package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Orchestrator is the core engine that manages task lifecycle.
type Orchestrator struct {
	repo         Repository
	depManager   *DependencyManager
	waveManager  *WaveManager
	retryManager *RetryManager
	ttlManager   *TTLManager
	logger       zerolog.Logger
	config       *OrchestratorConfig
}

// Repository defines the interface for persistence operations.
type Repository interface {
	// Task operations
	CreateTask(ctx context.Context, card *TaskCard) (string, error)
	GetTaskByID(ctx context.Context, taskID string) (TaskInfo, error)
	UpdateTaskState(ctx context.Context, taskID, fromState, toState, reason string, eventData *EventData) error
	UpdateTaskTopoRank(ctx context.Context, taskID string, topoRank int) error
	ListTasksByDispatchRef(ctx context.Context, dispatchRef string) ([]TaskInfo, error)
	ListTasksByDispatchRefAndWave(ctx context.Context, dispatchRef string, wave int) ([]TaskInfo, error)

	// Event operations
	CreateEvent(ctx context.Context, eventData *EventData) error

	// Cleanup operations
	CleanupTaskResources(ctx context.Context, taskID string) (bool, error)

	// Wave operations
	UpsertWave(ctx context.Context, dispatchRef string, waveNum int) error
	GetWave(ctx context.Context, dispatchRef string, waveNum int) (WaveInfo, error)
	SealWave(ctx context.Context, dispatchRef string, waveNum int) error
}

// TaskCard represents the JSON structure for task creation.
type TaskCard struct {
	ID                 string         `json:"id"`
	DispatchRef        string         `json:"dispatch_ref"`
	State              string         `json:"state"`
	RetryCount         int            `json:"retry_count"`
	LoopIterationCount int            `json:"loop_iteration_count"`
	Transport          string         `json:"transport"`
	Wave               int            `json:"wave"`
	TopoRank           int            `json:"topo_rank"`
	WorkspacePath      string         `json:"workspace_path,omitempty"`
	ArtifactPath       string         `json:"artifact_path,omitempty"`
	LastErrorReason    string         `json:"last_error_reason,omitempty"`
	CardJSON           string         `json:"card_json"`
	Relations          []TaskRelation `json:"relations,omitempty"`
}

func (t *TaskCard) GetID() string                { return t.ID }
func (t *TaskCard) GetWave() int                 { return t.Wave }
func (t *TaskCard) GetState() string             { return t.State }
func (t *TaskCard) GetDispatchRef() string       { return t.DispatchRef }
func (t *TaskCard) GetRelations() []TaskRelation { return t.Relations }
func (t *TaskCard) GetTopoRank() int             { return t.TopoRank }
func (t *TaskCard) GetRetryCount() int           { return t.RetryCount }
func (t *TaskCard) GetLoopIterationCount() int   { return t.LoopIterationCount }
func (t *TaskCard) GetCardJSON() string          { return t.CardJSON }

// TaskInfo provides information about a task.
type TaskInfo interface {
	GetID() string
	GetWave() int
	GetState() string
	GetDispatchRef() string
	GetRelations() []TaskRelation
	GetTopoRank() int
	GetRetryCount() int
	GetLoopIterationCount() int
}

// EventData represents event data for logging.
type EventData struct {
	EventID   string    `json:"event_id"`
	TaskID    string    `json:"task_id"`
	EventType string    `json:"event_type"`
	FromState string    `json:"from_state"`
	ToState   string    `json:"to_state"`
	Timestamp time.Time `json:"timestamp"`
	Reason    string    `json:"reason,omitempty"`
	Attempt   int       `json:"attempt"`
	Transport string    `json:"transport,omitempty"`
	RunnerID  string    `json:"runner_id,omitempty"`
	Details   string    `json:"details,omitempty"`
}

// OrchestratorConfig holds configuration for the orchestrator.
type OrchestratorConfig struct {
	RetryConfig *RetryConfig
	TTLConfig   *TTLConfig
}

// NewOrchestrator creates a new orchestrator.
func NewOrchestrator(
	repo Repository,
	depManager *DependencyManager,
	waveManager *WaveManager,
	retryManager *RetryManager,
	ttlManager *TTLManager,
	logger zerolog.Logger,
	config *OrchestratorConfig,
) *Orchestrator {
	if config == nil {
		config = &OrchestratorConfig{
			RetryConfig: DefaultRetryConfig(),
			TTLConfig:   DefaultTTLConfig(),
		}
	}

	return &Orchestrator{
		repo:         repo,
		depManager:   depManager,
		waveManager:  waveManager,
		retryManager: retryManager,
		ttlManager:   ttlManager,
		logger:       logger,
		config:       config,
	}
}

// EnqueueTask creates a new task with initial state "queued".
// Validates dependencies and computes topo_rank.
func (o *Orchestrator) EnqueueTask(ctx context.Context, card *TaskCard) (string, error) {
	if o.repo == nil {
		return "", ErrOrchestratorRepoNotConfigured
	}
	if o.depManager == nil {
		return "", ErrDependencyManagerNotConfigured
	}
	if o.waveManager == nil {
		return "", ErrWaveManagerNotConfigured
	}

	// Validate dependencies
	if err := o.depManager.ValidateDependencies(ctx, card); err != nil {
		o.logger.Warn().Err(err).Str("task_id", card.ID).Msg("dependency validation failed")
		return "", fmt.Errorf("dependency validation failed: %w", err)
	}

	// Calculate topo_rank
	topoRank, err := o.depManager.CalculateTopoRank(ctx, card)
	if err != nil {
		o.logger.Warn().Err(err).Str("task_id", card.ID).Msg("topo_rank calculation failed")
		return "", fmt.Errorf("topo_rank calculation failed: %w", err)
	}

	// Set initial state
	card.State = StateQueued
	card.TopoRank = topoRank

	// Ensure wave exists
	if err := o.waveManager.EnsureWave(ctx, card.DispatchRef, card.Wave); err != nil {
		o.logger.Error().Err(err).
			Str("dispatch_ref", card.DispatchRef).
			Int("wave", card.Wave).
			Msg("failed to ensure wave")
		return "", fmt.Errorf("wave management failed: %w", err)
	}

	// Create task in repository
	taskID, err := o.repo.CreateTask(ctx, card)
	if err != nil {
		o.logger.Error().Err(err).Str("task_id", card.ID).Msg("failed to create task")
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	o.logger.Info().
		Str("task_id", taskID).
		Int("topo_rank", topoRank).
		Msg("task enqueued")

	return taskID, nil
}

// RouteTask attempts to transition a task from "queued" to "routed".
// Validates wave seal status before routing.
func (o *Orchestrator) RouteTask(ctx context.Context, taskID string) error {
	if o.repo == nil {
		return ErrOrchestratorRepoNotConfigured
	}
	if o.waveManager == nil {
		return ErrWaveManagerNotConfigured
	}

	// Get task info
	task, err := o.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Check wave seal status
	canRoute, err := o.waveManager.CanRouteToWave(ctx, task.GetDispatchRef(), task.GetWave())
	if err != nil {
		return fmt.Errorf("failed to check wave state: %w", err)
	}

	if !canRoute {
		return ErrWaveNotSealed
	}

	// Perform state transition
	return o.TransitionState(ctx, taskID, StateQueued, StateRouted, "")
}

// TransitionState performs a state transition with validation and event logging.
func (o *Orchestrator) TransitionState(ctx context.Context, taskID, fromState, toState, reason string) error {
	// Validate transition
	if err := ValidateTransition(fromState, toState); err != nil {
		return err
	}

	// Create event data
	eventData := &EventData{
		EventID:   generateEventID(),
		TaskID:    taskID,
		EventType: "state_transition",
		FromState: fromState,
		ToState:   toState,
		Timestamp: time.Now().UTC(),
		Reason:    reason,
	}

	// Perform atomic update in repository
	if err := o.repo.UpdateTaskState(ctx, taskID, fromState, toState, reason, eventData); err != nil {
		return fmt.Errorf("failed to update task state: %w", err)
	}

	o.logger.Info().
		Str("task_id", taskID).
		Str("from", fromState).
		Str("to", toState).
		Str("reason", reason).
		Msg("state transition completed")

	return nil
}

// HandleRetry processes a retry for a task.
func (o *Orchestrator) HandleRetry(ctx context.Context, taskID string, state *RetryState, reason string) error {
	if o.retryManager == nil {
		return ErrRetryManagerNotConfigured
	}

	// Check if we should retry
	if !o.retryManager.ShouldRetry(state, reason) {
		// Max retries exceeded, mark as failed
		return o.TransitionState(ctx, taskID, StateRetryWaiting, StateFailed, "max_retries_exceeded")
	}

	// Check if backoff has elapsed
	if !state.LastRetryAt.IsZero() && !o.retryManager.IsBackoffElapsed(state.RetryCount, state.LastRetryAt) {
		return ErrBackoffNotElapsed
	}

	// Transition to retry_waiting or back to routed
	return o.TransitionState(ctx, taskID, StateRetryWaiting, StateRouted, reason)
}

// CheckTTL checks if a task has expired and should be cleaned up.
func (o *Orchestrator) CheckTTL(ctx context.Context, taskID string, terminalAt time.Time) error {
	if o.ttlManager == nil {
		return ErrTTLManagerNotConfigured
	}
	if !o.ttlManager.IsExpired(terminalAt) {
		return nil
	}

	task, err := o.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task for ttl cleanup: %w", err)
	}

	// Task has expired, trigger cleanup
	o.logger.Info().
		Str("task_id", taskID).
		Time("terminal_at", terminalAt).
		Msg("task TTL expired, triggering cleanup")

	cleaned, err := o.repo.CleanupTaskResources(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to cleanup task resources: %w", err)
	}
	if !cleaned {
		return nil
	}

	eventData := &EventData{
		EventID:   generateEventID(),
		TaskID:    taskID,
		EventType: "ttl_cleanup",
		FromState: task.GetState(),
		ToState:   task.GetState(),
		Timestamp: time.Now().UTC(),
		Reason:    "ttl_expired",
		Attempt:   task.GetRetryCount(),
	}
	if err := o.repo.CreateEvent(ctx, eventData); err != nil {
		return fmt.Errorf("failed to record ttl cleanup event: %w", err)
	}

	return nil
}

// PropagateDependencyFailure handles dependency failure propagation.
func (o *Orchestrator) PropagateDependencyFailure(ctx context.Context, failedTaskID string) error {
	if o.repo == nil {
		return ErrOrchestratorRepoNotConfigured
	}
	if o.depManager == nil {
		return ErrDependencyManagerNotConfigured
	}

	// Get the failed task
	failedTask, err := o.repo.GetTaskByID(ctx, failedTaskID)
	if err != nil {
		return fmt.Errorf("failed to get failed task: %w", err)
	}

	// Get all affected tasks
	affectedTaskIDs, err := o.depManager.PropagateDependencyFailure(ctx, failedTask)
	if err != nil {
		return fmt.Errorf("failed to propagate dependency failure: %w", err)
	}

	// Mark each affected task as failed
	for _, taskID := range affectedTaskIDs {
		task, err := o.repo.GetTaskByID(ctx, taskID)
		if err != nil {
			o.logger.Error().Err(err).Str("task_id", taskID).Msg("failed to get task for failure propagation")
			continue
		}

		if err := o.TransitionState(ctx, taskID, task.GetState(), StateFailed, "dependency_failed"); err != nil {
			o.logger.Error().Err(err).Str("task_id", taskID).Msg("failed to propagate dependency failure")
		}
	}

	return nil
}

// Errors.
var (
	// ErrOrchestratorRepoNotConfigured is returned when the orchestrator repository is missing.
	ErrOrchestratorRepoNotConfigured = errors.New("orchestrator repository is not configured")
	// ErrDependencyManagerNotConfigured is returned when the dependency manager is missing.
	ErrDependencyManagerNotConfigured = errors.New("dependency manager is not configured")
	// ErrWaveManagerNotConfigured is returned when the wave manager is missing.
	ErrWaveManagerNotConfigured = errors.New("wave manager is not configured")
	// ErrRetryManagerNotConfigured is returned when the retry manager is missing.
	ErrRetryManagerNotConfigured = errors.New("retry manager is not configured")
	// ErrTTLManagerNotConfigured is returned when the TTL manager is missing.
	ErrTTLManagerNotConfigured = errors.New("ttl manager is not configured")
	// ErrBackoffNotElapsed is returned when backoff period has not elapsed.
	ErrBackoffNotElapsed = errors.New("backoff period has not elapsed")
)

// generateEventID generates a unique event ID.
func generateEventID() string {
	return "evt_" + uuid.New().String()[:8]
}
