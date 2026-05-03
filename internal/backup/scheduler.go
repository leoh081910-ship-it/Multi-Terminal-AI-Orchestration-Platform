package backup

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Scheduler runs backups on a periodic schedule
type Scheduler struct {
	service  *Service
	interval time.Duration
	logger   zerolog.Logger

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	done    chan struct{}
}

// NewScheduler creates a backup scheduler
func NewScheduler(service *Service, interval time.Duration, logger zerolog.Logger) *Scheduler {
	if interval <= 0 {
		interval = time.Hour
	}
	return &Scheduler{
		service:  service,
		interval: interval,
		logger:   logger,
	}
}

// Start begins the periodic backup loop
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.done = make(chan struct{})
	s.running = true
	s.mu.Unlock()

	go s.loop(runCtx)
	s.logger.Info().Dur("interval", s.interval).Msg("backup scheduler started")
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.cancel()
	s.running = false
	s.mu.Unlock()
	<-s.done
	s.logger.Info().Msg("backup scheduler stopped")
	return nil
}

func (s *Scheduler) loop(ctx context.Context) {
	defer close(s.done)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run immediately on start
	if _, err := s.service.CreateBackup(ctx); err != nil {
		s.logger.Error().Err(err).Msg("initial backup failed")
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.service.CreateBackup(ctx); err != nil {
				s.logger.Error().Err(err).Msg("scheduled backup failed")
			}
		}
	}
}
