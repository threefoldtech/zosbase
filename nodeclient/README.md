# NodeClient

NodeClient provides a simple interface to interact with ThreeFold nodes through JSON-RPC calls. It supports various operations including system information, network configuration, deployment management, and performance monitoring.

## Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/threefoldtech/tfgrid-sdk-go/messenger"
    "github.com/threefoldtech/zosbase/nodeclient"
)

func main() {
    // Create messenger instance
    msgr := messenger.NewMessenger(/* messenger config */)
    
    // Create node client
    client := nodeclient.NewNodeClient(msgr, "node-destination-id")
    
    ctx := context.Background()
    
    // Get node version
    version, err := client.GetNodeVersion(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Node version: %+v\n", version)
    
    // Get system diagnostics
    diag, err := client.GetSystemDiagnostics(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Diagnostics: %+v\n", diag)
    
    // List deployments
    deployments, err := client.DeploymentList(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Found %d deployments\n", len(deployments))
}
```
