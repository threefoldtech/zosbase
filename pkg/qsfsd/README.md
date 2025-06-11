# QSFS Package

The QSFS (Quantum Safe File System) package provides a secure and distributed file system implementation for the ZOS ecosystem. It leverages the [quantum-storage](https://github.com/threefoldtech/quantum-storage) technology to create file systems that aims to support unlimited local storage with remote backend capabilities for offload and backup.

QSFS is a distributed, encrypted file system that provides:

1. **Distributed Storage**: Data is written locally to 0‑DB and archived remotely via 0‑Stor in segments, making local storage effectively unlimited
2. **Standard Filesystem Interface**: Appears as a standard FUSE filesystem, making it seamless for user applications while managing all remote synchronization under the hood

### Use Cases

- **Decentralized Backup & Archiving**: Offload storage securely to remote nodes, ensuring long-term data retention with strong quantum-resistant safeguards.

## Core Components

### Main Structures

#### `QSFS` Struct
The primary service structure that manages QSFS operations:

```go
type QSFS struct {
    cl zbus.Client
    mountsPath string
    tombstonesPath string
}
```

**Fields:**
- `cl`: ZBus client for communicating with other ZOS modules
- `mountsPath`: Directory where all QSFS mounts are created (typically under `/mounts`)
- `tombstonesPath`: Directory storing tombstone files for workloads marked for deletion

**Methods:**
- `Mount(wlID string, cfg zos.QuantumSafeFS) (pkg.QSFSInfo, error)`: Creates and mounts a new QSFS volume
- `UpdateMount(wlID string, cfg zos.QuantumSafeFS) (pkg.QSFSInfo, error)`: Updates an existing QSFS mount configuration
- `SignalDelete(wlID string) error`: Marks a QSFS volume for deletion
- `Metrics() (pkg.QSFSMetrics, error)`: Retrieves network metrics for all QSFS volumes
- `Unmount(wlID string)`: Completely removes a QSFS volume and cleans up resources

#### `zstorConfig` Struct
Configuration structure for the zstor daemon:

```go
type zstorConfig struct {
    zos.QuantumSafeFSConfig
    ZDBDataDirPath  string `toml:"zdb_data_dir_path"`
    Socket          string `toml:"socket"`
    MetricsPort     uint32 `toml:"prometheus_port"`
    ZDBFSMountpoint string `toml:"zdbfs_mountpoint"`
}
```

**Fields:**
- `QuantumSafeFSConfig`: Embedded quantum-safe filesystem configuration
- `ZDBDataDirPath`: Path for ZDB data storage (default: `/data/data/zdbfs-data`)
- `Socket`: Unix socket path for zstor communication (default: `/var/run/zstor.sock`)
- `MetricsPort`: Port for Prometheus metrics (9100)
- `ZDBFSMountpoint`: Internal mount point for ZDB filesystem (`/mnt`)

### Configuration Types (from gridtypes/zos/qsfs.go)

#### `QuantumSafeFS` Struct
Main workload configuration:

```go
type QuantumSafeFS struct {
    Cache  gridtypes.Unit      `json:"cache"`
    Config QuantumSafeFSConfig `json:"config"`
}
```

**Fields:**
- `Cache`: Local cache size allocation
- `Config`: Detailed quantum-safe filesystem configuration

**Methods:**
- `Valid(getter gridtypes.WorkloadGetter) error`: Validates configuration parameters
- `Challenge(w io.Writer) error`: Generates cryptographic challenge for workload verification
- `Capacity() (gridtypes.Capacity, error)`: Returns resource capacity requirements

#### `QuantumSafeFSConfig` Struct
Detailed configuration for the quantum-safe filesystem:

```go
type QuantumSafeFSConfig struct {
    MinimalShards     uint32             `json:"minimal_shards" toml:"minimal_shards"`
    ExpectedShards    uint32             `json:"expected_shards" toml:"expected_shards"`
    RedundantGroups   uint32             `json:"redundant_groups" toml:"redundant_groups"`
    RedundantNodes    uint32             `json:"redundant_nodes" toml:"redundant_nodes"`
    MaxZDBDataDirSize uint32             `json:"max_zdb_data_dir_size" toml:"max_zdb_data_dir_size"`
    Encryption        Encryption         `json:"encryption" toml:"encryption"`
    Meta              QuantumSafeMeta    `json:"meta" toml:"meta"`
    Groups            []ZdbGroup         `json:"groups" toml:"groups"`
    Compression       QuantumCompression `json:"compression" toml:"compression"`
}
```

**Fields:**
- `MinimalShards`: Minimum number of data shards required for reconstruction
- `ExpectedShards`: Expected total number of data shards
- `RedundantGroups`: Number of redundant storage groups for fault tolerance
- `RedundantNodes`: Number of redundant nodes per group
- `MaxZDBDataDirSize`: Maximum size for ZDB data directory
- `Encryption`: Quantum-safe encryption configuration
- `Meta`: Metadata storage configuration
- `Groups`: ZDB backend group configurations
- `Compression`: Compression algorithm settings


### Metrics and Monitoring

#### `QSFSMetrics` Struct
Network consumption metrics for QSFS workloads:

```go
type QSFSMetrics struct {
    Consumption map[string]NetMetric
}
```

**Fields:**
- `Consumption`: Map of workload ID to network metrics

**Methods:**
- `Nu(wlID string) uint64`: Calculates total network usage for a workload

#### `QSFSInfo` Struct
Information about a mounted QSFS volume:

```go
type QSFSInfo struct {
    Path            string
    MetricsEndpoint string
}
```

**Fields:**
- `Path`: Local filesystem path where the volume is mounted
- `MetricsEndpoint`: HTTP endpoint for Prometheus metrics

## Helper Functions

### Core Functions

#### `New(ctx context.Context, cl zbus.Client, root string) (pkg.QSFSD, error)`
Creates a new QSFS service instance:
- Initializes mount and tombstone directories
- Starts periodic cleanup goroutine
- Migrates any existing tombstone files

#### `setQSFSDefaults(cfg *zos.QuantumSafeFS) zstorConfig`
Applies default configuration values for zstor daemon:
- Sets socket path to `/var/run/zstor.sock`
- Configures metrics port to 9100
- Sets ZDB mount point to `/mnt`
- Sets ZDB data directory to `/data/data/zdbfs-data`

### Mount Management

#### `mountPath(wlID string) string`
Generates the local mount path for a workload: `{mountsPath}/{wlID}`

#### `prepareMountPath(path string) error`
Prepares a directory for mounting:
- Creates the directory if it doesn't exist
- Sets up bind mount for container visibility
- Configures mount propagation as shared

#### `writeQSFSConfig(root string, cfg zstorConfig) error`
Writes zstor configuration to TOML file at `{root}/data/zstor.toml`

#### `waitUntilMounted(ctx context.Context, path string) error`
Polls the filesystem until the QSFS volume is successfully mounted

#### `isMounted(path string) (bool, error)`
Checks if a path is currently a mount point

### Cleanup and Lifecycle

#### `markDelete(ctx context.Context, wlID string) error`
Creates a tombstone file to mark a workload for deletion

#### `isMarkedForDeletion(ctx context.Context, wlID string) (bool, error)`
Checks if a workload is marked for deletion by looking for tombstone file

#### `tombstone(wlID string) string`
Returns the path to a workload's tombstone file

### Metrics Functions

#### `Metrics() (pkg.QSFSMetrics, error)`
Collects network metrics for all active QSFS volumes:
- Scans the mounts directory for active volumes
- Retrieves network statistics from each volume's network namespace
- Returns aggregated metrics including RX/TX bytes and packets

#### `qsfsMetrics(ctx context.Context, wlID string) (pkg.NetMetric, error)`
Retrieves network metrics for a specific QSFS workload:
- Enters the workload's network namespace
- Collects statistics from public and ygg0 network interfaces
- Returns combined network usage metrics

#### `metricsForNics(nics []string) (pkg.NetMetric, error)`
Aggregates network statistics from specified network interfaces:
- Iterates through the provided network interface names
- Sums up RX/TX bytes and packets from all interfaces
- Returns consolidated network metrics


## Network Environment and Preparation

### QSFSPrepare Environment

The `QSFSPrepare` function (in the networker package) creates an isolated network environment for QSFS containers:

1. **Network Namespace Creation**: Creates a dedicated network namespace named `qfs-ns-{id}`
2. **NDMZ Bridge Attachment**: Connects to the node's NDMZ from user network
3. **Yggdrasil Integration**: Attaches to the Yggdrasil overlay network for peer-to-peer communication
4. **Firewall Configuration**: Applies restrictive firewall rules allowing only:
   - Established/related connections
   - Prometheus metrics access on port 9100 via Yggdrasil
   - Local loopback traffic
   - ICMPv6 traffic

### Network Isolation Benefits

- **Security**: Complete network isolation prevents unauthorized access
- **Resource Control**: Dedicated namespace allows precise traffic monitoring
- **Global Connectivity**: Yggdrasil provides secure global connectivity
- **Firewall Protection**: Minimal attack surface with restrictive firewall rules

## Periodic Cleanup System

### Cleanup Process

The QSFS package implements a sophisticated cleanup system to handle failed or abandoned QSFS instances:

#### Default Timer Configuration
- **Check Period**: 10 minutes (`checkPeriod = 10 * time.Minute`)
- **Upload Threshold**: 100 MB (`UploadLimit = 100 * 1024 * 1024`)
- **Metrics Failure Threshold**: 10 consecutive failures

#### Cleanup Algorithm

The `periodicCleanup` function runs continuously and:

1. **Scans Tombstones**: Checks the tombstones directory for workloads marked for deletion
2. **Metrics Collection**: Attempts to collect network metrics for each marked workload
3. **Activity Detection**: Monitors network upload activity to determine if the QSFS is active
4. **Cleanup Decision**: Removes inactive or failed QSFS instances based on:
   - Upload activity below threshold (< 100 MB in 10 minutes)
   - Consecutive metrics collection failures (≥ 10 failures)

#### Cleanup State Management

The cleanup system maintains state for each workload:

```go
type failedQSFSState struct {
    lastUploadMap       map[string]uint64  // Tracks last upload bytes
    metricsFailureCount map[string]uint    // Counts consecutive failures
}
```

#### Cleanup Actions

When a QSFS is determined to be dead or inactive:
1. **Container Deletion**: Removes the QSFS container
2. **Mount Cleanup**: Unmounts the filesystem (attempts twice for safety)
3. **Directory Removal**: Removes mount point directories
4. **Network Cleanup**: Destroys the network namespace and releases resources
5. **FList Cleanup**: Unmounts the associated flist
6. **Tombstone Removal**: Removes the deletion marker

### Benefits of Periodic Cleanup

- **Resource Recovery**: Automatically reclaims resources from failed deployments
- **System Stability**: Prevents accumulation of zombie processes and mounts
- **Network Hygiene**: Cleans up abandoned network namespaces
- **Storage Management**: Releases allocated storage and cache space
- **Monitoring Health**: Provides insights into QSFS deployment success rates

## Constants and Configuration

### FList Configuration
- **QSFS FList URL**: `https://hub.grid.tf/tf-autobuilder/qsfs-0.2.0-rc2.flist`
- **Container Namespace**: `qsfs`
- **Root FS Propagation**: `RootFSPropagationSlave`

### Network Configuration
- **Zstor Socket**: `/var/run/zstor.sock`
- **ZDB FS Mount Point**: `/mnt`
- **Metrics Port**: `9100`
- **ZDB Data Directory**: `/data/data/zdbfs-data`

### Directory Structure
- **Mounts Directory**: `{root}/mounts`
- **Tombstones Directory**: `{root}/tombstones`

### Cleanup Constants
- **Upload Limit**: `100 * 1024 * 1024` (100 MB) - Minimum upload activity required in cleanup period
- **Check Period**: `10 * time.Minute` - Interval between cleanup checks
- **Metrics Failure Threshold**: 10 consecutive metrics collection failures before cleanup

## Integration with ZOS Ecosystem

The QSFS package integrates seamlessly with other ZOS modules:

- **Container Module**: Runs QSFS as containerized workloads
- **Network Module**: Provides isolated network environments
- **FList Module**: Manages QSFS runtime images
- **Storage Module**: Coordinates with local storage systems
- **Provisioning Module**: Handles workload lifecycle management

This comprehensive implementation ensures that QSFS provides enterprise-grade, quantum-safe distributed storage for the ThreeFold Grid ecosystem.