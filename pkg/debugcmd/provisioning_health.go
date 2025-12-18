package debugcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cnins "github.com/containernetworking/plugins/pkg/ns"
	"github.com/threefoldtech/zosbase/pkg"
	"github.com/threefoldtech/zosbase/pkg/gridtypes"
	"github.com/threefoldtech/zosbase/pkg/gridtypes/zos"
	"github.com/threefoldtech/zosbase/pkg/network/namespace"
	"github.com/threefoldtech/zosbase/pkg/network/nr"
	"github.com/threefoldtech/zosbase/pkg/versioned"
	"github.com/threefoldtech/zosbase/pkg/vm"
	"github.com/threefoldtech/zosbase/pkg/zinit"
	"github.com/vishvananda/netlink"
)

type ProvisioningHealthRequest struct {
	TwinID     uint32 `json:"twin_id"`
	ContractID uint64 `json:"contract_id"`
}

type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthDegraded  HealthStatus = "degraded"
	HealthUnhealthy HealthStatus = "unhealthy"
)

type HealthCheck struct {
	Name     string                 `json:"name"`
	OK       bool                   `json:"ok"`
	Message  string                 `json:"message,omitempty"`
	Evidence map[string]interface{} `json:"evidence,omitempty"`
}

type WorkloadHealth struct {
	WorkloadID string        `json:"workload_id"`
	Type       string        `json:"type"`
	Name       string        `json:"name"`
	Status     HealthStatus  `json:"status"`
	Checks     []HealthCheck `json:"checks"`
}

type ProvisioningHealthResponse struct {
	TwinID     uint32           `json:"twin_id"`
	ContractID uint64           `json:"contract_id"`
	Workloads  []WorkloadHealth `json:"workloads"`
}

func ParseProvisioningHealthRequest(payload []byte) (ProvisioningHealthRequest, error) {
	var req ProvisioningHealthRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return req, err
	}
	return req, nil
}

func ProvisioningHealth(ctx context.Context, deps Deps, req ProvisioningHealthRequest) (ProvisioningHealthResponse, error) {
	if req.TwinID == 0 {
		return ProvisioningHealthResponse{}, fmt.Errorf("twin_id is required")
	}
	if req.ContractID == 0 {
		return ProvisioningHealthResponse{}, fmt.Errorf("contract_id is required")
	}

	deployment, err := deps.Provision.Get(ctx, req.TwinID, req.ContractID)
	if err != nil {
		return ProvisioningHealthResponse{}, fmt.Errorf("failed to get deployment: %w", err)
	}

	out := ProvisioningHealthResponse{TwinID: req.TwinID, ContractID: req.ContractID}
	for _, wl := range deployment.Workloads {
		switch wl.Type {
		case zos.NetworkType:
			out.Workloads = append(out.Workloads, checkNetworkWorkload(ctx, deps, req.TwinID, req.ContractID, wl))
		case zos.ZMachineType, zos.ZMachineLightType:
			out.Workloads = append(out.Workloads, checkZMachineWorkload(ctx, deps, req.TwinID, req.ContractID, wl))
		default:
		}
	}

	return out, nil
}

type checkBuilder struct {
	checks []HealthCheck
}

func (b *checkBuilder) add(name string, ok bool, msg string, evidence map[string]interface{}) {
	b.checks = append(b.checks, HealthCheck{Name: name, OK: ok, Message: msg, Evidence: evidence})
}

func (b *checkBuilder) status() HealthStatus {
	fail := 0
	for _, c := range b.checks {
		if !c.OK {
			fail++
		}
	}
	if fail == 0 {
		return HealthHealthy
	}
	if fail == 1 {
		return HealthDegraded
	}
	return HealthUnhealthy
}

