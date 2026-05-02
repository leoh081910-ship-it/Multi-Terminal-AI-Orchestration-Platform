package decompose

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
)

// SubtaskSpec represents a decomposed subtask from the Agent.
type SubtaskSpec struct {
	Title              string   `json:"title"`
	Description        string   `json:"description,omitempty"`
	Type               string   `json:"type"` // task, review, research, communication
	DependsOn          []string `json:"depends_on,omitempty"`
	ConflictsWith      []string `json:"conflicts_with,omitempty"`
	FilesToModify      []string `json:"files_to_modify,omitempty"`
	AssignedAgentID    string   `json:"assigned_agent_id,omitempty"`
	AssignedRoleID     string   `json:"assigned_role_id,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
}

// DecompositionResult is the Agent's response to a decomposition request.
type DecompositionResult struct {
	Subtasks []SubtaskSpec `json:"subtasks"`
	Rationale string       `json:"rationale,omitempty"`
}

// Decomposer handles calling an Agent to decompose goals into subtasks.
type Decomposer struct {
	client *ent.Client
}

// NewDecomposer creates a new Decomposer.
func NewDecomposer(client *ent.Client) *Decomposer {
	return &Decomposer{client: client}
}

// DecomposeGoal triggers decomposition of a goal task.
// It transitions the task to decomposing, calls the Agent, and stores the result.
// Returns the decomposition result for user review.
func (d *Decomposer) DecomposeGoal(ctx context.Context, goalID string, agentID string) (*DecompositionResult, error) {
	goal, err := d.client.Task.Get(ctx, goalID)
	if err != nil {
		return nil, fmt.Errorf("get goal: %w", err)
	}
	if goal.State != engine.StateQueued {
		return nil, fmt.Errorf("goal must be in queued state, got %s", goal.State)
	}

	// Transition to decomposing
	now := time.Now().UTC()
	if _, err := d.client.Task.UpdateOneID(goalID).
		SetState(engine.StateDecomposing).
		SetDecompositionStatus("decomposing").
		SetUpdatedAt(now).
		Save(ctx); err != nil {
		return nil, fmt.Errorf("transition to decomposing: %w", err)
	}

	// In a real implementation, this would call the Agent API.
	// For now, we return a placeholder that the API handler will populate.
	return &DecompositionResult{
		Subtasks:  nil, // Agent fills this in
		Rationale: "awaiting agent response",
	}, nil
}

// CreateSubtasks creates the actual subtask records from a decomposition result.
// Called after the user approves the decomposition.
func (d *Decomposer) CreateSubtasks(ctx context.Context, goalID string, result *DecompositionResult) ([]string, error) {
	goal, err := d.client.Task.Get(ctx, goalID)
	if err != nil {
		return nil, fmt.Errorf("get goal: %w", err)
	}

	rootID := goalID
	if goal.RootID != "" {
		rootID = goal.RootID
	}
	newDepth := goal.Depth + 1
	now := time.Now().UTC()

	var childIDs []string
	for i, spec := range result.Subtasks {
		childID := uuid.New().String()
		childIDs = append(childIDs, childID)

		subtaskState := engine.StateQueued
		if len(spec.DependsOn) > 0 {
			// Has dependencies — stays queued until deps resolve
			subtaskState = engine.StateQueued
		}

		specJSON, _ := json.Marshal(spec)

		create := d.client.Task.Create().
			SetID(childID).
			SetProjectID(goal.ProjectID).
			SetDispatchRef(goal.DispatchRef).
			SetState(subtaskState).
			SetTransport("api").
			SetWave(goal.Wave).
			SetTopoRank(i).
			SetDepth(newDepth).
			SetDecompositionStatus("none").
			SetParentID(goalID).
			SetRootID(rootID).
			SetCardJSON(string(specJSON)).
			SetCreatedAt(now).
			SetUpdatedAt(now)

		if spec.AssignedAgentID != "" {
			create.SetAssignedAgentID(spec.AssignedAgentID)
		}
		if spec.AssignedRoleID != "" {
			create.SetAssignedRoleID(spec.AssignedRoleID)
		}
		if goal.OrgID != "" {
			create.SetOrgID(goal.OrgID)
		}

		if _, err := create.Save(ctx); err != nil {
			return nil, fmt.Errorf("create subtask %d: %w", i, err)
		}
	}

	now = time.Now().UTC()
	// Transition goal to running (children are now executing)
	if _, err := d.client.Task.UpdateOneID(goalID).
		SetState(engine.StateRunning).
		SetDecompositionStatus("confirmed").
		SetUpdatedAt(now).
		Save(ctx); err != nil {
		return nil, fmt.Errorf("transition goal to running: %w", err)
	}

	return childIDs, nil
}

// RejectDecomposition marks a decomposition as rejected.
func (d *Decomposer) RejectDecomposition(ctx context.Context, goalID string) error {
	now := time.Now().UTC()
	_, err := d.client.Task.UpdateOneID(goalID).
		SetState(engine.StateFailed).
		SetDecompositionStatus("rejected").
		SetUpdatedAt(now).
		Save(ctx)
	return err
}

var now = time.Now().UTC
