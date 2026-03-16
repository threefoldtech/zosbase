# External Services

ZOS communicates with several external services for blockchain operations, messaging, package management, and node registration. All services have per-environment endpoints (dev, test, QA, prod) and most support redundant URLs with automatic failover.

Endpoint configuration is defined in [environment.go](../../pkg/environment/environment.go) and can be overridden at runtime via the [zos-config](https://github.com/threefoldtech/zos-config) repository (cached for 6 hours) or kernel boot parameters.

## TFChain (Substrate Blockchain)

| Environment | Endpoints                                                                                                                                   |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| prod        | `wss://tfchain.grid.tf/`, `wss://tfchain.02.grid.tf`, `wss://02.tfchain.grid.tf/`, `wss://03.tfchain.grid.tf/`, `wss://04.tfchain.grid.tf/` |
| test        | `wss://tfchain.test.grid.tf/`, `wss://tfchain.02.test.grid.tf`                                                                              |
| qa          | `wss://tfchain.qa.grid.tf/`, `wss://tfchain.02.qa.grid.tf/`                                                                                 |
| dev         | `wss://tfchain.dev.grid.tf/`, `wss://tfchain.02.dev.grid.tf`                                                                                |

- **Protocol**: WebSocket Secure (WSS)
- **Purpose**: Node registration, twin management, capacity reporting, contract management
- **Client**: `github.com/threefoldtech/tfchain/clients/tfchain-client-go`
- **Retry**: Exponential backoff (`cenkalti/backoff`), 500ms initial interval, 2s max interval, 5s max elapsed time. Applied to all substrate operations.
- **Override**: kernel param `substrate=` or env var `ZOS_SUBSTRATE_URL`

## RMB Relay (Reliable Message Bus)

| Environment | Endpoint                   |
| ----------- | -------------------------- |
| prod        | `wss://relay.grid.tf`      |
| test        | `wss://relay.test.grid.tf` |
| qa          | `wss://relay.qa.grid.tf`   |
| dev         | `wss://relay.dev.grid.tf`  |

- **Protocol**: WebSocket Secure (WSS)
- **Purpose**: P2P messaging between nodes, request routing from users to nodes
- **Client**: `github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go`
- **Retry**: Handled by the external RMB SDK library (WebSocket reconnection)
- **Override**: kernel param `relay=`
- **Note**: Relay URLs are stored on-chain with limited space — max 4 relays per environment

## Hub (Package & Flist Repository)

| Service             | V3 URL                          | V4 URL                             |
| ------------------- | ------------------------------- | ---------------------------------- |
| HTTP API            | `https://hub.threefold.me`      | `https://v4.hub.threefold.me`      |
| Redis (flist index) | `redis://hub.threefold.me:9900` | `redis://v4.hub.threefold.me:9940` |
| ZDB (storage)       | `zdb://hub.threefold.me:9900`   | `zdb://v4.hub.threefold.me:9940`   |

- **Purpose**: Downloading OS/service packages (flists), container base images, system binaries
- **API endpoints**: `/api/flist/{repo}`, `/api/flist/{repo}/{name}/light`, `/api/flist/{repo}/tags/{tag}`
- **Binary repos**: `tf-zos-v3-bins` (prod), `tf-zos-v3-bins.test`, `tf-zos-v3-bins.qanet`, `tf-zos-v3-bins.dev`
- **Retry**: `go-retryablehttp`, 5 retries with exponential backoff, 20s HTTP timeout
- **Override**: env var `ZOS_FLIST_URL`, `ZOS_BIN_REPO`

## GraphQL Gateway

| Environment | Endpoints                                                                                                            |
| ----------- | -------------------------------------------------------------------------------------------------------------------- |
| prod        | `https://graphql.grid.threefold.me/graphql`, `https://graphql.grid.tf/graphql`, `https://graphql.02.grid.tf/graphql` |
| test        | `https://graphql.test.grid.tf/graphql`, `https://graphql.02.test.grid.tf/graphql`                                    |
| qa          | `https://graphql.qa.grid.tf/graphql`, `https://graphql.02.qa.grid.tf/graphql`                                        |
| dev         | `https://graphql.dev.grid.tf/graphql`, `https://graphql.02.dev.grid.tf/graphql`                                      |

- **Protocol**: HTTPS
- **Purpose**: Grid metadata queries, node information, contract queries
- **Retry**: No per-request retry. Sequential URL fallback — tries each endpoint in order until one succeeds

## Activation Service

| Environment | Endpoints                                                                                                                                                         |
| ----------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| prod        | `https://activation.grid.threefold.me/activation/activate`, `https://activation.grid.tf/activation/activate`, `https://activation.02.grid.tf/activation/activate` |
| test        | `https://activation.test.grid.tf/activation/activate`, `https://activation.02.test.grid.tf/activation/activate`                                                   |
| qa          | `https://activation.qa.grid.tf/activation/activate`, `https://activation.02.qa.grid.tf/activation/activate`                                                       |
| dev         | `https://activation.dev.grid.tf/activation/activate`, `https://activation.02.dev.grid.tf/activation/activate`                                                     |

- **Protocol**: HTTPS
- **Purpose**: Twin account creation and activation
- **Retry**: Exponential backoff (`cenkalti/backoff`), 500ms initial, 2s max interval, 5s max elapsed time. Tries multiple URLs — moves to next URL only on activation service errors
- **Override**: kernel param `activation=`

## Registrar

| Environment | Endpoint                               |
| ----------- | -------------------------------------- |
| prod        | `https://registrar.prod4.threefold.me` |
| qa          | `https://registrar.qa4.grid.tf`        |
| test        | `http://registrar.test4.grid.tf`       |
| dev         | `http://registrar.dev4.grid.tf`        |

- **Purpose**: Node registration, identity management
- **Retry**: Exponential backoff (`cenkalti/backoff`), 2min max interval, indefinite retry (`MaxElapsedTime=0`) — retries forever until registration succeeds
- **Terms of Service**: `http://zos.tf/terms/v0.1`

## KYC (Know Your Customer)

| Environment | Endpoint                   |
| ----------- | -------------------------- |
| prod        | `https://kyc.threefold.me` |
| test        | `https://kyc.test.grid.tf` |
| qa          | `https://kyc.qa.grid.tf`   |
| dev         | `https://kyc.dev.grid.tf`  |

- **Purpose**: Identity verification, KYC compliance checks (twin verification at `/api/v1/status`)
- **Retry**: `go-retryablehttp`, 5 retries with exponential backoff, 10s HTTP timeout

## GeoIP

Shared across all environments:

- `https://geoip.threefold.me/`
- `https://geoip.grid.tf/`
- `https://02.geoip.grid.tf/`
- `https://03.geoip.grid.tf/`

- **Purpose**: Node geographic location detection (longitude, latitude, country, city)
- **Retry**: `go-retryablehttp`, 5 retries with exponential backoff, 10s HTTP timeout. Also falls back to next URL in the list
- **Source**: [geoip.go](../../pkg/geoip/geoip.go)

## ZOS Config (Runtime Configuration)

- **Base URL**: `https://raw.githubusercontent.com/threefoldtech/zos-config/main/`
- **Files**: `dev.json`, `test.json`, `qa.json`, `prod.json`
- **Purpose**: Runtime override of all service endpoints, peer lists (Yggdrasil, Mycelium), authorized users, admin twins, rollout upgrade farms
- **Retry**: `go-retryablehttp`, 5 retries with exponential backoff, 10s HTTP timeout. Falls back to expired cache if all retries fail
- **Cache**: 6 hours
- **Source**: [config.go](../../pkg/environment/config.go)

## Overlay Networks

### Yggdrasil

- **Listen ports**: TCP 9943, TLS 9944, LinkLocal 9945
- **Interface**: `ygg0`, MTU 65535
- **Peer list**: sourced from zos-config `yggdrasil.peers`
- **Purpose**: IPv6 mesh networking (`200::/7`), only in the full network module (not network-light)

### Mycelium

- **Peer list**: sourced from zos-config `mycelium.peers`
- **Purpose**: End-to-end encrypted mesh networking (`400::/7`)
- **Used by**: both full network and network-light modules

## Local Services

### Redis

- **Default**: `unix:///var/run/redis.sock` or `redis://localhost:6379`
- **Purpose**: IPC message bus (zbus), event queuing, stats aggregation

### Node HTTP API

- **Endpoint**: `http://[{node_ipv6}]:2021/api/v1/`
- **Purpose**: Node management API accessible over Yggdrasil/Mycelium mesh
- **Retry**: `go-retryablehttp` default client (1 retry)

## Summary

| Service    | Protocol        | Redundancy           | Retry                                         | Purpose                              |
| ---------- | --------------- | -------------------- | --------------------------------------------- | ------------------------------------ |
| TFChain    | WSS             | 2-5 endpoints        | Exponential backoff, 5s window                | Blockchain, contracts, node registry |
| RMB Relay  | WSS             | 1 endpoint           | External SDK (WebSocket reconnect)            | P2P messaging                        |
| Hub        | HTTPS/Redis/ZDB | 1 endpoint (V3 + V4) | 5 retries, 20s timeout                        | Package distribution                 |
| GraphQL    | HTTPS           | 2-3 endpoints        | Sequential URL fallback only                  | Grid metadata queries                |
| Activation | HTTPS           | 2-3 endpoints        | Exponential backoff, 5s window + URL fallback | Account activation                   |
| Registrar  | HTTP/HTTPS      | 1 endpoint           | Exponential backoff, indefinite               | Node registration                    |
| KYC        | HTTPS           | 1 endpoint           | 5 retries, 10s timeout                        | Identity verification                |
| GeoIP      | HTTPS           | 4 endpoints          | 5 retries + URL fallback                      | Location detection                   |
| ZOS Config | HTTPS           | 1 endpoint (GitHub)  | 5 retries + 6hr cache fallback                | Runtime configuration                |
| Yggdrasil  | TCP/TLS         | Peer mesh            | Peer reconnection                             | IPv6 overlay network                 |
| Mycelium   | TCP             | Peer mesh            | Peer reconnection                             | Encrypted overlay network            |
