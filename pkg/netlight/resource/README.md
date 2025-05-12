# Network Resource Package

## Overview

This package implements network resource management for ZOS Network Light, providing isolated network namespaces for workloads with mycelium and Wireguard connectivity.

## Network Resource

A network resource consists of:

- Network namespace (`n{name}`)
- Private network bridge (`r{name}`)
- Mycelium bridge (`m{name}`)
- Interfaces (public, private, mycelium, wireguard)
- NFT rules for proper routing and security

## Creation

`Create()` sets up a network resource by:

1. Creating bridges for private network and mycelium
2. Creating a network namespace
3. Setting up veth pairs to connect namespace to bridges
4. Configuring IP addresses and routing
5. Applying NFT rules

## Wireguard Integration

To create network resource with wireguard user needs should be

- Providing the subnet for the network resource (e.g., 10.1.3.0/24)
- Defining the overall IP range for the network (e.g., 10.1.0.0/16)
- Generating and providing the Wireguard private key
- Selecting an available port for Wireguard to listen on
- Configuring the list of peers with their public keys and allowed IPs

### Implementation

Wireguard interfaces are added to a network resource through:

1. `WGName()`: Generates the Wireguard interface name (`w-{name}`)
2. `SetWireguard()`: Creates Wireguard interface in the host namespace and moves it into the network namespace
3. `ConfigureWG()`: Sets up the Wireguard interface with:
   - The user-provided private key
   - The user-selected listen port
   - Peer configurations (public keys, allowed IPs, endpoints)
4. `HasWireguard()`: Checks if the Wireguard interface exists in the namespace

The Wireguard interface is created in the host namespace and then moved into the network resource namespace. Once configured with the user-provided private key, listen port, and peer information, it enables secure communication between network resources across different nodes by establishing encrypted tunnels to other network resources on different nodes, creating a secure mesh network.

## Cleanup

`Delete()`

- Destroys mycelium service
- Removes network namespace
- Deletes all created bridges

The cleanup process continues even if some steps fail, collecting all errors for proper reporting.
