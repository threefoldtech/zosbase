# Primitives Package

## Introduction

The primitives package implements the workload managers for all deployable workload types in 0-OS. It acts as the glue between the [provision engine](../provision/readme.md) and the underlying system daemons (storage, networking, VM, container, etc.), which it reaches over [zbus](https://github.com/threefoldtech/zbus).

Each workload type has a dedicated manager that knows how to provision, deprovision, and optionally update or pause that specific resource. The top-level `NewPrimitivesProvisioner` function wires all managers together into a single `provision.Provisioner` that the engine dispatches to by workload type.

## Architecture

```
                    provision engine
                         |
                   mapProvisioner
                    (dispatches by type)
                         |
    +--------+--------+--+--+--------+--------+
    |        |        |     |        |        |
  zmount  volume   network vm     zdb     gateway ...
    |        |        |     |        |        |
    v        v        v     v        v        v
  StorageStub      NetStub VMStub ContainerStub GatewayStub
    (zbus)          (zbus)  (zbus)   (zbus)      (zbus)
```

## Manager Interface

Every workload manager must implement the `provision.Manager` interface:

```go
type Manager interface {
    Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error)
    Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error
}
```

Managers may optionally implement additional interfaces:

| Interface | Methods | Purpose |
|-----------|---------|---------|
| `Initializer` | `Initialize(ctx)` | Called once before the engine starts (e.g., GPU VFIO binding, ZDB container recovery) |
| `Updater` | `Update(ctx, wl)` | Live update of workload parameters (e.g., resize volume, change ZDB password) |
| `Pauser` | `Pause(ctx, wl)` / `Resume(ctx, wl)` | Suspend/resume a workload without deleting it |

## Workload Type Registration

All managers are registered in `provisioner.go`:

```go
func NewPrimitivesProvisioner(zbus zbus.Client) provision.Provisioner {
    managers := map[gridtypes.WorkloadType]provision.Manager{
        zos.ZMountType:           zmount.NewManager(zbus),
        zos.ZLogsType:            zlogs.NewManager(zbus),
        zos.QuantumSafeFSType:    qsfs.NewManager(zbus),
        zos.ZDBType:              zdb.NewManager(zbus),
        zos.NetworkType:          network.NewManager(zbus),
        zos.PublicIPType:         pubip.NewManager(zbus),
        zos.PublicIPv4Type:       pubip.NewManager(zbus),
        zos.ZMachineType:         vm.NewManager(zbus),
        zos.NetworkLightType:     netlight.NewManager(zbus),
        zos.ZMachineLightType:    vmlight.NewManager(zbus),
        zos.VolumeType:           volume.NewManager(zbus),
        zos.GatewayNameProxyType: gateway.NewNameManager(zbus),
        zos.GatewayFQDNProxyType: gateway.NewFQDNManager(zbus),
    }
    return provision.NewMapProvisioner(managers)
}
```

## Workload Types

### ZMount (`zmount`)

Allocates a raw virtual disk (sparse file with COW disabled) via `StorageModule.DiskCreate`. Used as boot disk or additional data disk for VMs.

- **Supports**: Provision, Deprovision, Update (grow only, not while VM is running)
- **Storage**: SSD pools only
- **zbus stubs**: `StorageModuleStub`, `VMModuleStub`

### Volume (`volume`)

Creates a btrfs subvolume with quota via `StorageModule.VolumeCreate`. Unlike a zmount (block device), a volume is a filesystem path that can be bind-mounted into VMs as a shared directory (virtio-fs).

- **Supports**: Provision, Deprovision, Update (grow only)
- **Storage**: SSD pools only
- **zbus stubs**: `StorageModuleStub`

### Network (`network`)

Creates a WireGuard-based private network resource via `Networker.CreateNR`. Carries the full `zos.Network` config including WireGuard peers and subnet allocations.

- **Supports**: Provision, Deprovision, Update (upsert semantics)
- **zbus stubs**: `NetworkerStub`

### Network Light (`network-light`)

Creates a lightweight Mycelium-only network via `NetworkerLight.Create`. Used by light VMs that only need overlay networking.

- **Supports**: Provision, Deprovision, Update (upsert semantics)
- **zbus stubs**: `NetworkerLightStub`

### ZMachine (`vm`)

The most complex workload type. Provisions either:

- **Container VM** (flist without `/image.raw`): Mounts flist read-write with a btrfs volume overlay, injects cloud-container kernel + initrd, boots with VirtioFS.
- **Full VM** (flist with `/image.raw`): Writes the disk image to the first ZMount, boots directly from disk using `hypervisor-fw`.

Network interfaces are set up as tap devices: private network (WireGuard), optional Yggdrasil (planetary), optional Mycelium, optional public IPv4/IPv6. GPU passthrough is supported on rented nodes via VFIO.

- **Supports**: Provision, Deprovision, Initialize (GPU VFIO binding), Pause/Resume (VM lock)
- **zbus stubs**: `VMModuleStub`, `FlisterStub`, `StorageModuleStub`, `NetworkerStub`

### ZMachine Light (`vm-light`)

Same as ZMachine but uses `NetworkerLightStub`. Only supports Mycelium and private network interfaces (no Yggdrasil, no public IP).

- **Supports**: Provision, Deprovision, Initialize, Pause/Resume
- **zbus stubs**: `VMModuleStub`, `FlisterStub`, `StorageModuleStub`, `NetworkerLightStub`

### ZDB (`zdb`)

Manages [0-db](https://github.com/threefoldtech/0-DB) namespaces. Multiple ZDB namespaces share a single container per storage device. On provision:

1. Finds a container with free space, or allocates a new HDD device and starts a new container (via flist mount + `ContainerModule.Run`)
2. Connects to the ZDB unix socket, creates the namespace with the requested mode/password/size
3. Returns IPs (public, yggdrasil, mycelium) + port 9900

- **Supports**: Provision, Deprovision, Initialize (restart crashed containers, upgrade flist), Update (resize, change password/public — not mode), Pause/Resume (namespace lock)
- **Storage**: HDD pools only
- **zbus stubs**: `StorageModuleStub`, `FlisterStub`, `ContainerModuleStub`, `NetworkerStub` or `NetworkerLightStub`

### QSFS (`qsfs`)

Mounts a [Quantum Safe Filesystem](https://github.com/threefoldtech/quantum-storage) via `QSFSDModule.Mount`. Returns a mount path and metrics endpoint.

- **Supports**: Provision, Deprovision, Update
- **zbus stubs**: `QSFSDStub`

### Public IP (`pubip`)

Allocates and configures a public IPv4 and/or IPv6 address for a VM. Selects IPs from the contract's reserved pool, computes IPv6 SLAAC address from the node's public prefix, and sets up nftables filter rules.

`PublicIPv4Type` is kept for backward compatibility and maps to the same manager.

- **Supports**: Provision, Deprovision
- **zbus stubs**: `NetworkerStub`, `SubstrateGatewayStub`

### Gateway Name Proxy (`gateway/name`)

Sets up a reverse proxy where a subdomain is allocated by the grid. Calls `Gateway.SetNamedProxy` and returns the assigned FQDN.

- **Supports**: Provision, Deprovision
- **zbus stubs**: `GatewayStub`

### Gateway FQDN Proxy (`gateway/fqdn`)

Sets up a reverse proxy for a user-owned FQDN. Calls `Gateway.SetFQDNProxy`.

- **Supports**: Provision, Deprovision
- **zbus stubs**: `GatewayStub`

### ZLogs (`zlogs`)

Attaches a log stream from a running VM to an external destination. Finds the referenced ZMachine workload, extracts its network namespace, then calls `VMModule.StreamCreate`.

- **Supports**: Provision, Deprovision
- **zbus stubs**: `VMModuleStub`, `NetworkerStub`

## Helper Packages

### vmgpu

Shared GPU utility used by both `vm` and `vm-light`:

- `InitGPUs()`: Loads VFIO kernel modules, unbinds boot VGA if needed, binds all GPUs in each IoMMU group to the `vfio-pci` driver.
- `ExpandGPUs(gpus)`: For each requested GPU, returns all PCI devices in the same IoMMU group that must be passed through together (excludes PCI bridges and audio controllers).

## Statistics Interceptor

`Statistics` wraps the inner `Provisioner` as middleware. Before provisioning, it checks whether the node has enough capacity (memory, primarily) to satisfy the workload's requirements. It computes usable memory as:

```
usable = total_ram - max(theoretical_reserved, actual_used)
```

Where `theoretical_reserved` is the sum of all active workload MRU claims. This prevents over-commitment even when VMs haven't yet used their full allocation.

It also injects the current consumed capacity into the context so downstream managers can access it via `primitives.GetCapacity(ctx)`.

`NewStatisticsStream` provides a streaming interface (`pkg.Statistics`) that:
- Streams capacity updates every 2 minutes
- Reports total/used/system capacity, deployment counts, open TCP connections
- Lists GPUs with their allocation status (which contract is using each GPU)

## Key Patterns

### Idempotent Provision

Most managers check if a resource already exists before creating it. If it does, they return `provision.ErrNoActionNeeded` and the engine skips writing a new transaction. This makes re-provisioning on reboot safe.

### Workload ID as Resource Name

All resources are named `wl.ID.String()` (format: `<twin>-<contractID>-<name>`), making them globally unique and deterministic across reboots.

### Full vs Light Stack

The codebase has two parallel stacks:

| | Full | Light |
|--|------|-------|
| Network | `NetworkType` (WireGuard) | `NetworkLightType` (Mycelium only) |
| VM | `ZMachineType` | `ZMachineLightType` |
| Stubs | `NetworkerStub` | `NetworkerLightStub` |
| Features | WireGuard + Yggdrasil + Mycelium + Public IP | Mycelium only |

ZDB detects which stack to use via `kernel.GetParams().IsLight()`.

### Provision Order

The engine provisions workloads in type order (networks before VMs, storage before VMs) and deprovisions in reverse order. Within the same type, ZMount and Volume workloads are sorted largest-first.
