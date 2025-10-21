# Rotate package

## Introduction

rotate package provides a very simple tool to truncate a given file to 0 after it copies the last *configurable* part of this file to a new file with suffix .tail

The idea is that services need to have their log files (or redirection) be open in append mode. So truncation of the log file should be enough.

There is no guarantee that some logs will be lost between the copying of the file tail and the truncation of the file.

## Structure

### Configuration and Initialization

This section implements the options interface, which allow flexible configuration of the Rotator without a complex parameter list.

```go
type Option interface {
	apply(cfg *Rotator)
}

type optFn func(cfg *Rotator)

func (fn optFn) apply(cfg *Rotator) {
	fn(cfg)
}
```

The package provides a default Rotator and a constructor function that accepts a variable number of Option values to override
the required settings only

```go
  // Create a rotator with custom limits
  r := NewRotator(
      MaxSize(50 * Megabytes),     // rotate after 50MB
      TailSize(25 * Megabytes),    // keep last 25MB
      Suffix(".anything"),         // name rotated files "*.anything"
  )
```

The key components are:
  - `Option` interface: defines a configuration option that can be changes in a Rotator.
  - `optFn` type: a function adapter that implements the Option interface.
  - `MaxSize`, `TailSize`, `Suffix`: helpers that return Option values to adjust specific fields.
  - `NewRotator`: overrides passed options only over the default configuration and returns a Rotator.


### Rotator Interface
```go

type Rotator struct {
    maxsize Size     // maxsize is the max file size. If file size exceeds maxsize rotation is applied
                     // otherwise file is not touched
                     // default : 20 MB
    
    tailsize Size    // tailsize is size of the chunk to keep with Suffix before truncation of the file.
                     // If value is bigger than MaxSize, it will be set to MaxSize.
                     // default : 10 MB

    suffix string    // suffix of the tail chunk, default to .0
}

// Rotate takes a file and compares it to maxsize, it only rotates if file size > max size
// It copies the tailsize to tail file and truncates the old file to 0
func (r *Rotator) Rotate(file string) error

// RotateAll will rotate all files in the directory 
// if a set of names is given only named files will be rotated, other unknown files will be deleted
func (r *Rotator) RotateAll(dir string, names ...string) error
```

### Constructor 
```go
func NewRotator(opt ...Option) Rotator 
```