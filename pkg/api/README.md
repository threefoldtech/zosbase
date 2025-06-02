# ZOS API Package

The ZOS API package provides a transport-agnostic interface for interacting with ZOS (Zero OS) nodes. This package is designed with a clear separation between API definitions and transport implementations, ensuring flexibility and maintainability.

## Overview

The ZOS API package serves as the core interface layer for ZOS node operations. It provides a comprehensive set of methods for:

- System information and diagnostics
- Network configuration and monitoring
- Deployment management (create, update, delete, query)
- Storage pool management
- Performance monitoring and benchmarking

## Transport Agnostic Design

The API package is designed to be **transport-layer agnostic**, meaning it doesn't depend on specific transport protocols (RPC, REST, WebSocket, etc.). This design principle ensures:

### Key Benefits

1. **Protocol Independence**: The core API logic is separated from transport concerns
2. **Clear Interfaces**: Well-defined parameter and return types that work with any transport
4. **Future Flexibility**: New transport protocols can be added without changing core API logic
5. **Maintainability**: Changes to transport don't affect API business logic

### Architecture

```
┌─────────────────────────────────────────┐
│           Transport Layer               │
│  (JSON-RPC, REST, gRPC, WebSocket...)   │
├─────────────────────────────────────────┤
│           API Package                   │
│  • Method definitions                   │
│  • Parameter/Return types               │
│  • Business logic                       │
│  • Mode handling (full/light)           │
├─────────────────────────────────────────┤
│           Service Layer                 │
│  • ZBus stubs                           │
│  • Resource oracle                      │
│  • Diagnostics manager                  │
└─────────────────────────────────────────┘
```


## Operating Modes

either passing `light` or anything else will considered full/normal zos mode

## Current Protocol Support

### JSON-RPC Implementation

The primary transport protocol currently supported is **JSON-RPC**, implemented in the `pkg/api/jsonrpc` directory.

#### Handler Registration

The JSON-RPC handlers are registered in `pkg/api/jsonrpc/handlers.go`:

```go
func RegisterHandlers(s *messenger.JSONRPCServer, r *RpcHandler) {
    // System endpoints
    s.RegisterHandler("system.version", r.handleSystemVersion)
    s.RegisterHandler("system.dmi", r.handleSystemDMI)
    
    // Network endpoints  
    s.RegisterHandler("network.interfaces", r.handleNetworkInterfaces)
    
    // Deployment endpoints
    s.RegisterHandler("deployment.deploy", r.handleDeploymentDeploy)
    
    // ... and many more
}
```

#### Integration Pattern

1. **RpcHandler**: Wraps the API instance and provides JSON-RPC specific handling
2. **Parameter Extraction**: Converts JSON-RPC parameters to Go types
3. **Method Delegation**: Calls the appropriate API method
4. **Response Formatting**: Returns results in JSON-RPC format

This pattern allows the core API to remain transport-agnostic while providing a clean JSON-RPC interface.

## API Documentation Using the OpenRPC Specification

The `openrpc.json` file can be used with various tools:

- **API Documentation Generators**: Generate human-readable docs
- **Client Code Generation**: Auto-generate client libraries
- **API Testing Tools**: Validate requests and responses
- **IDE Integration**: Provide autocomplete and validation

Example method definition from `openrpc.json`:
```json
{
  "name": "system.version",
  "params": [],
  "result": {
    "name": "Version",
    "schema": {
      "$ref": "#/components/schemas/Version"
    }
  }
}
```

## Usage Examples

### Basic API Initialization

```go
package main

import (
    "context"
    "log"

    "github.com/threefoldtech/zbus"
    "github.com/threefoldtech/zosbase/pkg/api"
)

func main() {
    // Initialize ZBus client
    client, err := zbus.NewClient("unix:///var/run/redis.sock")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Create API instance in full mode
    apiInstance, err := api.NewAPI(client, "tcp://localhost:6379", "")
    if err != nil {
        log.Fatal(err)
    }

    // Create API instance in light mode
    lightAPI, err := api.NewAPI(client, "tcp://localhost:6379", "light")
    if err != nil {
        log.Fatal(err)
    }

    // Create JSON-RPC server
    server := messenger.NewJSONRPCServer()

    // Create RPC handler
    rpcHandler := jsonrpc.NewRpcHandler(apiInstance)

    // Register all handlers
    jsonrpc.RegisterHandlers(server, rpcHandler)

    server.Start(ctx)
}
```

## Available API Commands

The following commands are available through the API. For complete parameter and return type specifications, refer to the `openrpc.json` file.

### System Commands

| Command | Description | Parameters | Returns |
|---------|-------------|------------|---------|
| `system.version` | Get ZOS and Zinit versions | None | Version object |
| `system.dmi` | Get DMI (Desktop Management Interface) information | None | DMI object |
| `system.hypervisor` | Detect hypervisor type | None | String |
| `system.diagnostics` | Get comprehensive system diagnostics | None | Diagnostics object |
| `system.features` | Get supported node features | None | NodeFeatures array |

### Network Commands

| Command | Description | Parameters | Returns |
|---------|-------------|------------|---------|
| `network.wg_ports` | List reserved WireGuard ports | None | Array of port numbers |
| `network.public_config` | Get public network configuration | None | PublicConfig object |
| `network.has_ipv6` | Check IPv6 support | None | Boolean |
| `network.public_ips` | List public IP addresses | None | Array of IP addresses |
| `network.private_ips` | List private IPs for network | `network_name` | Array of IP addresses |
| `network.interfaces` | List network interfaces | None | Map of interface names to IPs |
| `network.set_public_nic` | Set public network interface | `device` | Success/Error |
| `network.get_public_nic` | Get current public interface | None | Interface name |
| `network.admin.interfaces` | List admin interfaces | None | Interface information |

### Deployment Commands

| Command | Description | Parameters | Returns |
|---------|-------------|------------|---------|
| `deployment.deploy` | Deploy new workload | Deployment object | Success/Error |
| `deployment.update` | Update existing deployment | Deployment object | Success/Error |
| `deployment.get` | Get deployment by contract ID | `contract_id` | Deployment object |
| `deployment.list` | List all deployments | None | Deployments array |
| `deployment.changes` | Get deployment changes | `contract_id` | Workloads array |
| `deployment.delete` | Delete deployment | `contract_id` | Success/Error |

### Performance Monitoring Commands

| Command | Description | Parameters | Returns |
|---------|-------------|------------|---------|
| `monitor.speed` | Run network speed test | None | Speed test results |
| `monitor.health` | Run health check | None | Health check results |
| `monitor.publicip` | Test public IP connectivity | None | Public IP test results |
| `monitor.benchmark` | Run CPU benchmark | None | Benchmark results |
| `monitor.all` | Run all performance tests | None | Combined test results |

### Storage Commands

| Command | Description | Parameters | Returns |
|---------|-------------|------------|---------|
| `storage.pools` | Get storage pool metrics | None | Pool metrics |

### GPU Commands

| Command | Description | Parameters | Returns |
|---------|-------------|------------|---------|
| `gpu.list` | List available GPUs | None | GPU information array |

### Statistics Commands

| Command | Description | Parameters | Returns |
|---------|-------------|------------|---------|
| `statistics` | Get node resource statistics | None | Statistics object |

### Location Commands

| Command | Description | Parameters | Returns |
|---------|-------------|------------|---------|
| `location.get` | Get node location information | None | Location object |

### VM Log Commands

| Command | Description | Parameters | Returns |
|---------|-------------|------------|---------|
| `vm.logs` | Get VM log content | `file_name` | Log content string |
