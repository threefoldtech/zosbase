package vmlight

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosbase/pkg"
	"github.com/threefoldtech/zosbase/pkg/environment"
	"github.com/threefoldtech/zosbase/pkg/gridtypes"
	"github.com/threefoldtech/zosbase/pkg/gridtypes/zos"
	"github.com/threefoldtech/zosbase/pkg/primitives/vmgpu"
	provision "github.com/threefoldtech/zosbase/pkg/provision"
	"github.com/threefoldtech/zosbase/pkg/stubs"
)

const (
	cloudContainerName = "cloud-container"
)

// ZMachine type
type ZMachine = zos.ZMachineLight

var (
	_ provision.Manager     = (*Manager)(nil)
	_ provision.Initializer = (*Manager)(nil)
)

type Manager struct {
	zbus zbus.Client
}

func NewManager(zbus zbus.Client) *Manager {
	return &Manager{zbus}
}

func (m *Manager) Initialize(ctx context.Context) error {
	return vmgpu.InitGPUs()
}

func (p *Manager) Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	return p.virtualMachineProvisionImpl(ctx, wl)
}

func (p *Manager) vmMounts(ctx context.Context, deployment *gridtypes.Deployment, mounts []zos.MachineMount, format bool, vm *pkg.VM) error {
	for _, mount := range mounts {
		wl, err := deployment.Get(mount.Name)
		if err != nil {
			return errors.Wrapf(err, "failed to get mount '%s' workload", mount.Name)
		}
		if wl.Result.State != gridtypes.StateOk {
			return fmt.Errorf("invalid disk '%s' state", mount.Name)
		}
		switch wl.Type {
		case zos.ZMountType:
			if err := p.mountDisk(ctx, wl, mount, format, vm); err != nil {
				return err
			}
		case zos.QuantumSafeFSType:
			if err := p.mountQsfs(wl, mount, vm); err != nil {
				return err
			}
		case zos.VolumeType:
			if err := p.mountVolume(ctx, wl, mount, vm); err != nil {
				return err
			}
		default:
			return fmt.Errorf("expecting a reservation of type '%s' or '%s' for disk '%s'", zos.ZMountType, zos.QuantumSafeFSType, mount.Name)
		}
	}
	return nil
}

func (p *Manager) mountDisk(ctx context.Context, wl *gridtypes.WorkloadWithID, mount zos.MachineMount, format bool, vm *pkg.VM) error {
	storage := stubs.NewStorageModuleStub(p.zbus)

	info, err := storage.DiskLookup(ctx, wl.ID.String())
	if err != nil {
		return errors.Wrapf(err, "failed to inspect disk '%s'", mount.Name)
	}

	if format {
		if err := storage.DiskFormat(ctx, wl.ID.String()); err != nil {
			return errors.Wrap(err, "failed to prepare mount")
		}
	}

	vm.Disks = append(vm.Disks, pkg.VMDisk{Path: info.Path, Target: mount.Mountpoint})

	return nil
}

func (p *Manager) mountVolume(ctx context.Context, wl *gridtypes.WorkloadWithID, mount zos.MachineMount, vm *pkg.VM) error {
	storage := stubs.NewStorageModuleStub(p.zbus)

	volume, err := storage.VolumeLookup(ctx, wl.ID.String())
	if err != nil {
		return fmt.Errorf("failed to lookup volume %q: %w", wl.ID.String(), err)
	}

	vm.Shared = append(vm.Shared, pkg.SharedDir{ID: wl.Name.String(), Path: volume.Path, Target: mount.Mountpoint})
	return nil
}

func (p *Manager) mountQsfs(wl *gridtypes.WorkloadWithID, mount zos.MachineMount, vm *pkg.VM) error {
	var info zos.QuatumSafeFSResult
	if err := wl.Result.Unmarshal(&info); err != nil {
		return fmt.Errorf("invalid qsfs result '%s': %w", mount.Name, err)
	}

	vm.Shared = append(vm.Shared, pkg.SharedDir{ID: wl.Name.String(), Path: info.Path, Target: mount.Mountpoint})
	return nil
}

