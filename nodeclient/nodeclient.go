package nodeclient

import (
	"context"

	"github.com/threefoldtech/tfgrid-sdk-go/messenger"
	"github.com/threefoldtech/zosbase/pkg"
	"github.com/threefoldtech/zosbase/pkg/api"
	"github.com/threefoldtech/zosbase/pkg/capacity/dmi"
	"github.com/threefoldtech/zosbase/pkg/diagnostics"
	"github.com/threefoldtech/zosbase/pkg/geoip"
	"github.com/threefoldtech/zosbase/pkg/gridtypes"
)

type NodeClient struct {
	rpcClient   *messenger.JSONRPCClient
	destination string
}

func NewNodeClient(msgr *messenger.Messenger, destination string) *NodeClient {
	return &NodeClient{
		rpcClient:   messenger.NewJSONRPCClient(msgr),
		destination: destination,
	}
}

// System Methods

func (c *NodeClient) GetNodeVersion(ctx context.Context) (api.Version, error) {
	var version api.Version
	if err := c.rpcClient.Call(ctx, c.destination, "system.version", nil, &version); err != nil {
		return api.Version{}, err
	}
	return version, nil
}

func (c *NodeClient) GetSystemDMI(ctx context.Context) (dmi.DMI, error) {
	var dmiInfo dmi.DMI
	if err := c.rpcClient.Call(ctx, c.destination, "system.dmi", nil, &dmiInfo); err != nil {
		return dmi.DMI{}, err
	}
	return dmiInfo, nil
}

func (c *NodeClient) GetSystemHypervisor(ctx context.Context) (string, error) {
	var hypervisor string
	if err := c.rpcClient.Call(ctx, c.destination, "system.hypervisor", nil, &hypervisor); err != nil {
		return "", err
	}
	return hypervisor, nil
}

func (c *NodeClient) GetSystemDiagnostics(ctx context.Context) (diagnostics.Diagnostics, error) {
	var diag diagnostics.Diagnostics
	if err := c.rpcClient.Call(ctx, c.destination, "system.diagnostics", nil, &diag); err != nil {
		return diagnostics.Diagnostics{}, err
	}
	return diag, nil
}

func (c *NodeClient) GetSystemNodeFeatures(ctx context.Context) ([]pkg.NodeFeature, error) {
	var features []pkg.NodeFeature
	if err := c.rpcClient.Call(ctx, c.destination, "system.features", nil, &features); err != nil {
		return nil, err
	}
	return features, nil
}

// Monitor/Performance Methods

func (c *NodeClient) GetPerfSpeed(ctx context.Context) (pkg.IperfTaskResult, error) {
	var result pkg.IperfTaskResult
	if err := c.rpcClient.Call(ctx, c.destination, "monitor.speed", nil, &result); err != nil {
		return pkg.IperfTaskResult{}, err
	}
	return result, nil
}

func (c *NodeClient) GetPerfHealth(ctx context.Context) (pkg.HealthTaskResult, error) {
	var result pkg.HealthTaskResult
	if err := c.rpcClient.Call(ctx, c.destination, "monitor.health", nil, &result); err != nil {
		return pkg.HealthTaskResult{}, err
	}
	return result, nil
}

func (c *NodeClient) GetPerfPublicIp(ctx context.Context) (pkg.PublicIpTaskResult, error) {
	var result pkg.PublicIpTaskResult
	if err := c.rpcClient.Call(ctx, c.destination, "monitor.publicip", nil, &result); err != nil {
		return pkg.PublicIpTaskResult{}, err
	}
	return result, nil
}

func (c *NodeClient) GetPerfBenchmark(ctx context.Context) (pkg.CpuBenchTaskResult, error) {
	var result pkg.CpuBenchTaskResult
	if err := c.rpcClient.Call(ctx, c.destination, "monitor.benchmark", nil, &result); err != nil {
		return pkg.CpuBenchTaskResult{}, err
	}
	return result, nil
}

func (c *NodeClient) GetPerfAll(ctx context.Context) (pkg.AllTaskResult, error) {
	var result pkg.AllTaskResult
	if err := c.rpcClient.Call(ctx, c.destination, "monitor.all", nil, &result); err != nil {
		return pkg.AllTaskResult{}, err
	}
	return result, nil
}

// Network Methods

func (c *NodeClient) GetNetworkWGPorts(ctx context.Context) ([]uint, error) {
	var ports []uint
	if err := c.rpcClient.Call(ctx, c.destination, "network.wg_ports", nil, &ports); err != nil {
		return nil, err
	}
	return ports, nil
}

func (c *NodeClient) GetNetworkPublicConfig(ctx context.Context) (pkg.PublicConfig, error) {
	var config pkg.PublicConfig
	if err := c.rpcClient.Call(ctx, c.destination, "network.public_config", nil, &config); err != nil {
		return pkg.PublicConfig{}, err
	}
	return config, nil
}

