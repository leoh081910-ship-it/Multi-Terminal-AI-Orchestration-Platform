package server

import (
	"path/filepath"
	"strings"

	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
)

func syncCompatPayloadState(payload map[string]interface{}, state string) {
	if payload == nil {
		return
	}

	payload["state"] = state
	payload["status"] = mapCompatWorkflowStatus(state)
	payload["dispatch_status"] = mapCompatDispatchStatus(state)
	if state != engine.StateRunning {
		payload["stalled"] = false
	}
}

func compatSystemReportPath(taskID string, taskType string) string {
	name := strings.TrimSpace(strings.ToLower(taskType))
	if name == "" {
		name = "report"
	}
	name = strings.ReplaceAll(name, " ", "-")
	return filepath.ToSlash(filepath.Join(".orchestrator", "reports", taskID+"-"+name+".md"))
}
