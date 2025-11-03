# Registrar package

The `registrar` pkg is used to handle node registration on ThreeFold Grid. It is used by the `noded` module, which triggers the registration process by first collecting node capacity information, creating a Redis client, and then adding the `registrar` to the intercommunication process using `zbus` for remote procedure calls (RPC).
re-registration occurs automatically every 24 hours or immediately after an IP address change, ensuring the node's information stays up-to-date.
registration process can fail due to various reasons, such as RPC calls failure.

The registration process includes:

1. Collecting node information
2. Creating/Ensuring a twinID exists for the node
3. Registering the node on the blockchain

## Error Handling

`ErrInProgress`: Error raised if the node registration is still in progress.

`ErrFailed`: Error raised if the node registration fails.

## Constants

### Node registration state constants

`Failed`: Node registration failed

`InProgress`: Node registration is in progress

`Done`: Node registration is completed

## Structs

### State Struct

Used to store the state of the node registration.


```go
type State struct {
	NodeID uint32               // The ID of the node.
	TwinID uint32               // The twin ID of the node.
	State  RegistrationState    // The state of the node registration.
	Msg    string               // The message associated with the node registration state.
}
```

### RegistrationInfo Struct

Used to store the capacity, location, and other information of the node.


```go
type RegistrationInfo struct {
	Capacity     gridtypes.Capacity  // The capacity of the node.
	Location     geoip.Location      // The location of the node.
	SecureBoot   bool                // State whether the node is booted via efi or not.
	Virtualized  bool                // State whether the node has hypervisor on it or not.
	SerialNumber string              // The serial number of the node.
	GPUs map[string]interface{}      // List of gpus short name
}
// Set the capacity of the node
WithCapacity(v gridtypes.Capacity) RegistrationInfo

// Set the GPUs of the node
WithGPU(short string) RegistrationInfo

// Set the location of the node
WithLocation(v geoip.Location) RegistrationInfo

// Set the secure boot status of the node
WithSecureBoot(v bool) RegistrationInfo

// Set the serial number of the node
WithSerialNumber(v string) RegistrationInfo

// Set the virtualized status of the node
WithVirtualized(v bool) RegistrationInfo

```

### Registrar Struct

The registrar is used to register nodes on the ThreeFold Grid.

```go
type Registrar struct {
	state State         // The state of the registrar.
	mutex sync.RWMutex  // A mutex for synchronizing access to the registrar.
}

NodeID() (uint32, error) // Returns the node ID if the registrar is in the done state
TwinID() (uint32, error) // Returns the twin ID if the registrar is in the done state
```

## Functions

```go
// Creates a new registrar with the given context, client, environment, and registration information, starts the registration process and returns the registrar.
NewRegistrar(ctx context.Context, cl Zbus.Client, env environment.Environment, info RegistrationInfo) *Registrar


// Returns a failed state with the given error.
FailedState(err error) State

// Returns an in progress state
InProgressState() State

// Returns a done state with the given node ID and twin ID.
DoneState(nodeID , twinID uint32) State
```


