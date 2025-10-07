# zbusdebug

`zbusdebug` show status summery for running `zbus` modules

## How it works

- Connects to `Redis` message broker over `zbus`
- Prints this specific module's status if `--module` flag was passed, else prints all modules' status

## Usage

```sh
zbusdebug  --broker unix:///var/run/redis.sock \
>          --module flist
```

### Sample Output

```yaml
## Status for  flist
objects:
- name: flist
  version: 0.0.1
workers:
- state: free
  time: 2025-10-07T06:31:39.56952271Z
```

### Command-line flags

| Flag        | Description                                | Default                      |
| ----------- | -------------------------------------------| ---------------------------- |
| `--broker`  | Connection string to the message BROKER    | `unix:///var/run/redis.sock` |
| `--module`  | debug specific `MODULE`                    | ""                           |
