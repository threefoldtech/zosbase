package network

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/vishvananda/netlink"
)

func CleanupUnusedLinks() error {
	links, err := netlink.LinkList()
	if err != nil {
		return fmt.Errorf("failed to list network interfaces: %w", err)
	}

	for _, link := range links {
		attrs := link.Attrs()
		if attrs == nil {
			continue
		}

		interfaceName := attrs.Name

		if attrs.OperState == netlink.OperDown &&
			hasNoCarrier(interfaceName) &&
			isOrphanVMTap(interfaceName) {
			if err := netlink.LinkDel(link); err != nil {
				return fmt.Errorf("failed to delete interface %s: %w", interfaceName, err)
			}
		}
	}

	return nil
}

func hasNoCarrier(interfaceName string) bool {
	cmd := exec.Command("ip", "link", "show", interfaceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "NO-CARRIER")
}

func isOrphanVMTap(interfaceName string) bool {
	return len(interfaceName) == 15 && strings.HasPrefix(interfaceName, "t-")
}
