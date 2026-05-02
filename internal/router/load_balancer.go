package router

import (
	"context"
	"sync"
	"time"

	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/task"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
)

// LoadBalancer tracks active tasks per agent and scores agents
// inversely to their current load.
type LoadBalancer struct {
	client *ent.Client
	cache  sync.Map // agentID -> cachedLoad
}

type cachedLoad struct {
	count     int
	updatedAt time.Time
}

func NewLoadBalancer(client *ent.Client) *LoadBalancer {
	return &LoadBalancer{client: client}
}

// Score returns 0.0-1.0 where 1.0 means the agent has zero load.
func (lb *LoadBalancer) Score(ctx context.Context, agentID string) float64 {
	count := lb.activeTaskCount(ctx, agentID)

	// Score: 1.0 for 0 tasks, decreasing as load increases
	// Cap at 10 concurrent tasks
	switch {
	case count == 0:
		return 1.0
	case count == 1:
		return 0.9
	case count <= 3:
		return 0.7
	case count <= 5:
		return 0.5
	case count <= 8:
		return 0.3
	default:
		return 0.1
	}
}

// ActiveTaskCount returns the number of active tasks for an agent.
func (lb *LoadBalancer) ActiveTaskCount(ctx context.Context, agentID string) int {
	return lb.activeTaskCount(ctx, agentID)
}

func (lb *LoadBalancer) activeTaskCount(ctx context.Context, agentID string) int {
	// Check cache (valid for 5 seconds)
	if v, ok := lb.cache.Load(agentID); ok {
		c := v.(cachedLoad)
		if time.Since(c.updatedAt) < 5*time.Second {
			return c.count
		}
	}

	activeStates := []string{
		engine.StateRouted,
		engine.StateWorkspacePrepared,
		engine.StateRunning,
		engine.StatePatchReady,
		engine.StateReviewPending,
		engine.StateTriage,
		engine.StateDecomposing,
	}

	count, _ := lb.client.Task.Query().
	 Where(
		task.And(
			task.AssignedAgentID(agentID),
			task.StateIn(activeStates...),
		),
	 ).
	 Count(ctx)

	lb.cache.Store(agentID, cachedLoad{count: count, updatedAt: time.Now()})
	return count
}

// InvalidateCache clears the cached load for an agent.
func (lb *LoadBalancer) InvalidateCache(agentID string) {
	lb.cache.Delete(agentID)
}
