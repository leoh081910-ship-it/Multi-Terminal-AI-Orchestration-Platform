package transport

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (w *lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.w.Write(p)
}

func executeCommand(ctx context.Context, dir, command, shell string, env map[string]string, outputPath string) (string, int, error) {
	cmd, err := BuildShellCommand(ctx, command, shell)
	if err != nil {
		return "", -1, err
	}

	cmd.Dir = dir
	cmd.Env = mergeCommandEnv(env)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", -1, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", -1, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	var buffer bytes.Buffer
	writers := []io.Writer{&buffer}
	var logFile *os.File
	if outputPath != "" {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return "", -1, fmt.Errorf("failed to create output directory: %w", err)
		}

		logFile, err = os.Create(outputPath)
		if err != nil {
			return "", -1, fmt.Errorf("failed to create output log: %w", err)
		}
		defer logFile.Close()
		writers = append(writers, logFile)
	}

	outputWriter := &lockedWriter{w: io.MultiWriter(writers...)}
	writeExecutionBanner(outputWriter, "started", dir, shell, command, "")

	if err := cmd.Start(); err != nil {
		writeExecutionBanner(outputWriter, "start_failed", dir, shell, command, err.Error())
		return buffer.String(), -1, err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(outputWriter, stdout)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(outputWriter, stderr)
	}()

	waitErr := cmd.Wait()
	wg.Wait()

	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	if waitErr != nil {
		writeExecutionBanner(outputWriter, "failed", dir, shell, command, waitErr.Error())
		return buffer.String(), exitCode, waitErr
	}

	writeExecutionBanner(outputWriter, "completed", dir, shell, command, "")
	return buffer.String(), exitCode, nil
}

func writeExecutionBanner(w io.Writer, phase, dir, shell, command, message string) {
	lines := []string{
		fmt.Sprintf("[%s] %s", time.Now().UTC().Format(time.RFC3339), phase),
		fmt.Sprintf("cwd: %s", dir),
	}
	if shell != "" {
		lines = append(lines, fmt.Sprintf("shell: %s", shell))
	}
	lines = append(lines, fmt.Sprintf("command: %s", command))
	if message != "" {
		lines = append(lines, fmt.Sprintf("message: %s", message))
	}
	lines = append(lines, "")
	_, _ = io.WriteString(w, fmt.Sprintf("%s\n", bytes.Join(func() [][]byte {
		items := make([][]byte, 0, len(lines))
		for _, line := range lines {
			items = append(items, []byte(line))
		}
		return items
	}(), []byte("\n"))))
}
