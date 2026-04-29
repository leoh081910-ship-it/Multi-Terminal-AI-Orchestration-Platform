package server

import (
	"context"
	"fmt"

	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
)

const (
	// MaxAutoRepairs is the maximum number of auto-generated remediation tasks per root task.
	// PRD §11 rule 1: "同一 root task 自动补救次数上限 2"
	MaxAutoRepairs = 2

	// MaxAutoRetries is the maximum number of automatic retries for an original task.
	// PRD §11 rule 3: "同一原任务最多自动重试 3 次"
	MaxAutoRetries = 3
)

// FailurePolicy enforces stop-loss rules per PRD §11.
type FailurePolicy struct {
	repo taskLister
}

type taskLister interface {
	ListAllTasks(ctx context.Context) ([]*ent.Task, error)
}

// NewFailurePolicy creates a new FailurePolicy.
func NewFailurePolicy(repo taskLister) *FailurePolicy {
	return &FailurePolicy{repo: repo}
}

// CheckRepairLimit checks if the root task has exceeded the auto-repair limit.
// PRD §11 rule 1: same root task max 2 auto repairs.
func (p *FailurePolicy) CheckRepairLimit(ctx context.Context, rootTaskID string) (int, bool, error) {
	tasks, err := p.repo.ListAllTasks(ctx)
	if err != nil {
		return 0, false, fmt.Errorf("failed to list tasks for repair limit check: %w", err)
	}

	repairCount := 0
	for _, t := range tasks {
		payload := decodeCompatPayload(t.CardJSON)
		rootID := readString(payload, "root_task_id")
		if rootID != rootTaskID {
			continue
		}
		parentID := readString(payload, "parent_task_id")
		if parentID == "" {
			continue // not a derived task
		}
		taskType := readString(payload, "type")
		if IsSystemTaskType(taskType) {
			repairCount++
		}
	}

	return repairCount, repairCount < MaxAutoRepairs, nil
}

// CheckRetryLimit checks if the original task has exceeded the auto-retry limit.
// PRD §11 rule 3: same task max 3 auto retries.
func (p *FailurePolicy) CheckRetryLimit(retryCount int) bool {
	return retryCount < MaxAutoRetries
}

// CheckSignatureDuplication checks if a remediation task for this failure signature already exists.
// PRD §11 rule 2: "同一 failure_signature 不重复派生"
// excludeTaskID is the task currently being triaged — it must be excluded to avoid self-matching.
func (p *FailurePolicy) CheckSignatureDuplication(ctx context.Context, rootTaskID, signature, excludeTaskID string) (bool, error) {
	tasks, err := p.repo.ListAllTasks(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to list tasks for signature dedup: %w", err)
	}

	for _, t := range tasks {
		// Never match the task being triaged against itself
		if t.ID == excludeTaskID {
			continue
		}
		payload := decodeCompatPayload(t.CardJSON)
		rootID := readString(payload, "root_task_id")
		if rootID != rootTaskID {
			continue
		}
		// Only check derived (remediation) tasks, not the original
		parentID := readString(payload, "parent_task_id")
		if parentID == "" {
			continue
		}
		existingSig := readString(payload, "failure_signature")
		if existingSig == signature {
			// Found existing derived task with same signature
			// If it's terminal (done/verified/failed/verify_failed/apply_failed), we can create a new one
			if t.State == engine.StateDone || t.State == engine.StateVerified || t.State == engine.StateFailed ||
				t.State == engine.StateVerifyFailed || t.State == engine.StateApplyFailed {
				continue
			}
			// If it's still active, don't duplicate
			return true, nil
		}
	}

	return false, nil
}

// ShouldStopLoss determines if a task should be stopped (moved to blocked/failed)
// based on all stop-loss rules.
func (p *FailurePolicy) ShouldStopLoss(ctx context.Context, task *ent.Task) (bool, string) {
	payload := decodeCompatPayload(task.CardJSON)
	rootTaskID := readString(payload, "root_task_id")
	if rootTaskID == "" {
		rootTaskID = task.ID
	}

	// Rule 3: retry limit
	retryCount := task.RetryCount
	if !p.CheckRetryLimit(retryCount) {
		return true, fmt.Sprintf("auto-retry limit reached (%d >= %d)", retryCount, MaxAutoRetries)
	}

	// Rule 1: repair limit
	repairCount, canRepair, err := p.CheckRepairLimit(ctx, rootTaskID)
	if err == nil && !canRepair {
		return true, fmt.Sprintf("auto-repair limit reached (%d >= %d)", repairCount, MaxAutoRepairs)
	}

	return false, ""
}
