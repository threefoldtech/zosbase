# Provisiond

This module is responsible for provision/decommission workload on the node. It accepts new deployment over `rmb` and applies them based on the workload type.

## How it works

- Connects to `Redis` message broker over `zbus`.
- Runs integrity checks if the flag was passed and return an error if checks did not pass
- Provisiond is disabled for orphan nodes
- Creates local reservation store `BoltStorage`
- Migrates deprecated workloads to the new bolt storage
- Clean up deleted contracts that has no active workloads anymore
- Collects information about workload statistics
- Sets the capacity for active contracts
- Creates a new engine and runs it, now the engine will continue processing all reservations and try to apply them.
- Ensures workloads are recreated after reboot
- Listens for blockchain contract events and calls the engine to apply them.
- Regularly reports used capacity back to the blockchain

## Usage

```sh
provisiond --root /var/cache/modules/provisiond \
>          --broker unix:///var/run/redis.sock 
```

```sh
provisiond --integrity
```


### Command-line flags

| Flag          | Description                               | Default                           |
| ----------    | ----------------------------------------  | --------------------------------  |
| `--root`      | `ROOT` working directory of the module    | `/var/cache/modules/provisiond`   |
| `--broker`    | Connection string to the message BROKER   | `unix:///var/run/redis.sock`      |
| `--integrity` | Runs some integrity checks on some files  | false                             |