func (c *NodeClient) GetNetworkHasIPv6(ctx context.Context) (bool, error) {
	var hasIPv6 bool
	if err := c.rpcClient.Call(ctx, c.destination, "network.has_ipv6", nil, &hasIPv6); err != nil {
		return false, err
	}
	return hasIPv6, nil
}

func (c *NodeClient) GetNetworkPublicIPs(ctx context.Context) ([]string, error) {
	var ips []string
	if err := c.rpcClient.Call(ctx, c.destination, "network.public_ips", nil, &ips); err != nil {
		return nil, err
	}
	return ips, nil
}

func (c *NodeClient) GetNetworkPrivateIPs(ctx context.Context, networkName string) ([]string, error) {
	params := map[string]any{
		"network_name": networkName,
	}
	var ips []string
	if err := c.rpcClient.Call(ctx, c.destination, "network.private_ips", params, &ips); err != nil {
		return nil, err
	}
	return ips, nil
}

func (c *NodeClient) GetNetworkInterfaces(ctx context.Context) (pkg.Interfaces, error) {
	var interfaces pkg.Interfaces
	if err := c.rpcClient.Call(ctx, c.destination, "network.interfaces", nil, &interfaces); err != nil {
		return pkg.Interfaces{}, err
	}
	return interfaces, nil
}

func (c *NodeClient) SetNetworkPublicNIC(ctx context.Context, device string) error {
	params := map[string]any{
		"device": device,
	}
	return c.rpcClient.Call(ctx, c.destination, "network.set_public_nic", params, nil)
}

func (c *NodeClient) GetNetworkPublicNIC(ctx context.Context) (pkg.ExitDevice, error) {
	var device pkg.ExitDevice
	if err := c.rpcClient.Call(ctx, c.destination, "network.get_public_nic", nil, &device); err != nil {
		return pkg.ExitDevice{}, err
	}
	return device, nil
}

// Deployment Methods

func (c *NodeClient) DeploymentDeploy(ctx context.Context, deployment gridtypes.Deployment) error {
	return c.rpcClient.Call(ctx, c.destination, "deployment.deploy", deployment, nil)
}

func (c *NodeClient) DeploymentUpdate(ctx context.Context, deployment gridtypes.Deployment) error {
	return c.rpcClient.Call(ctx, c.destination, "deployment.update", deployment, nil)
}

func (c *NodeClient) DeploymentGet(ctx context.Context, contractID uint64) (gridtypes.Deployment, error) {
	params := map[string]any{
		"contract_id": contractID,
	}
	var deployment gridtypes.Deployment
	if err := c.rpcClient.Call(ctx, c.destination, "deployment.get", params, &deployment); err != nil {
		return gridtypes.Deployment{}, err
	}
	return deployment, nil
}

func (c *NodeClient) DeploymentList(ctx context.Context) ([]gridtypes.Deployment, error) {
	var deployments []gridtypes.Deployment
	if err := c.rpcClient.Call(ctx, c.destination, "deployment.list", nil, &deployments); err != nil {
		return nil, err
	}
	return deployments, nil
}

func (c *NodeClient) DeploymentChanges(ctx context.Context, contractID uint64) ([]gridtypes.Workload, error) {
	params := map[string]any{
		"contract_id": contractID,
	}
	var changes []gridtypes.Workload
	if err := c.rpcClient.Call(ctx, c.destination, "deployment.changes", params, &changes); err != nil {
		return nil, err
	}
	return changes, nil
}

// Other Methods

func (c *NodeClient) GetGpuList(ctx context.Context) ([]pkg.GPUInfo, error) {
	var gpus []pkg.GPUInfo
	if err := c.rpcClient.Call(ctx, c.destination, "gpu.list", nil, &gpus); err != nil {
		return nil, err
	}
	return gpus, nil
}

func (c *NodeClient) GetStoragePools(ctx context.Context) ([]pkg.PoolMetrics, error) {
	var pools []pkg.PoolMetrics
	if err := c.rpcClient.Call(ctx, c.destination, "storage.pools", nil, &pools); err != nil {
		return nil, err
	}
	return pools, nil
}

func (c *NodeClient) GetStatistics(ctx context.Context) (pkg.Counters, error) {
	var stats pkg.Counters
	if err := c.rpcClient.Call(ctx, c.destination, "statistics", nil, &stats); err != nil {
		return pkg.Counters{}, err
	}
	return stats, nil
}

func (c *NodeClient) GetLocation(ctx context.Context) (geoip.Location, error) {
	var location geoip.Location
	if err := c.rpcClient.Call(ctx, c.destination, "location.get", nil, &location); err != nil {
		return geoip.Location{}, err
	}
	return location, nil
}