func checkNetworkWorkload(ctx context.Context, deps Deps, twin uint32, contract uint64, wl gridtypes.Workload) WorkloadHealth {
	const (
		networkdVolatileDir = "/var/run/cache/networkd"
		networksDir         = "networks"
		myceliumKeyDir      = "mycelium-key"

		prefixBridgeMycelium = "m-"
		prefixTap            = "t-"

		ifaceMyceliumBridge = "br-my"
		ifaceMyceliumTun    = "my"
		ifacePublic         = "public"
	)

	netID := zos.NetworkID(twin, wl.Name)
	workloadID, _ := gridtypes.NewWorkloadID(twin, contract, wl.Name)

	var b checkBuilder
	b.checks = make([]HealthCheck, 0, 16)

	netCfgPath := filepath.Join(networkdVolatileDir, networksDir, netID.String())
	ver, raw, err := versioned.ReadFile(netCfgPath)
	if err != nil {
		b.add("network.config.read", false, fmt.Sprintf("failed to read network config file: %v", err), map[string]interface{}{"path": netCfgPath, "netid": netID.String()})
	}
	var netCfg pkg.Network
	if err == nil {
		if err := json.Unmarshal(raw, &netCfg); err != nil {
			b.add("network.config.parse", false, fmt.Sprintf("failed to parse network config file: %v", err), map[string]interface{}{"path": netCfgPath, "version": ver.String()})
		} else if netCfg.NetID != netID {
			b.add("network.config.netid", false, "network config netid mismatch", map[string]interface{}{"expected": netID.String(), "got": netCfg.NetID.String(), "path": netCfgPath, "version": ver.String()})
		} else {
			b.add("network.config.netid", true, "network config exists and matches netid", map[string]interface{}{"path": netCfgPath, "netid": netID.String(), "version": ver.String()})
		}
	}
	myceliumConfigured := netCfg.Mycelium != nil

	nsName := deps.Network.Namespace(ctx, netID)
	if !namespace.Exists(nsName) {
		b.add("network.netns.exists", false, "network namespace not found", map[string]interface{}{"namespace": nsName})
	} else {
		b.add("network.netns.exists", true, "network namespace exists", map[string]interface{}{"namespace": nsName})
	}

	nrr := nr.New(pkg.Network{NetID: netID}, filepath.Join(networkdVolatileDir, myceliumKeyDir))
	wgIface, _ := nrr.WGName()
	nrIface, _ := nrr.NRIface()
	brName, _ := nrr.BridgeName()
	myBridgeName := fmt.Sprintf("%s%s", prefixBridgeMycelium, netID.String())

	netnsLinks := map[string]struct{}{}
	if netNS, err := namespace.GetByName(nsName); err != nil {
		b.add("network.netns.links", false, fmt.Sprintf("failed to open netns: %v", err), map[string]interface{}{"namespace": nsName})
	} else {
		_ = netNS.Do(func(_ cnins.NetNS) error {
			links, err := netlink.LinkList()
			if err != nil {
				return err
			}
			for _, l := range links {
				netnsLinks[l.Attrs().Name] = struct{}{}
			}
			return nil
		})
		_ = netNS.Close()
	}

	_, hasWg := netnsLinks[wgIface]
	_, hasNr := netnsLinks[nrIface]
	_, hasPublic := netnsLinks[ifacePublic]
	b.add("network.netns.iface.wg", hasWg, "wireguard interface presence in netns", map[string]interface{}{"namespace": nsName, "iface": wgIface})
	b.add("network.netns.iface.nr", hasNr, "netresource interface presence in netns", map[string]interface{}{"namespace": nsName, "iface": nrIface})
	b.add("network.netns.iface.public", hasPublic, "public iface presence in netns", map[string]interface{}{"namespace": nsName, "iface": ifacePublic})
	if myceliumConfigured {
		_, hasBrMy := netnsLinks[ifaceMyceliumBridge]
		_, hasMy := netnsLinks[ifaceMyceliumTun]
		b.add("network.netns.iface.br-my", hasBrMy, "mycelium bridge iface presence in netns", map[string]interface{}{"namespace": nsName, "iface": ifaceMyceliumBridge})
		b.add("network.netns.iface.my", hasMy, "mycelium tun iface presence in netns", map[string]interface{}{"namespace": nsName, "iface": ifaceMyceliumTun})
	}

	if _, err := os.Stat(filepath.Join("/sys/class/net", brName)); err != nil {
		b.add("network.bridge.exists", false, fmt.Sprintf("network bridge missing: %v", err), map[string]interface{}{"bridge": brName})
	} else {
		b.add("network.bridge.exists", true, "network bridge exists", map[string]interface{}{"bridge": brName})
	}
	if myceliumConfigured {
		if _, err := os.Stat(filepath.Join("/sys/class/net", myBridgeName)); err != nil {
			b.add("network.mycelium_bridge.exists", false, fmt.Sprintf("mycelium bridge missing: %v", err), map[string]interface{}{"bridge": myBridgeName})
		} else {
			b.add("network.mycelium_bridge.exists", true, "mycelium bridge exists", map[string]interface{}{"bridge": myBridgeName})
		}
	}

	checkBridgeMembers := func(checkPrefix, bridge string) {
		brifDir := filepath.Join("/sys/class/net", bridge, "brif")
		ents, err := os.ReadDir(brifDir)
		if err != nil {
			b.add(checkPrefix+".members", false, fmt.Sprintf("failed to read bridge members: %v", err), map[string]interface{}{"bridge": bridge, "path": brifDir})
			return
		}
		members := make([]string, 0, len(ents))
		for _, e := range ents {
			members = append(members, e.Name())
		}
		if len(members) == 0 {
			b.add(checkPrefix+".members", false, "bridge has no attached interfaces", map[string]interface{}{"bridge": bridge})
			return
		}
		b.add(checkPrefix+".members", true, "bridge has attached interfaces", map[string]interface{}{"bridge": bridge, "members": members})

		for _, m := range members {
			if !strings.HasPrefix(m, prefixTap) {
				b.add(checkPrefix+".member.tap_prefix", false, "bridge member does not have expected tap prefix (t-)", map[string]interface{}{"bridge": bridge, "member": m})
			} else {
				b.add(checkPrefix+".member.tap_prefix", true, "bridge member has expected tap prefix (t-)", map[string]interface{}{"bridge": bridge, "member": m})
			}

			oper := filepath.Join("/sys/class/net", m, "operstate")
			ob, err := os.ReadFile(oper)
			if err != nil {
				b.add(checkPrefix+".member.operstate", false, fmt.Sprintf("failed to read operstate: %v", err), map[string]interface{}{"bridge": bridge, "member": m, "path": oper})
				continue
			}
			state := strings.TrimSpace(string(ob))
			b.add(checkPrefix+".member.operstate", state == "up", "member operstate", map[string]interface{}{"bridge": bridge, "member": m, "operstate": state})
		}
	}

	checkBridgeMembers("network.bridge", brName)
	if myceliumConfigured {
		checkBridgeMembers("network.mycelium_bridge", myBridgeName)
	}

	if myceliumConfigured {
		service := fmt.Sprintf("mycelium-%s", netID.String())
		z := zinit.Default()
		exists, err := z.Exists(service)
		if err != nil {
			b.add("network.mycelium.service.exists", false, fmt.Sprintf("failed to query zinit: %v", err), map[string]interface{}{"service": service})
		} else if !exists {
			b.add("network.mycelium.service.exists", false, "mycelium service is not monitored in zinit", map[string]interface{}{"service": service})
		} else {
			st, err := z.Status(service)
			if err != nil {
				b.add("network.mycelium.service.status", false, fmt.Sprintf("failed to get service status: %v", err), map[string]interface{}{"service": service})
			} else {
				b.add("network.mycelium.service.running", st.State.Is(zinit.ServiceStateRunning), "mycelium service state", map[string]interface{}{"service": service, "state": st.State.String(), "pid": st.Pid})
			}
		}
	} else {
		b.add("network.mycelium.configured", true, "mycelium not configured for this network (skipped service check)", map[string]interface{}{"netid": netID.String()})
	}

	return WorkloadHealth{
		WorkloadID: workloadID.String(),
		Type:       string(wl.Type),
		Name:       string(wl.Name),
		Status:     b.status(),
		Checks:     b.checks,
	}
}

