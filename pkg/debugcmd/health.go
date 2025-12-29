package debugcmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosbase/pkg/debugcmd/checks"
	"github.com/threefoldtech/zosbase/pkg/gridtypes"
	"github.com/threefoldtech/zosbase/pkg/gridtypes/zos"
)

type HealthRequest struct {
	Deployment string                 `json:"deployment"`        // Format: "twin-id:contract-id"
	Options    map[string]interface{} `json:"options,omitempty"` // Optional configuration for health checks
}

type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthUnhealthy HealthStatus = "unhealthy"
)

type WorkloadHealth struct {
	WorkloadID string               `json:"workload_id"`
	Type       string               `json:"type"`
	Name       string               `json:"name"`
	Status     HealthStatus         `json:"status"`
	Checks     []checks.HealthCheck `json:"checks"`
}

type HealthResponse struct {
	TwinID     uint32           `json:"twin_id"`
	ContractID uint64           `json:"contract_id"`
	Workloads  []WorkloadHealth `json:"workloads"`
}

func ParseHealthRequest(payload []byte) (HealthRequest, error) {
	var req HealthRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return req, err
	}
	return req, nil
}

func Health(ctx context.Context, deps Deps, req HealthRequest) (HealthResponse, error) {
	twinID, contractID, err := ParseDeploymentID(req.Deployment)
	if err != nil {
		return HealthResponse{}, err
	}

	out := HealthResponse{TwinID: twinID, ContractID: contractID}

	if req.Options != nil && req.Options["system_probe"] != nil {
		probeCmd, ok := req.Options["system_probe"].(string)
		if !ok {
			return HealthResponse{}, fmt.Errorf("system_probe must be a string")
		}
		hc := checks.SystemProbeCheck[0](ctx, &checks.SystemProbeData{Command: probeCmd})
		status := HealthUnhealthy
		if hc.OK {
			status = HealthHealthy
		}
		out.Workloads = append(out.Workloads, WorkloadHealth{
			WorkloadID: "system",
			Type:       "diagnostic",
			Name:       "system.probe",
			Status:     status,
			Checks:     []checks.HealthCheck{hc},
		})
	}

	deployment, err := deps.Provision.Get(ctx, twinID, contractID)
	if err != nil {
		return HealthResponse{}, fmt.Errorf("failed to get deployment: %w", err)
	}

	for _, wl := range deployment.Workloads {
		workloadID, err := gridtypes.NewWorkloadID(twinID, contractID, wl.Name)
		if err != nil {
			log.Debug().Err(err).Str("workload_name", string(wl.Name)).Msg("failed to create workload ID")
			continue
		}
		checkData := &checks.CheckData{
			Network:  deps.Network.Namespace,
			VM:       deps.VM.Exists,
			Twin:     twinID,
			Contract: contractID,
			Workload: wl,
		}

		var allChecks []checks.HealthCheck
		var status HealthStatus

		switch wl.Type {
		case zos.NetworkType:
			allChecks, status = runNetworkChecks(ctx, checkData)
		case zos.ZMachineType, zos.ZMachineLightType:
			allChecks, status = runVMChecks(ctx, checkData)
		default:
			continue
		}

		out.Workloads = append(out.Workloads, WorkloadHealth{
			WorkloadID: workloadID.String(),
			Type:       string(wl.Type),
			Name:       string(wl.Name),
			Status:     status,
			Checks:     allChecks,
		})
	}

	return out, nil
}

func runNetworkChecks(ctx context.Context, data *checks.CheckData) ([]checks.HealthCheck, HealthStatus) {
	allChecks := []checks.HealthCheck{}

	for _, check := range checks.NetworkChecks {
		hc := check(ctx, data)
		allChecks = append(allChecks, hc)
	}

	status := HealthHealthy
	for _, c := range allChecks {
		if !c.OK {
			status = HealthUnhealthy
		}
	}

	return allChecks, status
}

func runVMChecks(ctx context.Context, data *checks.CheckData) ([]checks.HealthCheck, HealthStatus) {
	allChecks := []checks.HealthCheck{}

	for _, check := range checks.VMChecks {
		hc := check(ctx, data)
		allChecks = append(allChecks, hc)
	}

	status := HealthHealthy
	for _, c := range allChecks {
		if !c.OK {
			status = HealthUnhealthy
		}
	}

	return allChecks, status
}
