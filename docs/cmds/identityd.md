# Identityd

`Identity manager` is responsible for maintaining the node identity (public key). The manager make sure the node has one valid ID during the entire lifetime of the node. It also provide service to sign, encrypt and decrypt data using the node identity.
On first boot, the identity manager will generate an ID and then persist this ID for life.

`Identityd` is also responsible for Node live software updates.

## How it works

- Connects to `Redis` message broker over `zbus`
- Prints its output according to the passed bool flags
- Do upgrade to latest version (this might means it needs to restart itself), that's why it's implemented as a standalone binary
- Register the node to `BCDB`
- Start `zbus` server to serve identity interface
- Start watcher for new version
- On update, re-register the node with new version to `BCDB`

## Usage

```sh
identityd   -root /var/cache/modules/identityd \
>           -broker unix:///var/run/redis.sock \
>           -interval 600
```

```sh
identityd   -v
```
```sh
identityd   -d
```
### Command-line flags

| Flag          | Description                                               | Default                         |
| ------------- | --------------------------------------------------------- | ------------------------------- |
| `-broker`     | Connection string to the message BROKER                   | `unix:///var/run/redis.sock`    |
| `-root`       | ROOT working directory of the module                      | `/var/cache/modules/identityd`  |
| `-interval`   | interval in seconds between update check                  | `600`                           |
| `-address`    | prints the node ss58 address and exits                    | `false`                         |
| `-farm`       | prints the node farm id and exits                         | `false`                         |
| `-d`          | (debug) when set, no self update is done before upgrading | `false`                         |
| `-id`         | [deprecated] prints the node ID and exits                 | `false`                         |
| `-net`        | prints the node network and exits                         | `false`                         |
| `-v`          | show version and exit                                     | `false`                         |