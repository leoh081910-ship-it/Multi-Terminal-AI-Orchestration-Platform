package router

import (
	"context"
	"fmt"
	"sort"

	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/org"
	"github.com/rs/zerolog"
)

// Strategy defines how the router selects agents.
type Strategy string

const (
	StrategyCapabilityMatch Strategy = "capability_match"
	StrategyLoadBalance     Strategy = "load_balance"
	StrategyAffinity        Strategy = "affinity"
	StrategyWeighted        Strategy = "weighted"
	StrategyManual          Strategy = "manual"
)

// Config holds router configuration.
type Config struct {
	Strategy Strategy `json:"strategy"`
	Weights  Weights  `json:"weights"`
	Fallback Strategy `json:"fallback"`
}

// Weights for the weighted scoring strategy.
type Weights struct {
	Capability float64 `json:"capability"`
	Load       float64 `json:"load"`
	Affinity   float64 `json:"affinity"`
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() Config {
	return Config{
		Strategy: StrategyWeighted,
		Weights: Weights{
			Capability: 0.5,
			Load:       0.3,
			Affinity:   0.2,
		},
		Fallback: StrategyManual,
	}
}

// AgentScore represents a candidate agent with its computed score.
type AgentScore struct {
	AgentID       string
	AgentName     string
	Score         float64
	Capability    float64
	Load          float64
	Affinity      float64
	Available     bool
}

// RouteInput contains the information needed to route a task.
type RouteInput struct {
	TaskID         string
	TaskType       string
	Capabilities   []string
	OrgID          string
	AssignedRoleID string
}

// Router selects the best agent for a task based on configured strategy.
type Router struct {
	cfg       Config
	orgSvc    *org.Service
	client    *ent.Client
	capMatch  *CapabilityMatcher
	loadBal   *LoadBalancer
	affinity  *AffinityScorer
	logger    zerolog.Logger
}

func NewRouter(cfg Config, orgSvc *org.Service, client *ent.Client, logger zerolog.Logger) *Router {
	return &Router{
		cfg:      cfg,
		orgSvc:   orgSvc,
		client:   client,
		capMatch: NewCapabilityMatcher(),
		loadBal:  NewLoadBalancer(client),
		affinity: NewAffinityScorer(client),
		logger:   logger,
	}
}

// Config returns the router's configuration.
func (r *Router) Config() Config {
	return r.cfg
}

// SelectBestAgent finds the best agent for the given task.
// Returns the agent ID, or an error if no suitable agent is found.
func (r *Router) SelectBestAgent(ctx context.Context, input RouteInput) (string, error) {
	if r.cfg.Strategy == StrategyManual {
		return "", fmt.Errorf("routing strategy is manual, agent must be specified")
	}

	agents, err := r.orgSvc.ListAgents(ctx, input.OrgID)
	if err != nil {
		return "", fmt.Errorf("list agents: %w", err)
	}

	if len(agents) == 0 {
		return "", fmt.Errorf("no agents registered in org %s", input.OrgID)
	}

	// Filter to online agents (heartbeat within last 2 minutes)
	candidates := r.filterOnline(agents)
	if len(candidates) == 0 {
		// Fall back to all agents if none are online
		r.logger.Warn().Str("org_id", input.OrgID).Msg("no online agents, falling back to all agents")
		candidates = agents
	}

	// Filter by role if specified
	if input.AssignedRoleID != "" {
		filtered := make([]*org.AgentView, 0)
		for _, a := range candidates {
			if a.RoleID == input.AssignedRoleID {
				filtered = append(filtered, a)
			}
		}
		if len(filtered) > 0 {
			candidates = filtered
		}
	}

	// Score each candidate
	scores := r.scoreAgents(ctx, candidates, input)

	// Sort by score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	best := scores[0]
	r.logger.Info().
		Str("task_id", input.TaskID).
		Str("agent_id", best.AgentID).
		Str("agent_name", best.AgentName).
		Float64("score", best.Score).
		Float64("capability", best.Capability).
		Float64("load", best.Load).
		Float64("affinity", best.Affinity).
		Str("strategy", string(r.cfg.Strategy)).
		Msg("router selected agent")

	return best.AgentID, nil
}

// ScoreAgents returns scored candidates for debugging/display.
func (r *Router) ScoreAgents(ctx context.Context, input RouteInput) ([]AgentScore, error) {
	agents, err := r.orgSvc.ListAgents(ctx, input.OrgID)
	if err != nil {
		return nil, err
	}
	return r.scoreAgents(ctx, agents, input), nil
}

func (r *Router) scoreAgents(ctx context.Context, agents []*org.AgentView, input RouteInput) []AgentScore {
	scores := make([]AgentScore, len(agents))

	for i, a := range agents {
		capScore := r.capMatch.Score(a, input.TaskType, input.Capabilities)
		loadScore := r.loadBal.Score(ctx, a.ID)
		affinityScore := r.affinity.Score(ctx, a.ID, input.TaskType)

		var total float64
		switch r.cfg.Strategy {
		case StrategyCapabilityMatch:
			total = capScore
		case StrategyLoadBalance:
			total = loadScore
		case StrategyAffinity:
			total = affinityScore
		default:
			w := r.cfg.Weights
			total = w.Capability*capScore + w.Load*loadScore + w.Affinity*affinityScore
		}

		scores[i] = AgentScore{
			AgentID:    a.ID,
			AgentName:  a.Name,
			Score:      total,
			Capability: capScore,
			Load:       loadScore,
			Affinity:   affinityScore,
			Available:  a.Status != "offline" && a.Status != "error",
		}
	}

	return scores
}

func (r *Router) filterOnline(agents []*org.AgentView) []*org.AgentView {
	online := make([]*org.AgentView, 0, len(agents))
	for _, a := range agents {
		if a.Status != "offline" && a.Status != "error" {
			online = append(online, a)
		}
	}
	return online
}
