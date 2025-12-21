package debugcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/threefoldtech/zosbase/pkg"
	"github.com/threefoldtech/zosbase/pkg/gridtypes"
	"github.com/threefoldtech/zosbase/pkg/gridtypes/zos"
)

type VMInfoRequest struct {
	Deployment string `json:"deployment"` // Format: "twin-id:contract-id"
	VMName     string `json:"vm_name"`
	FullLogs   bool   `json:"full_logs"`
}

type VMInfoResponse struct {
	VMID string     `json:"vm_id"`
	Info pkg.VMInfo `json:"info"`
	Logs string     `json:"logs"`
}

func ParseVMInfoRequest(payload []byte) (VMInfoRequest, error) {
	var req VMInfoRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return req, err
	}
	return req, nil
}

func VMInfo(ctx context.Context, deps Deps, req VMInfoRequest) (VMInfoResponse, error) {
	if req.VMName == "" {
		return VMInfoResponse{}, fmt.Errorf("vm_name is required")
	}

	twinID, contractID, err := ParseDeploymentID(req.Deployment)
	if err != nil {
		return VMInfoResponse{}, err
	}

	deployment, err := deps.Provision.Get(ctx, twinID, contractID)
	if err != nil {
		return VMInfoResponse{}, fmt.Errorf("failed to get deployment: %w", err)
	}
	vmwl, err := deployment.GetType(gridtypes.Name(req.VMName), zos.ZMachineType)
	if err != nil {
		return VMInfoResponse{}, fmt.Errorf("failed to get zmachine workload: %w", err)
	}
	vmID := vmwl.ID.String()

	info, err := deps.VM.Inspect(ctx, vmID)
	if err != nil {
		return VMInfoResponse{}, fmt.Errorf("failed to inspect vm: %w", err)
	}

	var raw string
	if req.FullLogs {
		raw, err = deps.VM.LogsFull(ctx, vmID)
	} else {
		raw, err = deps.VM.Logs(ctx, vmID)
	}
	if err != nil {
		return VMInfoResponse{}, fmt.Errorf("failed to get vm logs: %w", err)
	}

	logs := sanitizeLogs(raw)
	return VMInfoResponse{VMID: vmID, Info: info, Logs: logs}, nil
}

func sanitizeLogs(raw string) string {
	// Sanitize logs:
	// - strip NUL bytes
	// - drop invalid UTF-8 bytes
	// - normalize CRLF -> LF
	b := []byte(raw)
	sanitized := make([]byte, 0, len(b))
	for _, c := range b {
		if c != 0x00 {
			sanitized = append(sanitized, c)
		}
	}
	if !utf8.Valid(sanitized) {
		valid := make([]byte, 0, len(sanitized))
		for len(sanitized) > 0 {
			r, size := utf8.DecodeRune(sanitized)
			if r == utf8.RuneError && size == 1 {
				sanitized = sanitized[1:]
				continue
			}
			valid = append(valid, sanitized[:size]...)
			sanitized = sanitized[size:]
		}
		sanitized = valid
	}
	logs := string(sanitized)
	logs = strings.ReplaceAll(logs, "\r\n", "\n")
	logs = strings.ReplaceAll(logs, "\r", "\n")
	return logs
}
