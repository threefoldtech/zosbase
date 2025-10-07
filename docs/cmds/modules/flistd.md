# Flistd

The flist module is responsible for mounting of flists. The mounted directory contains all the files required by containers or VMs.

Flistd provides the entrypoint of the flist module, exposing it as a service over the message bus (zbus) so that other modules can mount and manage flists.

## How it works

- Connects to `Redis` message broker over `zbus`
- Registers itself as the `flist` module to expose its methods through zbus
- A background cleaner runs every **24h** to remove cached flists that are older than **90 days**

## Usage

```sh
flistd --root /var/cache/modules/flistd \
       --broker unix:///var/run/redis.sock \
       --workers 2
```

### Command-line flags

| Flag        | Description                                | Default                      |
| ----------- | -------------------------------------------| ---------------------------- |
| `--broker`  | Connection string to the message BROKER    | `unix:///var/run/redis.sock` |
| `--root`    | ROOT working directory of the module       | `/var/cache/modules/flistd`  |
| `--workers` | Number of workers N                        | `1`                          |
