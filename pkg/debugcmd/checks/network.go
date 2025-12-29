package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	cnins "github.com/containernetworking/plugins/pkg/ns"
	"github.com/threefoldtech/zosbase/pkg"
	"github.com/threefoldtech/zosbase/pkg/gridtypes/zos"
	"github.com/threefoldtech/zosbase/pkg/network/namespace"
	"github.com/threefoldtech/zosbase/pkg/network/nr"
	"github.com/threefoldtech/zosbase/pkg/versioned"
	"github.com/threefoldtech/zosbase/pkg/zinit"
	"github.com/vishvananda/netlink"
)

const (
	networkdVolatileDir = "/var/run/cache/networkd"
	networksDir         = "networks"
	myceliumKeyDir      = "mycelium-key"
)

// CheckNetworkConfig verifies network configuration file exists and is valid
func CheckNetworkConfig(ctx context.Context, data *CheckData) HealthCheck {
	result := HealthCheck{
		Name: "network.config",
		OK:   false,
	}

	netID := zos.NetworkID(data.Twin, data.Workload.Name)
	netCfgPath := filepath.Join(networkdVolatileDir, networksDir, netID.String())

	_, raw, err := versioned.ReadFile(netCfgPath)
	if err != nil {
		result.Message = fmt.Sprintf("config file not found: %v", err)
		result.Evidence = map[string]interface{}{"path": netCfgPath, "netid": netID.String()}
		return result
	}

	var netCfg pkg.Network
	if err := json.Unmarshal(raw, &netCfg); err != nil {
		result.Message = fmt.Sprintf("config file invalid or unparseable: %v", err)
		result.Evidence = map[string]interface{}{"path": netCfgPath, "netid": netID.String()}
		return result
	}

	if netCfg.NetID != netID {
		result.Message = fmt.Sprintf("config netid mismatch: expected %s, got %s", netID.String(), netCfg.NetID.String())
		result.Evidence = map[string]interface{}{"expected": netID.String(), "got": netCfg.NetID.String()}
		return result
	}

	result.OK = true
	result.Message = "config valid"
	result.Evidence = map[string]interface{}{"path": netCfgPath, "netid": netID.String()}
	return result
}

// CheckNetworkNamespace verifies network namespace exists and is accessible
func CheckNetworkNamespace(ctx context.Context, data *CheckData) HealthCheck {
	result := HealthCheck{
		Name: "network.namespace",
		OK:   false,
	}

	netID := zos.NetworkID(data.Twin, data.Workload.Name)
	nsName := data.Network(ctx, netID)

	if !namespace.Exists(nsName) {
		result.Message = "namespace not found"
		result.Evidence = map[string]interface{}{"namespace": nsName}
		return result
	}

	result.OK = true
	result.Message = "namespace exists and accessible"
	result.Evidence = map[string]interface{}{"namespace": nsName}
	return result
}

// CheckNetworkInterfaces verifies required network interfaces exist inside namespace
func CheckNetworkInterfaces(ctx context.Context, data *CheckData) HealthCheck {
	result := HealthCheck{
		Name: "network.interfaces",
		OK:   false,
	}

	netID := zos.NetworkID(data.Twin, data.Workload.Name)
	nsName := data.Network(ctx, netID)

	nrr := nr.New(pkg.Network{NetID: netID}, filepath.Join(networkdVolatileDir, myceliumKeyDir))
	wgIface, _ := nrr.WGName()  // `w-*` iface
	nrIface, _ := nrr.NRIface() // `n-*` iface
	pubIface := "public"        // `public` iface
	// TODO: add mycelium iface if configured for the network

	netnsLinks := map[string]struct{}{}
	if netNS, err := namespace.GetByName(nsName); err == nil {
		_ = netNS.Do(func(_ cnins.NetNS) error {
			links, err := netlink.LinkList()
			if err == nil {
				for _, l := range links {
					netnsLinks[l.Attrs().Name] = struct{}{}
				}
			}
			return nil
		})
		netNS.Close()
	}

	missing := []string{}
	if _, ok := netnsLinks[wgIface]; !ok {
		missing = append(missing, wgIface)
	}
	if _, ok := netnsLinks[nrIface]; !ok {
		missing = append(missing, nrIface)
	}
	if _, ok := netnsLinks[pubIface]; !ok {
		missing = append(missing, pubIface)
	}

	if len(missing) > 0 {
		result.Message = fmt.Sprintf("missing interfaces: %v", missing)
		result.Evidence = map[string]interface{}{"namespace": nsName, "missing": missing}
		return result
	}

	result.OK = true
	result.Message = "all required interfaces present"
	result.Evidence = map[string]interface{}{"namespace": nsName}
	return result
}

// CheckNetworkBridge verifies network bridge exists and has members
func CheckNetworkBridge(ctx context.Context, data *CheckData) HealthCheck {
	netDir := "/sys/class/net"
	brIfaceDir := "brif"

	result := HealthCheck{
		Name: "network.bridge",
		OK:   false,
	}

	netID := zos.NetworkID(data.Twin, data.Workload.Name)
	nrr := nr.New(pkg.Network{NetID: netID}, filepath.Join(networkdVolatileDir, myceliumKeyDir))
	brName, _ := nrr.BridgeName()

	if _, err := os.Stat(filepath.Join(netDir, brName)); err != nil {
		result.Message = fmt.Sprintf("bridge not found: %v", err)
		result.Evidence = map[string]interface{}{"bridge": brName}
		return result
	}

	brifDir := filepath.Join(netDir, brName, brIfaceDir)
	ents, err := os.ReadDir(brifDir)
	if err != nil || len(ents) == 0 {
		result.Message = fmt.Sprintf("bridge has no members: %v", err)
		result.Evidence = map[string]interface{}{"bridge": brName}
		return result
	}

	// TODO: check if the members are up interfaces

	result.OK = true
	result.Message = "bridge has members"
	result.Evidence = map[string]interface{}{"bridge": brName}
	return result
}

// CheckNetworkMycelium verifies mycelium service is running (if configured)
func CheckNetworkMycelium(ctx context.Context, data *CheckData) HealthCheck {
	result := HealthCheck{
		Name: "network.mycelium",
		OK:   false,
	}

	netID := zos.NetworkID(data.Twin, data.Workload.Name)
	nrr := nr.New(pkg.Network{NetID: netID}, filepath.Join(networkdVolatileDir, myceliumKeyDir))
	service := nrr.MyceliumServiceName()

	st, err := zinit.Default().Status(service)
	if err != nil {
		result.Message = fmt.Sprintf("cannot get service status: %v", err)
		result.Evidence = map[string]interface{}{"service": service}
		return result
	}

	if !st.State.Is(zinit.ServiceStateRunning) {
		result.Message = fmt.Sprintf("service not running: %s", st.State.String())
		result.Evidence = map[string]interface{}{"service": service, "state": st.State.String()}
		return result
	}

	result.OK = true
	result.Message = "service running"
	result.Evidence = map[string]interface{}{"service": service, "pid": st.Pid}
	return result
}
