package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/threefoldtech/tfgrid-sdk-go/messenger"
	"github.com/threefoldtech/zosbase/pkg/api"
	"github.com/threefoldtech/zosbase/pkg/gridtypes"
)

// TODO: review the api again, rename the pkg rpcapi
type RpcHandler struct {
	api *api.API
}

// NewRpcHandler creates a new RpcHandler instance with the provided API.
func NewRpcHandler(api *api.API) *RpcHandler {
	return &RpcHandler{
		api: api,
	}
}

func RegisterHandlers(s *messenger.JSONRPCServer, r *RpcHandler) {
	s.RegisterHandler("system.version", r.handleSystemVersion)
	s.RegisterHandler("system.dmi", r.handleSystemDMI)
	s.RegisterHandler("system.hypervisor", r.handleSystemHypervisor)
	s.RegisterHandler("system.diagnostics", r.handleSystemDiagnostics)
	s.RegisterHandler("system.features", r.handleSystemNodeFeatures)

	s.RegisterHandler("monitor.speed", r.handlePerfSpeed)
	s.RegisterHandler("monitor.health", r.handlePerfHealth)
	s.RegisterHandler("monitor.publicip", r.handlePerfPublicIp)
	s.RegisterHandler("monitor.benchmark", r.handlePerfBenchmark)
	s.RegisterHandler("monitor.all", r.handlePerfAll)

	s.RegisterHandler("network.wg_ports", r.handleNetworkWGPorts)
	s.RegisterHandler("network.public_config", r.handleNetworkPublicConfig)
	s.RegisterHandler("network.has_ipv6", r.handleNetworkHasIPv6)
	s.RegisterHandler("network.public_ips", r.handleNetworkPublicIPs)
	s.RegisterHandler("network.private_ips", r.handleNetworkPrivateIPs)
	s.RegisterHandler("network.interfaces", r.handleNetworkInterfaces)
	s.RegisterHandler("network.set_public_nic", r.handleAdminSetPublicNIC)
	s.RegisterHandler("network.get_public_nic", r.handleAdminGetPublicNIC)
	// s.RegisterHandler("network.admin.interfaces", r.handleAdminInterfaces)

	s.RegisterHandler("deployment.deploy", r.handleDeploymentDeploy)
	s.RegisterHandler("deployment.update", r.handleDeploymentUpdate)
	s.RegisterHandler("deployment.get", r.handleDeploymentGet)
	s.RegisterHandler("deployment.list", r.handleDeploymentList)
	s.RegisterHandler("deployment.changes", r.handleDeploymentChanges)
	// s.RegisterHandler("deployment.delete", r.handleDeploymentDelete)

	s.RegisterHandler("gpu.list", r.handleGpuList)
	s.RegisterHandler("storage.pools", r.handleStoragePools)
	s.RegisterHandler("statistics", r.handleStatistics)
	s.RegisterHandler("location.get", r.handleLocationGet)
	// s.RegisterHandler("vm.logs", r.handleVmLogs)

}

func extractObject(params json.RawMessage, obj interface{}) error {
	if err := json.Unmarshal(params, obj); err != nil {
		return fmt.Errorf("invalid object parameter: %w", err)
	}
	return nil
}

func (r *RpcHandler) handleSystemVersion(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.SystemVersion(ctx)
}

func (r *RpcHandler) handleSystemDMI(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.SystemDMI(ctx)
}

func (r *RpcHandler) handleSystemHypervisor(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.SystemHypervisor(ctx)
}

func (r *RpcHandler) handleSystemDiagnostics(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.SystemDiagnostics(ctx)
}

func (r *RpcHandler) handleSystemNodeFeatures(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.SystemNodeFeatures(ctx), nil
}

func (r *RpcHandler) handlePerfSpeed(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.PerfSpeed(ctx)
}

func (r *RpcHandler) handlePerfHealth(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.PerfHealth(ctx)
}

func (r *RpcHandler) handlePerfPublicIp(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.PerfPublicIp(ctx)
}

func (r *RpcHandler) handlePerfBenchmark(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.PerfBenchmark(ctx)
}

