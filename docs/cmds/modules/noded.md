# Noded

This module is responsible of registering the node on the grid, and handling of grid events. It reports the node’s capacity (CPU, memory, storage, GPU, etc.), monitors performance and health, registers the node on the blockchain.

## How it works

- Connects to `Redis` message broker over `zbus`
- Prints **Node ID** or **Node network** if their flags were passed
- Collects node info:
  - Node capacity (CPU, memory, disk)
  - Whether the node is booted via `efi`
  - `DMI` Info (Desktop Management Interface)
  - Name of the hypervisor used on the node
  - List of available GPUs, logs GPUs info
- Starts a registrar service and publishes node info
- Registers the node on the blockchain
- Monitor node performance:
  - `NTP` check
  - `cpubench` (CPU benchmark)
  - `public ip` (validity of public ip)
- Streams events to the blockchain
- Keeps track of environment changes (substrate URLs)

## Usage

```sh
noded --broker unix:///var/run/redis.sock
```

```sh
noded --id
```

```sh
noded --net
```

### Command-line flags

| Flag       | Description                             | Default                      |
| ---------- | --------------------------------------- | ---------------------------- |
| `--broker` | Connection string to the message BROKER | `unix:///var/run/redis.sock` |
| `--id`     | print node id and exit                  | false                        |
| `--net`    | print node network and exit             | false                        |
