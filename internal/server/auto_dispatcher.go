package server

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
	"github.com/rs/zerolog"
)

type AutoDispatchConfig struct {
	Interval time.Duration
}

type AutoDispatcher struct {
	server   *Server
	logger   zerolog.Logger
	interval time.Duration
	running  bool
	stopCh   chan struct{}
}

type autoDispatchCandidate struct {
	task   *ent.Task
	compat compatSchedulerTask
}

func NewAutoDispatcher(server *Server, logger zerolog.Logger, cfg AutoDispatchConfig) *AutoDispatcher {
	interval := cfg.Interval
	if interval <= 0 {
		interval = 2 * time.Second
	}

	return &AutoDispatcher{
		server:   server,
		logger:   logger,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (d *AutoDispatcher) Start(ctx context.Context) error {
	if d.running {
		return fmt.Errorf("auto dispatcher already running")
	}

	d.running = true
	d.logger.Info().Dur("interval", d.interval).Msg("auto dispatcher started")
	go d.loop(ctx)
	return nil
}

func (d *AutoDispatcher) Stop() error {
	if !d.running {
		return nil
	}

	close(d.stopCh)
	d.running = false
	d.logger.Info().Msg("auto dispatcher stopped")
	return nil
}

func (d *AutoDispatcher) loop(ctx context.Context) {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.logger.Info().Msg("auto dispatcher context cancelled")
			return
		case <-d.stopCh:
			return
		case <-ticker.C:
			if err := d.dispatchEligibleTasks(ctx); err != nil {
				d.logger.Error().Err(err).Msg("auto dispatcher scan failed")
			}
		}
	}
}

func (d *AutoDispatcher) dispatchEligibleTasks(ctx context.Context) error {
	tasks, err := d.server.repo.ListAllTasks(ctx)
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		return nil
	}

	statesByID := make(map[string]string, len(tasks))
	activeByAgent := make(map[string]bool, len(compatAgents))
	candidates := make([]autoDispatchCandidate, 0, len(tasks))

	for _, task := range tasks {
		statesByID[task.ID] = task.State
		compatTask := d.server.mapCompatTask(task)

		activeKey := compatTask.ProjectID + "::" + compatTask.OwnerAgent
		if compatDispatchInFlight(compatTask.DispatchStatus) {
			activeByAgent[activeKey] = true
		}

		if !compatTask.AutoDispatchEnabled || compatTask.DispatchMode != "auto" {
			continue
		}
		if compatTask.DispatchStatus != "pending" {
			continue
		}
		if compatTask.ExecutionSessionID != "" {
			continue
		}
		if !compatAutoDispatchStateEligible(task.State) {
			continue
		}
		if !depsDone(statesByID, extractDependsOnFromCardJSON(task.CardJSON)) {
			continue
		}

		candidates = append(candidates, autoDispatchCandidate{
			task:   task,
			compat: compatTask,
		})
	}

	if len(candidates) == 0 {
		return nil
	}

	slices.SortStableFunc(candidates, func(a, b autoDispatchCandidate) int {
		if a.compat.OwnerAgent != b.compat.OwnerAgent {
			return stringsCompare(a.compat.OwnerAgent, b.compat.OwnerAgent)
		}
		if a.compat.Priority != b.compat.Priority {
			if a.compat.Priority > b.compat.Priority {
				return -1
			}
			return 1
		}
		if a.task.TopoRank != b.task.TopoRank {
			if a.task.TopoRank < b.task.TopoRank {
				return -1
			}
			return 1
		}
		if cmp := a.task.CreatedAt.Compare(b.task.CreatedAt); cmp != 0 {
			return cmp
		}
		return stringsCompare(a.task.ID, b.task.ID)
	})

	for _, candidate := range candidates {
		activeKey := candidate.compat.ProjectID + "::" + candidate.compat.OwnerAgent
		if activeByAgent[activeKey] {
			continue
		}

		if _, err := d.server.dispatchCompatTask(ctx, candidate.task.ID, false, candidate.compat.ProjectID); err != nil {
			d.logger.Warn().
				Err(err).
				Str("task_id", candidate.task.ID).
				Str("owner_agent", candidate.compat.OwnerAgent).
				Msg("auto dispatch skipped task")
			continue
		}

		activeByAgent[activeKey] = true
	}

	return nil
}

func compatAutoDispatchStateEligible(state string) bool {
	switch state {
	case engine.StateQueued, engine.StateRouted:
		return true
	default:
		return false
	}
}

func compatDispatchInFlight(dispatchStatus string) bool {
	switch dispatchStatus {
	case "queued", "dispatched", "running":
		return true
	default:
		return false
	}
}

func depsDone(statesByID map[string]string, dependsOn []string) bool {
	for _, depID := range dependsOn {
		if statesByID[depID] != engine.StateDone {
			return false
		}
	}
	return true
}

func stringsCompare(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
