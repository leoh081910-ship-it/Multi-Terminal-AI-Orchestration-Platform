package server

import (
	"context"
	"sync"
	"time"

	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/store"
	"github.com/rs/zerolog"
)

type TTLCleanupConfig struct {
	Interval time.Duration
}

type TTLCleanupRunner struct {
	repo       *store.Repository
	ttlManager *engine.TTLManager
	logger     zerolog.Logger
	config     TTLCleanupConfig

	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
}

func NewTTLCleanupRunner(repo *store.Repository, logger zerolog.Logger, cfg TTLCleanupConfig) *TTLCleanupRunner {
	ttlManager := engine.NewTTLManager(engine.DefaultTTLConfig())
	if cfg.Interval <= 0 {
		cfg.Interval = time.Minute
	}

	return &TTLCleanupRunner{
		repo:       repo,
		ttlManager: ttlManager,
		logger:     logger,
		config:     cfg,
		done:       make(chan struct{}),
	}
}

func (r *TTLCleanupRunner) Start(ctx context.Context) error {
	if r.cancel != nil {
		return nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	go func() {
		defer close(r.done)
		ticker := time.NewTicker(r.config.Interval)
		defer ticker.Stop()

		r.runOnce(runCtx)
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				r.runOnce(runCtx)
			}
		}
	}()

	r.logger.Info().
		Dur("interval", r.config.Interval).
		Msg("ttl cleanup runner started")

	return nil
}

func (r *TTLCleanupRunner) Stop() error {
	r.once.Do(func() {
		if r.cancel != nil {
			r.cancel()
		}
	})

	select {
	case <-r.done:
	case <-time.After(5 * time.Second):
		r.logger.Warn().Msg("ttl cleanup runner stop timed out")
	}

	return nil
}

func (r *TTLCleanupRunner) runOnce(ctx context.Context) {
	tasks, err := r.repo.ListAllTasks(ctx)
	if err != nil {
		r.logger.Error().Err(err).Msg("ttl cleanup scan failed")
		return
	}

	for _, item := range tasks {
		if item == nil || item.TerminalAt.IsZero() {
			continue
		}
		if !r.ttlManager.IsExpired(item.TerminalAt) {
			continue
		}

		cleaned, err := r.repo.CleanupTaskResources(ctx, item.ID)
		if err != nil {
			r.logger.Error().
				Str("task_id", item.ID).
				Err(err).
				Msg("ttl cleanup failed")
			continue
		}
		if !cleaned {
			continue
		}

		if err := r.repo.CreateEvent(ctx, &store.EventData{
			EventID:   "evt_ttl_" + item.ID,
			ProjectID: item.ProjectID,
			TaskID:    item.ID,
			EventType: "ttl_cleanup",
			FromState: item.State,
			ToState:   item.State,
			Timestamp: time.Now().UTC(),
			Reason:    "ttl_expired",
			Attempt:   item.RetryCount,
			Transport: item.Transport,
			Details:   "workspace and artifact paths cleared",
		}); err != nil {
			r.logger.Error().
				Str("task_id", item.ID).
				Err(err).
				Msg("ttl cleanup event failed")
		}
	}
}
