// Package engine provides the core orchestration engine for the AI orchestration platform.
// It implements the 12-state state machine, dependency management, wave coordination,
// retry logic, and task lifecycle management.
package engine

// Task states — 8 main states + 6 exception/coordination states.
const (
	// Main states.
	StateQueued            = "queued"
	StateRouted            = "routed"
	StateWorkspacePrepared = "workspace_prepared"
	StateRunning           = "running"
	StatePatchReady        = "patch_ready"
	StateVerified          = "verified"
	StateMerged            = "merged"
	StateDone              = "done"

	// Exception states.
	StateRetryWaiting = "retry_waiting"
	StateVerifyFailed = "verify_failed"
	StateApplyFailed  = "apply_failed"
	StateFailed       = "failed"

	// Coordination states (PRD-DA-001).
	StateTriage        = "triage"
	StateReviewPending = "review_pending"
)

// terminalStates are the only valid end states for a task.
// PR-3: failed is recoverable via recovery path (failed → retry_waiting),
// so it is removed from terminal states. Only done is truly terminal.
var terminalStates = map[string]bool{
	StateDone: true,
}

// validTransitions defines all legal state transitions per the PRD state diagram.
// Key = from state, Value = set of allowed to states.
var validTransitions = map[string]map[string]bool{
	StateQueued: {
		StateRouted: true,
	},
	StateRouted: {
		StateWorkspacePrepared: true,
	},
	StateWorkspacePrepared: {
		StateRunning: true,
	},
	StateRunning: {
		StatePatchReady:    true,
		StateRetryWaiting:  true,
		StateTriage:        true, // PRD-DA-001: failure enters triage
		StateReviewPending: true, // PRD-DA-001: success enters review
	},
	StateTriage: {
		StateRetryWaiting: true, // triage complete, remediation created
		StateFailed:       true, // non-retryable, give up
	},
	StateReviewPending: {
		StateVerified:     true, // review passed
		StateRetryWaiting: true, // review rejected, needs rework
	},
	StatePatchReady: {
		StateVerified:     true,
		StateVerifyFailed: true,
	},
	StateVerifyFailed: {
		StateRetryWaiting: true,
		StateFailed:       true,
	},
	StateVerified: {
		StateMerged:      true,
		StateApplyFailed: true,
	},
	StateMerged: {
		StateDone: true,
	},
	StateRetryWaiting: {
		StateRouted: true, // back to execution
	},
	StateApplyFailed: {
		StateFailed: true,
	},
	StateFailed: {
		StateRetryWaiting: true, // PR-3: recovery path for platform-defect blocked tasks
	},
}

// mainStates lists the 8 main states in execution order.
var mainStates = []string{
	StateQueued,
	StateRouted,
	StateWorkspacePrepared,
	StateRunning,
	StatePatchReady,
	StateVerified,
	StateMerged,
	StateDone,
}

// exceptionStates lists the 6 exception/coordination states.
var exceptionStates = []string{
	StateRetryWaiting,
	StateVerifyFailed,
	StateApplyFailed,
	StateFailed,
	StateTriage,
	StateReviewPending,
}

// AllStates returns all 12 state names.
func AllStates() []string {
	states := make([]string, 0, 12)
	states = append(states, mainStates...)
	states = append(states, exceptionStates...)
	return states
}

// ValidateTransition checks whether transitioning from `from` to `to` is legal.
// Returns nil if valid, an error otherwise.
func ValidateTransition(from, to string) error {
	if IsTerminal(from) {
		return &TerminalStateError{State: from}
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return &InvalidTransitionError{From: from, To: to}
	}

	if !allowed[to] {
		return &InvalidTransitionError{From: from, To: to}
	}

	return nil
}

// IsTerminal returns true if the state is a terminal state (done or failed).
func IsTerminal(state string) bool {
	return terminalStates[state]
}

// CanRetry returns true if a task in the given state can transition to retry_waiting.
func CanRetry(state string) bool {
	allowed, exists := validTransitions[state]
	if !exists {
		return false
	}
	return allowed[StateRetryWaiting]
}

// IsValidState checks if the given string is a recognized state.
func IsValidState(state string) bool {
	for _, s := range AllStates() {
		if s == state {
			return true
		}
	}
	return false
}

// TerminalStateError indicates an attempt to transition from a terminal state.
type TerminalStateError struct {
	State string
}

func (e *TerminalStateError) Error() string {
	return "cannot transition from terminal state: " + e.State
}

// InvalidTransitionError indicates an illegal state transition.
type InvalidTransitionError struct {
	From string
	To   string
}

func (e *InvalidTransitionError) Error() string {
	return "invalid state transition: " + e.From + " -> " + e.To
}
