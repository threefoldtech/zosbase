# Qsfsd

The `QSFS` (Quantum Safe File System) package provides a secure and distributed file system implementation for the ZOS ecosystem while maintaining the standard filesystem interface.

`qsfsd` manages the `qsfs`package

## How it works

- Connects to `Redis` message broker over `zbus`
- Registers itself as the `qsfsd` module to expose its methods through `zbus`

## Usage

```sh
qsfsd  --root /var/cache/modules/qsfsd  \
       --broker unix:///var/run/redis.sock \
       --workers 2
```

### Command-line flags

| Flag        | Description                                | Default                      |
| ----------- | -------------------------------------------| ---------------------------- |
| `--broker`  | Connection string to the message BROKER    | `unix:///var/run/redis.sock` |
| `--root`    | ROOT working directory of the module       | `/var/cache/modules/qsfsd`  |
| `--workers` | Number of workers N                        | `1`                          |
