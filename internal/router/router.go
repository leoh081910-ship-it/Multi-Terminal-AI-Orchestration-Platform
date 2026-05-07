package router

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/org"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/registry"
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
	StrategyCapabilityV2    Strategy = "capability_v2" // enhanced manifest-based matching
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
	AgentID       string  `json:"agent_id"`
	AgentName     string  `json:"agent_name"`
	Score         float64 `json:"score"`
	Capability    float64 `json:"capability"`
	Load          float64 `json:"load"`
	Affinity      float64 `json:"affinity"`
	Available     bool    `json:"available"`
	ManifestScore float64 `json:"manifest_score,omitempty"` // enhanced capability score
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
	mu         sync.RWMutex
	cfg        Config
	orgSvc     *org.Service
	client     *ent.Client
	registry   *registry.RunnerRegistry
	capMatch   *CapabilityMatcher
	capMatchV2 *EnhancedCapabilityMatcher
	loadBal    *LoadBalancer
	affinity   *AffinityScorer
	logger     zerolog.Logger
}

// NewRouter creates a new Router. If registry is provided, it uses manifest-based capability scoring.
func NewRouter(cfg Config, orgSvc *org.Service, client *ent.Client, logger zerolog.Logger, registry *registry.RunnerRegistry) *Router {
	r := &Router{
		cfg:      cfg,
		orgSvc:   orgSvc,
		client:   client,
		registry: registry,
		capMatch: NewCapabilityMatcher(),
		loadBal:  NewLoadBalancer(client),
		affinity: NewAffinityScorer(client),
		logger:   logger,
	}
	if registry != nil {
		r.capMatchV2 = NewEnhancedCapabilityMatcher(registry)
	}
	return r
}

// Config returns the router's current configuration.
func (r *Router) Config() Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cfg
}

// SetConfig updates the router configuration at runtime.
func (r *Router) SetConfig(cfg Config) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cfg = cfg
}

// SelectBestAgent finds the best agent for the given task.
// Returns the agent ID, or an error if no suitable agent is found.
func (r *Router) SelectBestAgent(ctx context.Context, input RouteInput) (string, error) {
	r.mu.RLock()
	strategy := r.cfg.Strategy
	r.mu.RUnlock()

	if strategy == StrategyManual {
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
		Str("strategy", string(strategy)).
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
	r.mu.RLock()
	cfg := r.cfg
	r.mu.RUnlock()

	scores := make([]AgentScore, len(agents))

	for i, a := range agents {
		var capScore, manifestScore float64

		if r.capMatchV2 != nil && (cfg.Strategy == StrategyCapabilityV2 || cfg.Strategy == StrategyWeighted) {
			manifestScore = r.capMatchV2.ScoreV2(ctx, a.ID, input.TaskType, input.Capabilities)
			capScore = manifestScore
		} else {
			capScore = r.capMatch.Score(a, input.TaskType, input.Capabilities)
		}

		loadScore := r.loadBal.Score(ctx, a.ID)
		affinityScore := r.affinity.Score(ctx, a.ID, input.TaskType)

		var total float64
		switch cfg.Strategy {
		case StrategyCapabilityMatch:
			total = capScore
		case StrategyLoadBalance:
			total = loadScore
		case StrategyAffinity:
			total = affinityScore
		case StrategyCapabilityV2:
			total = capScore
		default:
			w := cfg.Weights
			total = w.Capability*capScore + w.Load*loadScore + w.Affinity*affinityScore
		}

		scores[i] = AgentScore{
			AgentID:       a.ID,
			AgentName:     a.Name,
			Score:         total,
			Capability:    capScore,
			Load:          loadScore,
			Affinity:      affinityScore,
			Available:     a.Status != string(registry.StatusOffline) && a.Status != string(registry.StatusError),
			ManifestScore: manifestScore,
		}
	}

	return scores
}

func (r *Router) filterOnline(agents []*org.AgentView) []*org.AgentView {
	online := make([]*org.AgentView, 0, len(agents))
	for _, a := range agents {
		if a.Status != string(registry.StatusOffline) && a.Status != string(registry.StatusError) {
			online = append(online, a)
		}
	}
	return online
}