func (r *RpcHandler) handlePerfAll(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.PerfAll(ctx)
}

func (r *RpcHandler) handleGpuList(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.GpuList(ctx)
}

func (r *RpcHandler) handleStoragePools(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.StoragePoolsHandler(ctx)
}

func (r *RpcHandler) handleNetworkWGPorts(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.NetworkWGPorts(ctx)
}

func (r *RpcHandler) handleNetworkPublicConfig(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.NetworkPublicConfigGet(ctx, nil)
}

func (r *RpcHandler) handleNetworkHasIPv6(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.NetworkHasIPv6(ctx)
}

func (r *RpcHandler) handleNetworkPublicIPs(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.NetworkListPublicIPs(ctx)
}

func (r *RpcHandler) handleNetworkPrivateIPs(ctx context.Context, params json.RawMessage) (interface{}, error) {
	type networkParams struct {
		NetworkName string `json:"network_name"`
	}

	var p networkParams
	if err := extractObject(params, &p); err != nil {
		return nil, err
	}

	return r.api.NetworkListPrivateIPs(ctx, p.NetworkName)
}

func (r *RpcHandler) handleNetworkInterfaces(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.NetworkInterfaces(ctx)
}

func (r *RpcHandler) handleAdminInterfaces(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.AdminInterfaces(ctx)
}

func (r *RpcHandler) handleAdminSetPublicNIC(ctx context.Context, params json.RawMessage) (interface{}, error) {
	type deviceParams struct {
		Device string `json:"device"`
	}

	var p deviceParams
	if err := extractObject(params, &p); err != nil {
		return nil, err
	}

	return nil, r.api.AdminSetPublicNIC(ctx, p.Device)
}

func (r *RpcHandler) handleAdminGetPublicNIC(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.AdminGetPublicNIC(ctx)
}

func (r *RpcHandler) handleDeploymentDeploy(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var deployment gridtypes.Deployment
	if err := extractObject(params, &deployment); err != nil {
		return nil, err
	}

	return nil, r.api.DeploymentDeployHandler(ctx, deployment)
}

func (r *RpcHandler) handleDeploymentUpdate(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var deployment gridtypes.Deployment
	if err := extractObject(params, &deployment); err != nil {
		return nil, err
	}

	return nil, r.api.DeploymentUpdateHandler(ctx, deployment)
}

func (r *RpcHandler) handleDeploymentGet(ctx context.Context, params json.RawMessage) (interface{}, error) {
	type contractParams struct {
		ContractID uint64 `json:"contract_id"`
	}

	var p contractParams
	if err := extractObject(params, &p); err != nil {
		return nil, err
	}

	return r.api.DeploymentGetHandler(ctx, p.ContractID)
}

func (r *RpcHandler) handleDeploymentList(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.DeploymentListHandler(ctx)
}

func (r *RpcHandler) handleDeploymentChanges(ctx context.Context, params json.RawMessage) (interface{}, error) {
	type contractParams struct {
		ContractID uint64 `json:"contract_id"`
	}

	var p contractParams
	if err := extractObject(params, &p); err != nil {
		return nil, err
	}

	return r.api.DeploymentChangesHandler(ctx, p.ContractID)
}

func (r *RpcHandler) handleDeploymentDelete(ctx context.Context, params json.RawMessage) (interface{}, error) {
	type contractParams struct {
		ContractID uint64 `json:"contract_id"`
	}

	var p contractParams
	if err := extractObject(params, &p); err != nil {
		return nil, err
	}

	return nil, r.api.DeploymentDeleteHandler(ctx, p.ContractID)
}

func (r *RpcHandler) handleStatistics(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.Statistics(ctx)
}

func (r *RpcHandler) handleVmLogs(ctx context.Context, params json.RawMessage) (interface{}, error) {
	type fileParams struct {
		FileName string `json:"file_name"`
	}

	var p fileParams
	if err := extractObject(params, &p); err != nil {
		return nil, err
	}

	return r.api.GetVmLogsHandler(ctx, p.FileName)
}

func (r *RpcHandler) handleLocationGet(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return r.api.LocationGet(ctx)
}
