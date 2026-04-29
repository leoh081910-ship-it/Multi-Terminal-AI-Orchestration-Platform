package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/mCP-DevOS/ai-orchestration-platform/internal/mergequeue"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/store"
	"github.com/rs/zerolog"
)

type ProjectQueueManager struct {
	repo             *store.Repository
	logger           zerolog.Logger
	defaultProjectID string

	mu      sync.Mutex
	ctx     context.Context
	running bool
	queues  map[string]*mergequeue.MergeQueue
}

func NewProjectQueueManager(repo *store.Repository, logger zerolog.Logger, defaultProjectID string) *ProjectQueueManager {
	return &ProjectQueueManager{
		repo:             repo,
		logger:           logger,
		defaultProjectID: normalizeCompatProjectID(firstCompatNonEmpty(defaultProjectID, compatDefaultProjectID)),
		queues:           make(map[string]*mergequeue.MergeQueue),
	}
}

func (m *ProjectQueueManager) Start(ctx context.Context, projects []CompatProjectConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ctx = ctx
	m.running = true

	for _, project := range projects {
		if err := m.startProjectLocked(project); err != nil {
			return err
		}
	}
	return nil
}

func (m *ProjectQueueManager) AddProject(project CompatProjectConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startProjectLocked(project)
}

func (m *ProjectQueueManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for projectID, queue := range m.queues {
		if err := queue.Stop(); err != nil {
			return fmt.Errorf("failed to stop merge queue for %s: %w", projectID, err)
		}
	}
	m.running = false
	m.queues = make(map[string]*mergequeue.MergeQueue)
	return nil
}

func (m *ProjectQueueManager) startProjectLocked(project CompatProjectConfig) error {
	projectID := normalizeCompatProjectID(project.ID)
	if projectID == "" {
		return fmt.Errorf("project id is required")
	}
	if _, exists := m.queues[projectID]; exists {
		return fmt.Errorf("merge queue already exists for project %s", projectID)
	}

	queue := mergequeue.NewMergeQueue(
		NewMergeQueueRepositoryAdapter(m.repo, projectID, m.defaultProjectID),
		project.MainRepoPath,
		m.logger.With().Str("project_id", projectID).Logger(),
	)

	if m.running && m.ctx != nil {
		if err := queue.Start(m.ctx); err != nil {
			return err
		}
	}

	m.queues[projectID] = queue
	return nil
}
