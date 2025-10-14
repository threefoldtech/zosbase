# Zui

The zui module is a gui tool that launches a terminal-based dashboard for monitoring and visualizing system information.

## How it works

- Connects to `Redis` message broker over `zbus`
- Defines some paragraphs/widgets:

    | Widget            | Purpose                             |
    | ----------------- | ----------------------------------- |
    | `header`          | Displays general system info        |
    | `services`        | Monitors the state of some services |
    | `errorsParagraph` | Displays any reported errors        |
    | `netgrid`         | Shows network info                  |
    | `resources`       | Shows system resources              |

<img src="../assets/zui.png" alt="zui" height="200"/>

- `services` widget only shows when the services are starting, then they're replaced with `Network` and `resources` after are services are active
## Usage

```sh
zui --broker unix:///var/run/redis.sock \
    --workers 2
```

### Command-line flags

| Flag        | Description                                | Default                      |
| ----------- | -------------------------------------------| ---------------------------- |
| `--broker`  | Connection string to the message BROKER    | `unix:///var/run/redis.sock` |
| `--workers` | Number of workers N                        | `1`                          |
