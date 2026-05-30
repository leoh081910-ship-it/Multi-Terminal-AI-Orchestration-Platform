package server

import (
	"fmt"
	"strings"
)

const (
	// MaxAutoRepairCount is the maximum number of automatic rework cycles
	// before the system stops and escalates to manual triage.
	MaxAutoRepairCount = 2
)

// DefectTicket holds structured information extracted from a rejected review.
// It is embedded into rework tasks so the executor has full context for fixing.
type DefectTicket struct {
	ParentTaskID     string   `json:"parent_task_id"`
	ReviewTaskID     string   `json:"review_task_id"`
	RejectionReason  string   `json:"rejection_reason"`
	ArtifactPath     string   `json:"artifact_path"`
	OriginalGoal     string   `json:"original_goal"`
	OutputArtifacts  []string `json:"output_artifacts"`
	AcceptCriteria   []string `json:"acceptance_criteria"`
	RepairAttempt    int      `json:"repair_attempt"`
}

// buildReworkDescription produces a detailed rework task description that
// includes the full defect context — not just "fix issues found in review".
func buildReworkDescription(ticket DefectTicket) string {
	var b strings.Builder

	fmt.Fprintf(&b, "## Rework Task (Attempt %d/%d)\n\n", ticket.RepairAttempt, MaxAutoRepairCount)

	fmt.Fprintf(&b, "### Original Goal\n%s\n\n", firstNonEmpty(ticket.OriginalGoal, "No description provided."))

	fmt.Fprintf(&b, "### Defect Report\n%s\n\n", firstNonEmpty(ticket.RejectionReason, "No rejection reason provided."))

	if ticket.ArtifactPath != "" {
		fmt.Fprintf(&b, "### Artifact Location\n`%s`\n\n", ticket.ArtifactPath)
	}

	if len(ticket.OutputArtifacts) > 0 {
		fmt.Fprintf(&b, "### Required Output Artifacts\n")
		for _, a := range ticket.OutputArtifacts {
			fmt.Fprintf(&b, "- `%s`\n", a)
		}
		b.WriteString("\n")
	}

	if len(ticket.AcceptCriteria) > 0 {
		fmt.Fprintf(&b, "### Acceptance Criteria\n")
		for _, c := range ticket.AcceptCriteria {
			fmt.Fprintf(&b, "- %s\n", c)
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "### Instructions\n")
	fmt.Fprintf(&b, "1. Read the existing artifacts at the artifact path above.\n")
	fmt.Fprintf(&b, "2. Fix ALL issues mentioned in the Defect Report.\n")
	fmt.Fprintf(&b, "3. Ensure the output artifacts listed above are present and correct.\n")
	fmt.Fprintf(&b, "4. Do not introduce new issues while fixing existing ones.\n")

	return b.String()
}

// buildReworkTitle produces a descriptive rework task title.
func buildReworkTitle(parentTaskID string, attempt int, reason string) string {
	// Truncate reason to first 60 chars for title
	short := reason
	if len(short) > 60 {
		short = short[:57] + "..."
	}
	if short == "" {
		short = "review rejected"
	}
	return fmt.Sprintf("[rework #%d] %s: %s", attempt, parentTaskID, short)
}

// shouldBlockRework checks whether the rework cycle should be stopped.
// Returns (block, reason) — if block is true, the system should escalate
// to manual triage instead of creating another rework task.
func shouldBlockRework(payload map[string]interface{}, rejectionReason string) (bool, string) {
	// Check 1: auto_repair_count limit
	repairCount := readIntDefault(payload, 0, "auto_repair_count")
	if repairCount >= MaxAutoRepairCount {
		return true, fmt.Sprintf(
			"auto-repair limit reached (%d >= %d). Escalating to manual triage.",
			repairCount, MaxAutoRepairCount,
		)
	}

	// Check 2: duplicate rejection reason (same defect repeating)
	lastRejection := readString(payload, "last_rejection_reason")
	if lastRejection != "" && rejectionReason != "" {
		// Normalize and compare first 200 chars to detect repeated defects
		norm1 := normalizeForComparison(lastRejection)
		norm2 := normalizeForComparison(rejectionReason)
		if norm1 == norm2 {
			return true, fmt.Sprintf(
				"same rejection reason repeated. Defect persists after repair attempt %d. Escalating to manual triage.",
				repairCount,
			)
		}
	}

	return false, ""
}

// normalizeForComparison strips whitespace, lowercases, and truncates for dedup.
func normalizeForComparison(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	// Collapse multiple spaces
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}
	if len(text) > 200 {
		text = text[:200]
	}
	return text
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
