# Contd

Container module is reponsible for handling starting, stoppping and inspecting of containers.

Contd provides the entrypoint of the contd module, exposing it as a service over the message bus (zbus) so that other modules can handle the containers.

## How it works

- Ensure that `shim-logs` binary is available, backoff until it's found.
- Creates module root dirctory.
- Connects to `Redis` message broker over `zbus`.
- Register itself as a `containerd` module.
- Starts watching for events coming from `containerd`.


## Usage

```sh
contd --root /var/cache/modules/contd \
>    --broker unix:///var/run/redis.sock \
>    --congainerd /run/containerd/containerd.sock \
>    --workers 2
```

### Command-line flags

| Flag              | Description                                     | Default                           |
| -----------       | ------------------------------------------------| ----------------------------      |
| `--root`          | `ROOT` working directory of the module          | `/var/cache/modules/contd`        |
| `--broker`        | Connection string to the message BROKER         | `unix:///var/run/redis.sock`      |
| `--congainerd`    | connection string to containerd `CONTAINERD`    | `/run/containerd/containerd.sock` |
| `--workers`       | Number of workers N                             | `1`                               |
