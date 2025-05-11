package api

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/threefoldtech/zosbase/pkg"
	"github.com/threefoldtech/zosbase/pkg/gridtypes"
)

func (a *API) NetworkWGPorts(ctx context.Context) ([]uint, error) {
	if a.isLightMode() {
		return nil, ErrNotSupportedInLightMode
	}
	return a.networkerStub.WireguardPorts(ctx)
}

func (a *API) NetworkPublicConfigGet(ctx context.Context, _ any) (pkg.PublicConfig, error) {
	if a.isLightMode() {
		return pkg.PublicConfig{}, ErrNotSupportedInLightMode
	}

	return a.networkerStub.GetPublicConfig(ctx)
}

func (a *API) NetworkHasIPv6(ctx context.Context) (bool, error) {
	if a.isLightMode() {
		return false, nil
	}

	ipData, err := a.networkerStub.GetPublicIPv6Subnet(ctx)
	hasIP := ipData.IP != nil && err == nil
	return hasIP, nil

}

func (a *API) NetworkListPublicIPs(ctx context.Context) ([]string, error) {
	if a.isLightMode() {
		return nil, ErrNotSupportedInLightMode
	}

	return a.provisionStub.ListPublicIPs(ctx)
}

func (a *API) NetworkListPrivateIPs(ctx context.Context, networkName string) ([]string, error) {
	name := gridtypes.Name(networkName)
	twinID, ok := ctx.Value("twin_id").(uint32)
	if !ok {
		return nil, errors.New("could not get twin_id from context")
	}
	return a.provisionStub.ListPrivateIPs(ctx, twinID, name)
}

func (a *API) NetworkInterfaces(ctx context.Context) (pkg.Interfaces, error) {
	if a.isLightMode() {
		return a.networkerLightStub.Interfaces(ctx, "zos", "")
	}

	type q struct {
		inf    string
		ns     string
		rename string
	}
	result := pkg.Interfaces{
		Interfaces: make(map[string]pkg.Interface),
	}
	for _, i := range []q{{"zos", "", "zos"}, {"nygg6", "ndmz", "ygg"}} {
		ips, mac, err := a.networkerStub.Addrs(ctx, i.inf, i.ns)
		if err != nil {
			return pkg.Interfaces{Interfaces: make(map[string]pkg.Interface)}, fmt.Errorf("failed to get ips for '%s' interface: %w", i.inf, err)
		}

		iface := pkg.Interface{
			Name: i.rename,
			Mac:  mac,
			IPs:  []net.IPNet{},
		}

		for _, item := range ips {
			ipNet := net.IPNet{
				IP:   item,
				Mask: nil,
			}
			iface.IPs = append(iface.IPs, ipNet)
		}

		result.Interfaces[i.rename] = iface
	}

	return result, nil
}

// all interfaces on the node
func (a *API) AdminInterfaces(ctx context.Context) (pkg.Interfaces, error) {
	if a.isLightMode() {
		return a.networkerLightStub.Interfaces(ctx, "", "")
	}

	return a.networkerStub.Interfaces(ctx, "", "")
}

func (a *API) AdminSetPublicNIC(ctx context.Context, device string) error {
	if a.isLightMode() {
		return ErrNotSupportedInLightMode
	}
	return a.networkerStub.SetPublicExitDevice(ctx, device)
}

func (a *API) AdminGetPublicNIC(ctx context.Context) (pkg.ExitDevice, error) {
	if a.isLightMode() {
		return pkg.ExitDevice{}, ErrNotSupportedInLightMode
	}

	return a.networkerStub.GetPublicExitDevice(ctx)
}
