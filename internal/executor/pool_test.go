package executor

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerPoolBasic(t *testing.T) {
	logger := zerolog.Nop()
	exec := NewStandardExecutor(WithLogger(logger))
	pool := NewPool(exec, logger, PoolConfig{Workers: 2, QueueSize: 10})
	pool.Start(context.Background())
	defer pool.Stop()

	// Submit a task
	task := ExecTask{
		ID:      "test-1",
		Command: "echo hello",
		WorkDir: "",
		Timeout: 5 * time.Second,
	}

	select {
	case pool.taskQueue <- task:
		// Successfully queued
	case <-time.After(1 * time.Second):
		t.Fatal("failed to queue task")
	}

	// Wait for result
	select {
	case result := <-pool.ResultsChan():
		assert.Equal(t, "test-1", result.TaskID)
		assert.NoError(t, result.Error)
		assert.Contains(t, result.Output, "hello")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for result")
	}
}

func TestWorkerPoolMultipleTasks(t *testing.T) {
	logger := zerolog.Nop()
	exec := NewStandardExecutor(WithLogger(logger))
	pool := NewPool(exec, logger, PoolConfig{Workers: 3, QueueSize: 20})
	pool.Start(context.Background())
	defer pool.Stop()

	taskCount := 10
	for i := 0; i < taskCount; i++ {
		pool.Submit(ExecTask{
			ID:      "task-" + string(rune('0'+i)),
			Command: "echo " + string(rune('0'+i)),
		})
	}

	results := make(map[string]bool)
	timeout := time.After(10 * time.Second)

	for len(results) < taskCount {
		select {
		case result := <-pool.ResultsChan():
			results[result.TaskID] = true
			assert.NoError(t, result.Error)
		case <-timeout:
			t.Fatalf("timeout: got %d/%d results", len(results), taskCount)
		}
	}
	assert.Equal(t, taskCount, len(results))
	assert.Equal(t, 0, pool.TaskQueueLen())
}

func TestWorkerPoolConcurrent(t *testing.T) {
	logger := zerolog.Nop()
	exec := NewStandardExecutor(WithLogger(logger))
	pool := NewPool(exec, logger, PoolConfig{Workers: 4, QueueSize: 50})
	pool.Start(context.Background())
	defer pool.Stop()

	taskCount := 20
	start := time.Now()

	for i := 0; i < taskCount; i++ {
		pool.Submit(ExecTask{
			ID:      "sleep-task-" + string(rune('0'+i)),
			Command: "sleep 0.05; echo done",
		})
	}

	results := 0
	for results < taskCount {
		select {
		case <-pool.ResultsChan():
			results++
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout: got %d results", results)
		}
	}

	duration := time.Since(start)
	// With 4 workers, 20 * 0.05s = 1s of total work; should finish in ~1s if parallelism works
	// Give 3x margin for CI
	assert.Less(t, duration, 2*time.Second, "parallel execution should complete within 2 seconds")
}
