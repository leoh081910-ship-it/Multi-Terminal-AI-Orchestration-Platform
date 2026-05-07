// Package router provides intelligent task-to-agent routing.
// Enhanced capability matching using Registry CapabilityManifests.
package router

import (
	"context"
	"strings"

	"github.com/mCP-DevOS/ai-orchestration-platform/internal/registry"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/runner"
)

// EnhancedCapabilityMatcher scores agents based on their CapabilityManifest
// from the Registry, rather than the flat specialties string array. It evaluates
// multiple dimensions: task type coverage, model family, context window fit,
// streaming support, latency, and cost.
type EnhancedCapabilityMatcher struct {
	registry *registry.RunnerRegistry
}

func NewEnhancedCapabilityMatcher(r *registry.RunnerRegistry) *EnhancedCapabilityMatcher {
	return &EnhancedCapabilityMatcher{registry: r}
}

// ScoreV2 returns a multi-dimensional match score 0.0-1.0 for an agent.
// It uses the agent's CapabilityManifest from the registry.
// For agents not in the registry, falls back to zero score.
func (m *EnhancedCapabilityMatcher) ScoreV2(ctx context.Context, agentID, taskType string, taskCapabilities []string) float64 {
	if taskType == "" {
		return 0.5
	}

	manifest, err := m.registry.Manifest(ctx, agentID)
	if err != nil {
		// Agent not in registry — use low score
		return 0.1
	}

	// === Dimension 1: Task Type Coverage (weight: 50%) ===
	typeScore := m.scoreTaskType(manifest, taskType)

	// === Dimension 2: Context Window Fit (weight: 20%) ===
	// If task has an estimated input size, check against manifest's MaxInputSize
	// For now, use a heuristic: if manifest.ContextWindow > 0, give bonus
	ctxScore := m.scoreContextWindow(manifest, taskCapabilities)

	// === Dimension 3: Feature Support (weight: 20%) ===
	featureScore := m.scoreFeatures(manifest, taskCapabilities)

	// === Dimension 4: Cost/Latency Efficiency (weight: 10%) ===
	efficiencyScore := m.scoreEfficiency(manifest)

	return typeScore*0.5 + ctxScore*0.2 + featureScore*0.2 + efficiencyScore*0.1
}

// scoreTaskType evaluates how well the agent's TaskTypes match the requested task type.
// 1.0 = exact match, 0.7 = partial (contains), 0.4 = all-types fallback, 0.1 = no match
func (m *EnhancedCapabilityMatcher) scoreTaskType(manifest *runner.CapabilityManifest, taskType string) float64 {
	if manifest == nil || len(manifest.TaskTypes) == 0 {
		return 0.4 // all types
	}

	lt := strings.ToLower(taskType)

	// Exact match
	for _, t := range manifest.TaskTypes {
		if strings.ToLower(t) == lt {
			return 1.0
		}
	}

	// Category match (e.g., "code-review" matches "code")
	// Group task types into categories
	categories := map[string][]string{
		"feature":  {"feature", "coding", "development", "code"},
		"bugfix":   {"bugfix", "fix", "repair", "debug"},
		"refactor": {"refactor", "restructure", "cleanup"},
		"code-review": {"code-review", "review", "audit"},
		"documentation": {"documentation", "docs", "readme"},
		"analysis": {"analysis", "analyze", "research"},
		"reverse": {"reverse", "reversing", "static-analysis"},
	}

	for _, t := range manifest.TaskTypes {
		lt2 := strings.ToLower(t)
		if cats, ok := categories[lt2]; ok {
			for _, c := range cats {
				if c == lt {
					return 0.8
				}
			}
		}
	}

	// Partial contains match
	for _, t := range manifest.TaskTypes {
		lt2 := strings.ToLower(t)
		if strings.Contains(lt2, lt) || strings.Contains(lt, lt2) {
			return 0.6
		}
	}

	// All types means they handle anything
	return 0.4
}

