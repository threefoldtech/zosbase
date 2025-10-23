# Update Worker Tool
 
## Introduction

Update worker tool is responsible for managing and updating zos versions, it handles symbolic links between deployed zos versions and their corresponding network environments.
Each network directory in the destination is represented as a symlink that points to a specific zos version directory in the src. When a new zos version is deployed, it updates the symlinks to point to the latest release.

```perl
  ├── src/   
  │   ├── v3.1.0/
  │   └── v3.2.0/
  └── dst/  
      ├── production -> ../releases/v3.2.0
      └── qa -> ../releases/v3.1.0
```

# Code Flow
1. Tool user creates a new worker using `NewWorker` and use it to call `UpdateWithInterval`
```go
		worker, err := internal.NewWorker(src, dst, params)
		if err != nil {
			return err
		}
		worker.UpdateWithInterval(cmd.Context())
		return nil
```

2. `UpdateWithInterval` updates zos version for each network through `updateZosVersion` private method, which fetches the latest zos version from the chain and update the symbolic link to point to the correct version file

3. `updateZosVersion` calculates relative path between src, dst to pass the correct link to `updateLink` private method

4. `updateLink` ensures that the symlink at **latest** points to the correct **current** version


## Structure
### Worker Interface
```go
type Worker struct {
    src string  // .tag-<version> files for each zos version
    dst string  // contains symlinks that points to files in src

    interval  time.Duration
    substrate map[Network]client.Manager
}

// UpdateWithInterval updates the latest zos flist for a specific network with the updated zos version with a specific interval between each update
// It uses exponential backoff for retrying failed updates
func (w *Worker) UpdateWithInterval(ctx context.Context)
```

#### Constructor

```go
func NewWorker(src string, dst string, params Params) (*Worker, error)
```