package server

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mCP-DevOS/ai-orchestration-platform/internal/store"
	"github.com/rs/zerolog"
)

const compatDefaultProjectID = "default"

type CompatProjectConfig struct {
	ID                string
	Name              string
	MainRepoPath      string
	WorktreeBasePath  string
	WorkspaceBasePath string
	ArtifactBasePath  string
	ClaudeCommand     string
	ClaudeShell       string
	GeminiCommand     string
	GeminiShell       string
	CodexCommand      string
	CodexShell        string
}

type CompatProjectsConfig struct {
	DefaultProjectID string
	Projects         []CompatProjectConfig
}

type compatProjectSummary struct {
	ID       string `json:"project_id"`
	Name     string `json:"name"`
	RepoRoot string `json:"repo_root"`
	Default  bool   `json:"default"`
}

type compatProjectEntry struct {
	summary   compatProjectSummary
	config    CompatProjectConfig
	execution *compatExecutionManager
}

type compatProjectRegistry struct {
	defaultProjectID string
	projects         map[string]*compatProjectEntry
}

func newCompatProjectRegistry(repo *store.Repository, logger zerolog.Logger, cfg CompatProjectsConfig) (*compatProjectRegistry, error) {
	registry := &compatProjectRegistry{
		defaultProjectID: strings.TrimSpace(cfg.DefaultProjectID),
		projects:         make(map[string]*compatProjectEntry),
	}

	for _, item := range cfg.Projects {
		projectID := normalizeCompatProjectID(item.ID)
		if projectID == "" {
			return nil, fmt.Errorf("project id is required")
		}
		if _, exists := registry.projects[projectID]; exists {
			return nil, fmt.Errorf("duplicate project id: %s", projectID)
		}

		repoRoot := filepath.Clean(firstCompatNonEmpty(item.MainRepoPath, "."))
		name := firstCompatNonEmpty(item.Name, projectID)
		execution := newCompatExecutionManager(repo, logger.With().Str("project_id", projectID).Logger(), CompatExecutionConfig{
			MainRepoPath:      repoRoot,
			WorktreeBasePath:  item.WorktreeBasePath,
			WorkspaceBasePath: item.WorkspaceBasePath,
			ArtifactBasePath:  item.ArtifactBasePath,
			ClaudeCommand:     item.ClaudeCommand,
			ClaudeShell:       item.ClaudeShell,
			GeminiCommand:     item.GeminiCommand,
			GeminiShell:       item.GeminiShell,
			CodexCommand:      item.CodexCommand,
			CodexShell:        item.CodexShell,
		})

		registry.projects[projectID] = &compatProjectEntry{
			summary: compatProjectSummary{
				ID:       projectID,
				Name:     name,
				RepoRoot: repoRoot,
			},
			config: CompatProjectConfig{
				ID:                projectID,
				Name:              name,
				MainRepoPath:      repoRoot,
				WorktreeBasePath:  firstCompatNonEmpty(item.WorktreeBasePath, filepath.Join(repoRoot, ".orchestrator", "worktrees")),
				WorkspaceBasePath: firstCompatNonEmpty(item.WorkspaceBasePath, filepath.Join(repoRoot, ".orchestrator", "workspaces")),
				ArtifactBasePath:  firstCompatNonEmpty(item.ArtifactBasePath, filepath.Join(repoRoot, ".orchestrator", "artifacts")),
				ClaudeCommand:     item.ClaudeCommand,
				ClaudeShell:       item.ClaudeShell,
				GeminiCommand:     item.GeminiCommand,
				GeminiShell:       item.GeminiShell,
				CodexCommand:      item.CodexCommand,
				CodexShell:        item.CodexShell,
			},
			execution: execution,
		}
	}

	if len(registry.projects) == 0 {
		return nil, fmt.Errorf("at least one project must be configured")
	}
	if registry.defaultProjectID == "" {
		for projectID := range registry.projects {
			registry.defaultProjectID = projectID
			break
		}
	}
	registry.defaultProjectID = normalizeCompatProjectID(registry.defaultProjectID)
	if _, ok := registry.projects[registry.defaultProjectID]; !ok {
		return nil, fmt.Errorf("default project %q is not configured", registry.defaultProjectID)
	}
	registry.projects[registry.defaultProjectID].summary.Default = true
	return registry, nil
}

func normalizeCompatProjectID(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func (r *compatProjectRegistry) defaultProject() string {
	if r == nil || r.defaultProjectID == "" {
		return compatDefaultProjectID
	}
	return r.defaultProjectID
}

func (r *compatProjectRegistry) resolve(projectID string) (*compatProjectEntry, error) {
	if r == nil {
		return nil, fmt.Errorf("project registry is not configured")
	}
	projectID = normalizeCompatProjectID(firstCompatNonEmpty(projectID, r.defaultProjectID))
	entry, ok := r.projects[projectID]
	if !ok {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}
	return entry, nil
}

func (r *compatProjectRegistry) summaries() []compatProjectSummary {
	if r == nil {
		return nil
	}
	items := make([]compatProjectSummary, 0, len(r.projects))
	for _, entry := range r.projects {
		items = append(items, entry.summary)
	}
	slices.SortStableFunc(items, func(a, b compatProjectSummary) int {
		if a.Default != b.Default {
			if a.Default {
				return -1
			}
			return 1
		}
		switch {
		case a.ID < b.ID:
			return -1
		case a.ID > b.ID:
			return 1
		default:
			return 0
		}
	})
	return items
}

func (r *compatProjectRegistry) configs() []CompatProjectConfig {
	if r == nil {
		return nil
	}

	items := make([]CompatProjectConfig, 0, len(r.projects))
	for _, entry := range r.projects {
		items = append(items, entry.config)
	}
	slices.SortStableFunc(items, func(a, b CompatProjectConfig) int {
		switch {
		case a.ID < b.ID:
			return -1
		case a.ID > b.ID:
			return 1
		default:
			return 0
		}
	})
	return items
}
