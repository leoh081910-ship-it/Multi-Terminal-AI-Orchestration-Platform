// Package engine provides dependency management and topological ordering.
package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// RelationType represents the type of relationship between tasks.
type RelationType string

const (
	RelationDependsOn     RelationType = "depends_on"
	RelationConflictsWith RelationType = "conflicts_with"
)

// TaskRelation represents a relationship between tasks.
type TaskRelation struct {
	TaskID string       `json:"task_id"`
	Type   RelationType `json:"type"`
	Reason string       `json:"reason,omitempty"`
}

// TaskWithRelations is the minimal interface needed for dependency operations.
type TaskWithRelations interface {
	GetID() string
	GetWave() int
	GetState() string
	GetDispatchRef() string
	GetRelations() []TaskRelation
}

// DependencyManager handles dependency validation and topo_rank computation.
type DependencyManager struct {
	// resolver resolves task lookups for dependency validation
	resolver TaskResolver
}

// TaskResolver provides task lookup capabilities for dependency validation.
type TaskResolver interface {
	// GetTaskByID retrieves a task by its ID
	GetTaskByID(ctx context.Context, taskID string) (TaskWithRelations, error)
	// ListTasksByDispatchRefAndWave retrieves all tasks for a given dispatch reference and wave
	ListTasksByDispatchRefAndWave(ctx context.Context, dispatchRef string, wave int) ([]TaskWithRelations, error)
	// ListTasksByDispatchRef retrieves all tasks for a given dispatch reference
	ListTasksByDispatchRef(ctx context.Context, dispatchRef string) ([]TaskWithRelations, error)
}

// NewDependencyManager creates a new dependency manager.
func NewDependencyManager(resolver TaskResolver) *DependencyManager {
	return &DependencyManager{
		resolver: resolver,
	}
}

// ValidateDependencies validates task dependencies at enqueue time.
// REQUIRES (DEPD-03): depends_on can only point to same wave or earlier wave.
// REJECTS: Cross-wave forward dependencies.
func (dm *DependencyManager) ValidateDependencies(ctx context.Context, task TaskWithRelations) error {
	for _, rel := range task.GetRelations() {
		if rel.Type != RelationDependsOn {
			continue
		}

		depTask, err := dm.resolver.GetTaskByID(ctx, rel.TaskID)
		if err != nil {
			return fmt.Errorf("failed to resolve dependency %s: %w", rel.TaskID, err)
		}

		// DEPD-03: depends_on can only point to same wave or earlier wave
		if depTask.GetWave() > task.GetWave() {
			return &InvalidDependencyError{
				Reason:     "invalid_dependency",
				TaskID:     task.GetID(),
				Dependency: rel.TaskID,
				Detail:     fmt.Sprintf("depends_on cannot point to later wave (target: %d, current: %d)", depTask.GetWave(), task.GetWave()),
			}
		}
	}

	return nil
}

// CalculateTopoRank computes the topological rank for a task based on its depends_on relations.
// Uses Kahn's algorithm conceptually but for a single task update.
// REQUIRES: topo_rank is computed from dependency graph only.
// DEFAULT: tasks with no dependencies have topo_rank = 0.
func (dm *DependencyManager) CalculateTopoRank(ctx context.Context, task TaskWithRelations) (int, error) {
	deps := task.GetRelations()
	if len(deps) == 0 {
		return 0, nil
	}

	maxDepRank := -1
	depCount := 0
	for _, rel := range deps {
		if rel.Type != RelationDependsOn {
			continue
		}

		depCount++
		depTask, err := dm.resolver.GetTaskByID(ctx, rel.TaskID)
		if err != nil {
			return 0, fmt.Errorf("failed to resolve dependency %s for topo_rank: %w", rel.TaskID, err)
		}

		if rankedTask, ok := depTask.(interface{ GetTopoRank() int }); ok {
			if rank := rankedTask.GetTopoRank(); rank > maxDepRank {
				maxDepRank = rank
			}
		}
	}

	if depCount == 0 {
		return 0, nil
	}
	if maxDepRank < 0 {
		return 1, nil
	}

	return maxDepRank + 1, nil
}

// PropagateDependencyFailure propagates task failures to dependent tasks.
// REQUIRES: When a prerequisite task fails, dependent non-terminal tasks are immediately marked as failed.
// REASON: "dependency_failed"
func (dm *DependencyManager) PropagateDependencyFailure(ctx context.Context, failedTask TaskWithRelations) ([]string, error) {
	affectedTasks := []string{}

	// Get all tasks in the same dispatch ref
	tasks, err := dm.resolver.ListTasksByDispatchRef(ctx, failedTask.GetDispatchRef())
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks for dependency check: %w", err)
	}

	// Find all tasks that depend on the failed task
	for _, t := range tasks {
		if IsTerminal(t.GetState()) {
			continue
		}

		for _, rel := range t.GetRelations() {
			if rel.Type == RelationDependsOn && rel.TaskID == failedTask.GetID() {
				affectedTasks = append(affectedTasks, t.GetID())
				break
			}
		}
	}

	return affectedTasks, nil
}

