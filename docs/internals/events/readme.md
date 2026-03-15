# Events Package

## Introduction

The events package bridges TFChain (substrate) on-chain events to local node modules via Redis Streams. It subscribes to new blocks on the chain, decodes relevant events, filters them for this node/farm, and publishes them to Redis streams that other modules can consume.

## Architecture

```
TFChain (substrate)
    |
    |  WebSocket subscription (new block heads)
    v
Processor (events.go)
    |  Decodes EventRecords per block
    v
RedisStream (redis.go)
    |  Filters events for this node/farm
    |  Publishes to Redis streams (GOB-encoded)
    v
Redis Streams
    |
    +-- stream:public-config        → gateway, network modules
    +-- stream:contract-cancelled   → provision engine (noded)
    +-- stream:contract-lock        → provision engine (noded)
    +-- stream:power-target         → power module
    |
    v
RedisConsumer (redis.go)
    |  XREADGROUP with consumer groups
    v
Module-specific event channels
```

## Components

### Processor (`events.go`)

The core block-processing engine. Subscribes to new block headers via substrate WebSocket, then for each new block:

1. Gets the last processed block number from `State`
2. Iterates from `last + 1` to the new block number
3. Queries storage changes for `System.Events` at each block hash
4. Decodes the raw event records into `substrate.EventRecords`
5. Calls the registered `Callback` with the decoded events
6. Persists the new block number to `State`

If the subscription drops (connection lost, substrate manager updated), it waits 10 seconds and reconnects. Blocks that are too old and no longer in the archive (RPC error code -32000) are silently skipped.

### State (`events.go`)

Tracks the last processed block number to avoid reprocessing on restart.

```go
type State interface {
    Set(num types.BlockNumber) error
    Get(cl *gsrpc.SubstrateAPI) (types.BlockNumber, error)
}
```

`FileState` persists the block number as a 4-byte big-endian uint32 to a file. On first run (no file), it starts from the latest block on chain.

### RedisStream (`redis.go`) — Producer

Wraps `Processor` and publishes filtered events to Redis streams.

The `process` callback filters the following on-chain events:

| On-chain Event | Filter | Redis Stream | Event Type |
|----------------|--------|-------------|------------|
| `NodePublicConfigStored` | `event.Node == this node` | `stream:public-config` | `PublicConfigEvent` |
| `NodeContractCanceled` | `event.Node == this node` | `stream:contract-cancelled` | `ContractCancelledEvent` |
| `ContractGracePeriodStarted` | `event.NodeID == this node` | `stream:contract-lock` | `ContractLockedEvent` (Lock=true) |
| `ContractGracePeriodEnded` | `event.NodeID == this node` | `stream:contract-lock` | `ContractLockedEvent` (Lock=false) |
| `PowerTargetChanged` | `event.Farm == this farm` | `stream:power-target` | `PowerTargetChangeEvent` |

Events are GOB-encoded and pushed via `XADD` with `MAXLEN ~ 1024` (approximate trimming to keep the stream bounded).

The substrate manager can be hot-swapped via `UpdateSubstrateManager()` when the chain connection needs to be re-established with new URLs.

### RedisConsumer (`redis.go`) — Consumer

Provides typed Go channels for each event stream. Each consumer uses Redis consumer groups (`XREADGROUP`) for reliable delivery with acknowledgement.

```go
func (r *RedisConsumer) PublicConfig(ctx context.Context) (<-chan pkg.PublicConfigEvent, error)
func (r *RedisConsumer) ContractCancelled(ctx context.Context) (<-chan pkg.ContractCancelledEvent, error)
func (r *RedisConsumer) ContractLocked(ctx context.Context) (<-chan pkg.ContractLockedEvent, error)
func (r *RedisConsumer) PowerTargetChange(ctx context.Context) (<-chan pkg.PowerTargetChangeEvent, error)
```

Each consumer:
1. Creates a consumer group for the stream (idempotent, ignores `BUSYGROUP` error)
2. First reads any pending (unacknowledged) messages from ID `0`
3. Then blocks waiting for new messages from ID `>`
4. Decodes GOB payload and sends on the typed channel
5. Acknowledges each message after processing

The consumer ID must be unique per module to ensure independent delivery.

## Event Types

Defined in `pkg/events.go`:

```go
type PublicConfigEvent struct {
    PublicConfig substrate.OptionPublicConfig
}

type ContractCancelledEvent struct {
    Contract uint64
    TwinId   uint32
}

type ContractLockedEvent struct {
    Contract uint64
    TwinId   uint32
    Lock     bool    // true = grace period started, false = ended
}

type PowerTargetChangeEvent struct {
    FarmID pkg.FarmID
    NodeID uint32
    Target substrate.Power
}
```

## Consumers

| Module | Stream | Purpose |
|--------|--------|---------|
| Gateway / Network | `stream:public-config` | Reconfigure gateway when farmer updates public config |
| Provision engine (noded) | `stream:contract-cancelled` | Deprovision workloads when contract is cancelled |
| Provision engine (noded) | `stream:contract-lock` | Pause/resume workloads during grace period |
| Power module | `stream:power-target` | Handle power on/off commands from the farmer |
