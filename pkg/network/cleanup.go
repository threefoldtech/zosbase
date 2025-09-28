package network

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
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

// CleanupOrphanedNamespaces removes network namespaces that start with "n-"
// but don't have corresponding files in /var/run/cache/networkd/networks/
func CleanupOrphanedNamespaces() error {
	const networksDir = "/var/run/cache/networkd/networks/"

	// Get list of files in the networks directory
	validNetworkIDs := make(map[string]bool)
	entries, err := os.ReadDir(networksDir)
	if err != nil {
		return fmt.Errorf("failed to read networks directory %s: %w", networksDir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			validNetworkIDs[entry.Name()] = true
		}
	}

	// Get list of network namespaces
	cmd := exec.Command("ip", "netns", "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list network namespaces: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse namespace name (format: "n-EB2AJhw1cGbYf (id: 15)" or just "n-EB2AJhw1cGbYf")
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		nsName := parts[0]

		// Only process namespaces that start with "n-"
		if !strings.HasPrefix(nsName, "n-") {
			continue
		}

		// Extract the ID part after "n-"
		networkID := strings.TrimPrefix(nsName, "n-")

		// Check if corresponding file exists in networks directory
		if !validNetworkIDs[networkID] {
			// Remove the orphaned namespace
			delCmd := exec.Command("ip", "netns", "del", nsName)
			if err := delCmd.Run(); err != nil {
				log.Debug().Str("namespace", nsName).Err(err).Msg("failed to delete namespace")
				continue
			}
		}
	}

	return nil
}
