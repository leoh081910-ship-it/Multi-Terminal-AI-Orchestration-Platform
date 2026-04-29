package server

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/mergequeue"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/store"
)

type MergeQueueRepositoryAdapter struct {
	repo             *store.Repository
	projectID        string
	defaultProjectID string
}

func NewMergeQueueRepositoryAdapter(repo *store.Repository, projectID, defaultProjectID string) *MergeQueueRepositoryAdapter {
	return &MergeQueueRepositoryAdapter{
		repo:             repo,
		projectID:        normalizeCompatProjectID(projectID),
		defaultProjectID: normalizeCompatProjectID(firstCompatNonEmpty(defaultProjectID, compatDefaultProjectID)),
	}
}

func (a *MergeQueueRepositoryAdapter) GetVerifiedTasksReadyForMerge(ctx context.Context) ([]mergequeue.TaskInfo, error) {
	tasks, err := a.repo.ListAllTasks(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]mergequeue.TaskInfo, 0, len(tasks))
	for _, item := range tasks {
		if a.projectID != "" && normalizeCompatProjectID(firstCompatNonEmpty(item.ProjectID, a.defaultProjectID)) != a.projectID {
			continue
		}
		if item.State != "verified" {
			continue
		}
		result = append(result, mergequeue.TaskInfo{
			ID:           item.ID,
			DispatchRef:  item.DispatchRef,
			State:        item.State,
			TopoRank:     item.TopoRank,
			CreatedAt:    item.CreatedAt,
			ArtifactPath: item.ArtifactPath,
		})
	}
	return result, nil
}

func (a *MergeQueueRepositoryAdapter) GetTaskDependencies(ctx context.Context, taskID string) ([]string, error) {
	item, err := a.repo.GetTaskByID(ctx, taskID)
	if err != nil || item == nil {
		return nil, err
	}
	return extractDependsOnFromCardJSON(item.CardJSON), nil
}

func (a *MergeQueueRepositoryAdapter) CheckTasksInState(ctx context.Context, taskIDs []string, state string) (bool, error) {
	if len(taskIDs) == 0 {
		return true, nil
	}

	tasks, err := a.repo.ListAllTasks(ctx)
	if err != nil {
		return false, err
	}

	states := make(map[string]string, len(tasks))
	for _, item := range tasks {
		if a.projectID != "" && normalizeCompatProjectID(firstCompatNonEmpty(item.ProjectID, a.defaultProjectID)) != a.projectID {
			continue
		}
		states[item.ID] = item.State
	}
	for _, taskID := range taskIDs {
		if states[taskID] != state {
			return false, nil
		}
	}
	return true, nil
}

func (a *MergeQueueRepositoryAdapter) UpdateTaskState(ctx context.Context, taskID, fromState, toState, reason string) error {
	item, err := a.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		return err
	}
	if item == nil {
		return nil
	}

	if err := a.repo.UpdateTaskState(ctx, taskID, fromState, toState, reason, &store.EventData{
		EventID:   uuid.NewString(),
		TaskID:    taskID,
		EventType: "state_transition",
		FromState: fromState,
		ToState:   toState,
		Timestamp: time.Now().UTC(),
		Reason:    reason,
		Attempt:   item.RetryCount,
		Transport: item.Transport,
	}); err != nil {
		return err
	}

	if !requiresCompatPayloadSync(toState) {
		return nil
	}

	return a.syncCompatPayload(ctx, taskID, toState, reason)
}

func requiresCompatPayloadSync(state string) bool {
	switch state {
	case engine.StateDone, engine.StateFailed, engine.StateApplyFailed, engine.StateVerifyFailed:
		return true
	default:
		return false
	}
}

func (a *MergeQueueRepositoryAdapter) syncCompatPayload(ctx context.Context, taskID, state, reason string) error {
	item, err := a.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		return err
	}
	if item == nil || strings.TrimSpace(item.CardJSON) == "" {
		return nil
	}

	payload := mergeCompatPayload(decodeCompatPayload(item.CardJSON), map[string]interface{}{})
	if len(payload) == 0 {
		return nil
	}

	payload["status"] = mapCompatWorkflowStatus(state)
	payload["dispatch_status"] = mapCompatDispatchStatus(state)

	switch state {
	case engine.StateDone:
		payload["execution_session_id"] = nil
		payload["last_dispatch_error"] = nil
		payload["last_error_reason"] = nil
	case engine.StateFailed, engine.StateApplyFailed, engine.StateVerifyFailed:
		if strings.TrimSpace(reason) != "" {
			payload["last_dispatch_error"] = reason
			payload["last_error_reason"] = reason
		}
	}

	view, err := store.BuildTaskView(item)
	if err != nil {
		return err
	}

	card, err := buildCompatTaskCard(payload, view, taskID, a.defaultProjectID)
	if err != nil {
		return err
	}
	return a.repo.UpdateTask(ctx, taskID, card)
}

type mergeQueueCardPayload struct {
	DependsOn []string `json:"depends_on"`
	Relations []struct {
		TaskID string `json:"task_id"`
		Type   string `json:"type"`
	} `json:"relations"`
}

func extractDependsOnFromCardJSON(cardJSON string) []string {
	if strings.TrimSpace(cardJSON) == "" {
		return nil
	}

	var payload mergeQueueCardPayload
	if err := json.Unmarshal([]byte(cardJSON), &payload); err != nil {
		return nil
	}

	if len(payload.DependsOn) > 0 {
		return payload.DependsOn
	}

	result := make([]string, 0, len(payload.Relations))
	for _, relation := range payload.Relations {
		if relation.Type == "depends_on" && strings.TrimSpace(relation.TaskID) != "" {
			result = append(result, relation.TaskID)
		}
	}
	return result
}
