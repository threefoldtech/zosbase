package zosapi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

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

type debugDeploymentsListItem struct {
	TwinID     uint32                     `json:"twin_id"`
	ContractID uint64                     `json:"contract_id"`
	Workloads  []debugDeploymentsWorkload `json:"workloads"`
}

type debugDeploymentsWorkload struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	State string `json:"state"`
}

type debugWorkloadTransaction struct {
	Seq     int                   `json:"seq"`
	Type    string                `json:"type"`
	Name    string                `json:"name"`
	Created gridtypes.Timestamp   `json:"created"`
	State   gridtypes.ResultState `json:"state"`
	Message string                `json:"message"`
}

func (g *ZosAPI) debugDeploymentsListHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var args struct {
		TwinID uint32 `json:"twin_id"`
	}
	if len(payload) != 0 {
		// optional filter
		_ = json.Unmarshal(payload, &args)
	}

	twins := []uint32{args.TwinID}
	if args.TwinID == 0 {
		var err error
		twins, err = g.provisionStub.ListTwins(ctx)
		if err != nil {
			return nil, err
		}
	}

	items := make([]debugDeploymentsListItem, 0)
	for _, twin := range twins {
		deployments, err := g.provisionStub.List(ctx, twin)
		if err != nil {
			return nil, err
		}

		for _, deployment := range deployments {
			workloads := make([]debugDeploymentsWorkload, 0, len(deployment.Workloads))
			for _, wl := range deployment.Workloads {
				workloads = append(workloads, debugDeploymentsWorkload{
					Type:  string(wl.Type),
					Name:  string(wl.Name),
					State: string(wl.Result.State),
				})
			}

			items = append(items, debugDeploymentsListItem{
				TwinID:     deployment.TwinID,
				ContractID: deployment.ContractID,
				Workloads:  workloads,
			})
		}
	}

	return struct {
		Items []debugDeploymentsListItem `json:"items"`
	}{Items: items}, nil
}

func (g *ZosAPI) debugDeploymentGetHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var args struct {
		TwinID      uint32 `json:"twin_id"`
		ContractID  uint64 `json:"contract_id"`
		WithHistory bool   `json:"withhistory"`
	}
	if err := json.Unmarshal(payload, &args); err != nil {
		return nil, err
	}
	if args.TwinID == 0 {
		return nil, fmt.Errorf("twin_id is required")
	}
	if args.ContractID == 0 {
		return nil, fmt.Errorf("contract_id is required")
	}

	deployment, err := g.provisionStub.Get(ctx, args.TwinID, args.ContractID)
	if err != nil {
		return nil, err
	}

	if !args.WithHistory {
		return struct {
			Deployment gridtypes.Deployment `json:"deployment"`
		}{Deployment: deployment}, nil
	}

	history, err := g.provisionStub.Changes(ctx, args.TwinID, args.ContractID)
	if err != nil {
		return nil, err
	}

	transactions := make([]debugWorkloadTransaction, 0, len(history))
	for idx, wl := range history {
		transactions = append(transactions, debugWorkloadTransaction{
			Seq:     idx + 1,
			Type:    string(wl.Type),
			Name:    string(wl.Name),
			Created: wl.Result.Created,
			State:   wl.Result.State,
			Message: wl.Result.Error,
		})
	}

	return struct {
		Deployment gridtypes.Deployment       `json:"deployment"`
		History    []debugWorkloadTransaction `json:"history"`
	}{
		Deployment: deployment,
		History:    transactions,
	}, nil
}

