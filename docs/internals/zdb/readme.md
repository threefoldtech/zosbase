# ZDB Module

## Introduction

Package zdb provides a Go client for interacting with [0-DB](https://github.com/threefoldtech/0-DB), which is an efficient key-value store.
The package offers a simple API for creating, managing, and inspecting 0-DB namespaces abstracting manual Redis commands, in order to control namespace size limits, running modes, passwords, and public visibility.


## Interface

A namespace is a complete set of key/data. Each namespace can be optionally protected by a password and size limited.
You are always attached to a namespace, by default, it's namespace `default`.
```go
type Namespace struct {
    Name              string         `yaml:"name"`                     // The namespace name
    DataLimit         gridtypes.Unit `yaml:"data_limits_bytes"`        // Maximum data size in bytes
    DataDiskFreespace gridtypes.Unit `yaml:"data_disk_freespace_bytes"`// Disk space available in bytes
    Mode              string         `yaml:"mode"`                     // Running mode (user or seq)
    PasswordProtected bool           `yaml:"password"`                 // Whether password protection is enabled
    Public            bool           `yaml:"public"`                   // Whether the namespace is public
}
```

The zdb package exposes a `Client` interface that defines all 0-DB operations, including creating a 0-DB connection, managing namespaces, configuration, and retrieving database size.

```go
type Client interface {
    // Connect creates the connection pool and verifies connectivity by pinging the 0-DB server.
	Connect() error

    // Close releases the resources used by the client.
	Close() error

    // CreateNamespace creates a new namespace. Only admin can do this.
    // By default, a namespace is not password protected, is public and not size limited.
	CreateNamespace(name string) error

    // Exist checks if namespace exists
	Exist(name string) (bool, error)

    // DeleteNamespace deletes a namespace. Only admin can do this.
    // You can't remove the namespace you're currently using.
    // Any other clients using this namespace will be moved to a special state, awaiting to be disconnected.
	DeleteNamespace(name string) error

    // Namespaces returns a slice of all available namespaces name.
	Namespaces() ([]string, error)

    // Namespace retrieves returns basic informations about a specific namespace
	Namespace(name string) (Namespace, error)

    // NamespaceSetSize sets the maximum size in bytes, of the namespace's data set
	NamespaceSetSize(name string, size uint64) error

    // NamespaceSetPassword locks the namespace by a password, use * password to clear it
	NamespaceSetPassword(name, password string) error

    // NamespaceSetMode sets the mode of the namespace
	NamespaceSetMode(name, mode string) error

    // NamespaceSetPublic changes the public flag, a public namespace can be read-only if a password is set
	NamespaceSetPublic(name string, public bool) error

    // NamespaceSetLock set namespace in read-write protected or normal mode (0 or 1)
	NamespaceSetLock(name string, lock bool) error

    // DBSize returns the size of the database in bytes
	DBSize() (uint64, error)
}
```

