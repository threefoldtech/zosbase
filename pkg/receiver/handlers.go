package receiver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/threefoldtech/zosbase/pkg/gridtypes"
)

func (r *Receiver) registerHandler(method string, handler HandlerFunc) {
	r.handlers[method] = handler
}

func (r *Receiver) registerHandlers() {
	handlers := map[string]HandlerFunc{
		"system.version":     r.handleSystemVersion,
		"system.dmi":         r.handleSystemDMI,
		"system.hypervisor":  r.handleSystemHypervisor,
		"system.diagnostics": r.handleSystemDiagnostics,
		"system.features":    r.handleSystemNodeFeatures,

		"monitor.speed":     r.handlePerfSpeed,
		"monitor.health":    r.handlePerfHealth,
		"monitor.publicip":  r.handlePerfPublicIp,
		"monitor.benchmark": r.handlePerfBenchmark,
		"monitor.all":       r.handlePerfAll,

		"network.wg_ports":       r.handleNetworkWGPorts,
		"network.public_config":  r.handleNetworkPublicConfig,
		"network.has_ipv6":       r.handleNetworkHasIPv6,
		"network.public_ips":     r.handleNetworkPublicIPs,
		"network.private_ips":    r.handleNetworkPrivateIPs,
		"network.interfaces":     r.handleNetworkInterfaces,
		"network.set_public_nic": r.handleAdminSetPublicNIC,
		"network.get_public_nic": r.handleAdminGetPublicNIC,
		// "network.admin.interfaces":     r.handleAdminInterfaces,

		"deployment.deploy":  r.handleDeploymentDeploy,
		"deployment.update":  r.handleDeploymentUpdate,
		"deployment.get":     r.handleDeploymentGet,
		"deployment.list":    r.handleDeploymentList,
		"deployment.changes": r.handleDeploymentChanges,
		// "deployment.delete":  r.handleDeploymentDelete,

		"gpu.list":      r.handleGpuList,
		"storage.pools": r.handleStoragePools,
		"statistics":    r.handleStatistics,
		"location.get":  r.handleLocationGet,
		// "vm.logs":       r.handleVmLogs,
	}

	for method, handler := range handlers {
		r.registerHandler(method, handler)
	}
}

func extractObject(params json.RawMessage, obj interface{}) error {
	if err := json.Unmarshal(params, obj); err != nil {
		return fmt.Errorf("invalid object parameter: %w", err)
	}
	return nil
}

func (r *Receiver) handleSystemVersion(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.SystemVersion(ctx)
}

func (r *Receiver) handleSystemDMI(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.SystemDMI(ctx)
}

func (r *Receiver) handleSystemHypervisor(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.SystemHypervisor(ctx)
}

func (r *Receiver) handleSystemDiagnostics(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.SystemDiagnostics(ctx)
}

func (r *Receiver) handleSystemNodeFeatures(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.SystemNodeFeatures(ctx), nil
}

func (r *Receiver) handlePerfSpeed(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.PerfSpeed(ctx)
}

func (r *Receiver) handlePerfHealth(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.PerfHealth(ctx)
}

func (r *Receiver) handlePerfPublicIp(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.PerfPublicIp(ctx)
}

func (r *Receiver) handlePerfBenchmark(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.PerfBenchmark(ctx)
}

func (r *Receiver) handlePerfAll(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.PerfAll(ctx)
}

func (r *Receiver) handleGpuList(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.GpuList(ctx)
}

func (r *Receiver) handleStoragePools(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.StoragePoolsHandler(ctx)
}

func (r *Receiver) handleNetworkWGPorts(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.NetworkWGPorts(ctx)
}

func (r *Receiver) handleNetworkPublicConfig(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.NetworkPublicConfigGet(ctx, nil)
}

func (r *Receiver) handleNetworkHasIPv6(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.NetworkHasIPv6(ctx)
}

func (r *Receiver) handleNetworkPublicIPs(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.NetworkListPublicIPs(ctx)
}

func (r *Receiver) handleNetworkPrivateIPs(ctx context.Context, params json.RawMessage) (interface{}, error) {
	type networkParams struct {
		NetworkName string `json:"network_name"`
	}

	var p networkParams
	if err := extractObject(params, &p); err != nil {
		return nil, err
	}

	return r.api.NetworkListPrivateIPs(ctx, p.NetworkName)
}

func (r *Receiver) handleNetworkInterfaces(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.NetworkInterfaces(ctx)
}

func (r *Receiver) handleAdminInterfaces(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.AdminInterfaces(ctx)
}

func (r *Receiver) handleAdminSetPublicNIC(ctx context.Context, params json.RawMessage) (interface{}, error) {
	type deviceParams struct {
		Device string `json:"device"`
	}

	var p deviceParams
	if err := extractObject(params, &p); err != nil {
		return nil, err
	}

	return nil, r.api.AdminSetPublicNIC(ctx, p.Device)
}

func (r *Receiver) handleAdminGetPublicNIC(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.AdminGetPublicNIC(ctx)
}

func (r *Receiver) handleDeploymentDeploy(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var deployment gridtypes.Deployment
	if err := extractObject(params, &deployment); err != nil {
		return nil, err
	}

	return nil, r.api.DeploymentDeployHandler(ctx, deployment)
}

func (r *Receiver) handleDeploymentUpdate(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var deployment gridtypes.Deployment
	if err := extractObject(params, &deployment); err != nil {
		return nil, err
	}

	return nil, r.api.DeploymentUpdateHandler(ctx, deployment)
}

func (r *Receiver) handleDeploymentGet(ctx context.Context, params json.RawMessage) (interface{}, error) {
	type contractParams struct {
		ContractID uint64 `json:"contract_id"`
	}

	var p contractParams
	if err := extractObject(params, &p); err != nil {
		return nil, err
	}

	return r.api.DeploymentGetHandler(ctx, p.ContractID)
}

func (r *Receiver) handleDeploymentList(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.DeploymentListHandler(ctx)
}

func (r *Receiver) handleDeploymentChanges(ctx context.Context, params json.RawMessage) (interface{}, error) {
	type contractParams struct {
		ContractID uint64 `json:"contract_id"`
	}

	var p contractParams
	if err := extractObject(params, &p); err != nil {
		return nil, err
	}

	return r.api.DeploymentChangesHandler(ctx, p.ContractID)
}

func (r *Receiver) handleDeploymentDelete(ctx context.Context, params json.RawMessage) (interface{}, error) {
	type contractParams struct {
		ContractID uint64 `json:"contract_id"`
	}

	var p contractParams
	if err := extractObject(params, &p); err != nil {
		return nil, err
	}

	return nil, r.api.DeploymentDeleteHandler(ctx, p.ContractID)
}

func (r *Receiver) handleStatistics(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.Statistics(ctx)
}

func (r *Receiver) handleVmLogs(ctx context.Context, params json.RawMessage) (interface{}, error) {
	type fileParams struct {
		FileName string `json:"file_name"`
	}

	var p fileParams
	if err := extractObject(params, &p); err != nil {
		return nil, err
	}

	return r.api.GetVmLogsHandler(ctx, p.FileName)
}

func (r *Receiver) handleLocationGet(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.LocationGet(ctx)
}
