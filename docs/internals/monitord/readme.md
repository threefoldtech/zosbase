# Monitord Package

## Introduction

monitord package is responsible for continuously monitoring a node’s system resources.
It provides a monitor stream for components such as CPU, memory, disk, network cards, and system uptime.


## Interface
### System Monitor

```go
// systemMonitor implements the pkg.SystemMonitor interface
type systemMonitor struct {
    duration time.Duration // duration between statistics collection
    node     uint32        // node ID
    cl       zbus.Client   // client used to request node data
}

type SystemMonitor interface {

// CPU starts cpu monitor stream
// It periodically emits CPU usage percentage and time statistics
CPU(ctx context.Context) <-chan pkg.TimesStat

// Disks starts disk monitor stream
// It periodically emits disk stats for all mounted disks, including number of read/write operations, IO time, serial number
Disks(ctx context.Context) <-chan pkg.DisksIOCountersStat

// Get the types of workloads can be deployed depending on the network manager running on the node
GetNodeFeatures() []pkg.NodeFeature

// Memory starts memory monitor stream
// It periodically emits VirtualMemoryStat containing memory usage statistics like Total, Available and Used in bytes
Memory(ctx context.Context) <-chan pkg.VirtualMemoryStat

// Nics starts Nic monitor stream
// It periodically emits Nic stats for all available network cards, including number of packets sent, received or dropped
Nics(ctx context.Context) <-chan pkg.NicsIOCounterStat

// Returns node ID
NodeID() uint32
}
```

### Host Monitor

```go
// hostMonitor provides host-level monitoring such as system uptime.
type hostMonitor struct {
	duration time.Duration
}

type HostMonitor interface {
    // Uptime periodically reads /proc/uptime file, parses it and emits the uptime
	Uptime(ctx context.Context) <-chan time.Duration
}
```