// scoreContextWindow evaluates the agent's context window relative to task needs.
func (m *EnhancedCapabilityMatcher) scoreContextWindow(manifest *runner.CapabilityManifest, taskCapabilities []string) float64 {
	if manifest == nil {
		return 0.3
	}

	// If no context window specified, give neutral score
	if manifest.ContextWindow == 0 {
		return 0.5
	}

	// For now, higher context window is always better
	// CLI agents (Claude Code) get 200K default
	if manifest.ContextWindow >= 200000 {
		return 1.0
	}
	if manifest.ContextWindow >= 100000 {
		return 0.8
	}
	if manifest.ContextWindow >= 32000 {
		return 0.6
	}
	if manifest.ContextWindow >= 8000 {
		return 0.4
	}
	return 0.2
}

// scoreFeatures evaluates optional feature support.
// Checks SupportsThinking, SupportsMultimodal, SupportsStreaming.
func (m *EnhancedCapabilityMatcher) scoreFeatures(manifest *runner.CapabilityManifest, taskCapabilities []string) float64 {
	if manifest == nil {
		return 0.3
	}

	score := 0.0
	count := 0

	// If task needs thinking (complex analysis), check SupportsThinking
	for _, cap := range taskCapabilities {
		lower := strings.ToLower(cap)
		if strings.Contains(lower, "thinking") || strings.Contains(lower, "reasoning") {
			count++
			if manifest.SupportsThinking {
				score += 0.4
			}
		}
		if strings.Contains(lower, "multimodal") || strings.Contains(lower, "image") {
			count++
			if manifest.SupportsMultimodal {
				score += 0.3
			}
		}
		if strings.Contains(lower, "streaming") || strings.Contains(lower, "realtime") {
			count++
			if manifest.SupportsStreaming {
				score += 0.3
			}
		}
	}

	if count == 0 {
		// No specific feature requirements — use base capability score
		base := 0.3
		if manifest.SupportsThinking {
			base += 0.15
		}
		if manifest.SupportsMultimodal {
			base += 0.15
		}
		if manifest.SupportsStreaming {
			base += 0.1
		}
		return base
	}

	return score / float64(count) * 1.0
}

// scoreEfficiency evaluates cost and latency efficiency.
// Lower cost and latency = higher score.
func (m *EnhancedCapabilityMatcher) scoreEfficiency(manifest *runner.CapabilityManifest) float64 {
	if manifest == nil {
		return 0.5 // unknown = neutral
	}

	score := 0.5 // base score

	// Latency bonus
	if manifest.LatencyP50 > 0 {
		if manifest.LatencyP50 < 500 {
			score += 0.3
		} else if manifest.LatencyP50 < 2000 {
			score += 0.2
		} else if manifest.LatencyP50 < 5000 {
			score += 0.1
		}
		// > 5s = no bonus
	}

	// Cost penalty — higher cost = lower score
	// Use combined input+output cost per 1K tokens
	if manifest.CostPer1KInput > 0 || manifest.CostPer1KOutput > 0 {
		totalCost := manifest.CostPer1KInput + manifest.CostPer1KOutput
		if totalCost < 1.0 { // < $0.001 per token
			score += 0.2
		} else if totalCost < 5.0 {
			score += 0.1
		} else if totalCost > 20.0 {
			score -= 0.2
		}
	}

	// CLI runners are free (no API cost)
	if manifest.Runtime == "cli" {
		score += 0.1
	}

	// Clamp to 0.0-1.0
	if score > 1.0 {
		score = 1.0
	}
	if score < 0 {
		score = 0.1
	}

	return score
}

// BatchScore returns scores for multiple agents in parallel.
func (m *EnhancedCapabilityMatcher) BatchScore(ctx context.Context, agentIDs []string, taskType string, taskCapabilities []string) map[string]float64 {
	scores := make(map[string]float64, len(agentIDs))
	for _, id := range agentIDs {
		scores[id] = m.ScoreV2(ctx, id, taskType, taskCapabilities)
	}
	return scores
}