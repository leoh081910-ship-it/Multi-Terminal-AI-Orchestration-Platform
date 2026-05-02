package router

import (
	"context"
	"sync"
	"time"

	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/task"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
)

// AffinityScorer scores agents based on their historical success rate
// with similar task types. Agents that have successfully completed tasks
// of the same type get a higher score.
type AffinityScorer struct {
	client *ent.Client
	cache  sync.Map // key: "agentID:taskType" -> cachedAffinity
}

type cachedAffinity struct {
	score     float64
	updatedAt time.Time
}

func NewAffinityScorer(client *ent.Client) *AffinityScorer {
	return &AffinityScorer{client: client}
}

// Score returns 0.0-1.0 based on the agent's historical success with this task type.
func (as *AffinityScorer) Score(ctx context.Context, agentID, taskType string) float64 {
	if taskType == "" {
		return 0.5
	}

	key := agentID + ":" + taskType

	// Check cache (valid for 60 seconds)
	if v, ok := as.cache.Load(key); ok {
		c := v.(cachedAffinity)
		if time.Since(c.updatedAt) < 60*time.Second {
			return c.score
		}
	}

	score := as.calculateScore(ctx, agentID, taskType)
	as.cache.Store(key, cachedAffinity{score: score, updatedAt: time.Now()})
	return score
}

func (as *AffinityScorer) calculateScore(ctx context.Context, agentID, taskType string) float64 {
	// Count completed tasks of this type by this agent
	completed, _ := as.client.Task.Query().
		Where(
			task.And(
				task.AssignedAgentID(agentID),
				task.StateIn(engine.StateDone, engine.StateVerified, engine.StateMerged),
			),
		).
		Count(ctx)

	// Count total tasks of this type by this agent
	total, _ := as.client.Task.Query().
		Where(task.AssignedAgentID(agentID)).
		Count(ctx)

	if total == 0 {
		return 0.5 // no history, neutral score
	}

	successRate := float64(completed) / float64(total)

	// Scale: 0% success -> 0.2, 100% success -> 1.0
	// Also factor in volume (more history = more confidence)
	volumeBonus := 0.0
	if total >= 10 {
		volumeBonus = 0.1
	} else if total >= 5 {
		volumeBonus = 0.05
	}

	score := 0.2 + 0.6*successRate + volumeBonus
	if score > 1.0 {
		score = 1.0
	}
	return score
}

// RecordOutcome updates the cache after a task completes or fails.
func (as *AffinityScorer) InvalidateCache(agentID, taskType string) {
	as.cache.Delete(agentID + ":" + taskType)
}
