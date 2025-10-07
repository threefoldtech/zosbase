# Powerd

Power module handles the node power events (node uptime reporting, wake on lan, etc...)

## How it works
- Loads the running environment of the node to get the `farmID`
- Connects to `Redis` message broker over `zbus` 
- Get node information like `nodeID`, `twinID`
- Loads the node’s identity key.
- Creates a substrate manager, substrate gateway to handle blockchain interactions.
- Reports **node uptime**
- Enables **Wake On Lan** feature if supported

## Usage

```sh
powerd --broker unix:///var/run/redis.sock
```

### Command-line flags

| Flag              | Description                                     | Default                           |
| -----------       | ------------------------------------------------| ----------------------------      |
| `--broker`        | Connection string to the message BROKER         | `unix:///var/run/redis.sock`      |
