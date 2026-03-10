# Provision Module

## ZBus

The provision module exposes the `Provision` interface over zbus:

| module | object | version |
|--------|--------|---------|
| provision | [provision](#interface) | 0.0.1 |

## Introduction

This module is responsible for provisioning and decommissioning workloads on the node. It accepts new deployments over RMB (Reliable Message Bus), validates them against the TFChain contract, and brings them to reality by dispatching to per-type workload managers via zbus.

`provisiond` knows about all available daemons and contacts them over zbus to ask for the needed services. It pulls everything together and updates the deployment with the workload state.

If the node is restarted, `provisiond` re-provisions all active workloads to restore them to their original state.

## Supported Workloads

0-OS supports 13 workload types (see the [primitives package](../primitives/readme.md) for details):

- `network` — WireGuard private network
- `network-light` — Mycelium-only network
- `zmachine` — virtual machine (full networking)
- `zmachine-light` — virtual machine (mycelium only)
- `zmount` — virtual disk for a zmachine
- `volume` — btrfs subvolume (shared directory for a zmachine)
- `public-ip` / `public-ipv4` — public IPv4/IPv6 for a zmachine
- [`zdb`](https://github.com/threefoldtech/0-DB) — 0-db namespace
- [`qsfs`](https://github.com/threefoldtech/quantum-storage) — quantum safe filesystem
- `zlogs` — log stream from a zmachine to an external endpoint
- `gateway-name-proxy` — reverse proxy with grid-assigned subdomain
- `gateway-fqdn-proxy` — reverse proxy with user-owned FQDN

## Architecture

```
  User
   |  (RMB: zos.deployment.deploy / update)
   v
 ZOS API (pkg/zos_api)
   |  (zbus)
   v
 NativeEngine (pkg/provision)
   |  1. Validate & persist to BoltDB
   |  2. Enqueue job to disk queue
   |
   v  (run loop, single-threaded)
 mapProvisioner (pkg/provision)
   |  (dispatches by workload type)
   v
 Manager (pkg/primitives/*)
   |  (zbus)
   v
 System daemons (storaged, networkd, vmd, ...)
```

## Engine

The `NativeEngine` is the central orchestrator. It implements both the `Engine` interface (for scheduling) and the `pkg.Provision` interface (exposed over zbus).

### Job Queue

All operations go through a durable disk-backed queue (`dque`). Enqueue returns immediately after persisting to storage — execution happens asynchronously in the run loop.

| Operation | Description |
|-----------|-------------|
| `opProvision` | Install a new deployment (validates against chain) |
| `opDeprovision` | Uninstall a deployment |
| `opUpdate` | Diff current vs new, apply add/remove/update ops |
| `opProvisionNoValidation` | Re-install on boot (skips chain hash validation) |
| `opPause` | Pause all workloads in a deployment |
| `opResume` | Resume all paused workloads |

The run loop is **single-threaded**: one job at a time, FIFO order. A job is only dequeued after it completes, so if the node crashes mid-job, it will be retried on the next boot.

### Deployment Lifecycle

```
1. RMB message arrives at ZOS API
2. CreateOrUpdate validates:
   - Structural validity (no duplicate names, valid versions)
   - Ownership (deployment.TwinID == sender twin)
   - KYC verification (via env.KycURL)
   - Signature (ed25519 using twin's on-chain public key)
3. Engine.Provision persists to BoltDB and enqueues opProvision
4. Run loop picks up job:
   a. Chain validation:
      - Fetches NodeContract from substrate
      - Verifies contract is for this node
      - Compares deployment ChallengeHash with contract DeploymentHash
      - Checks node rent status
   b. Installs workloads in type order via provisioner
   c. Dequeues job, fires callback
```

### Startup Order

Workloads are installed in a deterministic type order (networks before VMs, storage before VMs) and uninstalled in **reverse** order. Within the same type, ZMount and Volume workloads are sorted largest-first.

Pause uses reverse order, resume uses forward order.

### Boot Recovery

When `rerunAll` is enabled, the engine on start:

1. Scans all persisted deployments
2. Re-enqueues active ones as `opProvisionNoValidation` (skips chain hash check since the deployment is already validated)
3. The run loop processes them normally, restoring all workloads

### Upgrade / Update

When a deployment update arrives:

1. The engine computes a diff: which workloads are added, removed, or updated
2. Operations are sorted: removes first (reverse type order), then adds/updates (forward type order)
3. Each operation dispatches to the provisioner accordingly
4. Workload type changes are not allowed; only managers implementing the `Updater` interface accept updates

## Interfaces

### Engine

```go
type Engine interface {
    Provision(ctx context.Context, wl gridtypes.Deployment) error
    Deprovision(ctx context.Context, twin uint32, id uint64, reason string) error
    Pause(ctx context.Context, twin uint32, id uint64) error
    Resume(ctx context.Context, twin uint32, id uint64) error
    Update(ctx context.Context, update gridtypes.Deployment) error
    Storage() Storage
    Twins() Twins
    Admins() Twins
}
```

### Provisioner

Operates at the per-workload level. Returned by `primitives.NewPrimitivesProvisioner`.

```go
type Provisioner interface {
    Initialize(ctx context.Context) error
    Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (gridtypes.Result, error)
    Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error
    Pause(ctx context.Context, wl *gridtypes.WorkloadWithID) (gridtypes.Result, error)
    Resume(ctx context.Context, wl *gridtypes.WorkloadWithID) (gridtypes.Result, error)
    Update(ctx context.Context, wl *gridtypes.WorkloadWithID) (gridtypes.Result, error)
    CanUpdate(ctx context.Context, typ gridtypes.WorkloadType) bool
}
```

### Manager

The interface each workload type must implement (see [primitives](../primitives/readme.md)):

```go
type Manager interface {
    Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error)
    Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error
}
```

Optional extensions: `Initializer`, `Updater`, `Pauser`.

### Storage

Persists deployment state. Primary implementation uses BoltDB.

```go
type Storage interface {
    Create(deployment gridtypes.Deployment) error
    Update(twin uint32, deployment uint64, fields ...Field) error
    Delete(twin uint32, deployment uint64) error
    Get(twin uint32, deployment uint64) (gridtypes.Deployment, error)
    Error(twin uint32, deployment uint64, err error) error
    Add(twin uint32, deployment uint64, workload gridtypes.Workload) error
    Remove(twin uint32, deployment uint64, name gridtypes.Name) error
    Transaction(twin uint32, deployment uint64, workload gridtypes.Workload) error
    Changes(twin uint32, deployment uint64) (changes []gridtypes.Workload, err error)
    Current(twin uint32, deployment uint64, name gridtypes.Name) (gridtypes.Workload, error)
    Twins() ([]uint32, error)
    ByTwin(twin uint32) ([]uint64, error)
    Capacity(exclude ...Exclude) (StorageCapacity, error)
}
```

## Storage Backend

### BoltDB (`provision/storage/`)

The primary storage backend uses BoltDB with an append-only transaction log:

```
<twin_id>                    (bucket)
  └── "global"               (bucket) — sharable workload name → deployment ID
  └── <deployment_id>        (bucket)
        ├── "version"
        ├── "metadata"
        ├── "description"
        ├── "signature_requirement"
        ├── "workloads"      (bucket) — name → type (active workload index)
        └── "transactions"   (bucket) — sequence → JSON(workload + result)
```

Every state change appends a new entry to the `transactions` bucket. `Current()` scans backward to find the latest state for each workload. `Remove()` deletes from the active `workloads` index but historical entries remain.

### Filesystem (`provision/storage.fs/`)

Legacy filesystem-based storage. Each deployment is a versioned JSON file. Used for migration to BoltDB.

## Context Enrichment

The engine injects values into the context before calling provisioner methods:

| Accessor | Value | Description |
|----------|-------|-------------|
| `GetEngine(ctx)` | `Engine` | Access to the engine (and storage) |
| `GetDeploymentID(ctx)` | `(twin, deployment)` | Current deployment IDs |
| `GetDeployment(ctx)` | `Deployment` | Fresh deployment from storage |
| `GetWorkload(ctx, name)` | `Workload` | Last state of a workload in the deployment |
| `GetContract(ctx)` | `NodeContract` | TFChain contract for this deployment |
| `IsRentedNode(ctx)` | `bool` | Whether the node has an active rent contract |

## Error Handling

| Error | Meaning |
|-------|---------|
| `ErrNoActionNeeded` | Workload already running correctly; skip writing a transaction |
| `ErrDeploymentExists` | Storage conflict: deployment already exists |
| `ErrDeploymentNotExists` | Deployment not found |
| `ErrWorkloadNotExist` | Workload not found |
| `ErrDeploymentUpgradeValidationError` | Upgrade diff failed validation |
| `ErrInvalidVersion` | Version number is wrong |

Workload failures are expressed via `gridtypes.Result` with `State = StateError`, not as Go errors. Special response types allow managers to communicate specific states:

- `Ok()` — explicit success (normally returning `nil` error suffices)
- `Paused()` — workload is paused
- `UnChanged(err)` — update failed but workload still running with previous config

## Authentication

### Twins (`auth.go`)

- **`substrateTwins`**: Fetches twin ed25519 public keys from TFChain via `substrateGateway.GetTwin()`. Caches up to 1024 entries in an LRU cache.
- **`substrateAdmins`**: Authorizes the farm owner twin only. Used for admin-only operations.

### HTTP Middleware (`mw/`)

JWT-based authentication for the HTTP API layer:
- Validates JWT signed with ed25519 (audience `"zos"`, max 2-minute expiry)
- Injects twin ID and public key into the request context

## Interface

The zbus-exposed interface:

```go
type Provision interface {
    DecommissionCached(id string, reason string) error
    GetWorkloadStatus(id string) (gridtypes.ResultState, bool, error)
    CreateOrUpdate(twin uint32, deployment gridtypes.Deployment, update bool) error
    Get(twin uint32, contractID uint64) (gridtypes.Deployment, error)
    List(twin uint32) ([]gridtypes.Deployment, error)
    Changes(twin uint32, contractID uint64) ([]gridtypes.Workload, error)
    ListTwins() ([]uint32, error)
    ListPublicIPs() ([]string, error)
    ListPrivateIPs(twin uint32, network gridtypes.Name) ([]string, error)
}
```