func checkZMachineWorkload(ctx context.Context, deps Deps, twin uint32, contract uint64, wl gridtypes.Workload) WorkloadHealth {
	workloadID, _ := gridtypes.NewWorkloadID(twin, contract, wl.Name)
	vmID := workloadID.String()

	var b checkBuilder
	b.checks = make([]HealthCheck, 0, 16)

	const vmdVolatileDir = "/var/run/cache/vmd"
	cfgPath := filepath.Join(vmdVolatileDir, vmID)
	if _, err := os.Stat(cfgPath); err != nil {
		b.add("vm.config.exists", false, fmt.Sprintf("vm config missing: %v", err), map[string]interface{}{"path": cfgPath})
	} else {
		b.add("vm.config.exists", true, "vm config exists", map[string]interface{}{"path": cfgPath})
	}

	b.add("vm.vmd.exists", deps.VM.Exists(ctx, vmID), "vmd reports VM exists", map[string]interface{}{"vm_id": vmID})

	if ps, err := vm.Find(vmID); err != nil {
		b.add("vm.process.cloud_hypervisor", false, fmt.Sprintf("cloud-hypervisor process not found: %v", err), map[string]interface{}{"vm_id": vmID})
	} else {
		b.add("vm.process.cloud_hypervisor", true, "cloud-hypervisor process found", map[string]interface{}{"vm_id": vmID, "pid": ps.Pid})
	}

	machine, err := vm.MachineFromFile(cfgPath)
	hasConsole := false
	if err != nil {
		b.add("vm.config.parse", false, fmt.Sprintf("failed to parse vm config: %v", err), map[string]interface{}{"path": cfgPath})
	} else {
		for _, nic := range machine.Interfaces {
			if nic.Console != nil {
				hasConsole = true
				break
			}
		}

		for _, d := range machine.Disks {
			if d.Path == "" {
				continue
			}
			if st, err := os.Stat(d.Path); err != nil {
				b.add("vm.disk.exists", false, fmt.Sprintf("disk path missing: %v", err), map[string]interface{}{"path": d.Path})
			} else if st.Size() == 0 {
				b.add("vm.disk.nonzero", false, "disk file size is 0", map[string]interface{}{"path": d.Path})
			} else {
				b.add("vm.disk.ok", true, "disk path exists", map[string]interface{}{"path": d.Path, "bytes": st.Size()})
			}
		}

		if len(machine.FS) == 0 {
			b.add("vm.virtiofsd.required", true, "no virtiofs shares configured (skipped virtiofsd check)", nil)
		} else {
			for i := range machine.FS {
				sock := filepath.Join("/var/run", fmt.Sprintf("virtio-%s-%d.socket", vmID, i))
				if _, err := os.Stat(sock); err != nil {
					b.add("vm.virtiofsd.socket", false, fmt.Sprintf("virtiofs socket missing: %v", err), map[string]interface{}{"socket": sock})
				} else {
					b.add("vm.virtiofsd.socket", true, "virtiofs socket exists", map[string]interface{}{"socket": sock})
				}
			}
		}
	}

	if err == nil {
		if hasConsole {
			if ok, pid := processExistsByName("cloud-console", vmID); !ok {
				b.add("vm.process.cloud_console", false, "cloud-console process not found (best-effort)", map[string]interface{}{"vm_id": vmID})
			} else {
				b.add("vm.process.cloud_console", true, "cloud-console process found (best-effort)", map[string]interface{}{"vm_id": vmID, "pid": pid})
			}
		} else {
			b.add("vm.console.configured", true, "vm has no console configured (skipped cloud-console check)", map[string]interface{}{"vm_id": vmID})
		}
	}

	return WorkloadHealth{
		WorkloadID: workloadID.String(),
		Type:       string(wl.Type),
		Name:       string(wl.Name),
		Status:     b.status(),
		Checks:     b.checks,
	}
}

func processExistsByName(binary, needle string) (bool, int) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false, 0
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := e.Name()
		pid := 0
		for _, r := range dir {
			if r < '0' || r > '9' {
				pid = 0
				break
			}
			pid = pid*10 + int(r-'0')
		}
		if pid == 0 {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join("/proc", dir, "cmdline"))
		if err != nil || len(cmdline) == 0 {
			continue
		}
		s := string(cmdline)
		if strings.Contains(s, binary) && strings.Contains(s, needle) {
			return true, pid
		}
	}
	return false, 0
}
