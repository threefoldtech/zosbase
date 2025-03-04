package nft

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"text/template"

	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"

	"github.com/pkg/errors"
)

// Apply applies the ntf configuration contained in the reader r
// if ns is specified, the nft command is execute in the network namespace names ns
func Apply(r io.Reader, ns string) error {
	var cmd *exec.Cmd

	if ns != "" {
		cmd = exec.Command("ip", "netns", "exec", ns, "nft", "-f", "-")
	} else {
		cmd = exec.Command("nft", "-f", "-")
	}

	cmd.Stdin = r

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Str("output", string(out)).Msg("error during nft")
		if eerr, ok := err.(*exec.ExitError); ok {
			return errors.Wrapf(err, "failed to execute nft: %v", string(eerr.Stderr))
		}
		return errors.Wrap(err, "failed to execute nft")
	}
	return nil
}

// DropTrafficToLAN drops all the outgoing traffic to any peers on
// the same lan network, but allow dicovery port for ygg/myc by accepting
// traffic to/from dest/src ports.
func DropTrafficToLAN(rules string) error {
	dgw, err := getDefaultGW()
	if err != nil {
		return fmt.Errorf("failed to find default gateway: %w", err)
	}

	if !dgw.IP.IsPrivate() {
		log.Warn().Msg("skip LAN security. default gateway is public")
		return nil
	}

	ipAddr := dgw.IP.String()
	netAddr := getNetworkRange(dgw)
	log.Debug().
		Str("ipAddr", ipAddr).
		Str("netAddr", netAddr).
		Msg("drop traffic to lan with the default gateway")

	templ, err := template.New("lanSecurityRules").Parse(rules)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := templ.Execute(&buf, map[string]string{
		"GatewayIP": ipAddr,
		"SubnetIP":  netAddr,
	}); err != nil {
		return err
	}

	return Apply(&buf, "")
}

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
