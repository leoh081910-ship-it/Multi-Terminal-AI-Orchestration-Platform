// Package mergequeue implements the merge queue for verified tasks.
// It provides single-consumer serial processing with dependency-aware ordering.
package mergequeue

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// TaskInfo represents minimal task information needed for merge queue.
type TaskInfo struct {
	ID           string    `json:"id"`
	DispatchRef  string    `json:"dispatch_ref"`
	State        string    `json:"state"`
	TopoRank     int       `json:"topo_rank"`
	CreatedAt    time.Time `json:"created_at"`
	ArtifactPath string    `json:"artifact_path"`
}

// MergeQueue manages the serial merge process for verified tasks.
type MergeQueue struct {
	// repo provides access to task data
	repo Repository
	// mainRepoPath is the path to the main git repository
	mainRepoPath string
	// logger logs merge operations
	logger zerolog.Logger
	// running indicates if the queue processor is running
	running bool
	// stopCh signals the processor to stop
	stopCh chan struct{}
}

// Repository defines the interface for task data access.
type Repository interface {
	// GetVerifiedTasksReadyForMerge returns verified tasks whose dependencies are all done
	GetVerifiedTasksReadyForMerge(ctx context.Context) ([]TaskInfo, error)
	// GetTaskDependencies returns the IDs of tasks that the given task depends on
	GetTaskDependencies(ctx context.Context, taskID string) ([]string, error)
	// CheckTasksInState checks if all given task IDs are in the specified state
	CheckTasksInState(ctx context.Context, taskIDs []string, state string) (bool, error)
	// UpdateTaskState updates the task state with event logging
	UpdateTaskState(ctx context.Context, taskID, fromState, toState, reason string) error
}

// NewMergeQueue creates a new merge queue.
func NewMergeQueue(repo Repository, mainRepoPath string, logger zerolog.Logger) *MergeQueue {
	return &MergeQueue{
		repo:         repo,
		mainRepoPath: mainRepoPath,
		logger:       logger,
		running:      false,
		stopCh:       make(chan struct{}),
	}
}

// Start starts the merge queue processor.
func (mq *MergeQueue) Start(ctx context.Context) error {
	if mq.running {
		return fmt.Errorf("merge queue already running")
	}

	mq.running = true
	mq.logger.Info().Msg("Merge queue processor started")

	// Start the processor loop
	go mq.processorLoop(ctx)

	return nil
}

// Stop stops the merge queue processor.
func (mq *MergeQueue) Stop() error {
	if !mq.running {
		return nil
	}

	close(mq.stopCh)
	mq.running = false
	mq.logger.Info().Msg("Merge queue processor stopped")

	return nil
}

// processorLoop is the main processing loop for the merge queue.
// REQUIRES (MERG-01~02): Single-consumer serial processing, ordered by topo_rank asc, then created_at asc.
func (mq *MergeQueue) processorLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			mq.logger.Info().Msg("Merge queue processor context cancelled")
			return
		case <-mq.stopCh:
			mq.logger.Info().Msg("Merge queue processor stopped")
			return
		case <-ticker.C:
			if err := mq.processNextTask(ctx); err != nil {
				mq.logger.Error().Err(err).Msg("Failed to process next task")
			}
		}
	}
}

// processNextTask processes the next available task in the queue.
// REQUIRES (MERG-03): Only consume tasks with all dependencies in 'done' state.
func (mq *MergeQueue) processNextTask(ctx context.Context) error {
	// Get verified tasks ready for merge
	tasks, err := mq.repo.GetVerifiedTasksReadyForMerge(ctx)
	if err != nil {
		return fmt.Errorf("failed to get verified tasks: %w", err)
	}

	if len(tasks) == 0 {
		return nil // No tasks to process
	}

	// Sort tasks by topo_rank asc, then created_at asc
	// REQUIRES (MERG-02): Order by topo_rank asc, then created_at asc.
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].TopoRank != tasks[j].TopoRank {
			return tasks[i].TopoRank < tasks[j].TopoRank
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})

	// Process the first task (single-consumer serial processing)
	task := tasks[0]

	mq.logger.Info().
		Str("task_id", task.ID).
		Int("topo_rank", task.TopoRank).
		Msg("Processing task for merge")

	// Verify dependencies are all done
	deps, err := mq.repo.GetTaskDependencies(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("failed to get task dependencies: %w", err)
	}

	if len(deps) > 0 {
		allDone, err := mq.repo.CheckTasksInState(ctx, deps, "done")
		if err != nil {
			return fmt.Errorf("failed to check dependency states: %w", err)
		}
		if !allDone {
			mq.logger.Info().Str("task_id", task.ID).Msg("Dependencies not all done, skipping")
			return nil
		}
	}

	// Execute merge
	if err := mq.executeMerge(ctx, task); err != nil {
		return fmt.Errorf("merge execution failed: %w", err)
	}

	return nil
}

