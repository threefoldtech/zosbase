package pkg

import (
	"context"
	"net"

	"github.com/threefoldtech/zosbase/pkg/gridtypes"
	"github.com/threefoldtech/zosbase/pkg/gridtypes/zos"
)

//go:generate mkdir -p stubs
//go:generate zbusc -module netlight -version 0.0.1 -name netlight -package stubs github.com/threefoldtech/zosbase/pkg+NetworkerLight stubs/network_light_stub.go

// NetworkerLight is the interface for the network light module
type NetworkerLight interface {
	// Create(name string, net zos.NetworkLight, seed []byte) error
	Create(name string, wl gridtypes.WorkloadID, net zos.NetworkLight) error
	Delete(name string) error
	AttachPrivate(name, id string, vmIp net.IP) (device TapDevice, err error)
	AttachMycelium(name, id string, seed []byte) (device TapDevice, err error)
	Detach(id string) error
	Interfaces(iface string, netns string) (Interfaces, error)
	AttachZDB(id string) (string, error)
	ZDBIPs(namespace string) ([]net.IP, error)
	Namespace(id string) string
	Ready() error
	ZOSAddresses(ctx context.Context) <-chan NetlinkAddresses
	SetPublicConfig(cfg PublicConfig) error
	UnSetPublicConfig() error
	LoadPublicConfig() (PublicConfig, error)

	WireguardPorts() ([]uint, error)
	GetDefaultGwIP(id NetID) (net.IP, error)
	GetNet(id NetID) (net.IPNet, error)
	GetSubnet(id NetID) (net.IPNet, error)
}

type TapDevice struct {
	Name   string
	Mac    net.HardwareAddr
	IP     *net.IPNet
	Routes []Route
}

// Interfaces struct to bypass zbus generation error
// where it generate a stub with map as interface instead of map
type Interfaces struct {
	Interfaces map[string]Interface
}
