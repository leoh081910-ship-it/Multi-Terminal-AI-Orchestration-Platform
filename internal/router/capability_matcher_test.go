package router

import (
	"math"
	"testing"

	"github.com/mCP-DevOS/ai-orchestration-platform/internal/org"
)

func TestCapabilityMatcherScore(t *testing.T) {
	matcher := NewCapabilityMatcher()

	tests := []struct {
		name         string
		agent        *org.AgentView
		taskType     string
		capabilities []string
		expected     float64
	}{
		{
			name:     "empty task type returns neutral score",
			agent:    &org.AgentView{Type: "claude", Specialties: []string{"bugfix"}},
			expected: 0.5,
		},
		{
			name:     "exact specialty match wins",
			agent:    &org.AgentView{Type: "claude", Specialties: []string{"BugFix"}},
			taskType: "bugfix",
			expected: 1.0,
		},
		{
			name:     "partial specialty match beats type match",
			agent:    &org.AgentView{Type: "code-review", Specialties: []string{"code"}},
			taskType: "code-review",
			expected: 0.7,
		},
		{
			name:     "agent type match",
			agent:    &org.AgentView{Type: "Claude", Specialties: []string{"documentation"}},
			taskType: "claude",
			expected: 0.8,
		},
		{
			name:     "generic agent with no specialties",
			agent:    &org.AgentView{Type: "claude"},
			taskType: "analysis",
			expected: 0.4,
		},
		{
			name:         "capability overlap scores by matched ratio",
			agent:        &org.AgentView{Type: "claude", Specialties: []string{"go", "react"}},
			taskType:     "feature",
			capabilities: []string{"go", "sqlite"},
			expected:     0.55,
		},
		{
			name:         "full capability overlap",
			agent:        &org.AgentView{Type: "claude", Specialties: []string{"go", "sqlite"}},
			taskType:     "feature",
			capabilities: []string{"go", "sqlite"},
			expected:     0.8,
		},
		{
			name:         "no capability overlap falls back low",
			agent:        &org.AgentView{Type: "claude", Specialties: []string{"python"}},
			taskType:     "feature",
			capabilities: []string{"go"},
			expected:     0.1,
		},
		{
			name:     "no match falls back low",
			agent:    &org.AgentView{Type: "http", Specialties: []string{"translation"}},
			taskType: "reverse",
			expected: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matcher.Score(tt.agent, tt.taskType, tt.capabilities)
			if math.Abs(got-tt.expected) > 0.0001 {
				t.Fatalf("Score() = %v, want %v", got, tt.expected)
			}
		})
	}
}
