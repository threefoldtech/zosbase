package nft

import (
	"bytes"
	"fmt"
	"io"
	"text/template"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"
)

func getDefaultGW() (netlink.Neigh, error) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return netlink.Neigh{}, fmt.Errorf("failed to list routes: %v", err)
	}

	var defaultRoute *netlink.Route
	for _, route := range routes {
		if route.Dst == nil {
			defaultRoute = &route
			break
		}
	}

	if defaultRoute == nil {
		return netlink.Neigh{}, fmt.Errorf("default route not found")
	}

	if defaultRoute.Gw == nil {
		return netlink.Neigh{}, fmt.Errorf("default route has no gateway")
	}

	neighs, err := netlink.NeighList(0, netlink.FAMILY_V4)
	if err != nil {
		return netlink.Neigh{}, fmt.Errorf("failed to list neighbors: %v", err)
	}

	for _, neigh := range neighs {
		if neigh.IP.Equal(defaultRoute.Gw) {
			return neigh, nil
		}
	}

	return netlink.Neigh{}, errors.New("failed to get default gw")
}

func getNetworkRange(ip netlink.Neigh) string {
	mask := ip.IP.DefaultMask()
	network := ip.IP.Mask(mask)
	ones, _ := mask.Size()
	networkRange := fmt.Sprintf("%s/%d", network.String(), ones)

	return networkRange
}

func renderRulesTemplate(tmpl string, gateway netlink.Neigh) (io.Reader, error) {
	GatewayIP := gateway.IP.String()
	SubnetIP := getNetworkRange(gateway)

	log.Debug().
		Str("GatewayIP", GatewayIP).
		Str("SubnetIP", SubnetIP).
		Msg("drop traffic to lan with the default gateway")

	templ, err := template.New("lanSecurityRules").Parse(tmpl)
	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	var buf bytes.Buffer
	if err := templ.Execute(&buf, map[string]string{
		"GatewayIP": GatewayIP,
		"SubnetIP":  SubnetIP,
	}); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return &buf, nil
}
