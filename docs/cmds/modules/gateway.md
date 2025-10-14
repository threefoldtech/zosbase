# Gateway

The gateway modules is used to register traefik routes and services to act as a reverse proxy, forwarding traffic to backend workloads.

cmds/gateway provides the entrypoint of the gateway module, exposing it as a service over the message bus (zbus) so that other modules can use it.

## How it works

- Connects to `Redis` message broker over `zbus`.
- Register itself as a `gateway` module.

## Usage

```sh
gateway --root /var/cache/modules/gateway \
>       --broker unix:///var/run/redis.sock \
>       --workers 2
```

### Command-line flags

| Flag              | Description                                     | Default                           |
| -----------       | ------------------------------------------------| ----------------------------      |
| `--root`          | `ROOT` working directory of the module          | `/var/cache/modules/gateway`        |
| `--broker`        | Connection string to the message BROKER         | `unix:///var/run/redis.sock`      |
| `--workers`       | Number of workers N                             | `1`                               |
