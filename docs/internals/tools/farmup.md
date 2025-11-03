# Farmup Tool
 
## Introduction

The farmup tool is responsible for getting all the nodes in a farm to power up.

## Code Flow
1. The command fires the `run` function with the passed options (flags) 
2. It then gets all nodes in the farm and updates their power target on the chain to `up`


## Command-line flags

| Flag          | Description                                | Default           |
| ------------- | ------------------------------------------ | ----------------- |
| `--network`   | network (main, test, dev)                  | main              |
| `--mnemonics` | mnemonics for the farmer (required)        | ""                |
| `--farm`      | farm ID (required)                         | 0                 |
| `--debug`     | show debugging logs                        | false             |
