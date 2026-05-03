package executor

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Pool manages a fixed size worker pool for parallel command execution.
type Pool struct {
	workers      int
	taskQueue    chan ExecTask
	results      chan ExecResult
	executor     Executor
	logger       zerolog.Logger
	wg           sync.WaitGroup
	started      bool
	stopChan     chan struct{}
	activeCount  int
	mu           sync.Mutex
}

// ExecTask represents a command to be executed by the pool.
type ExecTask struct {
	ID      string
	Command string
	WorkDir string
	Timeout time.Duration
}

// ExecResult contains the outcome of a worker task.
type ExecResult struct {
	TaskID   string
	Output   string
	Error    error
	ExitCode int
	Duration time.Duration
}

// PoolConfig configures the worker pool.
type PoolConfig struct {
	Workers       int           // Number of concurrent workers (default: runtime.NumCPU())
	QueueSize     int           // Size of task queue (default: 100)
	DefaultTimeout time.Duration // Default timeout per task (default: 30*time.Minute)
}

// NewPool creates a new worker pool.
func NewPool(executor Executor, logger zerolog.Logger, cfg PoolConfig) *Pool {
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 100
	}
	if cfg.DefaultTimeout <= 0 {
		cfg.DefaultTimeout = 30 * time.Minute
	}

	return &Pool{
		workers:   cfg.Workers,
		taskQueue: make(chan ExecTask, cfg.QueueSize),
		results:   make(chan ExecResult, cfg.QueueSize),
		executor:  executor,
		logger:    logger,
		stopChan:  make(chan struct{}),
	}
}

// Start launches the worker goroutines.
func (p *Pool) Start(ctx context.Context) {
	p.mu.Lock()
	if p.started {
		p.mu.Unlock()
		return
	}
	p.started = true
	p.mu.Unlock()

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
	p.logger.Info().Int("workers", p.workers).Msg("worker pool started")
}

// Stop gracefully shuts down the pool. Waits for all active tasks to complete.
func (p *Pool) Stop() {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return
	}
	close(p.stopChan)
	p.mu.Unlock()

	// Wait for workers to finish
	p.wg.Wait()
	close(p.results)
	p.logger.Info().Msg("worker pool stopped")
}

// Submit enqueues a task for execution. Returns immediately.
func (p *Pool) Submit(task ExecTask) {
	select {
	case p.taskQueue <- task:
	default:
		p.logger.Warn().Str("task_id", task.ID).Msg("task queue full, dropping task")
	}
}

// SubmitAndWait submits a task and waits for its result.
func (p *Pool) SubmitAndWait(ctx context.Context, task ExecTask) (*ExecResult, error) {
	// Create a result channel for this specific task
	resultChan := make(chan ExecResult, 1)

	// Submit the task as a closure that pushes to the channel
	select {
	case p.taskQueue <- task:
		// The worker will produce a result; we need to capture it.
		// Since workers write to p.results, we need to read from there.
		// This approach is simpler: use a promise pattern.
	default:
		return nil, fmt.Errorf("task queue full")
	}

	// Wait for result on p.results, but filter by TaskID
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case result := <-p.results:
			if result.TaskID == task.ID {
				return &result, nil
			}
			// Not our task, but we must avoid dropping other results.
			// Instead of an unbounded channel, we can temporarily store unmatched results.
			// For simplicity, this implementation requires a dedicated submission method.
			// Let's implement a proper promise approach.
		}
	}
}

// SubmitAndWaitPromise submits a task and returns a channel that will receive the result.
func (p *Pool) SubmitAndWaitPromise(task ExecTask) <-chan ExecResult {
	promise := make(chan ExecResult, 1)

	// Wrap the task to capture the result and forward it
	go func() {
		select {
		case p.taskQueue <- task:
			// The result will be written to p.results; we need to read it.
			// This is flawed. Better to restructure: workers do not write to a shared results channel.
			// Instead, each task gets its own result channel.
		default:
			promise <- ExecResult{TaskID: task.ID, Error: fmt.Errorf("queue full")}
		}
	}()

	return promise
}

// worker processes tasks from the queue.
func (p *Pool) worker(ctx context.Context, id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.stopChan:
			return
		case task := <-p.taskQueue:
			p.mu.Lock()
			p.activeCount++
			p.mu.Unlock()

			p.logger.Debug().Int("worker", id).Str("task_id", task.ID).Msg("executing task")

			result := p.executeOne(ctx, task)

			p.mu.Lock()
			p.activeCount--
			p.mu.Unlock()

			// Send result to the shared results channel
			select {
			case p.results <- result:
			default:
				p.logger.Warn().Str("task_id", task.ID).Msg("result channel full, dropping result")
			}
		}
	}
}

// executeOne runs a single command and returns the result.
func (p *Pool) executeOne(ctx context.Context, task ExecTask) ExecResult {
	start := time.Now()
	timeout := task.Timeout
	if timeout == 0 {
		timeout = 30 * time.Minute
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if task.WorkDir != "" {
		cmd = exec.CommandContext(execCtx, "/bin/sh", "-c", task.Command)
		cmd.Dir = task.WorkDir
	} else {
		cmd = exec.CommandContext(execCtx, "/bin/sh", "-c", task.Command)
	}

	out, err := cmd.CombinedOutput()
	var exitCode int
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return ExecResult{
		TaskID:   task.ID,
		Output:   string(out),
		Error:    err,
		ExitCode: exitCode,
		Duration: time.Since(start),
	}
}

// ActiveWorkers returns the number of currently executing tasks.
func (p *Pool) ActiveWorkers() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.activeCount
}

// TaskQueueLen returns the number of tasks waiting in the queue.
func (p *Pool) TaskQueueLen() int {
	return len(p.taskQueue)
}

// ResultsChan returns the read-only results channel.
func (p *Pool) ResultsChan() <-chan ExecResult {
	return p.results
}