func (g *ZosAPI) debugVMInfoHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var args struct {
		TwinID     uint32 `json:"twin_id"`
		ContractID uint64 `json:"contract_id"`
		VMName     string `json:"vm_name"`
		FullLogs   bool   `json:"full_logs"`
	}
	if err := json.Unmarshal(payload, &args); err != nil {
		return nil, err
	}
	if args.TwinID == 0 {
		return nil, fmt.Errorf("twin_id is required")
	}
	if args.ContractID == 0 {
		return nil, fmt.Errorf("contract_id is required")
	}
	if args.VMName == "" {
		return nil, fmt.Errorf("vm_name is required")
	}

	deployment, err := g.provisionStub.Get(ctx, args.TwinID, args.ContractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	vm, err := deployment.GetType(gridtypes.Name(args.VMName), zos.ZMachineType)
	if err != nil {
		return nil, fmt.Errorf("failed to get zmachine workload: %w", err)
	}
	vmID := vm.ID.String()

	info, err := g.vmStub.Inspect(ctx, vmID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect vm: %w", err)
	}

	// Logs: tailed by default, full only when requested.
	var raw string
	if args.FullLogs {
		raw, err = g.vmStub.LogsFull(ctx, vmID)
	} else {
		raw, err = g.vmStub.Logs(ctx, vmID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get vm logs: %w", err)
	}

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

	return struct {
		VMID string     `json:"vm_id"`
		Info pkg.VMInfo `json:"info"`
		Logs string     `json:"logs"`
	}{
		VMID: vmID,
		Info: info,
		Logs: logs,
	}, nil
}

type debugHealthStatus string

const (
	debugHealthHealthy   debugHealthStatus = "healthy"
	debugHealthDegraded  debugHealthStatus = "degraded"
	debugHealthUnhealthy debugHealthStatus = "unhealthy"
)

type debugHealthCheck struct {
	Name     string                 `json:"name"`
	OK       bool                   `json:"ok"`
	Message  string                 `json:"message,omitempty"`
	Evidence map[string]interface{} `json:"evidence,omitempty"`
}

type debugWorkloadHealth struct {
	WorkloadID string             `json:"workload_id"`
	Type       string             `json:"type"`
	Name       string             `json:"name"`
	Status     debugHealthStatus  `json:"status"`
	Checks     []debugHealthCheck `json:"checks"`
}

type debugCheckBuilder struct {
	checks []debugHealthCheck
}

func (b *debugCheckBuilder) add(name string, ok bool, msg string, evidence map[string]interface{}) {
	b.checks = append(b.checks, debugHealthCheck{
		Name:     name,
		OK:       ok,
		Message:  msg,
		Evidence: evidence,
	})
}

func (b *debugCheckBuilder) status() debugHealthStatus {
	return summarizeHealth(b.checks)
}

func (g *ZosAPI) debugProvisioningHealthHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var args struct {
		TwinID     uint32 `json:"twin_id"`
		ContractID uint64 `json:"contract_id"`
	}
	if err := json.Unmarshal(payload, &args); err != nil {
		return nil, err
	}
	if args.TwinID == 0 {
		return nil, fmt.Errorf("twin_id is required")
	}
	if args.ContractID == 0 {
		return nil, fmt.Errorf("contract_id is required")
	}

	deployment, err := g.provisionStub.Get(ctx, args.TwinID, args.ContractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	workloads := make([]debugWorkloadHealth, 0)
	for _, wl := range deployment.Workloads {
		switch wl.Type {
		case zos.NetworkType:
			workloads = append(workloads, g.checkNetworkWorkload(ctx, args.TwinID, args.ContractID, wl))
		case zos.ZMachineType, zos.ZMachineLightType:
			workloads = append(workloads, g.checkZMachineWorkload(ctx, args.TwinID, args.ContractID, wl))
		default:
			// ignore other workload types (for now)
		}
	}

	return struct {
		TwinID     uint32                `json:"twin_id"`
		ContractID uint64                `json:"contract_id"`
		Workloads  []debugWorkloadHealth `json:"workloads"`
	}{
		TwinID:     args.TwinID,
		ContractID: args.ContractID,
		Workloads:  workloads,
	}, nil
}

// Network workload checks:
// - config file exists and is versioned+parseable, contains correct netid
// - netns exists: n-<netid>
// - netns interfaces exist: n-<netid>, w-<netid>, public, (br-my, my optional if mycelium configured)
// - host bridges exist: b-<netid>, m-<netid>
// - host bridge members exist (brif not empty) and look sane:
//   - each member has t- prefix
//   - each member operstate is "up"
//
// - mycelium service exists and is running (only if mycelium configured in network config)
func (g *ZosAPI) checkNetworkWorkload(ctx context.Context, twin uint32, contract uint64, wl gridtypes.Workload) debugWorkloadHealth {
	const (
		networkdVolatileDir = "/var/run/cache/networkd"
		networksDir         = "networks"
		myceliumKeyDir      = "mycelium-key"

		prefixBridgeNetwork  = "b-"
		prefixBridgeMycelium = "m-"
		prefixTap            = "t-"

		ifaceMyceliumBridge = "br-my"
		ifaceMyceliumTun    = "my"
		ifacePublic         = "public"
	)

	netID := zos.NetworkID(twin, wl.Name)
	workloadID, _ := gridtypes.NewWorkloadID(twin, contract, wl.Name)

	var b debugCheckBuilder
	b.checks = make([]debugHealthCheck, 0, 16)

	// 1) config file exists and contains correct netid (versioned stream)
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

	// 2) wiring: namespace + core interfaces/bridges
	nsName := g.networkerStub.Namespace(ctx, netID)
	if !namespace.Exists(nsName) {
		b.add("network.netns.exists", false, "network namespace not found", map[string]interface{}{"namespace": nsName})
	} else {
		b.add("network.netns.exists", true, "network namespace exists", map[string]interface{}{"namespace": nsName})
	}

	// expected interface/bridge naming per nr.NetResource
	nrr := nr.New(pkg.Network{NetID: netID}, filepath.Join(networkdVolatileDir, myceliumKeyDir))
	wgIface, _ := nrr.WGName()
	nrIface, _ := nrr.NRIface()
	brName, _ := nrr.BridgeName()
	myBridgeName := fmt.Sprintf("%s%s", prefixBridgeMycelium, netID.String())
	networkBridgeName := fmt.Sprintf("%s%s", prefixBridgeNetwork, netID.String())
	_ = networkBridgeName // matches brName; kept for clarity

	// inside namespace: direct netlink probe (no filtering)
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

	// Only check mycelium-specific interfaces if mycelium is configured on the network.
	myceliumConfigured := netCfg.Mycelium != nil
	if myceliumConfigured {
		_, hasBrMy := netnsLinks[ifaceMyceliumBridge]
		_, hasMy := netnsLinks[ifaceMyceliumTun]
		b.add("network.netns.iface.br-my", hasBrMy, "mycelium bridge iface presence in netns", map[string]interface{}{"namespace": nsName, "iface": ifaceMyceliumBridge})
		b.add("network.netns.iface.my", hasMy, "mycelium tun iface presence in netns", map[string]interface{}{"namespace": nsName, "iface": ifaceMyceliumTun})
	}

	// host namespace bridges
	if _, err := os.Stat(filepath.Join("/sys/class/net", brName)); err != nil {
		b.add("network.bridge.exists", false, fmt.Sprintf("network bridge missing: %v", err), map[string]interface{}{"bridge": brName})
	} else {
		b.add("network.bridge.exists", true, "network bridge exists", map[string]interface{}{"bridge": brName})
	}
	if _, err := os.Stat(filepath.Join("/sys/class/net", myBridgeName)); err != nil {
		b.add("network.mycelium_bridge.exists", false, fmt.Sprintf("mycelium bridge missing: %v", err), map[string]interface{}{"bridge": myBridgeName})
	} else {
		b.add("network.mycelium_bridge.exists", true, "mycelium bridge exists", map[string]interface{}{"bridge": myBridgeName})
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

	// 3) mycelium zinit service (only if configured)
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
				ok := st.State.Is(zinit.ServiceStateRunning)
				b.add("network.mycelium.service.running", ok, "mycelium service state", map[string]interface{}{"service": service, "state": st.State.String(), "pid": st.Pid})
			}
		}
	} else {
		b.add("network.mycelium.configured", true, "mycelium not configured for this network (skipped service check)", map[string]interface{}{"netid": netID.String()})
	}

	return debugWorkloadHealth{
		WorkloadID: workloadID.String(),
		Type:       string(wl.Type),
		Name:       string(wl.Name),
		Status:     b.status(),
		Checks:     b.checks,
	}
}