// executeMerge performs the actual merge operation.
// REQUIRES (MERG-04): Copy artifacts to main checkout, run 'git add' and 'git commit'.
func (mq *MergeQueue) executeMerge(ctx context.Context, task TaskInfo) error {
	mq.logger.Info().Str("task_id", task.ID).Msg("Executing merge")

	// Copy artifacts to main checkout
	if err := mq.copyArtifactsToMainCheckout(task); err != nil {
		if updateErr := mq.repo.UpdateTaskState(ctx, task.ID, "verified", "apply_failed", err.Error()); updateErr != nil {
			mq.logger.Error().Err(updateErr).Str("task_id", task.ID).Msg("Failed to update task state to apply_failed")
		}
		return fmt.Errorf("failed to copy artifacts: %w", err)
	}

	// Run git add
	if err := mq.runGitCommand("add", "."); err != nil {
		if updateErr := mq.repo.UpdateTaskState(ctx, task.ID, "verified", "apply_failed", err.Error()); updateErr != nil {
			mq.logger.Error().Err(updateErr).Str("task_id", task.ID).Msg("Failed to update task state to apply_failed")
		}
		return fmt.Errorf("failed to run git add: %w", err)
	}

	// Run git commit
	commitMsg := fmt.Sprintf("Merge task %s", task.ID)
	if err := mq.runGitCommand("commit", "-m", commitMsg); err != nil {
		if !isGitNoChangesError(err) {
			if updateErr := mq.repo.UpdateTaskState(ctx, task.ID, "verified", "apply_failed", err.Error()); updateErr != nil {
				mq.logger.Error().Err(updateErr).Str("task_id", task.ID).Msg("Failed to update task state to apply_failed")
			}
			return fmt.Errorf("failed to run git commit: %w", err)
		}
		mq.logger.Info().
			Str("task_id", task.ID).
			Msg("Merge produced no git diff; treating as successful no-op")
	}

	// Update task state to 'merged' only after commit succeeds.
	if err := mq.repo.UpdateTaskState(ctx, task.ID, "verified", "merged", ""); err != nil {
		return fmt.Errorf("failed to update task state to merged: %w", err)
	}

	// Update task state to 'done'
	if err := mq.repo.UpdateTaskState(ctx, task.ID, "merged", "done", ""); err != nil {
		return fmt.Errorf("failed to update task state to done: %w", err)
	}

	mq.logger.Info().Str("task_id", task.ID).Msg("Merge completed successfully")

	return nil
}

func isGitNoChangesError(err error) bool {
	if err == nil {
		return false
	}

	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "nothing to commit") ||
		strings.Contains(lower, "working tree clean")
}

// copyArtifactsToMainCheckout copies task artifacts to the main git repository.
func (mq *MergeQueue) copyArtifactsToMainCheckout(task TaskInfo) error {
	if strings.TrimSpace(task.ArtifactPath) == "" {
		return fmt.Errorf("artifact_path is required")
	}

	artifactRoot := filepath.Clean(task.ArtifactPath)
	info, err := os.Stat(artifactRoot)
	if err != nil {
		return fmt.Errorf("failed to stat artifact path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("artifact_path must be a directory: %s", artifactRoot)
	}

	copiedFiles := 0
	err = filepath.WalkDir(artifactRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == artifactRoot {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink artifacts are not supported: %s", path)
		}

		relPath, err := filepath.Rel(artifactRoot, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path: %w", err)
		}
		if relPath == "." || filepath.IsAbs(relPath) || strings.HasPrefix(relPath, "..") {
			return fmt.Errorf("invalid artifact relative path: %s", relPath)
		}

		destPath := filepath.Join(mq.mainRepoPath, relPath)
		if d.IsDir() {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return fmt.Errorf("failed to create destination directory %s: %w", destPath, err)
			}
			return nil
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory %s: %w", filepath.Dir(destPath), err)
		}
		if err := copyFile(path, destPath); err != nil {
			return err
		}
		copiedFiles++
		return nil
	})
	if err != nil {
		return err
	}

	if copiedFiles == 0 {
		return fmt.Errorf("no artifact files found in %s", artifactRoot)
	}

	return nil
}

// runGitCommand runs a git command in the main repository.
func (mq *MergeQueue) runGitCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = mq.mainRepoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git command failed: %w (output: %s)", err, output)
	}
	return nil
}

func copyFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", srcPath, err)
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file %s: %w", srcPath, err)
	}

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dstPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy file %s -> %s: %w", srcPath, dstPath, err)
	}

	return nil
}
