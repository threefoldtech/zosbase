# Network Light Module

## ZBus

Network light module is available on zbus over the following channel

| module | object | version |
|--------|--------|---------|
| network | [network](#interface) | 0.0.1 |

## Home Directory

| directory | path |
|-----------|------|
| root | `/var/cache/modules/networkd` (volatile) |
| networks | `{root}/networks/` — persisted NR configs |
| link | `{root}/networks/link/` — workload-to-network symlinks |
| ndmz leases | `{root}/ndmz-lease/` — IPv4 IPAM allocations |
| mycelium seeds | `/tmp/network/mycelium/` — per-NR mycelium seed files |

## Introduction

Network light is a simplified version of the [network module](../network/readme.md) designed for cloud-hosted ZOS nodes. It uses Mycelium as the primary overlay network and drops features that require complex infrastructure (Yggdrasil, public IPs, dual-NIC detection with IPv6 SLAAC).

### Key differences from the full network module

| Feature | Network Light | Full Network |
|---------|--------------|--------------|
| Primary overlay | Mycelium (`400::/7`) | Yggdrasil (`200::/7`) + Mycelium |
| Yggdrasil | Not available | Full support with `br-ygg` bridge |
| Public IPs for VMs | Not supported | Supported via `br-pub` + nftables |
| NDMZ | Simple bridge at `100.127.0.1/16` | Full namespace with dual-stack (IPv4 DHCP + IPv6 SLAAC) |
| ZDB connectivity | Mycelium only (via `br-hmy`) | Public IPv6 + Yggdrasil + Mycelium |
| WireGuard | Optional per-NR | Always present per-NR |
| NIC setup | Single NIC assumed | Auto-detects single/dual NIC |

### What VMs get

- A **local private IP** (e.g., `10.x.x.x`) for internet via NAT through the node's IP
- A **mycelium IP** (`400::/7`) for end-to-end encrypted communication. The user controls the subnet via their seed
- Optional **WireGuard** access for direct private network connectivity

## Architecture

### Host Namespace

```
                        ┌──────────┐
                        │   NIC    │
                        └────┬─────┘
                             │ attached
                             ▼
  ┌──────────────────────────────────────────────────────────────────┐
  │                          Host                                    │
  │                                                                  │
  │  ┌──────────┐  ┌─────┐  ┌──────────┐          ┌──────────┐      │
  │  │  br-hmy  │  │ my0 │  │   zos    │- - - - - │ br-ndmz  │      │
  │  │  (ZDB)   │  │(tun)│  │  (DHCP)  │  NAT'd   │100.127.0.1      │
  │  └──────────┘  └─────┘  └──────────┘          └──────────┘      │
  │       │           │                                              │
  │       │      mycelium-host                                       │
  │       │      zinit service                                       │
  └──────────────────────────────────────────────────────────────────┘
```

- The physical NIC is attached to the `zos` bridge
- DHCP client runs on `zos` to get the node's IP
- `br-ndmz` bridge is created with static IP `100.127.0.1/16`
- `my0` tunnel and `br-hmy` bridge run directly on the host (moved from ndmz namespace in the original design)
- `mycelium-host` zinit service creates the `my0` tunnel for host-level mycelium
- `br-hmy` connects ZDB namespaces to the host mycelium
- nftables rule drops inbound connections to `100.127.0.1`

### Bridges

| Bridge | Purpose |
|--------|---------|
| `zos` | Management bridge; DHCP from physical network |
| `br-pub` | Public bridge; exit NIC or veth to zos |
| `br-ndmz` | NDMZ routing bridge at `100.127.0.1/16`; all NR namespaces connect here |
| `br-hmy` | Host mycelium bridge; ZDB namespaces connect here for mycelium IPs |
| `r<name>` | Per-NR private bridge; VMs attach TAPs here |
| `m<name>` | Per-NR mycelium bridge; VMs attach mycelium TAPs here |

### Namespaces

| Namespace | Purpose |
|-----------|---------|
| `public` | Optional — only when farmer sets a public config |
| `n<name>` | Per-user network resource with private bridge, mycelium, optional WireGuard |
| `n<hash>` | Per-ZDB container; wired to `br-hmy` for mycelium access |

Unlike the full network module, network light has **no ndmz namespace**. The `br-ndmz` bridge lives directly on the host at `100.127.0.1/16` and provides NAT'd internet to all NR namespaces through the node's IP. Mycelium runs inside each NR namespace individually (not in a shared daemon).

## Network Resources (NR)

```
                         ┌──────┐
                         │  VM  │
                         └──┬┬──┘
                            ││
              ┌─────────────┘└──────────┐
              │ tap (b-<hash>)     tap  │ (m-<hash>)
              ▼                         ▼
  r-<NR>                             m-<NR>       br-ndmz
    ○                                  ○             ○
    │ veth                             │ veth        │ veth
    │                                  │             │
  ┌─┼──────────────────────────────────┼─────────────┼──┐
  │ │              n-<NR>              │             │  │
  │ ▼                                  ▼             ▼  │
  │ ┌─────────┐  ┌──────────┐  ┌─────────┐             │
  │ │ private │  │ mycelium │  │ public  │             │
  │ │<sub>.1  │  │ (myc daemon  │100.127. │             │
  │ │  gw     │  │  runs here) │ x.x/16 │             │
  │ └─────────┘  └──────────┘  └─────────┘             │
  │       ▲                         │                   │
  │  VM private                  default route          │
  │  traffic                     via 100.127.0.1        │
  └─────────────────────────────────────────────────────┘
```

Each user network gets an isolated namespace `n<name>` with:

**Interfaces:**
- `private` veth → `r<name>` bridge (private VM traffic). Gateway at `<subnet>.1`
- `public` veth → `br-ndmz` (internet via NAT). IP from `100.127.0.2–100.127.255.254`
- `mycelium` veth → `m<name>` bridge (per-NR mycelium)
- `w-<name>` WireGuard interface (optional)

**Mycelium:**
Each NR runs its own mycelium daemon inside its namespace with the user's seed. This gives the user control over the IP range. VMs get deterministic IPs from the NR's `/64` subnet: `<prefix>::ff0f:<vm-seed>/64`.

**Private networking:**
- VMs in the same NR communicate directly over the `r<name>` bridge
- Different NRs are completely isolated — overlapping private IPs are fine
- Internet access is NAT'd through `public` → `br-ndmz` → node IP

**Firewall:** nftables masquerade on traffic from the `private` interface so VMs can reach the internet via NAT.

## ZDB Networking

```
     br-hmy
       ○
       │ veth
       │
  ┌────┼──────────────────┐
  │    │    n-<hash>       │
  │    ▼                   │
  │ ┌──────┐               │
  │ │ eth0 │               │
  │ └──────┘               │
  │    │                   │
  │  mycelium IP from      │
  │  DMZ /64 subnet        │
  │  route 400::/7 via gw  │
  └────────────────────────┘
```

ZDB containers get a dedicated namespace with mycelium-only connectivity:

1. Namespace `n<hash>` is created
2. A veth pair connects the namespace to `br-hmy` (host mycelium bridge)
3. A mycelium IP from the DMZ's `/64` subnet is assigned
4. Route `400::/7` via the mycelium gateway

ZDBs are accessible exclusively over mycelium.

## Private Networks (WireGuard)

WireGuard is optional in network light. To access VMs on a node using WireGuard:

1. Deploy a network with valid WireGuard peers:

```go
WGPrivateKey: wgKey,
WGListenPort: 3011,
Peers: []zos.Peer{
    {
        Subnet:      gridtypes.MustParseIPNet("10.1.2.0/24"),
        WGPublicKey: "4KTvZS2KPWYfMr+GbiUUly0ANVg8jBC7xP9Bl79Z8zM=",
        AllowedIPs: []gridtypes.IPNet{
            gridtypes.MustParseIPNet("10.1.2.0/24"),
            gridtypes.MustParseIPNet("100.64.1.2/32"),
        },
    },
},
```

2. Configure WireGuard on your local machine:

```conf
[Interface]
Address = 100.64.1.2/32
PrivateKey = <your private key>

[Peer]
PublicKey = cYvKjMRBLj3o3e4lxWOK6bbSyHWtgLNHkEBxIv7Olm4=
AllowedIPs = 10.1.1.0/24, 100.64.1.1/32
PersistentKeepalive = 25
Endpoint = 192.168.123.32:3011
```

3. Bring the interface up: `wg-quick up <config file>`
4. Test: `ping 10.1.1.2`

WireGuard IPs use CGNAT space: `100.64.<a>.<b>/16` derived from the NR subnet.

### Full Picture

```
                           ┌──────┐
                           │  VM  │
                           └─┬──┬─┘
                             │  │
            tap (b-<hash>)   │  │   tap (m-<hash>)
                             │  │
  ┌──────────────────────────┼──┼──────────────────────────────────────┐
  │                  Host    │  │                                      │
  │                          │  │                                      │
  │  ┌─────────┐  r-<NR>    │  │  m-<NR>  ┌──────────┐  ┌──────────┐ │
  │  │my0 (tun)│    ○───────┘  └────○     │ br-ndmz  │  │  br-hmy  │ │
  │  │mycelium │    │                │     │100.127.  │  └────┬─────┘ │
  │  │ -host   │    │                │     │  0.1/16  │       │       │
  │  └─────────┘    │                │     └────┬─────┘       │       │
  │            ┌────┘          ┌─────┘          │             │       │
  │   ┌────┐   │  veth         │  veth     veth │        veth │       │
  │   │NIC │   │               │                │             │       │
  │   └─┬──┘   │               │                │             │       │
  │     │att.  │               │                │             │       │
  │     ▼      │               │                │             │       │
  │  ┌──────┐  │               │                │             │       │
  │  │ zos  │  │               │                │             │       │
  │  │(DHCP)│  │               │                │             │       │
  │  └──────┘  │               │                │             │       │
  └────────────┼───────────────┼────────────────┼─────────────┼───────┘
               │               │                │             │
  ┌────────────┼───────────────┼────────────────┼──┐    ┌─────┼─────┐
  │            ▼    n-<NR>     ▼                ▼  │    │ n-<hash>  │
  │  ┌─────────┐  ┌──────────┐  ┌─────────┐       │    │     ▼     │
  │  │ private │  │ mycelium │  │ public  │       │    │  ┌──────┐ │
  │  │<sub>.1  │  │ (myc     │  │100.127. │       │    │  │ eth0 │ │
  │  │  gw     │  │  daemon) │  │ x.x/16 │       │    │  └──────┘ │
  │  └─────────┘  └──────────┘  └─────────┘       │    │  myc IP   │
  │                                  │             │    └───────────┘
  │                        default route           │
  │                        via 100.127.0.1         │
  └────────────────────────────────────────────────┘
```

## VM TAP Devices

VMs connect via TAP devices created by the networker:

| TAP name | Bridge | Purpose |
|----------|--------|---------|
| `b-<hash>` | `r<name>` | Private NR network (IPv4) |
| `m-<hash>` | `m<name>` | Per-NR mycelium (IPv6 in `400::/7`) |

TAP names are derived from MD5 hash of the workload ID, Base58-encoded, truncated to 13 characters.

## Subpackages

| Package | Purpose |
|---------|---------|
| `bootstrap/` | Node boot: detect NICs, create `zos` bridge, DHCP probing |
| `bridge/` | Bridge create/delete/attach with VLAN filtering |
| `dhcp/` | DHCP probing via `udhcpc` and `dhcpcd` zinit service |
| `ifaceutil/` | Veth pairs, deterministic MAC/device-name from input bytes |
| `ipam/` | IPv4 IPAM from `100.127.0.3–100.127.255.254` via CNI host-local |
| `iperf/` | iperf3 service in public namespace |
| `macvlan/` | Macvlan device creation and IP/route installation |
| `macvtap/` | Macvtap device creation |
| `namespace/` | Network namespace create/delete/list/monitor |
| `options/` | sysctl wrappers: IPv6 forwarding, accept_ra, proxy_arp |
| `public/` | Public namespace setup, config persistence, exit NIC detection |
| `resource/` | NR lifecycle: bridges, namespace, mycelium daemon, WireGuard, nftables |
| `tuntap/` | TAP device creation for VM network interfaces |
| `types/` | Bridge/namespace name constants |

## Interface

```go
type NetworkerLight interface {
    Create(name string, wl gridtypes.WorkloadWithID, network Network) (string, error)
    Delete(name string) error

    AttachPrivate(name, id string, vmIp net.IP) (TapDevice, error)
    AttachMycelium(name, id string, seed []byte) (TapDevice, error)
    Detach(id string) error

    AttachZDB(id string) (string, error)
    ZDBIPs(nsName string) ([]net.IP, error)

    GetSubnet(networkID NetID) (net.IPNet, error)
    GetNet(networkID NetID) (net.IPNet, string, error)
    GetDefaultGwIP(networkID NetID) (net.IP, error)

    Interfaces(iface string, netns string) (Interfaces, error)
    ZOSAddresses(ctx context.Context) (<-chan NetlinkAddresses, error)
    WireguardPorts() ([]uint, error)
    Namespace(id string) (string, error)

    SetPublicConfig(cfg PublicConfig) error
    LoadPublicConfig() (*PublicConfig, error)
    UnSetPublicConfig() error

    Ready() error
}
```
