// Package registry provides the runtime Agent registry for v3 multi-agent support.
// It manages dynamic registration/unregistration of Runner instances and
// maintains a per-Agent CapabilityManifest cache refreshed via health checks.
package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/runner"
	"github.com/rs/zerolog"
)

// Registry maintains the live set of registered Runner instances.
// It is safe for concurrent use by multiple goroutines.
//
// The Registry is the bridge between the database Agent records (from ent/agent)
// and the in-memory Runner instances used for execution. On startup, the Server
// loads all Agents from the DB and creates the corresponding Runner for each.
//
// Dynamic registration (via HTTP API) creates a new Runner and persists the
// Agent record to the DB in a single transaction.
type Registry struct {
	mu      sync.RWMutex
	agents  map[string]*AgentEntry // key: agent ID
	logger  zerolog.Logger
	client  *ent.Client
	heartbeatInterval time.Duration
}

// AgentEntry holds a Runner instance alongside its cached CapabilityManifest.
type AgentEntry struct {
	Runner      runner.Runner
	Manifest    *runner.CapabilityManifest
	LastCheck   time.Time // last successful HealthCheck
	HeartbeatAt time.Time // last heartbeat from the Runner itself
	Status      AgentStatus
}

// AgentStatus represents the current operational status of an Agent.
type AgentStatus string

const (
	StatusOnline  AgentStatus = "online"
	StatusIdle    AgentStatus = "idle"
	StatusRunning AgentStatus = "running"
	StatusOffline AgentStatus = "offline"
	StatusError   AgentStatus = "error"
)

// New creates a new Registry.
func New(logger zerolog.Logger, client *ent.Client) *Registry {
	return &Registry{
		agents:           make(map[string]*AgentEntry),
		logger:           logger,
		client:           client,
		heartbeatInterval: 2 * time.Minute,
	}
}

// Register adds a Runner to the registry. If an entry with the same ID already
// exists, it is replaced. This is safe for dynamic re-registration.
func (r *Registry) Register(id string, run runner.Runner) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry := &AgentEntry{
		Runner:      run,
		Manifest:    nil, // populated on first GetCapabilities call
		LastCheck:   time.Now(),
		HeartbeatAt: time.Now(),
		Status:      StatusIdle,
	}

	r.agents[id] = entry
	r.logger.Info().Str("agent_id", id).Str("runner_type", string(run.Type())).Msg("agent registered")
}

// Unregister removes a Runner from the registry. Returns the removed entry or nil.
func (r *Registry) Unregister(id string) *AgentEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.agents[id]
	if ok {
		delete(r.agents, id)
		r.logger.Info().Str("agent_id", id).Msg("agent unregistered")
	}
	return entry
}

// Get returns a registered Runner by ID. Returns nil if not found.
func (r *Registry) Get(id string) runner.Runner {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.agents[id]
	if !ok {
		return nil
	}
	return entry.Runner
}

// List returns all registered Runner instances.
func (r *Registry) List() []runner.Runner {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]runner.Runner, 0, len(r.agents))
	for _, entry := range r.agents {
		result = append(result, entry.Runner)
	}
	return result
}

// ListEntries returns all agent entries with manifest data.
func (r *Registry) ListEntries() []*AgentEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*AgentEntry, 0, len(r.agents))
	for _, entry := range r.agents {
		result = append(result, entry)
	}
	return result
}

// Manifest returns the cached CapabilityManifest for an agent.
// If not cached yet, calls Runner.GetCapabilities and caches it.
func (r *Registry) Manifest(ctx context.Context, id string) (*runner.CapabilityManifest, error) {
	r.mu.Lock()
	entry, ok := r.agents[id]
	if !ok {
		r.mu.Unlock()
		return nil, fmt.Errorf("agent %q not found in registry", id)
	}

	// Refresh if stale (older than heartbeat interval)
	stale := entry.Manifest == nil || time.Since(entry.LastCheck) > r.heartbeatInterval
	if stale {
		// Release lock during the potentially slow GetCapabilities call
		r.mu.Unlock()

		manifest, err := entry.Runner.GetCapabilities(ctx)
		if err != nil {
			// Re-acquire lock and update status on failure
			r.mu.Lock()
			entry.Status = StatusError
			entry.LastCheck = time.Now()
			r.mu.Unlock()
			return nil, fmt.Errorf("GetCapabilities for %q: %w", id, err)
		}

		r.mu.Lock()
		entry.Manifest = manifest
		entry.LastCheck = time.Now()
		entry.Status = StatusIdle
		r.mu.Unlock()

		return manifest, nil
	}
	r.mu.Unlock()
	return entry.Manifest, nil
}

// AllManifests returns capability manifests for all registered agents.
func (r *Registry) AllManifests(ctx context.Context) (map[string]*runner.CapabilityManifest, error) {
	r.mu.RLock()
	ids := make([]string, 0, len(r.agents))
	for id := range r.agents {
		ids = append(ids, id)
	}
	r.mu.RUnlock()

	result := make(map[string]*runner.CapabilityManifest, len(ids))
	for _, id := range ids {
		m, err := r.Manifest(ctx, id)
		if err != nil {
			r.logger.Warn().Err(err).Str("agent_id", id).Msg("failed to get manifest")
			continue
		}
		result[id] = m
	}
	return result, nil
}

// Count returns the number of registered agents.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.agents)
}

// StatusOf returns the current status for an agent.
func (r *Registry) StatusOf(id string) AgentStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.agents[id]
	if !ok {
		return StatusOffline
	}
	return entry.Status
}

