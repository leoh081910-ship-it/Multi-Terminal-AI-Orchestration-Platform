package server

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// FailureCode represents a structured failure classification per PRD §8.
type FailureCode string

const (
	FailureCodeArtifactMissing      FailureCode = "artifact_missing"
	FailureCodeArtifactSpecMismatch FailureCode = "artifact_spec_mismatch"
	FailureCodeGitNoChanges         FailureCode = "git_no_changes"
	FailureCodeCommandExitNonzero   FailureCode = "command_exit_nonzero"
	FailureCodeWorkspaceWriteFail   FailureCode = "workspace_write_failed"
	FailureCodeDependencyFailed     FailureCode = "dependency_failed"
	FailureCodeNonRetryable         FailureCode = "non_retryable_failure"
)

// RemediationTaskType maps failure codes to system task types per PRD §9.
type RemediationTaskType string

const (
	RemediationTypeArtifactFix  RemediationTaskType = "artifact-fix"
	RemediationTypeNoopReview   RemediationTaskType = "noop-review"
	RemediationTypeDebugFailure RemediationTaskType = "debug-failure"
	RemediationTypeCodeReview   RemediationTaskType = "code-review"
	RemediationTypeRework       RemediationTaskType = "rework"
	RemediationTypeTriage       RemediationTaskType = "triage-failure"
)

// StructuredFailure holds the classified failure information.
type StructuredFailure struct {
	FailureCode      FailureCode         `json:"failure_code"`
	FailureStage     string              `json:"error_stage"`
	Retryable        bool                `json:"retryable"`
	SuggestedAction  string              `json:"suggested_action"`
	RemediationType  RemediationTaskType `json:"remediation_type"`
	RemediationOwner string              `json:"remediation_owner"`
}

// ClassifyError maps a raw error message to a structured failure.
// Implements PRD §8.2 classification rules.
func ClassifyError(rawError string) StructuredFailure {
	lower := strings.ToLower(rawError)

	// Rule: empty_artifact_match / no files matched
	if strings.Contains(lower, "empty_artifact_match") ||
		strings.Contains(lower, "no files matched") {
		// Distinguish spec mismatch vs missing artifact
		if strings.Contains(lower, "path") || strings.Contains(lower, "spec") {
			return StructuredFailure{
				FailureCode:      FailureCodeArtifactSpecMismatch,
				FailureStage:     "artifact_verification",
				Retryable:        true,
				SuggestedAction:  "Fix artifact path configuration in task definition",
				RemediationType:  RemediationTypeArtifactFix,
				RemediationOwner: compatAgentCodex,
			}
		}
		return StructuredFailure{
			FailureCode:      FailureCodeArtifactMissing,
			FailureStage:     "artifact_verification",
			Retryable:        true,
			SuggestedAction:  "Regenerate missing artifacts",
			RemediationType:  RemediationTypeArtifactFix,
			RemediationOwner: compatAgentCodex,
		}
	}

	// Rule: nothing to commit, working tree clean
	if strings.Contains(lower, "nothing to commit") ||
		strings.Contains(lower, "working tree clean") {
		return StructuredFailure{
			FailureCode:      FailureCodeGitNoChanges,
			FailureStage:     "git_commit",
			Retryable:        true,
			SuggestedAction:  "Review if task is a no-op or needs different execution",
			RemediationType:  RemediationTypeNoopReview,
			RemediationOwner: compatAgentClaude,
		}
	}

	// Rule: command execution failed: exit status N
	if strings.Contains(lower, "exit status") ||
		strings.Contains(lower, "command execution failed") {
		return StructuredFailure{
			FailureCode:      FailureCodeCommandExitNonzero,
			FailureStage:     "execution",
			Retryable:        true,
			SuggestedAction:  "Debug command failure from execution logs",
			RemediationType:  RemediationTypeDebugFailure,
			RemediationOwner: compatAgentCodex,
		}
	}

	// Rule: workspace write failed
	if strings.Contains(lower, "workspace") && (strings.Contains(lower, "write") || strings.Contains(lower, "permission")) {
		return StructuredFailure{
			FailureCode:      FailureCodeWorkspaceWriteFail,
			FailureStage:     "workspace_preparation",
			Retryable:        true,
			SuggestedAction:  "Fix workspace permissions or path",
			RemediationType:  RemediationTypeDebugFailure,
			RemediationOwner: compatAgentCodex,
		}
	}

	// Rule: dependency failed
	if strings.Contains(lower, "dependency_failed") || strings.Contains(lower, "dependency failed") {
		return StructuredFailure{
			FailureCode:      FailureCodeDependencyFailed,
			FailureStage:     "dependency_check",
			Retryable:        false,
			SuggestedAction:  "Wait for dependency to recover, then retry",
			RemediationType:  RemediationTypeTriage,
			RemediationOwner: compatAgentCoordinator,
		}
	}

	// Default: non-retryable
	return StructuredFailure{
		FailureCode:      FailureCodeNonRetryable,
		FailureStage:     "unknown",
		Retryable:        false,
		SuggestedAction:  "Manual investigation required",
		RemediationType:  RemediationTypeDebugFailure,
		RemediationOwner: compatAgentCodex,
	}
}

// GenerateFailureSignature creates a unique hash for deduplication.
// Same root_task + same failure_code + same error pattern = same signature.
func GenerateFailureSignature(rootTaskID string, failureCode FailureCode, rawError string) string {
	// Normalize the error for signature stability
	normalized := strings.ToLower(strings.TrimSpace(rawError))
	// Truncate to first 256 chars to avoid overly long keys
	if len(normalized) > 256 {
		normalized = normalized[:256]
	}

	key := rootTaskID + "::" + string(failureCode) + "::" + normalized
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:12])
}

// System agent roles per PRD §6.2
const (
	compatAgentCoordinator = "Coordinator"
	compatAgentReviewer    = "Reviewer"
)

// System task types that should not be dispatched through normal execution
var systemTaskTypes = map[RemediationTaskType]bool{
	RemediationTypeArtifactFix:  true,
	RemediationTypeNoopReview:   true,
	RemediationTypeDebugFailure: true,
	RemediationTypeCodeReview:   true,
	RemediationTypeRework:       true,
	RemediationTypeTriage:       true,
}

// IsSystemTaskType checks if a task type is a system-generated coordination task.
func IsSystemTaskType(taskType string) bool {
	return systemTaskTypes[RemediationTaskType(taskType)]
}
