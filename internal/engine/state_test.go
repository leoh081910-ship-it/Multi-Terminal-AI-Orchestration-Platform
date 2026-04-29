package engine

import (
	"testing"
)

func TestAllStates(t *testing.T) {
	states := AllStates()
	if len(states) != 14 {
		t.Errorf("Expected 14 states, got %d", len(states))
	}
}

func TestValidateTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		wantErr bool
	}{
		{
			name:    "queued to routed is valid",
			from:    StateQueued,
			to:      StateRouted,
			wantErr: false,
		},
		{
			name:    "done to routed is invalid (terminal)",
			from:    StateDone,
			to:      StateRouted,
			wantErr: true,
		},
		{
			name:    "queued to done is invalid",
			from:    StateQueued,
			to:      StateDone,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransition(tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTransition(%q, %q) error = %v, wantErr %v",
					tt.from, tt.to, err, tt.wantErr)
			}
		})
	}
}

func TestIsTerminal(t *testing.T) {
	tests := []struct {
		state    string
		expected bool
	}{
		{StateDone, true},
		{StateFailed, false}, // PR-3: recoverable via recovery path
		{StateQueued, false},
		{StateRunning, false},
		{StateVerified, false},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			result := IsTerminal(tt.state)
			if result != tt.expected {
				t.Errorf("IsTerminal(%q) = %v, want %v", tt.state, result, tt.expected)
			}
		})
	}
}

func TestCanRetry(t *testing.T) {
	tests := []struct {
		state    string
		expected bool
	}{
		{StateRunning, true},
		{StatePatchReady, false},
		{StateVerifyFailed, true},
		{StateVerified, false},
		{StateQueued, false},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			result := CanRetry(tt.state)
			if result != tt.expected {
				t.Errorf("CanRetry(%q) = %v, want %v", tt.state, result, tt.expected)
			}
		})
	}
}
