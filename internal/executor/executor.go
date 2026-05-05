package executor

import (
	"context"
	"os/exec"
	"runtime"
	"time"
)

// Executor defines the interface for executing commands.
type Executor interface {
	Execute(ctx context.Context, task ExecTask) ExecResult
}

// StandardExecutor runs shell commands using os/exec.
type StandardExecutor struct{}

// NewStandardExecutor creates a new StandardExecutor.
func NewStandardExecutor() *StandardExecutor {
	return &StandardExecutor{}
}

// shellCommand creates a platform-appropriate command.
func shellCommand(ctx context.Context, command string, workDir string) *exec.Cmd {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", command)
	} else {
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", command)
	}
	if workDir != "" {
		cmd.Dir = workDir
	}
	return cmd
}

// Execute runs the command and returns the result.
func (e *StandardExecutor) Execute(ctx context.Context, task ExecTask) ExecResult {
	start := time.Now()
	timeout := task.Timeout
	if timeout == 0 {
		timeout = 30 * time.Minute
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := shellCommand(execCtx, task.Command, task.WorkDir)

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
