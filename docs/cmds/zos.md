# Zos

Zos contains the `main` function which is the main entry point of `ZOS`.

## How it works

- Creates a `cli` app named `zos` and adds its flags and subcommands (such as flistd, networkd, provisiond, etc.). Each module represents a system daemon or service that runs as part of ZOS.
- If no subcommand was passed, `zos` prints the help menu and exits.
- Otherwise, if `list` flag was true, `zos` lists all available modules names to automate building of symlinks
- Allow each submodule to be called directy so that these both commands work the same:
    ```sh
    zos flistd
    flistd
    ```

## Usage

```sh
zos   --list
```


### Command-line flags

| Flag        | Description                                                 |
| ----------  | ----------------------------------------------------------- |
| `--list`    | Hidden flag used internally to list available module names  |
| `--debug`   | Force debug level                                           |