// ZMachine workload checks:
// - VM config exists under vmd volatile config dir
// - VM exists according to vmd
// - cloud-hypervisor process exists for VM
// - VM config parse succeeds (MachineFromFile)
// - disk paths referenced by config exist and are non-zero
// - virtiofsd sockets exist if FS shares are configured
// - cloud-console process exists (best-effort)
func (g *ZosAPI) checkZMachineWorkload(ctx context.Context, twin uint32, contract uint64, wl gridtypes.Workload) debugWorkloadHealth {
	workloadID, _ := gridtypes.NewWorkloadID(twin, contract, wl.Name)
	vmID := workloadID.String()

	var b debugCheckBuilder
	b.checks = make([]debugHealthCheck, 0, 16)

	// 1) config file exists
	const vmdVolatileDir = "/var/run/cache/vmd"
	cfgPath := filepath.Join(vmdVolatileDir, vmID)
	if _, err := os.Stat(cfgPath); err != nil {
		b.add("vm.config.exists", false, fmt.Sprintf("vm config missing: %v", err), map[string]interface{}{"path": cfgPath})
	} else {
		b.add("vm.config.exists", true, "vm config exists", map[string]interface{}{"path": cfgPath})
	}

	// 2) vmd existence (zbus truth)
	vmdExists := g.vmStub.Exists(ctx, vmID)
	b.add("vm.vmd.exists", vmdExists, "vmd reports VM exists", map[string]interface{}{"vm_id": vmID})

	// 3) cloud-hypervisor process (host probe)
	if ps, err := vm.Find(vmID); err != nil {
		b.add("vm.process.cloud_hypervisor", false, fmt.Sprintf("cloud-hypervisor process not found: %v", err), map[string]interface{}{"vm_id": vmID})
	} else {
		b.add("vm.process.cloud_hypervisor", true, "cloud-hypervisor process found", map[string]interface{}{"vm_id": vmID, "pid": ps.Pid})
	}

	// 4) parse machine config to derive disks/fs and expected sockets
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

		// disks sanity
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

		// virtiofsd: if VM has FS entries, expect sockets under /var/run/virtio-<vmID>-<i>.socket
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

	// 5) cloud-console: only if the VM has console configured
	// (console is optional and not required for the VM to run).
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

	return debugWorkloadHealth{
		WorkloadID: workloadID.String(),
		Type:       string(wl.Type),
		Name:       string(wl.Name),
		Status:     b.status(),
		Checks:     b.checks,
	}
}

func summarizeHealth(checks []debugHealthCheck) debugHealthStatus {
	if len(checks) == 0 {
		return debugHealthHealthy
	}
	fail := 0
	for _, c := range checks {
		if !c.OK {
			fail++
		}
	}
	if fail == 0 {
		return debugHealthHealthy
	}
	// a single failed check is degraded; multiple is unhealthy
	if fail == 1 {
		return debugHealthDegraded
	}
	return debugHealthUnhealthy
}

// processExistsByName is a best-effort /proc scan for a process whose cmdline
// contains both `binary` and `needle`.
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
		// only numeric dirs
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
