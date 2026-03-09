# Storage Module

## ZBus

Storage module is available on zbus over the following channel

| module | object | version |
|--------|--------|---------|
| storage|[storage](#interface)| 0.0.1|

## Introduction

This module is responsible for managing everything related to storage. On start, storaged takes ownership of all node disks and separates them into two sets:

- **SSD pools**: One btrfs pool per SSD disk. Used for subvolumes (read-write layers), virtual disks (VM storage), and system cache.
- **HDD pools**: One btrfs pool per HDD disk. Used exclusively for 0-DB device allocation.

The module provides three storage primitives:

- **Subvolume** (with quota): A btrfs subvolume used by `flistd` to support read-write operations on flists. Used as rootfs for containers and VMs. Only created on SSD pools.
  - On boot, a permanent subvolume `zos-cache` is always created (starting at 5 GiB) and bind-mounted at `/var/cache`. This volume holds system state and downloaded file caches.
- **VDisk** (virtual disk): A sparse file with Copy-on-Write disabled (`FS_NOCOW_FL`), used as block storage for virtual machines. Only created on SSD pools inside a `vdisks` subvolume.
- **Device**: A btrfs subvolume named `zdb` inside an HDD pool, allocated to a single 0-DB service. One 0-DB instance can serve multiple namespaces for multiple users. Only created on HDD pools.

ZOS can operate without HDDs (it will not serve ZDB workloads), but not without SSDs. A node with no SSD will never register on the grid.

## Architecture

### Pool Organization

```
Physical Disk (SSD)          Physical Disk (HDD)
    |                            |
    v                            v
btrfs pool (mounted at       btrfs pool (mounted at
/mnt/<label>)                /mnt/<label>)
    |                            |
    +-- zos-cache (subvolume)    +-- zdb (subvolume -> 0-DB device)
    +-- <workload> (subvolume)
    +-- vdisks/ (subvolume)
         +-- <vm-disk> (sparse file)
```

### Device Type Detection

The module determines whether a disk is SSD or HDD using:
1. A `.seektime` file persisted at the pool root (survives reboots)
2. Fallback to the `seektime` tool or device rotational flag from lsblk

### Mount Points

| Resource | Path |
|----------|------|
| Pools | `/mnt/<pool-label>` |
| Cache | `/var/cache` (bind mount to `zos-cache` subvolume) |
| Volumes | `/mnt/<pool-label>/<volume-name>` |
| VDisks | `/mnt/<pool-label>/vdisks/<disk-id>` |
| Devices (0-DB) | `/mnt/<pool-label>/zdb` |

## On Node Booting

When the module boots:

1. Scans all available block devices using `lsblk`
2. For each device not already used by a pool, creates a new btrfs filesystem (all pools use `RaidSingle` policy)
3. Mounts all available pools
4. Detects device type (SSD/HDD) for each pool
5. Ensures a cache subvolume exists. If none is found, creates one on an SSD pool and bind-mounts it at `/var/cache`. Falls back to tmpfs if no SSD is available (sets `LimitedCache` flag)
6. Starts cache monitoring goroutine (checks every 5 minutes, auto-grows at 60% utilization, shrinks below 20%)
7. Shuts down and spins down unused HDD pools to save power
8. Starts periodic disk power management

### zinit unit

The zinit unit file specifies the command line, test command, and boot ordering.

Storage module is a dependency for almost all other system modules, hence it has high boot precedence (calculated on boot) by zinit based on the configuration.

The storage module is only considered running if (and only if) `/var/cache` is ready:

```yaml
exec: storaged
test: mountpoint /var/cache
```

## Cache Management

The system cache is a special btrfs subvolume (`zos-cache`) that stores persistent system state and downloaded files.

| Parameter | Value |
|-----------|-------|
| Initial size | 5 GiB |
| Check interval | 5 minutes |
| Grow threshold | 60% utilization |
| Shrink threshold | 20% utilization |
| Fallback | tmpfs (if no SSD available) |

## Pool Selection Policies

When creating volumes or disks, the module selects a pool using one of these policies:

- **SSD Only**: Only considers SSD pools (used for volumes and vdisks)
- **HDD Only**: Only considers HDD pools (used for 0-DB device allocation)
- **SSD First**: Prefers SSD pools, falls back to HDD

Mounted pools are always prioritized over unmounted ones to avoid unnecessary spin-ups.

## Error Handling

The module tracks two categories of failures:

- **Broken Pools**: Pools that fail to mount. Tracked and reported via `BrokenPools()`.
- **Broken Devices**: Devices that fail formatting, mounting, or type detection. Tracked and reported via `BrokenDevices()`.

These are exposed through the interface for monitoring and diagnostics.

## Thread Safety

All pool and volume operations are protected by a `sync.RWMutex`. Concurrent reads (lookups, listings) are allowed, while writes (create, delete, resize) are serialized.

## Consumers

Other modules access storage via zbus stubs:

| Consumer | Operations Used |
|----------|----------------|
| VM provisioner (`pkg/primitives/vm/`) | DiskCreate, DiskFormat, DiskWrite, DiskDelete |
| Volume provisioner (`pkg/primitives/volume/`) | VolumeCreate, VolumeDelete, VolumeLookup |
| ZMount provisioner (`pkg/primitives/zmount/`) | VolumeCreate, VolumeUpdate, VolumeDelete |
| ZDB provisioner (`pkg/primitives/zdb/`) | DeviceAllocate, DeviceLookup |
| Capacity oracle (`pkg/capacity/`) | Total, Metrics |

## Interface

```go
// StorageModule is the storage subsystem interface
// this should allow you to work with the following types of storage medium
// - full disks (device) (these are used by zdb)
// - subvolumes these are used as a read-write layers for 0-fs mounts
// - vdisks are used by zmachines
// this works as following:
// a storage module maintains a list of ALL disks on the system
// separated in 2 sets of pools (SSDs, and HDDs)
// ssd pools can only be used for
// - subvolumes
// - vdisks
// hdd pools are only used by zdb as one disk
type StorageModule interface {
	// Cache method return information about zos cache volume
	Cache() (Volume, error)

	// Total gives the total amount of storage available for a device type
	Total(kind DeviceType) (uint64, error)
	// BrokenPools lists the broken storage pools that have been detected
	BrokenPools() []BrokenPool
	// BrokenDevices lists the broken devices that have been detected
	BrokenDevices() []BrokenDevice
	//Monitor returns stats stream about pools
	Monitor(ctx context.Context) <-chan PoolsStats

	// Volume management

	// VolumeCreate creates a new volume
	VolumeCreate(name string, size gridtypes.Unit) (Volume, error)

	// VolumeUpdate updates the size of an existing volume
	VolumeUpdate(name string, size gridtypes.Unit) error

	// VolumeLookup return volume information for given name
	VolumeLookup(name string) (Volume, error)

	// VolumeDelete deletes a volume by name
	VolumeDelete(name string) error

	// VolumeList list all volumes
	VolumeList() ([]Volume, error)

	// Virtual disk management

	// DiskCreate creates a virtual disk given name and size
	DiskCreate(name string, size gridtypes.Unit) (VDisk, error)

	// DiskResize resizes the disk to given size
	DiskResize(name string, size gridtypes.Unit) (VDisk, error)

	// DiskWrite writes the given raw image to disk
	DiskWrite(name string, image string) error

	// DiskFormat makes sure disk has filesystem, if it already formatted nothing happens
	DiskFormat(name string) error

	// DiskLookup looks up vdisk by name
	DiskLookup(name string) (VDisk, error)

	// DiskExists checks if disk exists
	DiskExists(name string) bool

	// DiskDelete deletes a disk
	DiskDelete(name string) error

	DiskList() ([]VDisk, error)
	// Device management

	//Devices list all "allocated" devices
	Devices() ([]Device, error)

	// DeviceAllocate allocates a new device (formats and give a new ID)
	DeviceAllocate(min gridtypes.Unit) (Device, error)

	// DeviceLookup inspects a previously allocated device
	DeviceLookup(name string) (Device, error)
}
```