// SetStatus sets the agent status directly.
func (r *Registry) SetStatus(id string, status AgentStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if entry, ok := r.agents[id]; ok {
		entry.Status = status
	}
}

// LoadFromDB seeds the Registry with Runners from existing Agent DB records.
// It creates a CLIRunner for each DB agent and populates the cache from the
// agent.config JSON field (which may contain runner_type and runner_config).
//
// This is called once at server startup before the HTTP API is live.
func (r *Registry) LoadFromDB(ctx context.Context, cfg RegistryConfig) error {
	agents, err := r.client.Agent.Query().All(ctx)
	if err != nil {
		return fmt.Errorf("load agents from DB: %w", err)
	}

	for _, a := range agents {
		// prefer dedicated runner_type/runner_config columns (set by migration tool).
		// Fall back to config JSON for agents created before migration.
		runnerTypeStr := a.RunnerType
		if runnerTypeStr == "" {
			runnerTypeStr = getAgentFieldFromJSON(a.Config, "runner_type", string(runner.RunnerTypeCLI))
		}
		runnerType := RunnerTypeFromString(runnerTypeStr)

		cfgJSON := a.RunnerConfig
		if cfgJSON == "" || cfgJSON == "{}" {
			cfgJSON = getAgentFieldFromJSON(a.Config, "runner_config", "{}")
		}

		var config json.RawMessage
		if err := json.Unmarshal([]byte(cfgJSON), &config); err != nil {
			r.logger.Warn().Err(err).Str("agent_id", a.ID).Msg("failed to parse runner_config, using defaults")
			config = nil
		}

		run, err := BuildRunner(BuilderInput{
			AgentID:        a.ID,
			AgentName:      a.Name,
			RunnerType:     runnerType,
			Config:         config,
			RegistryConfig: cfg,
		})
		if err != nil {
			r.logger.Warn().Err(err).Str("agent_id", a.ID).Str("runner_type", string(runnerType)).Msg("failed to build runner for agent, skipping")
			continue
		}

		r.Register(a.ID, run)
	}

	r.logger.Info().Int("loaded", len(agents)).Msg("registry loaded from database")
	return nil
}

// RegistryConfig holds global configuration shared by all Runners.
type RegistryConfig struct {
	// BasePath is the parent directory for CLI worktrees.
	BasePath string
	// MainRepo is the path to the main git repository.
	MainRepo string
	// ArtifactBase is the parent directory for artifact storage.
	ArtifactBase string
}

// BuilderInput is passed to BuildRunner to construct a Runner from DB config.
type BuilderInput struct {
	AgentID        string
	AgentName      string
	RunnerType     runner.RunnerType
	Config         json.RawMessage
	RegistryConfig RegistryConfig
}

// BuildRunner constructs a Runner instance based on type and config.
// It is the factory function used by both LoadFromDB and the dynamic registration API.
func BuildRunner(in BuilderInput) (runner.Runner, error) {
	switch in.RunnerType {
	case runner.RunnerTypeCLI:
		cfg := CLIRunnerBuildConfig{
			ID:       in.AgentID,
			Name:     in.AgentName,
			BasePath: in.RegistryConfig.BasePath,
			MainRepo: in.RegistryConfig.MainRepo,
		}
		// Allow config JSON to override BasePath/MainRepo
		if in.Config != nil {
			_ = json.Unmarshal(in.Config, &cfg) // ignore errors, use defaults
		}
		return runner.NewCLIRunner(runner.CLIRunnerConfig{
			ID:       cfg.ID,
			Name:     cfg.Name,
			BasePath: cfg.BasePath,
			MainRepo: cfg.MainRepo,
		}), nil

	case runner.RunnerTypeHTTP:
		cfg := runner.HTTPRunnerConfig{}
		if in.Config != nil {
			if err := json.Unmarshal(in.Config, &cfg); err != nil {
				return nil, fmt.Errorf("invalid HTTP runner config: %w", err)
			}
		}
		return runner.NewHTTPRunner(in.AgentID, in.AgentName, cfg), nil

	case runner.RunnerTypeMCP:
		cfg := runner.MCPRunnerConfig{}
		if in.Config != nil {
			if err := json.Unmarshal(in.Config, &cfg); err != nil {
				return nil, fmt.Errorf("invalid MCP runner config: %w", err)
			}
		}
		return runner.NewMCPRunner(in.AgentID, in.AgentName, cfg), nil

	default:
		return nil, fmt.Errorf("unsupported runner type: %s", in.RunnerType)
	}
}

// CLIRunnerBuildConfig is the JSON shape for CLI runner config stored in agent.runner_config.
type CLIRunnerBuildConfig struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	BasePath string `json:"base_path"`
	MainRepo string `json:"main_repo"`
}

// RunnerTypeFromString converts a string to RunnerType, defaulting to CLI.
func RunnerTypeFromString(s string) runner.RunnerType {
	switch runner.RunnerType(s) {
	case runner.RunnerTypeCLI:
		return runner.RunnerTypeCLI
	case runner.RunnerTypeHTTP:
		return runner.RunnerTypeHTTP
	case runner.RunnerTypeMCP:
		return runner.RunnerTypeMCP
	default:
		return runner.RunnerTypeCLI
	}
}

// getAgentFieldFromJSON extracts a field value from a JSON config string.
func getAgentFieldFromJSON(configJSON, key, fallback string) string {
	if configJSON == "" || configJSON == "{}" {
		return fallback
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &m); err != nil {
		return fallback
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return fallback
}