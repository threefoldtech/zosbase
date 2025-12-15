# API Gateway Module

## ZBus

API Gateway module is available on ZBus over the following channel:
| module | object | version |
|--------|--------|---------|
| api-gateway|[api-gateway](#interface)| 0.0.1|

## Introduction

API Gateway module acts as the entrypoint for incoming and outgoing requests. Modules trying to reach Threefold Chain should do that through API Gateway, that way we can ensure no extrinsic chain requests are done at the same time causing some of them to be ignored. Incoming RMB requests should also go through API Gateway and be passed to each module through internal communication (ZBus). Having all routes defined on one place rather than being scattered around in every module highly improves readability and also the ability to traverse different API implementation. Developer of each module that requires an external API needs to define the entrypoint (i.e. `zos.deployment.deploy`) and pass user input after validation to the module internal API.

## zinit unit

`api-gateway` module needs to start after `identityd` has started as it needs the node identity for managing chain requests.

```yaml
exec: api-gateway --broker unix:///var/run/redis.sock
after:
  - boot
  - identityd
```

## Interface

```go
type SubstrateGateway interface {
    UpdateSubstrateGatewayConnection(ctx context.Context, manager substrate.Manager) (err error)
    CreateNode(ctx context.Context, node substrate.Node) (uint32, error)
    CreateTwin(ctx context.Context, relay string, pk []byte) (uint32, error)
    EnsureAccount(ctx context.Context, activationURL []string, termsAndConditionsLink string, termsAndConditionsHash string) (info substrate.AccountInfo, err error)
    GetContract(ctx context.Context, id uint64) (substrate.Contract, SubstrateError)
    GetContractIDByNameRegistration(ctx context.Context, name string) (uint64, SubstrateError)
    GetFarm(ctx context.Context, id uint32) (substrate.Farm, error)
    GetNode(ctx context.Context, id uint32) (substrate.Node, error)
    GetNodeByTwinID(ctx context.Context, twin uint32) (uint32, SubstrateError)
    GetNodeContracts(ctx context.Context, node uint32) ([]types.U64, error)
    GetNodeRentContract(ctx context.Context, node uint32) (uint64, SubstrateError)
    GetNodes(ctx context.Context, farmID uint32) ([]uint32, error)
    GetPowerTarget(ctx context.Context, nodeID uint32) (power substrate.NodePower, err error)
    GetTwin(ctx context.Context, id uint32) (substrate.Twin, error)
    GetTwinByPubKey(ctx context.Context, pk []byte) (uint32, SubstrateError)
    Report(ctx context.Context, consumptions []substrate.NruConsumption) (types.Hash, error)
    SetContractConsumption(ctx context.Context, resources ...substrate.ContractResources) error
    SetNodePowerState(ctx context.Context, up bool) (hash types.Hash, err error)
    UpdateNode(ctx context.Context, node substrate.Node) (uint32, error)
    UpdateNodeUptimeV2(ctx context.Context, uptime uint64, timestampHint uint64) (hash types.Hash, err error)
    GetTime(ctx context.Context) (time.Time, error)
    GetZosVersion(ctx context.Context) (string, error)
}
```

## Distributed Tracing

API Gateway implements distributed tracing to track requests across ZOS modules. Each request is assigned a unique trace ID that flows through the entire system.

### How It Works

1. **Context Propagation**: All interface methods accept `context.Context` as the first parameter
2. **Trace ID Generation**: A unique trace ID (format: `trace-{uuid}`) is generated or extracted from the context
3. **Automatic Logging**: All operations log the trace ID, enabling request correlation across modules

### Log Output

All logs include the `trace_id` field:

```json
{"level":"debug","trace_id":"trace-abc123","method":"CreateNode","twin_id":1234,"message":"method called"}
{"level":"debug","trace_id":"trace-abc123","message":"CreateNode failed, retrying"}
{"level":"debug","trace_id":"trace-abc123","message":"CreateNode completed successfully"}
```

### Searching Logs

To trace a complete request journey:

```bash
# Find all logs for a specific trace ID
zinit log | grep "trace-abc123"

# Or use journalctl
journalctl -u api-gateway | grep "trace-abc123"
```

### Benefits

- **Request Tracking**: Follow a provision request from arrival through flist mounting, disk preparation, and completion
- **Cross-Module Correlation**: Same trace ID flows through api-gateway → provision → flist → storage modules
- **Debugging**: Quickly identify which requests are causing issues
- **Performance Analysis**: Track request duration across the entire system
