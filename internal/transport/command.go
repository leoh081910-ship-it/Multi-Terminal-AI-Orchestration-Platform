package transport

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// BuildShellCommand builds an exec.Cmd using an OS default shell or an explicit shell override.
func BuildShellCommand(ctx context.Context, command, shell string) (*exec.Cmd, error) {
	selectedShell := strings.TrimSpace(strings.ToLower(shell))
	if selectedShell == "" {
		if runtime.GOOS == "windows" {
			selectedShell = "cmd"
		} else {
			selectedShell = "sh"
		}
	}

	switch selectedShell {
	case "cmd":
		return exec.CommandContext(ctx, "cmd", "/C", command), nil
	case "powershell":
		return exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", command), nil
	case "pwsh":
		return exec.CommandContext(ctx, "pwsh", "-NoProfile", "-Command", command), nil
	case "sh":
		return exec.CommandContext(ctx, "sh", "-c", command), nil
	case "bash":
		return exec.CommandContext(ctx, "bash", "-c", command), nil
	default:
		return nil, fmt.Errorf("unsupported shell: %s", shell)
	}
}
