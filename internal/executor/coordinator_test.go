package executor

import (
	"context"
	"testing"

	"github.com/mCP-DevOS/ai-orchestration-platform/internal/transport"
)

type stubExecutor struct {
	captured *transport.TaskConfig
	result   *transport.ExecutionResult
	err      error
}

func (s *stubExecutor) Execute(_ context.Context, config *transport.TaskConfig) (*transport.ExecutionResult, error) {
	s.captured = config
	if s.result != nil || s.err != nil {
		return s.result, s.err
	}
	return &transport.ExecutionResult{Success: true, ExitCode: 0}, nil
}

func (s *stubExecutor) Type() transport.TransportType {
	return transport.TransportCLI
}

func (s *stubExecutor) Validate(_ *transport.TaskConfig) error {
	return nil
}

func TestHandleTaskExecutionParsesFilesToModifyFromJSONSlice(t *testing.T) {
	cli := &stubExecutor{}
	handler := NewExecutionHandler(NewCoordinator(cli, &stubExecutor{}, nil, nil))

	err := handler.HandleTaskExecution(context.Background(), "task-json-slice", transport.TransportCLI, map[string]interface{}{
		"workspace_path":  `E:\04-Claude\Projects\多终端 AI 编排平台\workspace with spaces`,
		"artifact_path":   `E:\04-Claude\Projects\多终端 AI 编排平台\artifacts`,
		"files_to_modify": []interface{}{"a.go", "b.go"},
		"command":         "echo ok",
		"shell":           "powershell",
		"context":         map[string]interface{}{"k": "v"},
	})
	if err != nil {
		t.Fatalf("HandleTaskExecution failed: %v", err)
	}

	if cli.captured == nil {
		t.Fatal("expected executor to receive task config")
	}
	if len(cli.captured.FilesToModify) != 2 || cli.captured.FilesToModify[0] != "a.go" || cli.captured.FilesToModify[1] != "b.go" {
		t.Fatalf("expected parsed files_to_modify, got %+v", cli.captured.FilesToModify)
	}
	if cli.captured.Shell != "powershell" {
		t.Fatalf("expected shell to be preserved, got %q", cli.captured.Shell)
	}
	if cli.captured.Context["k"] != "v" {
		t.Fatalf("expected context to be forwarded, got %+v", cli.captured.Context)
	}
}

func TestHandleTaskExecutionRejectsInvalidFilesToModify(t *testing.T) {
	handler := NewExecutionHandler(NewCoordinator(&stubExecutor{}, &stubExecutor{}, nil, nil))

	err := handler.HandleTaskExecution(context.Background(), "task-invalid-slice", transport.TransportCLI, map[string]interface{}{
		"files_to_modify": []interface{}{"a.go", 1},
	})
	if err == nil {
		t.Fatal("expected invalid files_to_modify to fail")
	}
}
