package engine

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

type stubOrchestratorRepo struct {
	task           TaskInfo
	cleanupCalls   int
	createdEvents  []*EventData
	cleanupErr     error
	getTaskErr     error
	createEventErr error
}

func (s *stubOrchestratorRepo) CreateTask(ctx context.Context, card *TaskCard) (string, error) {
	return card.ID, nil
}

func (s *stubOrchestratorRepo) GetTaskByID(ctx context.Context, taskID string) (TaskInfo, error) {
	return s.task, s.getTaskErr
}

func (s *stubOrchestratorRepo) UpdateTaskState(ctx context.Context, taskID, fromState, toState, reason string, eventData *EventData) error {
	return nil
}

func (s *stubOrchestratorRepo) UpdateTaskTopoRank(ctx context.Context, taskID string, topoRank int) error {
	return nil
}

func (s *stubOrchestratorRepo) ListTasksByDispatchRef(ctx context.Context, dispatchRef string) ([]TaskInfo, error) {
	return nil, nil
}

func (s *stubOrchestratorRepo) ListTasksByDispatchRefAndWave(ctx context.Context, dispatchRef string, wave int) ([]TaskInfo, error) {
	return nil, nil
}

func (s *stubOrchestratorRepo) CreateEvent(ctx context.Context, eventData *EventData) error {
	if s.createEventErr != nil {
		return s.createEventErr
	}
	s.createdEvents = append(s.createdEvents, eventData)
	return nil
}

func (s *stubOrchestratorRepo) CleanupTaskResources(ctx context.Context, taskID string) (bool, error) {
	s.cleanupCalls++
	if s.cleanupErr != nil {
		return false, s.cleanupErr
	}
	return true, nil
}

func (s *stubOrchestratorRepo) UpsertWave(ctx context.Context, dispatchRef string, waveNum int) error {
	return nil
}

func (s *stubOrchestratorRepo) GetWave(ctx context.Context, dispatchRef string, waveNum int) (WaveInfo, error) {
	return nil, nil
}

func (s *stubOrchestratorRepo) SealWave(ctx context.Context, dispatchRef string, waveNum int) error {
	return nil
}

func TestOrchestratorGuards(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()
	repo := &stubOrchestratorRepo{
		task: &TaskCard{
			ID:          "task-1",
			DispatchRef: "dispatch-1",
			State:       StateQueued,
			Wave:        1,
		},
	}

	t.Run("EnqueueTask returns repo configuration error", func(t *testing.T) {
		o := &Orchestrator{logger: logger}
		_, err := o.EnqueueTask(ctx, &TaskCard{ID: "task-1"})
		if err != ErrOrchestratorRepoNotConfigured {
			t.Fatalf("expected %v, got %v", ErrOrchestratorRepoNotConfigured, err)
		}
	})

	t.Run("EnqueueTask returns dependency manager configuration error", func(t *testing.T) {
		o := &Orchestrator{repo: repo, logger: logger}
		_, err := o.EnqueueTask(ctx, &TaskCard{ID: "task-1"})
		if err != ErrDependencyManagerNotConfigured {
			t.Fatalf("expected %v, got %v", ErrDependencyManagerNotConfigured, err)
		}
	})

	t.Run("EnqueueTask returns wave manager configuration error", func(t *testing.T) {
		o := &Orchestrator{
			repo:       repo,
			depManager: NewDependencyManager(nil),
			logger:     logger,
		}
		_, err := o.EnqueueTask(ctx, &TaskCard{ID: "task-1"})
		if err != ErrWaveManagerNotConfigured {
			t.Fatalf("expected %v, got %v", ErrWaveManagerNotConfigured, err)
		}
	})

	t.Run("RouteTask returns wave manager configuration error", func(t *testing.T) {
		o := &Orchestrator{repo: repo, logger: logger}
		err := o.RouteTask(ctx, "task-1")
		if err != ErrWaveManagerNotConfigured {
			t.Fatalf("expected %v, got %v", ErrWaveManagerNotConfigured, err)
		}
	})

	t.Run("HandleRetry returns retry manager configuration error", func(t *testing.T) {
		o := &Orchestrator{logger: logger}
		err := o.HandleRetry(ctx, "task-1", &RetryState{}, "execution_failure")
		if err != ErrRetryManagerNotConfigured {
			t.Fatalf("expected %v, got %v", ErrRetryManagerNotConfigured, err)
		}
	})

	t.Run("CheckTTL returns ttl manager configuration error", func(t *testing.T) {
		o := &Orchestrator{logger: logger}
		err := o.CheckTTL(ctx, "task-1", time.Now())
		if err != ErrTTLManagerNotConfigured {
			t.Fatalf("expected %v, got %v", ErrTTLManagerNotConfigured, err)
		}
	})

	t.Run("PropagateDependencyFailure returns dependency manager configuration error", func(t *testing.T) {
		o := &Orchestrator{repo: repo, logger: logger}
		err := o.PropagateDependencyFailure(ctx, "task-1")
		if err != ErrDependencyManagerNotConfigured {
			t.Fatalf("expected %v, got %v", ErrDependencyManagerNotConfigured, err)
		}
	})
}

func TestCheckTTLPerformsCleanupAndRecordsEvent(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()
	repo := &stubOrchestratorRepo{
		task: &TaskCard{
			ID:          "task-ttl",
			DispatchRef: "dispatch-ttl",
			State:       StateDone,
			RetryCount:  2,
			Wave:        1,
		},
	}

	o := &Orchestrator{
		repo:       repo,
		ttlManager: NewTTLManager(&TTLConfig{DefaultTTL: time.Hour}),
		logger:     logger,
	}

	err := o.CheckTTL(ctx, "task-ttl", time.Now().Add(-2*time.Hour))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if repo.cleanupCalls != 1 {
		t.Fatalf("expected cleanup to run once, got %d", repo.cleanupCalls)
	}
	if len(repo.createdEvents) != 1 {
		t.Fatalf("expected one cleanup event, got %d", len(repo.createdEvents))
	}
	if repo.createdEvents[0].EventType != "ttl_cleanup" {
		t.Fatalf("expected ttl_cleanup event, got %q", repo.createdEvents[0].EventType)
	}
	if repo.createdEvents[0].Reason != "ttl_expired" {
		t.Fatalf("expected ttl_expired reason, got %q", repo.createdEvents[0].Reason)
	}
}

func TestCheckTTLDoesNothingWhenNotExpired(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()
	repo := &stubOrchestratorRepo{
		task: &TaskCard{
			ID:          "task-ttl",
			DispatchRef: "dispatch-ttl",
			State:       StateDone,
			Wave:        1,
		},
	}

	o := &Orchestrator{
		repo:       repo,
		ttlManager: NewTTLManager(&TTLConfig{DefaultTTL: 24 * time.Hour}),
		logger:     logger,
	}

	err := o.CheckTTL(ctx, "task-ttl", time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if repo.cleanupCalls != 0 {
		t.Fatalf("expected no cleanup call, got %d", repo.cleanupCalls)
	}
	if len(repo.createdEvents) != 0 {
		t.Fatalf("expected no events, got %d", len(repo.createdEvents))
	}
}