// CalculateConflicts calculates conflict relationships for tasks within the same wave.
// REQUIRES: conflicts_with is only calculated within the same wave.
// EXCLUDES: tasks that already have depends_on relationships.
func (dm *DependencyManager) CalculateConflicts(ctx context.Context, tasks []TaskWithRelations) map[string][]string {
	conflicts := make(map[string][]string)

	for i, t1 := range tasks {
		conflicts[t1.GetID()] = []string{}

		// Check if t1 already has depends_on relationships
		hasDepends := false
		for _, rel := range t1.GetRelations() {
			if rel.Type == RelationDependsOn {
				hasDepends = true
				break
			}
		}

		if hasDepends {
			continue
		}

		for j, t2 := range tasks {
			if i == j {
				continue
			}

			// Check if both tasks modify the same files
			if hasConflictingFiles(t1, t2) {
				conflicts[t1.GetID()] = append(conflicts[t1.GetID()], t2.GetID())
			}
		}
	}

	return conflicts
}

type taskCardPayload struct {
	FilesToModify []string `json:"files_to_modify"`
}

// hasConflictingFiles checks if two tasks have overlapping files_to_modify.
func hasConflictingFiles(t1, t2 TaskWithRelations) bool {
	files1 := filesToModifyFromTask(t1)
	files2 := filesToModifyFromTask(t2)
	if len(files1) == 0 || len(files2) == 0 {
		return false
	}

	seen := make(map[string]struct{}, len(files1))
	for _, file := range files1 {
		seen[file] = struct{}{}
	}

	for _, file := range files2 {
		if _, ok := seen[file]; ok {
			return true
		}
	}

	return false
}

func filesToModifyFromTask(task TaskWithRelations) []string {
	withCard, ok := task.(interface{ GetCardJSON() string })
	if !ok {
		return nil
	}

	var payload taskCardPayload
	if err := json.Unmarshal([]byte(withCard.GetCardJSON()), &payload); err != nil {
		return nil
	}

	return payload.FilesToModify
}

// InvalidDependencyError is returned when a dependency validation fails.
type InvalidDependencyError struct {
	Reason     string
	TaskID     string
	Dependency string
	Detail     string
}

func (e *InvalidDependencyError) Error() string {
	return fmt.Sprintf("invalid dependency: %s (task=%s, dependency=%s): %s", e.Reason, e.TaskID, e.Dependency, e.Detail)
}

// JSON helpers for relation serialization.

// RelationsFromJSON parses JSON relations into TaskRelation slice.
func RelationsFromJSON(data []byte) ([]TaskRelation, error) {
	var relations []TaskRelation
	if err := json.Unmarshal(data, &relations); err != nil {
		return nil, err
	}
	return relations, nil
}

// RelationsToJSON serializes TaskRelation slice to JSON.
func RelationsToJSON(relations []TaskRelation) ([]byte, error) {
	return json.Marshal(relations)
}

// TopologicalSorter performs Kahn's algorithm for topo_rank computation.
type TopologicalSorter struct {
	tasks    []TaskWithRelations
	inDegree map[string]int
	graph    map[string][]string
}

// NewTopologicalSorter creates a new topological sorter for a set of tasks.
func NewTopologicalSorter(tasks []TaskWithRelations) *TopologicalSorter {
	sorter := &TopologicalSorter{
		tasks:    tasks,
		inDegree: make(map[string]int),
		graph:    make(map[string][]string),
	}
	sorter.buildGraph()
	return sorter
}

// buildGraph constructs the dependency graph and computes in-degrees.
func (s *TopologicalSorter) buildGraph() {
	// Initialize in-degree for all tasks
	for _, t := range s.tasks {
		s.inDegree[t.GetID()] = 0
		s.graph[t.GetID()] = []string{}
	}

	// Build edges from depends_on relations
	for _, t := range s.tasks {
		for _, rel := range t.GetRelations() {
			if rel.Type == RelationDependsOn {
				s.graph[rel.TaskID] = append(s.graph[rel.TaskID], t.GetID())
				s.inDegree[t.GetID()]++
			}
		}
	}
}

// Sort performs Kahn's algorithm and returns tasks in topological order.
// Also returns a map of task ID to topo_rank.
func (s *TopologicalSorter) Sort() ([]string, map[string]int, error) {
	queue := make([]string, 0)
	ranks := make(map[string]int)
	result := make([]string, 0)

	// Initialize ranks to 0 for all tasks
	for _, t := range s.tasks {
		ranks[t.GetID()] = 0
	}

	// Start with tasks having in-degree 0
	for _, t := range s.tasks {
		if s.inDegree[t.GetID()] == 0 {
			queue = append(queue, t.GetID())
			ranks[t.GetID()] = 0
		}
	}

	for len(queue) > 0 {
		// Dequeue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Process neighbors
		for _, neighbor := range s.graph[current] {
			s.inDegree[neighbor]--

			// Update rank: max(current_rank + 1, existing_rank)
			newRank := ranks[current] + 1
			if newRank > ranks[neighbor] {
				ranks[neighbor] = newRank
			}

			if s.inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check for cycle
	if len(result) != len(s.tasks) {
		return nil, nil, errors.New("cycle detected in dependency graph")
	}

	return result, ranks, nil
}
