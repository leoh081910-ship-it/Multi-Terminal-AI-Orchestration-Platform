package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

type compatCreateProjectRequest struct {
	ProjectID     string `json:"project_id"`
	Name          string `json:"name"`
	RepoRoot      string `json:"repo_root"`
	WorktreeBase  string `json:"worktree_base,omitempty"`
	WorkspaceBase string `json:"workspace_base,omitempty"`
	ArtifactBase  string `json:"artifact_base,omitempty"`
}

func (s *Server) handleCompatCreateProject(w http.ResponseWriter, r *http.Request) {
	var req compatCreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid request body"})
		return
	}

	registry := s.getProjectRegistry()
	if registry == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"detail": "Project registry is not configured"})
		return
	}

	project, err := s.buildCompatProjectConfig(registry, req)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"detail": err.Error()})
		return
	}

	configs := append(registry.configs(), project)
	nextRegistry, err := newCompatProjectRegistry(s.repo, s.logger, CompatProjectsConfig{
		DefaultProjectID: registry.defaultProject(),
		Projects:         configs,
	})
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"detail": err.Error()})
		return
	}

	if s.projectConfigStore != nil {
		if err := s.projectConfigStore.AddProject(project); err != nil {
			status := http.StatusInternalServerError
			if isProjectAlreadyExistsError(err) {
				status = http.StatusConflict
			}
			s.writeJSON(w, status, map[string]string{"detail": err.Error()})
			return
		}
	}

	if s.projectQueueManager != nil {
		if err := s.projectQueueManager.AddProject(project); err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": fmt.Sprintf("failed to start project queue: %v", err)})
			return
		}
	}

	s.setProjectRegistry(nextRegistry)
	entry, _ := nextRegistry.resolve(project.ID)
	s.writeJSON(w, http.StatusCreated, entry.summary)
}

func (s *Server) buildCompatProjectConfig(registry *compatProjectRegistry, req compatCreateProjectRequest) (CompatProjectConfig, error) {
	repoRoot := filepath.Clean(firstCompatNonEmpty(req.RepoRoot))
	if repoRoot == "" {
		return CompatProjectConfig{}, fmt.Errorf("repo_root is required")
	}

	info, err := os.Stat(repoRoot)
	if err != nil {
		return CompatProjectConfig{}, fmt.Errorf("repo_root is invalid: %w", err)
	}
	if !info.IsDir() {
		return CompatProjectConfig{}, fmt.Errorf("repo_root must be a directory")
	}

	projectID := normalizeCompatProjectID(firstCompatNonEmpty(req.ProjectID, filepath.Base(repoRoot), req.Name))
	if projectID == "" {
		return CompatProjectConfig{}, fmt.Errorf("project_id is required")
	}
	if _, err := registry.resolve(projectID); err == nil {
		return CompatProjectConfig{}, fmt.Errorf("project already exists: %s", projectID)
	}

	template := registry.projects[registry.defaultProject()].config

	worktreeBase := filepath.Clean(firstCompatNonEmpty(req.WorktreeBase, filepath.Join(repoRoot, ".orchestrator", "worktrees")))
	workspaceBase := filepath.Clean(firstCompatNonEmpty(req.WorkspaceBase, filepath.Join(repoRoot, ".orchestrator", "workspaces")))
	artifactBase := filepath.Clean(firstCompatNonEmpty(req.ArtifactBase, filepath.Join(repoRoot, ".orchestrator", "artifacts")))

	for _, path := range []string{worktreeBase, workspaceBase, artifactBase} {
		if err := os.MkdirAll(path, 0755); err != nil {
			return CompatProjectConfig{}, fmt.Errorf("failed to prepare project directories: %w", err)
		}
	}

	return CompatProjectConfig{
		ID:                projectID,
		Name:              firstCompatNonEmpty(req.Name, projectID),
		MainRepoPath:      repoRoot,
		WorktreeBasePath:  worktreeBase,
		WorkspaceBasePath: workspaceBase,
		ArtifactBasePath:  artifactBase,
		ClaudeCommand:     template.ClaudeCommand,
		ClaudeShell:       template.ClaudeShell,
		GeminiCommand:     template.GeminiCommand,
		GeminiShell:       template.GeminiShell,
		CodexCommand:      template.CodexCommand,
		CodexShell:        template.CodexShell,
	}, nil
}

func isProjectAlreadyExistsError(err error) bool {
	return err != nil && len(err.Error()) >= len("project already exists:") && err.Error()[:len("project already exists:")] == "project already exists:"
}
