package server

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

type projectRuntimeCommandFile struct {
	Command string `yaml:"command,omitempty"`
	Shell   string `yaml:"shell,omitempty"`
}

type projectConfigFileEntry struct {
	ID            string                    `yaml:"id"`
	Name          string                    `yaml:"name"`
	RepoRoot      string                    `yaml:"repo_root"`
	WorktreeBase  string                    `yaml:"worktree_base"`
	WorkspaceBase string                    `yaml:"workspace_base"`
	ArtifactBase  string                    `yaml:"artifact_base"`
	Claude        projectRuntimeCommandFile `yaml:"claude,omitempty"`
	Gemini        projectRuntimeCommandFile `yaml:"gemini,omitempty"`
	Codex         projectRuntimeCommandFile `yaml:"codex,omitempty"`
}

type projectConfigFileDocument struct {
	Database     map[string]interface{} `yaml:"database,omitempty"`
	Server       map[string]interface{} `yaml:"server,omitempty"`
	Runtime      map[string]interface{} `yaml:"runtime,omitempty"`
	AutoDispatch map[string]interface{} `yaml:"auto_dispatch,omitempty"`
	TTLCleanup   map[string]interface{} `yaml:"ttl_cleanup,omitempty"`
	Projects     struct {
		Default string                   `yaml:"default"`
		Items   []projectConfigFileEntry `yaml:"items"`
	} `yaml:"projects"`
}

type ProjectConfigStore struct {
	path string
	mu   sync.Mutex
}

func NewProjectConfigStore(path string) *ProjectConfigStore {
	return &ProjectConfigStore{path: path}
}

func (s *ProjectConfigStore) AddProject(project CompatProjectConfig) error {
	if s == nil || s.path == "" {
		return fmt.Errorf("project config store is not configured")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.read()
	if err != nil {
		return err
	}

	projectID := normalizeCompatProjectID(project.ID)
	for _, item := range doc.Projects.Items {
		if normalizeCompatProjectID(item.ID) == projectID {
			return fmt.Errorf("project already exists: %s", projectID)
		}
	}

	doc.Projects.Items = append(doc.Projects.Items, projectConfigFileEntry{
		ID:            projectID,
		Name:          firstCompatNonEmpty(project.Name, projectID),
		RepoRoot:      filepath.Clean(project.MainRepoPath),
		WorktreeBase:  filepath.Clean(project.WorktreeBasePath),
		WorkspaceBase: filepath.Clean(project.WorkspaceBasePath),
		ArtifactBase:  filepath.Clean(project.ArtifactBasePath),
		Claude: projectRuntimeCommandFile{
			Command: project.ClaudeCommand,
			Shell:   project.ClaudeShell,
		},
		Gemini: projectRuntimeCommandFile{
			Command: project.GeminiCommand,
			Shell:   project.GeminiShell,
		},
		Codex: projectRuntimeCommandFile{
			Command: project.CodexCommand,
			Shell:   project.CodexShell,
		},
	})

	if doc.Projects.Default == "" {
		doc.Projects.Default = projectID
	}

	return s.write(doc)
}

func (s *ProjectConfigStore) read() (*projectConfigFileDocument, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var doc projectConfigFileDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	return &doc, nil
}

func (s *ProjectConfigStore) write(doc *projectConfigFileDocument) error {
	data, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to encode config file: %w", err)
	}
	return os.WriteFile(s.path, data, 0644)
}
