package mergequeue

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

type stateTransition struct {
	taskID    string
	fromState string
	toState   string
	reason    string
}

// Mock repository for testing.
type mockRepo struct {
	tasks       map[string]TaskInfo
	deps        map[string][]string
	transitions []stateTransition
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		tasks: make(map[string]TaskInfo),
		deps:  make(map[string][]string),
	}
}

func (m *mockRepo) GetVerifiedTasksReadyForMerge(ctx context.Context) ([]TaskInfo, error) {
	var result []TaskInfo
	for _, task := range m.tasks {
		if task.State == "verified" {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *mockRepo) GetTaskDependencies(ctx context.Context, taskID string) ([]string, error) {
	return append([]string(nil), m.deps[taskID]...), nil
}

func (m *mockRepo) CheckTasksInState(ctx context.Context, taskIDs []string, state string) (bool, error) {
	for _, taskID := range taskIDs {
		task, ok := m.tasks[taskID]
		if !ok || task.State != state {
			return false, nil
		}
	}
	return true, nil
}

func (m *mockRepo) UpdateTaskState(ctx context.Context, taskID, fromState, toState, reason string) error {
	task, exists := m.tasks[taskID]
	if !exists {
		return nil
	}
	m.transitions = append(m.transitions, stateTransition{
		taskID:    taskID,
		fromState: fromState,
		toState:   toState,
		reason:    reason,
	})
	task.State = toState
	m.tasks[taskID] = task
	return nil
}

func (m *mockRepo) AddTask(task TaskInfo) {
	m.tasks[task.ID] = task
}

func TestMergeQueueSortOrder(t *testing.T) {
	tasks := []TaskInfo{
		{ID: "task1", TopoRank: 1, CreatedAt: time.Now().Add(1 * time.Hour)},
		{ID: "task2", TopoRank: 0, CreatedAt: time.Now().Add(2 * time.Hour)},
		{ID: "task3", TopoRank: 1, CreatedAt: time.Now()},
	}

	sortTasks(tasks)

	if tasks[0].ID != "task2" {
		t.Errorf("Expected task2 first (rank 0), got %s", tasks[0].ID)
	}
	if tasks[1].ID != "task3" {
		t.Errorf("Expected task3 second (rank 1, earlier), got %s", tasks[1].ID)
	}
	if tasks[2].ID != "task1" {
		t.Errorf("Expected task1 third (rank 1, later), got %s", tasks[2].ID)
	}
}

func sortTasks(tasks []TaskInfo) {
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].TopoRank != tasks[j].TopoRank {
			return tasks[i].TopoRank < tasks[j].TopoRank
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
}

func TestMergeQueueDependencyCheck(t *testing.T) {
	repo := newMockRepo()

	repo.AddTask(TaskInfo{
		ID:        "task1",
		State:     "verified",
		TopoRank:  0,
		CreatedAt: time.Now(),
	})
	repo.AddTask(TaskInfo{
		ID:        "task2",
		State:     "verified",
		TopoRank:  1,
		CreatedAt: time.Now(),
	})

	tasks, err := repo.GetVerifiedTasksReadyForMerge(context.Background())
	if err != nil {
		t.Fatalf("Failed to get verified tasks: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 verified tasks, got %d", len(tasks))
	}
}

func TestCopyArtifactsToMainCheckoutCopiesNestedFiles(t *testing.T) {
	mainRepo := t.TempDir()
	artifactDir := filepath.Join(t.TempDir(), "artifacts")
	nestedDir := filepath.Join(artifactDir, "nested")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create artifact dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(artifactDir, "root.txt"), []byte("root"), 0644); err != nil {
		t.Fatalf("failed to write artifact root file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "child.txt"), []byte("child"), 0644); err != nil {
		t.Fatalf("failed to write artifact nested file: %v", err)
	}

	mq := NewMergeQueue(newMockRepo(), mainRepo, zerolog.Nop())
	err := mq.copyArtifactsToMainCheckout(TaskInfo{
		ID:           "task-copy",
		ArtifactPath: artifactDir,
	})
	if err != nil {
		t.Fatalf("copyArtifactsToMainCheckout failed: %v", err)
	}

	rootContent, err := os.ReadFile(filepath.Join(mainRepo, "root.txt"))
	if err != nil {
		t.Fatalf("failed to read copied root file: %v", err)
	}
	if string(rootContent) != "root" {
		t.Fatalf("expected root content to be copied, got %q", string(rootContent))
	}

	childContent, err := os.ReadFile(filepath.Join(mainRepo, "nested", "child.txt"))
	if err != nil {
		t.Fatalf("failed to read copied nested file: %v", err)
	}
	if string(childContent) != "child" {
		t.Fatalf("expected nested content to be copied, got %q", string(childContent))
	}
}

func TestCopyArtifactsToMainCheckoutRejectsEmptyDirectory(t *testing.T) {
	mainRepo := t.TempDir()
	artifactDir := t.TempDir()

	mq := NewMergeQueue(newMockRepo(), mainRepo, zerolog.Nop())
	err := mq.copyArtifactsToMainCheckout(TaskInfo{
		ID:           "task-empty",
		ArtifactPath: artifactDir,
	})
	if err == nil {
		t.Fatal("expected empty artifact directory to fail")
	}
}

func TestExecuteMergeTransitionsVerifiedToApplyFailedWhenCopyFails(t *testing.T) {
	repo := newMockRepo()
	repo.AddTask(TaskInfo{
		ID:           "task-missing-artifacts",
		State:        "verified",
		TopoRank:     0,
		CreatedAt:    time.Now(),
		ArtifactPath: filepath.Join(t.TempDir(), "missing"),
	})

	mq := NewMergeQueue(repo, t.TempDir(), zerolog.Nop())
	err := mq.executeMerge(context.Background(), repo.tasks["task-missing-artifacts"])
	if err == nil {
		t.Fatal("expected executeMerge to fail when artifacts are missing")
	}

	if repo.tasks["task-missing-artifacts"].State != "apply_failed" {
		t.Fatalf("expected task state apply_failed, got %q", repo.tasks["task-missing-artifacts"].State)
	}
	if len(repo.transitions) != 1 || repo.transitions[0].fromState != "verified" || repo.transitions[0].toState != "apply_failed" {
		t.Fatalf("expected single verified->apply_failed transition, got %+v", repo.transitions)
	}
}

func TestExecuteMergeCopiesArtifactsAndCommitsBeforeDone(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	mainRepo := t.TempDir()
	runGit(t, mainRepo, "init")
	runGit(t, mainRepo, "config", "user.email", "orchestrator@example.com")
	runGit(t, mainRepo, "config", "user.name", "orchestrator-bot")

	artifactDir := filepath.Join(t.TempDir(), "artifacts")
	if err := os.MkdirAll(filepath.Join(artifactDir, "dir"), 0755); err != nil {
		t.Fatalf("failed to create artifact dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(artifactDir, "dir", "merged.txt"), []byte("merged-content"), 0644); err != nil {
		t.Fatalf("failed to write artifact file: %v", err)
	}

	repo := newMockRepo()
	task := TaskInfo{
		ID:           "task-merge-success",
		State:        "verified",
		TopoRank:     0,
		CreatedAt:    time.Now(),
		ArtifactPath: artifactDir,
	}
	repo.AddTask(task)

	mq := NewMergeQueue(repo, mainRepo, zerolog.Nop())
	if err := mq.executeMerge(context.Background(), task); err != nil {
		t.Fatalf("executeMerge failed: %v", err)
	}

	if repo.tasks[task.ID].State != "done" {
		t.Fatalf("expected task state done, got %q", repo.tasks[task.ID].State)
	}
	if len(repo.transitions) != 2 {
		t.Fatalf("expected verified->merged->done transitions, got %+v", repo.transitions)
	}
	if repo.transitions[0].fromState != "verified" || repo.transitions[0].toState != "merged" {
		t.Fatalf("expected first transition verified->merged, got %+v", repo.transitions[0])
	}
	if repo.transitions[1].fromState != "merged" || repo.transitions[1].toState != "done" {
		t.Fatalf("expected second transition merged->done, got %+v", repo.transitions[1])
	}

	mergedContent, err := os.ReadFile(filepath.Join(mainRepo, "dir", "merged.txt"))
	if err != nil {
		t.Fatalf("failed to read merged file: %v", err)
	}
	if string(mergedContent) != "merged-content" {
		t.Fatalf("expected merged artifact content, got %q", string(mergedContent))
	}

	logOutput, err := exec.Command("git", "-C", mainRepo, "log", "--oneline", "-1").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to inspect git log: %v (output: %s)", err, string(logOutput))
	}
	if !strings.Contains(string(logOutput), "Merge task task-merge-success") {
		t.Fatalf("expected merge commit message, got %q", string(logOutput))
	}
}

func TestExecuteMergeTreatsNothingToCommitAsSuccess(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	mainRepo := t.TempDir()
	runGit(t, mainRepo, "init")
	runGit(t, mainRepo, "config", "user.email", "orchestrator@example.com")
	runGit(t, mainRepo, "config", "user.name", "orchestrator-bot")

	existingPath := filepath.Join(mainRepo, "dir", "same.txt")
	if err := os.MkdirAll(filepath.Dir(existingPath), 0755); err != nil {
		t.Fatalf("failed to create destination dir: %v", err)
	}
	if err := os.WriteFile(existingPath, []byte("same-content"), 0644); err != nil {
		t.Fatalf("failed to write existing file: %v", err)
	}
	runGit(t, mainRepo, "add", ".")
	runGit(t, mainRepo, "commit", "-m", "initial state")

	artifactDir := filepath.Join(t.TempDir(), "artifacts")
	if err := os.MkdirAll(filepath.Join(artifactDir, "dir"), 0755); err != nil {
		t.Fatalf("failed to create artifact dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(artifactDir, "dir", "same.txt"), []byte("same-content"), 0644); err != nil {
		t.Fatalf("failed to write artifact file: %v", err)
	}

	repo := newMockRepo()
	task := TaskInfo{
		ID:           "task-noop-merge",
		State:        "verified",
		TopoRank:     0,
		CreatedAt:    time.Now(),
		ArtifactPath: artifactDir,
	}
	repo.AddTask(task)

	mq := NewMergeQueue(repo, mainRepo, zerolog.Nop())
	if err := mq.executeMerge(context.Background(), task); err != nil {
		t.Fatalf("executeMerge failed: %v", err)
	}

	if repo.tasks[task.ID].State != "done" {
		t.Fatalf("expected task state done, got %q", repo.tasks[task.ID].State)
	}
	if len(repo.transitions) != 2 {
		t.Fatalf("expected verified->merged->done transitions, got %+v", repo.transitions)
	}

	statusOutput, err := exec.Command("git", "-C", mainRepo, "status", "--short").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to inspect git status: %v (output: %s)", err, string(statusOutput))
	}
	if strings.TrimSpace(string(statusOutput)) != "" {
		t.Fatalf("expected clean working tree after noop merge, got %q", string(statusOutput))
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v (output: %s)", args, err, string(output))
	}
}
