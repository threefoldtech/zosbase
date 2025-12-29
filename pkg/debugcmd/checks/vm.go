package checks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/threefoldtech/zosbase/pkg/gridtypes"
	"github.com/threefoldtech/zosbase/pkg/vm"
)

const vmdVolatileDir = "/var/run/cache/vmd"

// CheckVMConfig verifies VM configuration file exists and is valid
func CheckVMConfig(ctx context.Context, data *CheckData) HealthCheck {
	result := HealthCheck{
		Name: "vm.config",
		OK:   false,
	}

	workloadID, err := gridtypes.NewWorkloadID(data.Twin, data.Contract, data.Workload.Name)
	if err != nil { // this should never happen
		result.Message = fmt.Sprintf("invalid workload ID: %v", err)
		result.Evidence = map[string]interface{}{"workload_id": workloadID}
		return result
	}
	vmID := workloadID.String()
	cfgPath := filepath.Join(vmdVolatileDir, vmID)

	if _, err := os.Stat(cfgPath); err != nil {
		result.Message = fmt.Sprintf("config file not found: %v", err)
		result.Evidence = map[string]interface{}{"path": cfgPath}
		return result
	}

	_, err = vm.MachineFromFile(cfgPath)
	if err != nil {
		result.Message = "config file invalid or unparseable"
		result.Evidence = map[string]interface{}{"path": cfgPath}
		return result
	}

	result.OK = true
	result.Message = "config valid"
	result.Evidence = map[string]interface{}{"path": cfgPath}
	return result
}

// CheckVMVMD verifies VMD reports VM exists
func CheckVMVMD(ctx context.Context, data *CheckData) HealthCheck {
	result := HealthCheck{
		Name: "vm.vmd",
		OK:   false,
	}

	workloadID, err := gridtypes.NewWorkloadID(data.Twin, data.Contract, data.Workload.Name)
	if err != nil { // this should never happen
		result.Message = fmt.Sprintf("invalid workload ID: %v", err)
		result.Evidence = map[string]interface{}{"workload_id": workloadID}
		return result
	}
	vmID := workloadID.String()

	if !data.VM(ctx, vmID) {
		result.Message = "vmd reports VM does not exist"
		result.Evidence = map[string]interface{}{"vm_id": vmID}
		return result
	}

	result.OK = true
	result.Message = "vmd reports VM exists"
	result.Evidence = map[string]interface{}{"vm_id": vmID}
	return result
}

// CheckVMProcess verifies cloud-hypervisor process is running
func CheckVMProcess(ctx context.Context, data *CheckData) HealthCheck {
	result := HealthCheck{
		Name: "vm.process",
		OK:   false,
	}

	workloadID, err := gridtypes.NewWorkloadID(data.Twin, data.Contract, data.Workload.Name)
	if err != nil { // this should never happen
		result.Message = fmt.Sprintf("invalid workload ID: %v", err)
		result.Evidence = map[string]interface{}{"workload_id": workloadID}
		return result
	}
	vmID := workloadID.String()

	ps, err := vm.Find(vmID)
	if err != nil {
		result.Message = fmt.Sprintf("process not found: %v", err)
		result.Evidence = map[string]interface{}{"vm_id": vmID}
		return result
	}

	result.OK = true
	result.Message = "process running"
	result.Evidence = map[string]interface{}{"vm_id": vmID, "pid": ps.Pid}
	return result
}

// CheckVMDisks verifies all VM disk files exist
func CheckVMDisks(ctx context.Context, data *CheckData) HealthCheck {
	result := HealthCheck{
		Name: "vm.disks",
		OK:   false,
	}

	workloadID, err := gridtypes.NewWorkloadID(data.Twin, data.Contract, data.Workload.Name)
	if err != nil { // this should never happen
		result.Message = fmt.Sprintf("invalid workload ID: %v", err)
		result.Evidence = map[string]interface{}{"workload_id": workloadID}
		return result
	}
	vmID := workloadID.String()
	cfgPath := filepath.Join(vmdVolatileDir, vmID)

	machine, err := vm.MachineFromFile(cfgPath)
	if err != nil {
		result.Message = "config not available"
		result.Evidence = map[string]interface{}{"vm_id": vmID}
		return result
	}

	for _, disk := range machine.Disks {
		if disk.Path == "" {
			continue
		}

		_, err := os.Stat(disk.Path)
		if err != nil {
			result.Message = fmt.Sprintf("disk missing: %s", disk.Path)
			result.Evidence = map[string]interface{}{"path": disk.Path}
			return result
		}

		// TODO: could we check files on disk?
	}

	result.OK = true
	result.Message = "all disks valid"
	result.Evidence = map[string]interface{}{"vm_id": vmID}
	return result
}

// CheckVMVirtioFS verifies virtiofs sockets exist
func CheckVMVirtioFS(ctx context.Context, data *CheckData) HealthCheck {
	result := HealthCheck{
		Name: "vm.virtiofs",
		OK:   false,
	}

	workloadID, err := gridtypes.NewWorkloadID(data.Twin, data.Contract, data.Workload.Name)
	if err != nil { // this should never happen
		result.Message = fmt.Sprintf("invalid workload ID: %v", err)
		result.Evidence = map[string]interface{}{"workload_id": workloadID}
		return result
	}
	vmID := workloadID.String()
	cfgPath := filepath.Join(vmdVolatileDir, vmID)

	machine, err := vm.MachineFromFile(cfgPath)
	if err != nil {
		result.OK = true
		result.Message = fmt.Sprintf("config file invalid or unparseable: %v", err)
		result.Evidence = map[string]interface{}{"vm_id": vmID}
		return result
	}

	for i := range machine.FS {
		sock := filepath.Join("/var/run", fmt.Sprintf("virtio-%s-%d.socket", vmID, i))
		if _, err := os.Stat(sock); err != nil {
			result.Message = fmt.Sprintf("socket missing: %s", sock)
			result.Evidence = map[string]interface{}{"socket": sock}
			return result
		}
	}

	result.OK = true
	result.Message = "all virtiofs sockets present"
	result.Evidence = map[string]interface{}{"vm_id": vmID}
	return result
}

// TODO: add cloud-console check
