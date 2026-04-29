package engine

import (
	"testing"
)

// Mock implementations for testing
type mockTask struct {
	id          string
	wave        int
	state       string
	dispatchRef string
	relations   []TaskRelation
	topoRank    int
	cardJSON    string
}

func (m *mockTask) GetID() string                { return m.id }
func (m *mockTask) GetWave() int                 { return m.wave }
func (m *mockTask) GetState() string             { return m.state }
func (m *mockTask) GetDispatchRef() string       { return m.dispatchRef }
func (m *mockTask) GetRelations() []TaskRelation { return m.relations }
func (m *mockTask) GetTopoRank() int             { return m.topoRank }
func (m *mockTask) GetRetryCount() int           { return 0 }
func (m *mockTask) GetLoopIterationCount() int   { return 0 }
func (m *mockTask) GetCardJSON() string          { return m.cardJSON }

func TestTopologicalSorter(t *testing.T) {
	tests := []struct {
		name          string
		tasks         []TaskWithRelations
		expectedOrder []string
		expectError   bool
	}{
		{
			name: "simple chain",
			tasks: []TaskWithRelations{
				&mockTask{id: "task3", relations: []TaskRelation{{TaskID: "task2", Type: RelationDependsOn}}},
				&mockTask{id: "task2", relations: []TaskRelation{{TaskID: "task1", Type: RelationDependsOn}}},
				&mockTask{id: "task1", relations: []TaskRelation{}},
			},
			expectedOrder: []string{"task1", "task2", "task3"},
			expectError:   false,
		},
		{
			name: "independent tasks",
			tasks: []TaskWithRelations{
				&mockTask{id: "task2", relations: []TaskRelation{}},
				&mockTask{id: "task1", relations: []TaskRelation{}},
			},
			expectedOrder: []string{"task1", "task2"},
			expectError:   false,
		},
		{
			name: "diamond dependency",
			tasks: []TaskWithRelations{
				&mockTask{id: "D", relations: []TaskRelation{{TaskID: "B", Type: RelationDependsOn}, {TaskID: "C", Type: RelationDependsOn}}},
				&mockTask{id: "C", relations: []TaskRelation{{TaskID: "A", Type: RelationDependsOn}}},
				&mockTask{id: "B", relations: []TaskRelation{{TaskID: "A", Type: RelationDependsOn}}},
				&mockTask{id: "A", relations: []TaskRelation{}},
			},
			expectedOrder: []string{"A", "B", "C", "D"},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorter := NewTopologicalSorter(tt.tasks)
			order, ranks, err := sorter.Sort()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check order length
			if len(order) != len(tt.expectedOrder) {
				t.Errorf("Expected order length %d, got %d", len(tt.expectedOrder), len(order))
				return
			}

			// Verify ranks are consistent with dependencies
			for _, task := range tt.tasks {
				for _, rel := range task.GetRelations() {
					if rel.Type == RelationDependsOn {
						depRank := ranks[rel.TaskID]
						taskRank := ranks[task.GetID()]
						if taskRank <= depRank {
							t.Errorf("Rank violation: task %s (rank %d) should be > dep %s (rank %d)",
								task.GetID(), taskRank, rel.TaskID, depRank)
						}
					}
				}
			}
		})
	}
}

func TestCalculateConflicts(t *testing.T) {
	tests := []struct {
		name              string
		tasks             []TaskWithRelations
		expectedConflicts map[string][]string
	}{
		{
			name: "no conflicts",
			tasks: []TaskWithRelations{
				&mockTask{id: "task1", relations: []TaskRelation{}, cardJSON: `{"files_to_modify":["a.go"]}`},
				&mockTask{id: "task2", relations: []TaskRelation{}, cardJSON: `{"files_to_modify":["b.go"]}`},
			},
			expectedConflicts: map[string][]string{
				"task1": {},
				"task2": {},
			},
		},
		{
			name: "detects overlapping files_to_modify",
			tasks: []TaskWithRelations{
				&mockTask{id: "task1", relations: []TaskRelation{}, cardJSON: `{"files_to_modify":["shared.go","a.go"]}`},
				&mockTask{id: "task2", relations: []TaskRelation{}, cardJSON: `{"files_to_modify":["shared.go","b.go"]}`},
			},
			expectedConflicts: map[string][]string{
				"task1": {"task2"},
				"task2": {"task1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := NewDependencyManager(nil)
			conflicts := dm.CalculateConflicts(nil, tt.tasks)

			if len(conflicts) != len(tt.expectedConflicts) {
				t.Fatalf("Expected %d conflict entries, got %d", len(tt.expectedConflicts), len(conflicts))
			}

			for taskID, expected := range tt.expectedConflicts {
				actual := conflicts[taskID]
				if len(actual) != len(expected) {
					t.Fatalf("Task %s: expected %d conflicts, got %d (%v)", taskID, len(expected), len(actual), actual)
				}
				for i := range expected {
					if actual[i] != expected[i] {
						t.Fatalf("Task %s: expected conflicts %v, got %v", taskID, expected, actual)
					}
				}
			}
		})
	}
}

func TestRelationsFromJSON(t *testing.T) {
	jsonData := `[{"task_id": "task1", "type": "depends_on", "reason": "test"}]`

	relations, err := RelationsFromJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("RelationsFromJSON failed: %v", err)
	}

	if len(relations) != 1 {
		t.Errorf("Expected 1 relation, got %d", len(relations))
	}

	if relations[0].TaskID != "task1" {
		t.Errorf("Expected task_id 'task1', got %q", relations[0].TaskID)
	}

	if relations[0].Type != RelationDependsOn {
		t.Errorf("Expected type 'depends_on', got %q", relations[0].Type)
	}
}

func TestRelationsToJSON(t *testing.T) {
	relations := []TaskRelation{
		{TaskID: "task1", Type: RelationDependsOn, Reason: "test"},
	}

	data, err := RelationsToJSON(relations)
	if err != nil {
		t.Fatalf("RelationsToJSON failed: %v", err)
	}

	// Verify it can be parsed back
	parsed, err := RelationsFromJSON(data)
	if err != nil {
		t.Fatalf("Failed to parse JSON back: %v", err)
	}

	if len(parsed) != 1 || parsed[0].TaskID != "task1" {
		t.Errorf("Round-trip failed, got: %+v", parsed)
	}
}
