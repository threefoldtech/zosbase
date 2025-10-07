# Vmd

The vmd module, manages all virtual machines processes, it provide the interface to, create, inspect, and delete virtual machines. It also monitor the vms to make sure they are re-spawned if crashed. 

## How it works

- Connects to `Redis` message broker over `zbus`.
- Creates a 50MB volatile storage to store configs, so they are gone if the machine is rebooted.
    - **If vmd is starting because of an update:** vms must make sure current running vms should stay running
    - **If it's a fresh start:** config files should be ignored
- We assume this is an **update** if there are running VMs, in that case we need to move the files from the deprecated (persisted) moduleRoot directory to the volatile directory.
- The old config directory is removed
- The module registers itself as the `vmd` module to expose its methods through `zbus`

## Usage

```sh
vmd   --root /var/cache/modules/vmd \
>     --broker unix:///var/run/redis.sock \
>     --workers 2
```


### Command-line flags

| Flag        | Description                               | Default                           |
| ----------  | ----------------------------------------  | --------------------------------  |
| `--root`    | `ROOT` working directory of the module    | `/var/cache/modules/provisiond`   |
| `--broker`  | Connection string to the message BROKER   | `unix:///var/run/redis.sock`      |
| `--workers` | number of workers `N`                     | 1                                 |