// Package decompose provides task tree operations for the goal decomposition lifecycle.
package decompose

import (
	"context"
	"fmt"

	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/task"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
)

// TreeNode represents a task in the decomposition tree.
type TreeNode struct {
	ID                 string       `json:"id"`
	ParentID           string       `json:"parent_id,omitempty"`
	RootID             string       `json:"root_id,omitempty"`
	Depth              int          `json:"depth"`
	State              string       `json:"state"`
	DecompositionStatus string      `json:"decomposition_status"`
	Title              string       `json:"title"`
	AssignedAgentID    string       `json:"assigned_agent_id,omitempty"`
	Children           []*TreeNode  `json:"children,omitempty"`
}

// TreeOps provides task tree query operations.
type TreeOps struct {
	client *ent.Client
}

// NewTreeOps creates a new TreeOps instance.
func NewTreeOps(client *ent.Client) *TreeOps {
	return &TreeOps{client: client}
}

// GetChildren returns all direct children of a task.
func (t *TreeOps) GetChildren(ctx context.Context, parentID string) ([]*ent.Task, error) {
	return t.client.Task.Query().
		Where(task.ParentID(parentID)).
		Order(task.ByID()).
		All(ctx)
}

// GetTree builds the full task tree rooted at the given task ID.
func (t *TreeOps) GetTree(ctx context.Context, rootID string) (*TreeNode, error) {
	root, err := t.client.Task.Get(ctx, rootID)
	if err != nil {
		return nil, fmt.Errorf("get root task: %w", err)
	}

	node := taskToNode(root)
	if err := t.buildChildren(ctx, node); err != nil {
		return nil, err
	}
	return node, nil
}

// AllChildrenComplete checks if all direct children of a task are in a terminal state.
func (t *TreeOps) AllChildrenComplete(ctx context.Context, parentID string) (bool, error) {
	children, err := t.GetChildren(ctx, parentID)
	if err != nil {
		return false, err
	}
	if len(children) == 0 {
		return false, nil
	}
	for _, child := range children {
		if !engine.IsTerminal(child.State) {
			return false, nil
		}
	}
	return true, nil
}

// CountByState counts children of a task grouped by state.
func (t *TreeOps) CountByState(ctx context.Context, parentID string) (map[string]int, error) {
	children, err := t.GetChildren(ctx, parentID)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	for _, child := range children {
		counts[child.State]++
	}
	return counts, nil
}

func (t *TreeOps) buildChildren(ctx context.Context, node *TreeNode) error {
	children, err := t.GetChildren(ctx, node.ID)
	if err != nil {
		return err
	}
	for _, child := range children {
		childNode := taskToNode(child)
		node.Children = append(node.Children, childNode)
		if err := t.buildChildren(ctx, childNode); err != nil {
			return err
		}
	}
	return nil
}

func taskToNode(t *ent.Task) *TreeNode {
	return &TreeNode{
		ID:                  t.ID,
		ParentID:            t.ParentID,
		RootID:              t.RootID,
		Depth:               t.Depth,
		State:               t.State,
		DecompositionStatus: t.DecompositionStatus,
		AssignedAgentID:     t.AssignedAgentID,
	}
}
