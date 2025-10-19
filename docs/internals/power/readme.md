# Power Module

## ZBus

Power module is available on zbus over the following channel

| module | object            | version         |
|--------|-------------------|---------------- |
| power  |[power](#interface)| 0.0.1           |

## Introduction

Power module handles the node power events (node uptime reporting, wake on lan, etc...)

The power daemon should start after:

- **boot** : to ensures the base system is initialized
- **noded** : to ensure the node daemon is running and the node is registered on grid, since powerd depends on node state synchronization with the chain


```yaml
exec: powerd --broker unix://var/run/redis.sock
after:
  - boot
  - noded
```

### Power Server

The PowerServer is responsible for managing and synchronizing the power state of a node with its target state defined on the blockchain. It continuously listens to power events , updates its internal state, and performs power actions.

```go
type PowerServer struct { 
    consumer         *events.RedisConsumer
    substrateGateway *stubs.SubstrateGatewayStub

    // enabled means the node can power off!
    enabled bool
    farm    pkg.FarmID
    node    uint32
    twin    uint32
    ut      *Uptime
}


// This is the main entrypoint for the PowerServer runtime
func (p *PowerServer) Start(ctx context.Context) error


// event handles a single PowerTargetChangeEvent received from Redis
func (p *PowerServer) event(event *pkg.PowerTargetChangeEvent) error


// events manages continuous processing of power-related events
// It automatically retries the event stream loop if it fails, unless the context is canceled
func (p *PowerServer) events(ctx context.Context) error


// powerUp sends a wake-on-LAN (WOL) signal to a target node using its MAC address
func (p *PowerServer) powerUp(node *substrate.Node, reason string) error

// recv listens for PowerTargetChange events from Redis and forwards each event to the event handler
// It exits only when the context is canceled
func (p *PowerServer) recv(ctx context.Context) error


// Sets the node power state as provided or to up if power manegement is not enabled on this node
// It makes sure to compare the state with on chain state to not do un-necessary transactions
func (p *PowerServer) setNodePowerState(up bool) error


// shutdown powers down the node when the target is down on the blockchain
// It sends an uptime record first before shutting down
// Only shutdown if power management is enabled
func (p *PowerServer) shutdown() error


// Syncs the actual node state with the target state
// If target is up, and the node state is up, we do nothing
// If target is up, but the node is down, we set the state to up and return
// If target is down, we make sure state is down, then shutdown
func (p *PowerServer) syncSelf() error
```


