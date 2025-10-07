# API Gateway

API Gateway module acts as the entrypoint for incoming and outgoing requests to access Threefold chain and incoming rmb calls.

## How it works

- Connects to `Redis` message broker over `zbus`
- Loads the node’s identity key.
- Creates a substrate manager to handle blockchain interactions.
- Creates substrate gateway and registers it as the `api-gateway` module.
- Handles incoming `Rmb` calls from nodes and routes them.
- Periodically checks for updates in relay and substrate urls.

## Usage

```sh
api-gateway --broker unix:///var/run/redis.sock \
       --workers 2
```

### Command-line flags

| Flag        | Description                                | Default                      |
| ----------- | -------------------------------------------| ---------------------------- |
| `--broker`  | Connection string to the message BROKER    | `unix:///var/run/redis.sock` |
| `--workers` | Number of workers N                        | `1`                          |
