# RRD Package

## Introduction

Package rrd provides a Round Robin Database (RRD) implementation using BoltDB.

The RRD is designed to store monotonically increasing counter values, making it easy to compute the increase of a metric over a fixed time window. Data is stored in fixed-size time slots, and older data is automatically deleted after the retention duration. 

## Structure

**Interfaces:**                 
   - `RRD` : the main API for interacting with the DB 
   - `Slot` : represents one time window bucket      
   - `Printer` : for printing DB contents             
                             
**Structs:**              
   - `rrdBolt` implements `RRD`
   - `rrdSlot` implements `Slot`

### 1- RRD Interface
```go
type RRD interface {
	// Slot returns the current window (slot) to store values.
	Slot() (Slot, error)

	// Counters, return all stored counters since the given time (since) until now.
	Counters(since time.Time) (map[string]float64, error)

	// Last returns the last reported value for a metric given the metric name
	Last(key string) (value float64, ok bool, err error)

	// Close the db
	Close() error
}
```

#### Constructor

```go
// NewRRDBolt creates a new rrd database that uses bolt as a storage
// if window or retention are 0, the function will panic.
// If retnetion is smaller then window the function will panic.
// retention and window must be multiple of 1 minute.
func NewRRDBolt(path string, window time.Duration, retention time.Duration) (RRD, error)
```

#### rrdBolt Struct
`rrdBolt` implements `RRD` and also implements the `Printer` interface

```go
type rrdBolt struct { 
	db        *bolt.DB // BoltDB database handle
	window    uint64   // Window size in seconds
	retention uint64   // Retention duration in seconds
}

// RDD interface methods
Slot() (Slot, error)
Counters(since time.Time) (map[string]float64, error)
Last(key string) (value float64, ok bool, err error)
Close() error

// Printer interface methods 
Print(out io.Writer) error

// extra methods
// Slots return all the slot timestamps stored in database
Slots() ([]uint64, error)
```

### 2- Slot Interface

```go
type Slot interface {
	// Counter sets (or overrides) the current stored value for this key, with the passed value
	Counter(key string, value float64) error

	// Key return the key of the slot which is the window timestamp
	Key() uint64
}
```

#### rrdSlot Struct
`rrdSlot` implements `Slot`

```go
type rrdSlot struct {
	db  *bolt.DB // BoltDB database handle
	key uint64   // slot timestamp
}

// Slot interface methods
Counter(key string, value float64) error
Key() uint64
```

### 3- Printer Interface

```go
type Printer interface {
    // Print writes the content of the RRD to the passed writer
	Print(w io.Writer) error
}
```