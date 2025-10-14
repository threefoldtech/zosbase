# Storaged

The storage module is responsible for handling and managing the creation of disks and volumes.

Storaged provides the entrypoint of the storage module, exposing it as a service over the message bus (zbus) so that other modules can create and manage disks and volumes.

## How it works

- Connects to `Redis` message broker over `zbus`
- Registers itself as the `storage` module to expose its methods through zbus

## Usage

```sh
storaged --broker unix:///var/run/redis.sock \
       --workers 2
```

### Command-line flags

| Flag        | Description                                | Default                      |
| ----------- | -------------------------------------------| ---------------------------- |
| `--broker`  | Connection string to the message BROKER    | `unix:///var/run/redis.sock` |
| `--workers` | Number of workers N                        | `1`                          |
