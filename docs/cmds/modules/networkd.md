# Networkd
Network module handles network resources and user networks.

After the internet process sets up a bridge (zos) which will have an IP address that has Internet access, the rest of the system can be started, where ultimately, the networkd daemon is started.

The networkd daemon receives tasks from the provisioning daemon, so that it can create the necessary resources in the User Network.

## How it works

- Creates `root` directory
- Waits for `yggdrasil` binary to be available
- Checks that the `zos` bridge is created and of the correct type
- Connects to `Redis` message broker over `zbus`
- Ensures the system has the correct existing host `nft` rules
- Creates the public setup, to make sure bridges are created and wired correctly, and initialize public name space. If no exit nic is set the node tries to detect the exit br-pub nic based on the following criteria
     - physical nic 
     - wired and has a signal 
     - can get public slaac IPv6
if no nic is found zos is selected.
- Creates `ndmz` namespace which represents the router between private user networks and public internet
- Sets up and starts `yggdrasil`
- Sets up and starts `mycelium`
- Creates a `networker` module that can be used over `zbus` and registers itself as the `network` module

## Usage

```sh
networkd --root /var/cache/modules/networkd \
>       --broker unix:///var/run/redis.sock
```

### Command-line flags

| Flag              | Description                                     | Default                           |
| -----------       | ------------------------------------------------| ----------------------------      |
| `--root`          | `ROOT` working directory of the module          | `/var/cache/modules/networkd`     |
| `--broker`        | Connection string to the message BROKER         | `unix:///var/run/redis.sock`      |
