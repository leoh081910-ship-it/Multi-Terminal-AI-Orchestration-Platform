package executor

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func echoCmd(s string) string {
	if runtime.GOOS == "windows" {
		return "cmd /c echo " + s
	}
	return "echo " + s
}

func sleepCmd(d string) string {
	if runtime.GOOS == "windows" {
		return "cmd /c timeout /t " + d + " /nobreak >nul && echo done"
	}
	return "sleep " + d + "; echo done"
}

func TestWorkerPoolBasic(t *testing.T) {
	logger := zerolog.Nop()
	exec := NewStandardExecutor()
	pool := NewPool(exec, logger, PoolConfig{Workers: 2, QueueSize: 10})
	pool.Start(context.Background())
	defer pool.Stop()

	task := ExecTask{
		ID:      "test-1",
		Command: echoCmd("hello"),
		WorkDir: "",
		Timeout: 5 * time.Second,
	}

	select {
	case pool.taskQueue <- task:
	case <-time.After(1 * time.Second):
		t.Fatal("failed to queue task")
	}

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
	exec := NewStandardExecutor()
	pool := NewPool(exec, logger, PoolConfig{Workers: 3, QueueSize: 20})
	pool.Start(context.Background())
	defer pool.Stop()

	taskCount := 10
	for i := 0; i < taskCount; i++ {
		pool.Submit(ExecTask{
			ID:      fmt.Sprintf("task-%d", i),
			Command: echoCmd(fmt.Sprintf("%d", i)),
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
	exec := NewStandardExecutor()
	pool := NewPool(exec, logger, PoolConfig{Workers: 4, QueueSize: 50})
	pool.Start(context.Background())
	defer pool.Stop()

	taskCount := 20
	start := time.Now()

	for i := 0; i < taskCount; i++ {
		pool.Submit(ExecTask{
			ID:      fmt.Sprintf("sleep-task-%d", i),
			Command: sleepCmd("1"),
		})
	}

	results := 0
	for results < taskCount {
		select {
		case <-pool.ResultsChan():
			results++
		case <-time.After(15 * time.Second):
			t.Fatalf("timeout: got %d results", results)
		}
	}

	duration := time.Since(start)
	assert.Less(t, duration, 10*time.Second, "parallel execution should complete within 10 seconds")
}
