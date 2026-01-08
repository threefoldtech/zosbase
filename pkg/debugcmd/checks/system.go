package checks

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const systemProbeTimeout = 60 * time.Second

type SystemChecker struct {
	command string
}

func (sc *SystemChecker) Name() string { return "system" }

func (sc *SystemChecker) Run(ctx context.Context, data *CheckData) []HealthCheck {
	if sc.command == "" {
		return nil
	}

	parts := strings.Fields(sc.command)
	if len(parts) == 0 {
		return []HealthCheck{failure("system.probe", "empty probe command", nil)}
	}

	probeCtx, cancel := context.WithTimeout(ctx, systemProbeTimeout)
	defer cancel()

	cmd := exec.CommandContext(probeCtx, parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return []HealthCheck{failure("system.probe", fmt.Sprintf("probe failed: %v", err), map[string]interface{}{"error": err.Error()})}
	}

	return []HealthCheck{success("system.probe", "probe executed successfully", map[string]interface{}{
		"output":    string(output),
		"exit_code": cmd.ProcessState.ExitCode(),
	})}
}

func NewSystemChecker(command string) *SystemChecker {
	return &SystemChecker{command: command}
}
