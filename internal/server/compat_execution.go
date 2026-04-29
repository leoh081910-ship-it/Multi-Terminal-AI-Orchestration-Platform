package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mCP-DevOS/ai-orchestration-platform/internal/store"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/transport"
	"github.com/rs/zerolog"
)

type CompatExecutionConfig struct {
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

type compatRuntimeSpec struct {
	Name    string
	Command string
	Shell   string
}

type compatExecutionManager struct {
	repo          *store.Repository
	logger        zerolog.Logger
	mainRepoPath  string
	worktreeBase  string
	workspaceBase string
	artifactBase  string
	logsBase      string
	cliTransport  *transport.CLITransport
	apiTransport  *transport.APITransport
	runtimes      map[string]compatRuntimeSpec
}

func newCompatExecutionManager(repo *store.Repository, logger zerolog.Logger, cfg CompatExecutionConfig) *compatExecutionManager {
	mainRepo := filepath.Clean(firstCompatNonEmpty(cfg.MainRepoPath, "."))
	worktreeBase := filepath.Clean(firstCompatNonEmpty(cfg.WorktreeBasePath, filepath.Join(mainRepo, ".orchestrator", "worktrees")))
	workspaceBase := filepath.Clean(firstCompatNonEmpty(cfg.WorkspaceBasePath, filepath.Join(mainRepo, ".orchestrator", "workspaces")))
	artifactBase := filepath.Clean(firstCompatNonEmpty(cfg.ArtifactBasePath, filepath.Join(mainRepo, ".orchestrator", "artifacts")))
	logsBase := filepath.Join(filepath.Dir(artifactBase), "logs")

	return &compatExecutionManager{
		repo:          repo,
		logger:        logger,
		mainRepoPath:  mainRepo,
		worktreeBase:  worktreeBase,
		workspaceBase: workspaceBase,
		artifactBase:  artifactBase,
		logsBase:      logsBase,
		cliTransport:  transport.NewCLITransport(worktreeBase, mainRepo),
		apiTransport:  transport.NewAPITransport(workspaceBase),
		runtimes: map[string]compatRuntimeSpec{
			compatAgentClaude:   {Name: "claude", Command: strings.TrimSpace(cfg.ClaudeCommand), Shell: strings.TrimSpace(cfg.ClaudeShell)},
			compatAgentGemini:   {Name: "gemini", Command: strings.TrimSpace(cfg.GeminiCommand), Shell: strings.TrimSpace(cfg.GeminiShell)},
			compatAgentCodex:    {Name: "codex", Command: strings.TrimSpace(cfg.CodexCommand), Shell: strings.TrimSpace(cfg.CodexShell)},
			compatAgentReviewer: {Name: "reviewer", Command: strings.TrimSpace(cfg.ClaudeCommand), Shell: strings.TrimSpace(cfg.ClaudeShell)},
		},
	}
}

func (s *Server) ConfigureCompatExecution(cfg CompatExecutionConfig) {
	_ = s.ConfigureCompatProjects(CompatProjectsConfig{
		DefaultProjectID: compatDefaultProjectID,
		Projects: []CompatProjectConfig{
			{
				ID:                compatDefaultProjectID,
				Name:              compatDefaultProjectID,
				MainRepoPath:      cfg.MainRepoPath,
				WorktreeBasePath:  cfg.WorktreeBasePath,
				WorkspaceBasePath: cfg.WorkspaceBasePath,
				ArtifactBasePath:  cfg.ArtifactBasePath,
				ClaudeCommand:     cfg.ClaudeCommand,
				ClaudeShell:       cfg.ClaudeShell,
				GeminiCommand:     cfg.GeminiCommand,
				GeminiShell:       cfg.GeminiShell,
				CodexCommand:      cfg.CodexCommand,
				CodexShell:        cfg.CodexShell,
			},
		},
	})
}

func (s *Server) ConfigureCompatProjects(cfg CompatProjectsConfig) error {
	registry, err := newCompatProjectRegistry(s.repo, s.logger, cfg)
	if err != nil {
		return err
	}
	s.setProjectRegistry(registry)
	return nil
}

func (m *compatExecutionManager) runtimeForAgent(agent string) compatRuntimeSpec {
	spec, ok := m.runtimes[normalizeCompatAgent(agent)]
	if ok {
		return spec
	}
	return compatRuntimeSpec{Name: inferRuntimeFromAgent(agent, "")}
}

func (m *compatExecutionManager) commandConfigured(agent string) bool {
	return strings.TrimSpace(m.runtimeForAgent(agent).Command) != ""
}

func (m *compatExecutionManager) workspacePath(taskID string, transportType string) string {
	switch transportType {
	case string(transport.TransportAPI):
		return filepath.Join(m.workspaceBase, taskID)
	default:
		return filepath.Join(m.worktreeBase, taskID)
	}
}

func (m *compatExecutionManager) artifactPath(taskID string) string {
	return filepath.Join(m.artifactBase, taskID)
}

func (m *compatExecutionManager) logPath(taskID string) string {
	return filepath.Join(m.logsBase, taskID+".log")
}

func (m *compatExecutionManager) readLogTail(taskID string, maxBytes int64) string {
	path := m.logPath(taskID)
	data, err := readFileTail(path, maxBytes)
	if err != nil {
		return ""
	}
	return data
}

func readFileTail(path string, maxBytes int64) (string, error) {
	if maxBytes <= 0 {
		maxBytes = 32 * 1024
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	size := info.Size()
	offset := int64(0)
	if size > maxBytes {
		offset = size - maxBytes
	}

	buffer := make([]byte, size-offset)
	if _, err := file.ReadAt(buffer, offset); err != nil {
		return "", err
	}

	return string(buffer), nil
}

func (m *compatExecutionManager) executor(transportType string) (transport.Executor, error) {
	switch transportType {
	case "", string(transport.TransportCLI):
		return m.cliTransport, nil
	case string(transport.TransportAPI):
		return m.apiTransport, nil
	default:
		return nil, fmt.Errorf("unsupported transport: %s", transportType)
	}
}
