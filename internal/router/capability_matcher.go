// Package router provides intelligent task-to-agent routing.
package router

import (
	"strings"

	"github.com/mCP-DevOS/ai-orchestration-platform/internal/org"
)

// CapabilityMatcher scores agents based on how well their specialties
// match the task's type and required capabilities.
type CapabilityMatcher struct{}

func NewCapabilityMatcher() *CapabilityMatcher {
	return &CapabilityMatcher{}
}

// Score returns 0.0-1.0 indicating how well the agent matches the task.
// 1.0 = perfect match (specialty equals task type), 0.0 = no match.
func (m *CapabilityMatcher) Score(agent *org.AgentView, taskType string, taskCapabilities []string) float64 {
	if taskType == "" {
		return 0.5 // no type info, neutral score
	}

	lowerType := strings.ToLower(taskType)

	// Direct specialty match
	for _, s := range agent.Specialties {
		if strings.ToLower(s) == lowerType {
			return 1.0
		}
	}

	// Partial match (specialty contains task type or vice versa)
	for _, s := range agent.Specialties {
		lower := strings.ToLower(s)
		if strings.Contains(lower, lowerType) || strings.Contains(lowerType, lower) {
			return 0.7
		}
	}

	// Agent type match (e.g., "claude" agent for general tasks)
	if strings.ToLower(agent.Type) == lowerType {
		return 0.8
	}

	// Generic agents get a baseline score
	if len(agent.Specialties) == 0 {
		return 0.4
	}

	// Check required capabilities
	if len(taskCapabilities) > 0 {
		matched := 0
		for _, cap := range taskCapabilities {
			for _, s := range agent.Specialties {
				if strings.EqualFold(s, cap) {
					matched++
					break
				}
			}
		}
		if matched > 0 {
			return 0.3 + 0.5*(float64(matched)/float64(len(taskCapabilities)))
		}
	}

	return 0.1
}
