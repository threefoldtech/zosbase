package checks

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var systemProbeTimeout = 60 * time.Second

type SystemProbeData struct {
	Command string
}

// CheckSystemProbe executes a custom system probe command
func CheckSystemProbe(ctx context.Context, data *SystemProbeData) HealthCheck {
	result := HealthCheck{
		Name:    "system.probe.custom",
		OK:      false,
		Message: "system state probe execution",
		Evidence: map[string]interface{}{
			"probe_type": "custom",
			"exit_code":  0,
		},
	}

	probeCtx, cancel := context.WithTimeout(ctx, systemProbeTimeout)
	defer cancel()

	parts := strings.Fields(data.Command)
	if len(parts) == 0 {
		result.Message = "empty probe command"
		result.Evidence["error"] = "empty probe command"
		result.OK = false
		return result
	}

	execCmd := exec.CommandContext(probeCtx, parts[0], parts[1:]...)
	output, err := execCmd.CombinedOutput()
	if err != nil {
		result.Message = fmt.Sprintf("probe command failed: %v", err)
		result.Evidence["error"] = err.Error()
		result.OK = false
		return result
	}

	result.OK = true
	result.Message = "probe command executed successfully"
	result.Evidence["output"] = string(output)
	result.Evidence["exit_code"] = execCmd.ProcessState.ExitCode()
	return result
}
