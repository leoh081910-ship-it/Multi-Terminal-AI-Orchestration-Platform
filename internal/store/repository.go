// Package store provides the repository layer for the AI orchestration platform.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/task"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/wave"
	"github.com/rs/zerolog"
)

// TaskCard represents the JSON structure for task creation/updates.
type TaskCard struct {
	ID                 string `json:"id"`
	DispatchRef        string `json:"dispatch_ref"`
	State              string `json:"state"`
	RetryCount         int    `json:"retry_count"`
	LoopIterationCount int    `json:"loop_iteration_count"`
	Transport          string `json:"transport"`
	Wave               int    `json:"wave"`
	TopoRank           int    `json:"topo_rank"`
	WorkspacePath      string `json:"workspace_path,omitempty"`
	ArtifactPath       string `json:"artifact_path,omitempty"`
	LastErrorReason    string `json:"last_error_reason,omitempty"`
	CardJSON           string `json:"card_json"`
}

// TaskView is the API-facing task projection built from card_json plus runtime fields.
type TaskView struct {
	ID                 string    `json:"id"`
	DispatchRef        string    `json:"dispatch_ref"`
	State              string    `json:"state"`
	RetryCount         int       `json:"retry_count"`
	LoopIterationCount int       `json:"loop_iteration_count"`
	Transport          string    `json:"transport"`
	Wave               int       `json:"wave"`
	TopoRank           int       `json:"topo_rank"`
	WorkspacePath      string    `json:"workspace_path,omitempty"`
	ArtifactPath       string    `json:"artifact_path,omitempty"`
	LastErrorReason    string    `json:"last_error_reason,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	TerminalAt         time.Time `json:"terminal_at,omitempty"`
	CardJSON           string    `json:"card_json"`
}

type taskCardJSONPayload struct {
	ID                 *string `json:"id"`
	DispatchRef        *string `json:"dispatch_ref"`
	State              *string `json:"state"`
	RetryCount         *int    `json:"retry_count"`
	LoopIterationCount *int    `json:"loop_iteration_count"`
	Transport          *string `json:"transport"`
	Wave               *int    `json:"wave"`
	TopoRank           *int    `json:"topo_rank"`
	WorkspacePath      *string `json:"workspace_path"`
	ArtifactPath       *string `json:"artifact_path"`
	LastErrorReason    *string `json:"last_error_reason"`
}

// EventData represents the event structure for logging.
type EventData struct {
	EventID   string
	TaskID    string
	EventType string
	FromState string
	ToState   string
	Timestamp time.Time
	Reason    string
	Attempt   int
	Transport string
	RunnerID  string
	Details   string
}

// Repository wraps ent.Client and provides transactional CRUD operations.
type Repository struct {
	client *ent.Client
	logger *zerolog.Logger
}

// NewRepository creates a new Repository instance.
func NewRepository(client *ent.Client, logger *zerolog.Logger) *Repository {
	return &Repository{
		client: client,
		logger: logger,
	}
}

// WithTx executes the given function within a transaction.
// If the function returns an error, the transaction is rolled back.
func (r *Repository) WithTx(ctx context.Context, fn func(tx *ent.Tx) error) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}
	return nil
}

// CreateTask creates a new task.
func (r *Repository) CreateTask(ctx context.Context, card *TaskCard) (string, error) {
	derived, err := deriveTaskCard(card, nil)
	if err != nil {
		return "", err
	}

	if derived.ID == "" {
		return "", errors.New("task id is required")
	}
	if derived.DispatchRef == "" {
		return "", errors.New("dispatch_ref is required")
	}
	if derived.Transport == "" {
		return "", errors.New("transport is required")
	}

	now := time.Now().UTC()
	t, err := r.client.Task.Create().
		SetID(derived.ID).
		SetDispatchRef(derived.DispatchRef).
		SetState(derived.State).
		SetRetryCount(derived.RetryCount).
		SetLoopIterationCount(derived.LoopIterationCount).
		SetTransport(derived.Transport).
		SetWave(derived.Wave).
		SetTopoRank(derived.TopoRank).
		SetWorkspacePath(derived.WorkspacePath).
		SetArtifactPath(derived.ArtifactPath).
		SetLastErrorReason(derived.LastErrorReason).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetCardJSON(derived.CardJSON).
		Save(ctx)

	if err != nil {
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	r.logger.Debug().Str("task_id", t.ID).Msg("task created")
	return t.ID, nil
}

// UpdateTaskState updates task state with event logging in a single transaction.
// This is required by PERS-05: event and task updates must be in the same SQLite transaction.
func (r *Repository) UpdateTaskState(ctx context.Context, taskID, fromState, toState, reason string, eventData *EventData) error {
	return r.WithTx(ctx, func(tx *ent.Tx) error {
		// Update task state
		updated, err := tx.Task.Update().
			Where(task.ID(taskID), task.State(fromState)).
			SetState(toState).
			SetUpdatedAt(time.Now().UTC()).
			Save(ctx)

		if err != nil {
			return fmt.Errorf("failed to update task state: %w", err)
		}

		if updated == 0 {
			return errors.New("task not found or state mismatch")
		}

		// Create event in the same transaction
		if eventData != nil {
			now := time.Now().UTC()
			_, err = tx.Event.Create().
				SetEventID(eventData.EventID).
				SetTaskID(taskID).
				SetEventType(eventData.EventType).
				SetFromState(fromState).
				SetToState(toState).
				SetTimestamp(now).
				SetReason(reason).
				SetAttempt(eventData.Attempt).
				SetTransport(eventData.Transport).
				SetRunnerID(eventData.RunnerID).
				SetDetails(eventData.Details).
				Save(ctx)

			if err != nil {
				return fmt.Errorf("failed to create event: %w", err)
			}
		}

		return nil
	})
}

// GetTaskByID retrieves a task by its ID.
func (r *Repository) GetTaskByID(ctx context.Context, taskID string) (*ent.Task, error) {
	t, err := r.client.Task.Get(ctx, taskID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return t, nil
}

// ListTasksByDispatchRef retrieves all tasks for a given dispatch reference.
func (r *Repository) ListTasksByDispatchRef(ctx context.Context, dispatchRef string) ([]*ent.Task, error) {
	tasks, err := r.client.Task.Query().
		Where(task.DispatchRef(dispatchRef)).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	return tasks, nil
}

// ListAllTasks retrieves all tasks.
func (r *Repository) ListAllTasks(ctx context.Context) ([]*ent.Task, error) {
	tasks, err := r.client.Task.Query().
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	return tasks, nil
}

// UpdateTask updates task fields.
func (r *Repository) UpdateTask(ctx context.Context, taskID string, updates *TaskCard) error {
	existing, err := r.client.Task.Get(ctx, taskID)
	if err != nil {
		if ent.IsNotFound(err) {
			return err
		}
		return fmt.Errorf("failed to get task for update: %w", err)
	}

	derived, err := deriveTaskCard(updates, existing)
	if err != nil {
		return err
	}

	_, err = r.client.Task.UpdateOneID(taskID).
		SetDispatchRef(derived.DispatchRef).
		SetState(derived.State).
		SetRetryCount(derived.RetryCount).
		SetLoopIterationCount(derived.LoopIterationCount).
		SetTransport(derived.Transport).
		SetWave(derived.Wave).
		SetTopoRank(derived.TopoRank).
		SetWorkspacePath(derived.WorkspacePath).
		SetArtifactPath(derived.ArtifactPath).
		SetLastErrorReason(derived.LastErrorReason).
		SetUpdatedAt(time.Now().UTC()).
		SetCardJSON(derived.CardJSON).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// BuildTaskView reconstructs an API-facing task from card_json plus runtime state.
func BuildTaskView(t *ent.Task) (*TaskView, error) {
	if t == nil {
		return nil, nil
	}

	base := &TaskCard{
		ID:                 t.ID,
		DispatchRef:        t.DispatchRef,
		State:              t.State,
		RetryCount:         t.RetryCount,
		LoopIterationCount: t.LoopIterationCount,
		Transport:          t.Transport,
		Wave:               t.Wave,
		TopoRank:           t.TopoRank,
		WorkspacePath:      t.WorkspacePath,
		ArtifactPath:       t.ArtifactPath,
		LastErrorReason:    t.LastErrorReason,
		CardJSON:           t.CardJSON,
	}

	derived, err := deriveTaskCard(base, t)
	if err != nil {
		derived = base
	}

	view := &TaskView{
		ID:                 firstNonEmpty(derived.ID, t.ID),
		DispatchRef:        firstNonEmpty(derived.DispatchRef, t.DispatchRef),
		State:              t.State,
		RetryCount:         t.RetryCount,
		LoopIterationCount: t.LoopIterationCount,
		Transport:          firstNonEmpty(derived.Transport, t.Transport),
		Wave:               derived.Wave,
		TopoRank:           derived.TopoRank,
		WorkspacePath:      t.WorkspacePath,
		ArtifactPath:       t.ArtifactPath,
		LastErrorReason:    t.LastErrorReason,
		CreatedAt:          t.CreatedAt,
		UpdatedAt:          t.UpdatedAt,
		TerminalAt:         t.TerminalAt,
		CardJSON:           t.CardJSON,
	}

	return view, err
}

// CreateEvent creates a new event (standalone, not within transaction).
func (r *Repository) CreateEvent(ctx context.Context, eventData *EventData) error {
	if eventData.Timestamp.IsZero() {
		eventData.Timestamp = time.Now().UTC()
	}

	_, err := r.client.Event.Create().
		SetEventID(eventData.EventID).
		SetTaskID(eventData.TaskID).
		SetEventType(eventData.EventType).
		SetFromState(eventData.FromState).
		SetToState(eventData.ToState).
		SetTimestamp(eventData.Timestamp).
		SetReason(eventData.Reason).
		SetAttempt(eventData.Attempt).
		SetTransport(eventData.Transport).
		SetRunnerID(eventData.RunnerID).
		SetDetails(eventData.Details).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	return nil
}

// UpsertWave creates or updates a wave.
func (r *Repository) UpsertWave(ctx context.Context, dispatchRef string, waveNum int) error {
	now := time.Now().UTC()

	// Try to get existing wave
	existing, err := r.client.Wave.Query().
		Where(wave.DispatchRef(dispatchRef), wave.Wave(waveNum)).
		Only(ctx)

	if err == nil {
		// Update existing
		_, err = existing.Update().
			SetCreatedAt(existing.CreatedAt).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to update wave: %w", err)
		}
		return nil
	}

	if !ent.IsNotFound(err) {
		return fmt.Errorf("failed to query wave: %w", err)
	}

	// Create new wave
	_, err = r.client.Wave.Create().
		SetDispatchRef(dispatchRef).
		SetWave(waveNum).
		SetCreatedAt(now).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to create wave: %w", err)
	}

	return nil
}

// GetWave retrieves a wave by dispatch reference and wave number.
func (r *Repository) GetWave(ctx context.Context, dispatchRef string, waveNum int) (*ent.Wave, error) {
	w, err := r.client.Wave.Query().
		Where(wave.DispatchRef(dispatchRef), wave.Wave(waveNum)).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get wave: %w", err)
	}
	return w, nil
}

// SealWave marks a wave as sealed.
func (r *Repository) SealWave(ctx context.Context, dispatchRef string, waveNum int) error {
	w, err := r.client.Wave.Query().
		Where(wave.DispatchRef(dispatchRef), wave.Wave(waveNum)).
		Only(ctx)

	if err != nil {
		return fmt.Errorf("wave not found: %w", err)
	}

	now := time.Now().UTC()
	_, err = w.Update().
		SetSealedAt(now).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to seal wave: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (r *Repository) Close() error {
	return r.client.Close()
}

func deriveTaskCard(input *TaskCard, existing *ent.Task) (*TaskCard, error) {
	if input == nil {
		return nil, errors.New("task card is required")
	}
	if input.CardJSON == "" {
		return nil, errors.New("invalid card_json: empty")
	}

	var payload taskCardJSONPayload
	if err := json.Unmarshal([]byte(input.CardJSON), &payload); err != nil {
		return nil, fmt.Errorf("invalid card_json: %w", err)
	}

	derived := &TaskCard{
		ID:                 chooseString(payload.ID, existingString(existing, func(t *ent.Task) string { return t.ID })),
		DispatchRef:        chooseString(payload.DispatchRef, existingString(existing, func(t *ent.Task) string { return t.DispatchRef })),
		State:              chooseString(payload.State, existingString(existing, func(t *ent.Task) string { return t.State })),
		RetryCount:         chooseInt(payload.RetryCount, existingInt(existing, func(t *ent.Task) int { return t.RetryCount })),
		LoopIterationCount: chooseInt(payload.LoopIterationCount, existingInt(existing, func(t *ent.Task) int { return t.LoopIterationCount })),
		Transport:          chooseString(payload.Transport, existingString(existing, func(t *ent.Task) string { return t.Transport })),
		Wave:               chooseInt(payload.Wave, existingInt(existing, func(t *ent.Task) int { return t.Wave })),
		TopoRank:           chooseInt(payload.TopoRank, existingInt(existing, func(t *ent.Task) int { return t.TopoRank })),
		WorkspacePath:      chooseString(payload.WorkspacePath, existingString(existing, func(t *ent.Task) string { return t.WorkspacePath })),
		ArtifactPath:       chooseString(payload.ArtifactPath, existingString(existing, func(t *ent.Task) string { return t.ArtifactPath })),
		LastErrorReason:    chooseString(payload.LastErrorReason, existingString(existing, func(t *ent.Task) string { return t.LastErrorReason })),
		CardJSON:           input.CardJSON,
	}

	if derived.ID == "" {
		return nil, errors.New("invalid card_json: missing id")
	}
	if derived.DispatchRef == "" {
		return nil, errors.New("invalid card_json: missing dispatch_ref")
	}
	if derived.Transport == "" {
		return nil, errors.New("invalid card_json: missing transport")
	}
	if derived.State == "" {
		derived.State = "queued"
	}

	return derived, nil
}

func chooseString(value *string, existing string) string {
	if value != nil {
		return *value
	}
	return existing
}

func chooseInt(value *int, existing int) int {
	if value != nil {
		return *value
	}
	return existing
}

func existingString(t *ent.Task, getter func(*ent.Task) string) string {
	if t == nil {
		return ""
	}
	return getter(t)
}

func existingInt(t *ent.Task, getter func(*ent.Task) int) int {
	if t == nil {
		return 0
	}
	return getter(t)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