func (p *Manager) virtualMachineProvisionImpl(ctx context.Context, wl *gridtypes.WorkloadWithID) (result zos.ZMachineLightResult, err error) {
	var (
		network = stubs.NewNetworkerLightStub(p.zbus)
		flist   = stubs.NewFlisterStub(p.zbus)
		vm      = stubs.NewVMModuleStub(p.zbus)

		config ZMachine
	)
	if vm.Exists(ctx, wl.ID.String()) {
		return result, provision.ErrNoActionNeeded
	}

	if err := json.Unmarshal(wl.Data, &config); err != nil {
		return result, errors.Wrap(err, "failed to decode reservation schema")
	}

	if len(config.GPU) != 0 && !provision.IsRentedNode(ctx) {
		// you cannot use GPU unless this is a rented node
		return result, fmt.Errorf("usage of GPU is not allowed unless node is rented")
	}

	machine := pkg.VM{
		Name:       wl.ID.String(),
		CPU:        config.ComputeCapacity.CPU,
		Memory:     config.ComputeCapacity.Memory,
		Entrypoint: config.Entrypoint,
		KernelArgs: pkg.KernelArgs{},
	}

	// expand GPUs
	devices, err := vmgpu.ExpandGPUs(config.GPU)
	if err != nil {
		return result, errors.Wrap(err, "failed to prepare requested gpu device(s)")
	}

	for _, device := range devices {
		gpuDevice := fmt.Sprintf("%s,iommu=on", device.Slot)
		machine.Devices = append(machine.Devices, gpuDevice)
	}

	// the config is validated by the engine. we now only support only one
	// private network
	if len(config.Network.Interfaces) != 1 {
		return result, fmt.Errorf("only one private network is support")
	}
	netConfig := config.Network.Interfaces[0]

	result.ID = wl.ID.String()
	result.IP = netConfig.IP.String()

	deployment, err := provision.GetDeployment(ctx)
	if err != nil {
		return result, errors.Wrap(err, "failed to get deployment")
	}
	networkInfo := pkg.VMNetworkInfo{
		Nameservers: []net.IP{net.ParseIP("8.8.8.8"), net.ParseIP("1.1.1.1"), net.ParseIP("2001:4860:4860::8888")},
	}

	defer func() {
		tapName := wl.ID.Unique(string(config.Network.Mycelium.Network))
		if err != nil {
			_ = network.Detach(ctx, tapName)
		}
	}()

	for _, nic := range config.Network.Interfaces {
		inf, err := p.newPrivNetworkInterface(ctx, deployment, wl, nic)
		if err != nil {
			return result, err
		}
		networkInfo.Ifaces = append(networkInfo.Ifaces, inf)
	}

	if config.Network.Mycelium != nil {
		inf, err := p.newMyceliumNetworkInterface(ctx, deployment, wl, config.Network.Mycelium)
		if err != nil {
			return result, err
		}
		networkInfo.Ifaces = append(networkInfo.Ifaces, inf)
		result.MyceliumIP = inf.IPs[0].IP.String()
	}
	// - mount flist RO
	mnt, err := flist.Mount(ctx, wl.ID.String(), config.FList, pkg.ReadOnlyMountOptions)
	if err != nil {
		return result, errors.Wrapf(err, "failed to mount flist: %s", wl.ID.String())
	}

	var imageInfo FListInfo
	// - detect type (container or VM)
	imageInfo, err = getFlistInfo(mnt)
	if err != nil {
		return result, err
	}

	log.Debug().Msgf("detected flist type: %+v", imageInfo)

	// mount cloud-container flist (or reuse) which has kernel, initrd and also firmware
	env := environment.MustGet()
	cloudContainerFlist, err := url.JoinPath(env.HubURL, "tf-autobuilder", "cloud-container-9dba60e.flist")
	if err != nil {
		return zos.ZMachineLightResult{}, errors.Wrap(err, "failed to construct cloud-container flist url")
	}

	hash, err := flist.FlistHash(ctx, cloudContainerFlist)
	if err != nil {
		return zos.ZMachineLightResult{}, errors.Wrap(err, "failed to get cloud-container flist hash")
	}

	// if the name changes (because flist changed, a new mount will be created)
	name := fmt.Sprintf("%s:%s", cloudContainerName, hash)
	// now mount cloud image also
	cloudImage, err := flist.Mount(ctx, name, cloudContainerFlist, pkg.ReadOnlyMountOptions)
	if err != nil {
		return result, errors.Wrap(err, "failed to mount cloud container base image")
	}

	if imageInfo.IsContainer() {
		if err = p.prepContainer(ctx, cloudImage, imageInfo, &machine, &config, &deployment, wl); err != nil {
			return result, err
		}
	} else {
		if err = p.prepVirtualMachine(ctx, cloudImage, imageInfo, &machine, &config, &deployment, wl); err != nil {
			return result, err
		}
	}

	// - Attach mounts
	// - boot
	machine.Network = networkInfo
	machine.Environment = config.Env
	machine.Hostname = wl.Name.String()

	machineInfo, err := vm.Run(ctx, machine)
	if err != nil {
		// attempt to delete the vm, should the process still be lingering
		log.Error().Err(err).Msg("cleaning up vm deployment duo to an error")
		_ = vm.Delete(ctx, wl.ID.String())
	}
	result.ConsoleURL = machineInfo.ConsoleURL
	return result, err
}

func (p *Manager) copyFile(srcPath string, destPath string, permissions os.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return errors.Wrapf(err, "Coludn't find %s on the node", srcPath)
	}
	defer src.Close()
	dest, err := os.OpenFile(destPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, permissions)
	if err != nil {
		return errors.Wrapf(err, "Coludn't create %s file", destPath)
	}
	defer dest.Close()
	_, err = io.Copy(dest, src)
	if err != nil {
		return errors.Wrapf(err, "Couldn't copy to %s", destPath)
	}
	return nil
}

func (p *Manager) Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	var (
		flist   = stubs.NewFlisterStub(p.zbus)
		network = stubs.NewNetworkerLightStub(p.zbus)
		vm      = stubs.NewVMModuleStub(p.zbus)
		storage = stubs.NewStorageModuleStub(p.zbus)

		cfg ZMachine
	)

	if err := json.Unmarshal(wl.Data, &cfg); err != nil {
		return errors.Wrap(err, "failed to decode reservation schema")
	}

	if _, err := vm.Inspect(ctx, wl.ID.String()); err == nil {
		if err := vm.Delete(ctx, wl.ID.String()); err != nil {
			return errors.Wrapf(err, "failed to delete vm %s", wl.ID)
		}
	}

	if err := flist.Unmount(ctx, wl.ID.String()); err != nil {
		log.Error().Err(err).Msg("failed to unmount machine flist")
	}

	volName := fmt.Sprintf("rootfs:%s", wl.ID.String())
	if err := storage.VolumeDelete(ctx, volName); err != nil {
		log.Error().Err(err).Str("name", volName).Msg("failed to delete rootfs volume")
	}

	tapName := wl.ID.Unique(string(cfg.Network.Mycelium.Network))

	if err := network.Detach(ctx, tapName); err != nil {
		return errors.Wrap(err, "could not clean up tap device")
	}

	return nil
}
