# Certify Tool
 
## Introduction
Certify tool is responsible for setting the nodes certification state. 

## Code Flow
1. The command fires the `run` function with the passed options (flags) 
2. It then queries the postgres database using graphql with `certification == Diy` and `secure == true` to get all the nodes on the chain that needs manual certification
3. Sets the node certification state to true on the chain


## Command-line flags

| Flag          | Description                                | Default           |
| ------------- | ------------------------------------------ | ----------------- |
| `--network`   | network (main, test, dev)                  | main              |
| `--dry-run`   | print the list of the nodes to be migrated | false             |
| `--mnemonics` | mnemonics for the sudo key (required)      | ""                |
| `--debug`     | show debugging logs                        | false             |
