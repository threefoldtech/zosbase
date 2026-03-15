# Network Module

## ZBus

Network module is available on zbus over the following channel

| module | object | version |
|--------|--------|---------|
| network | [network](#interface) | 0.0.1 |

## Home Directory

| directory | path |
|-----------|------|
| root | `/var/cache/modules/networkd` |
| networks | `{root}/networks/` — persisted NR configs |
| link | `{root}/networks/link/` — workload-to-network symlinks |
| ndmz leases | `{root}/ndmz-lease/` — IPv4 IPAM allocations |
| mycelium keys | `{root}/mycelium-key/` — per-NR mycelium key files |

## Related Documentation

- [Definitions of vocabulary](definitions.md)
- [Introduction to networkd](introduction.md)
- [WireGuard mesh](mesh.md)
- [Farm network setup](setup_farm_network.md)
- [VLANs](vlans.md)
- [Yggdrasil](yggdrasil.md)

## Introduction

The network module (`networkd`) manages all networking infrastructure on the node. It creates and manages bridges, network namespaces, WireGuard tunnels, overlay networks (Yggdrasil and Mycelium), and provides network connectivity for all workloads (VMs, containers, ZDB, QSFS).

On boot, the node first establishes internet connectivity by finding a NIC with a working DHCP lease and creating the `zos` management bridge. Once connected, `networkd` starts and sets up the DMZ namespace, overlay networks, and registers itself on zbus to accept workload networking requests from the [provision engine](../provision/readme.md).

### zinit unit

```yaml
exec: networkd --broker unix:///var/run/redis.sock
after:
  - boot
```

## Architecture

### Network Layers

```
                    ┌─────────────────────────────────────────────┐
                    │              Physical NIC(s)                │
                    └──────┬──────────────────┬───────────────────┘
                           │                  │
                    ┌──────▼──────┐    ┌──────▼──────┐
                    │  zos bridge │    │  br-pub     │  (exit NIC or veth to zos)
                    │  (mgmt/dhcp)│    │  (public)   │
                    └──────┬──────┘    └──────┬──────┘
                           │                  │
           ┌───────────────┼──────────────────┼────────────────┐
           │               │                  │                │
    ┌──────▼──────┐ ┌──────▼──────┐ ┌─────────▼────────┐ ┌────▼─────┐
    │  ndmz       │ │  public     │ │  zdb-ns-*        │ │  br-ygg  │
    │  namespace  │ │  namespace  │ │  ZDB namespaces  │ │  br-my   │
    │  (NAT gw)   │ │  (pub IP)   │ │  (pub+ygg+myc)   │ │ (overlay)│
    └──────┬──────┘ └─────────────┘ └──────────────────┘ └────┬─────┘
           │                                                   │
    ┌──────▼──────┐                                           │
    │  br-ndmz    │                                           │
    │  (routing)  │                                           │
    └──────┬──────┘                                           │
           │                                                   │
    ┌──────▼───────────────────────────────────────────────────▼──┐
    │  n-<netID> namespaces  (per-user network resources)        │
    │  ├── macvlan on b-<netID> bridge (internal NR traffic)     │
    │  ├── macvlan "public" on br-ndmz (internet via NAT)        │
    │  ├── WireGuard w-<netID> (encrypted mesh to other nodes)   │
    │  └── mycelium on m-<netID> bridge (optional overlay)       │
    └──────┬─────────────────────────────────────────────────────┘
           │
    ┌──────▼──────┐
    │  VM TAPs    │  t-<name> on b-<netID> (private)
    │             │  p-<name> on br-pub    (public IP)
    │             │  t-<name> on br-ygg    (yggdrasil)
    │             │  t-<name> on m-<netID> (mycelium)
    └─────────────┘
```

### Bridges

| Bridge | Name constant | Purpose |
|--------|--------------|---------|
| `zos` | DefaultBridge | Management bridge; gets DHCP from router; default internet uplink |
| `br-pub` | PublicBridge | Public traffic; wired to exit NIC (dual-NIC) or zos via veth (single-NIC) |
| `br-ndmz` | NdmzBridge | Routing bridge connecting NR namespaces to the ndmz for NAT'd internet |
| `br-ygg` | YggBridge | Yggdrasil overlay; shared by all ygg interfaces (`200::/7`) |
| `br-my` | MyceliumBridge | Mycelium overlay; shared by all mycelium interfaces (`400::/7`) |
| `b-<netID>` | — | Per-NR bridge; connects VMs/containers within that private network |
| `m-<netID>` | — | Per-NR mycelium bridge; connects VMs' mycelium tap devices in that NR |

### Namespaces

| Namespace | Purpose |
|-----------|---------|
| `ndmz` | Node's DMZ — NAT gateway for all NR namespaces. Has `npub4` (IPv4 via zos) and `npub6` (IPv6 via br-pub) |
| `public` | Optional — only when farmer sets a public config. Hosts the node's public IP, yggdrasil daemon, iperf |
| `n-<netID>` | Per-user private network resource with WireGuard mesh, NAT via ndmz, optional mycelium |
| `zdb-ns-<hash>` | Per-ZDB container; wired to br-pub + ygg + mycelium |
| `qfs-ns-<hash>` | Per-QSFS mount; wired to ndmz + ygg + mycelium |

## NDMZ (DMZ Namespace)

The `ndmz` namespace is the node's NAT gateway providing internet access to all NR namespaces.

**Setup:**
1. Creates `br-ndmz` bridge on the host
2. Creates macvlan `tonrs` inside ndmz on `br-ndmz` with IP `100.127.0.1/16`
3. Creates macvlan `npub6` inside ndmz on `br-pub` for IPv6 (SLAAC)
4. Creates macvlan `npub4` inside ndmz on `zos` for IPv4 (DHCP)
5. Applies nftables: masquerade on `npub4`/`npub6`, exempt yggdrasil traffic
6. Starts DHCP monitoring goroutine for IPv4 lease renewal

**NR attachment:** Each NR namespace gets a macvlan `public` on `br-ndmz` with an IP allocated from `100.127.0.2–100.127.255.254/16` (IPAM via CNI host-local backend). Default gateway points to `100.127.0.1`.

## Public Namespace

The `public` namespace is optional and created only when the farmer configures a public IP for the node.

**Setup:**
1. Creates `br-pub` bridge and detects the exit NIC (physical NIC with public IPv6, or falls back to `zos`)
2. Creates `public` namespace with macvlan on `br-pub`
3. Assigns static IPv4/IPv6 from the public config
4. Starts yggdrasil daemon inside the namespace
5. Starts iperf service for network testing

**Single-NIC vs dual-NIC:** When no dedicated public NIC is found, `br-pub` is connected to `zos` via a veth pair (single-NIC mode). When a separate NIC with public IPv6 is detected, it's attached directly to `br-pub` (dual-NIC mode).

## Network Resources (NR)

Each user private network gets a dedicated namespace `n-<netID>` with:

- **Internal bridge** (`b-<netID>`): VMs connect here via TAP devices
- **NAT interface** (`public` macvlan on `br-ndmz`): Internet via ndmz
- **WireGuard** (`w-<netID>`): Encrypted mesh to the same network on other nodes. IP in `100.64.0.0/16` CGNAT space
- **Mycelium** (optional, `m-<netID>` bridge): Per-NR mycelium instance with its own key, running as a zinit service

**IP addressing within NR:**
- NR gateway: `<subnet>.1` (e.g., `10.1.2.1/24`)
- IPv6 (ULA): derived from MD5 of netID + IPv4 (`fd::/8` range)
- WireGuard: `100.64.<a>.<b>/16` derived from subnet

## Overlay Networks

### Yggdrasil

Address space: `200::/7`. Ports: TCP 9943, TLS 9944, link-local 9945.

Runs as a zinit service inside the `public` namespace (or `ndmz` if no public namespace). The node's yggdrasil identity is derived from its Ed25519 private key.

Each ZDB/QSFS/VM gets a deterministic yggdrasil IP by hashing its identifier through the node's `/64` subnet.

### Mycelium

Address space: `400::/7`. Port: TCP 9651.

Runs as a zinit service at the host level. Uses X25519 key derived from the node's Ed25519 key. Each NR can optionally run its own mycelium instance with a separate key.

Each ZDB/QSFS/VM gets a deterministic mycelium IP by hashing its identifier through the node's `/64` subnet.

## ZDB Networking

ZDB containers get a dedicated namespace (`zdb-ns-<hash>`) with:
- `eth0`: veth pair to `br-pub` (public IPv6 via SLAAC)
- `ygg0`: veth pair to `br-ygg` (yggdrasil access)
- `my0`: veth pair to `br-my` (mycelium access)

## QSFS Networking

QSFS mounts get a dedicated namespace (`qfs-ns-<hash>`) with:
- `public`: macvlan on `br-ndmz` (internet via NAT, IPv4 from IPAM)
- `ygg0`: veth pair to `br-ygg`
- `my0`: veth pair to `br-my`
- Strict nftables: drops all inbound except established, loopback, prometheus on ygg, and ICMPv6

## VM TAP Devices

VMs connect to networks via TAP devices created by the networker:

| Method | TAP name | Bridge | Purpose |
|--------|----------|--------|---------|
| `SetupPrivTap` | `t-<name>` | `b-<netID>` | Private NR network |
| `SetupPubTap` | `p-<name>` | `br-pub` | Public IPv4/IPv6 |
| `SetupYggTap` | `t-<name>` | `br-ygg` | Yggdrasil overlay |
| `SetupMyceliumTap` | `t-<name>` | `m-<netID>` | Per-NR mycelium |

## Public IP Filtering

When a VM has a public IP, the networker sets up nftables bridge filter rules on `br-pub` to enforce anti-spoofing: only traffic from the assigned MAC+IP pair is allowed through the VM's tap device.

## Subpackages

| Package | Purpose |
|---------|---------|
| `bootstrap/` | Node boot: detect NICs, create `zos` bridge, find internet connectivity |
| `bridge/` | Bridge create/delete/attach with VLAN filtering support |
| `dhcp/` | DHCP probing via `udhcpc` and `dhcpcd` zinit service management |
| `ifaceutil/` | Veth pair creation, MAC/IPv6 derivation from input bytes |
| `iperf/` | iperf3 service in public namespace for network testing |
| `macvlan/` | Macvlan device creation, IP/route installation |
| `macvtap/` | Macvtap device creation for VMs |
| `mycelium/` | Mycelium daemon management, config generation, namespace plumbing |
| `namespace/` | Linux network namespace create/delete/list, address monitoring |
| `ndmz/` | DMZ namespace setup, DHCP monitoring, IPAM, nftables |
| `nr/` | Network resource lifecycle: bridge, namespace, WireGuard, mycelium, firewall |
| `options/` | sysctl wrappers: IPv6 forwarding, accept_ra, proxy_arp |
| `portm/` | WireGuard port allocator with filesystem-backed persistence |
| `public/` | Public namespace setup, public config persistence, exit NIC detection |
| `tuntap/` | TAP device creation for VM network interfaces |
| `types/` | Bridge/namespace name constants, interface info types |
| `yggdrasil/` | Yggdrasil daemon management, config generation, namespace plumbing |

## Interface

```go
type Networker interface {
    CreateNR(wl gridtypes.WorkloadWithID, network Network) (string, error)
    DeleteNR(wl gridtypes.WorkloadWithID) error

    SetPublicConfig(cfg PublicConfig) error
    GetPublicConfig() (PublicConfig, error)
    UnsetPublicConfig()

    EnsureZDBPrepare(id string) (string, error)
    ZDBDestroy(ns string) error

    QSFSPrepare(id string) (string, string, error)
    QSFSDestroy(id string) error

    SetupPrivTap(networkID NetID, name string) (string, error)
    SetupPubTap(name string) (string, error)
    SetupYggTap(name string) (YggTapConfig, error)
    SetupMyceliumTap(name string, netID string, cfg MyceliumTapConfig) (string, error)
    RemoveTap(name string) error
    RemovePubTap(name string) error
    DisconnectPubTap(name string) error

    SetupPubIPFilter(filterName, tapName, mac string, ip4 net.IP, ip6 net.IP) error
    RemovePubIPFilter(filterName, tapName string) error
    PubIPFilterExists(filterName string) bool

    GetSubnet(networkID NetID) (net.IPNet, error)
    GetNet(networkID NetID) (net.IPNet, string, error)
    GetDefaultGwIP(networkID NetID) (net.IP, error)
    GetPublicIPv6Subnet() (net.IPNet, error)
    GetPublicIPV6Gateway() (net.IP, error)

    Interfaces(iface string, netns string) (Interfaces, error)
    Addrs(iface string, netns string) ([][]byte, string, error)
    ZOSAddresses(ctx context.Context) (<-chan NetlinkAddresses, error)
    DMZAddresses(ctx context.Context) (<-chan NetlinkAddresses, error)
    YggAddresses(ctx context.Context) (<-chan NetlinkAddresses, error)
    PublicAddresses(ctx context.Context) (<-chan OptionPublicConfig, error)
    WireguardPorts() ([]uint, error)

    Metrics() (NetResourceMetrics, error)
    Namespace(networkID NetID) (string, error)
    GetIPv6From4(networkID NetID, ip4 net.IP) (net.IPNet, error)
}
```
