# Storage Light Module

## ZBus

Storage light module is available on zbus over the same channel as the full storage module:

| module | object | version |
|--------|--------|---------|
| storage|[storage](#interface)| 0.0.1|

## Introduction

`storage_light` is a lightweight variant of the [storage module](../storage/readme.md). It implements the same `StorageModule` interface and provides identical functionality to consumers, but has enhanced device initialization logic designed for nodes with pre-partitioned disks.

Both modules are interchangeable at the zbus level — other modules access storage via the same `StorageModuleStub` regardless of which variant is running.

## Differences from Storage

The key difference is in the **device initialization** phase during boot. The standard storage module treats each whole disk as a single btrfs pool. The light variant adds:

### 1. Partition-Aware Initialization

Instead of requiring whole disks, `storage_light` can work with individual partitions:

- Detects if a disk is already partitioned (has child partitions)
- Scans for unallocated space on partitioned disks using `parted`
- Creates new partitions in free space (minimum 5 GiB) for btrfs pools
- Refreshes device info after partition table changes

This allows ZOS to coexist with other operating systems or PXE boot partitions on the same disk.

### 2. PXE Partition Detection

Partitions labeled `ZOSPXE` are automatically skipped during initialization. This prevents the storage module from claiming boot partitions used for PXE network booting.

### 3. Enhanced Device Manager

The filesystem subpackage in `storage_light` extends the device manager with:

- `Children []DeviceInfo` field on `DeviceInfo` to track child partitions
- `UUID` field for btrfs filesystem identification
- `IsPartitioned()` method to check if a disk has child partitions
- `IsPXEPartition()` method to detect PXE boot partitions
- `GetUnallocatedSpaces()` method using `parted` to find free disk space
- `AllocateEmptySpace()` method to create partitions in free space
- `RefreshDeviceInfo()` method to reload device info after changes
- `ClearCache()` on the device manager interface for refreshing the device list

## Initialization Flow

The boot process is similar to the standard storage module but with added partition handling:

1. Load kernel parameters (detect VM, check MissingSSD)
2. Scan devices via DeviceManager
3. For each device:
   - **If whole disk (not partitioned)**: Create btrfs pool on the entire device (same as standard)
   - **If partitioned**:
     - Skip partitions labeled `ZOSPXE`
     - Process existing partitions that have btrfs filesystems
     - Scan for unallocated space using `parted`
     - Create new partitions in free space >= 5 GiB
     - Create btrfs pools on new partitions
   - Mount pool, detect device type (SSD/HDD)
   - Add to SSD or HDD pool arrays
4. Ensure cache exists (create if needed, start monitoring)
5. Shut down unused HDD pools
6. Start periodic disk power management

## When to Use Storage Light

Use `storage_light` instead of `storage` when:

- The node has disks with existing partition tables that must be preserved
- PXE boot partitions exist on the same disks
- The node dual-boots or shares disks with other systems
- Disks have been partially allocated and have free space that should be used

## Architecture

The overall architecture (pool types, mount points, cache management, volume/disk/device operations) is identical to the [standard storage module](../storage/readme.md). Refer to that document for details on:

- Pool organization (SSD vs HDD)
- Storage primitives (subvolumes, vdisks, devices)
- Cache management and auto-sizing
- Pool selection policies
- Error handling and broken device tracking
- Thread safety
- The `StorageModule` interface definition

## Interface

Same as the [standard storage module](../storage/readme.md#interface). Both variants implement the same `StorageModule` interface defined in `pkg/storage.go`.
