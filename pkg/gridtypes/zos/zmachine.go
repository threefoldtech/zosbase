package zos

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zosbase/pkg/gridtypes"
)

// MachineNetwork structure
type MachineNetwork struct {
	// PublicIP optional public IP attached to this machine. If set
	// it must be a valid name of a PublicIP workload in the same deployment
	PublicIP gridtypes.Name `json:"public_ip"`
	// Planetary support planetary network
	Planetary bool `json:"planetary"`

	// Mycelium IP config, if planetary is true, but Mycelium is not set we fall back
	// to yggdrasil support. Otherwise (if mycelium is set) a mycelium ip is used instead.
	Mycelium *MyceliumIP `json:"mycelium,omitempty"`

	// Interfaces list of user znets to join
	Interfaces []MachineInterface `json:"interfaces"`
}

// Challenge builder
func (n *MachineNetwork) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s", n.PublicIP); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%t", n.Planetary); err != nil {
		return err
	}

	for _, inf := range n.Interfaces {
		if _, err := fmt.Fprintf(w, "%s", inf.Network); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(w, "%s", inf.IP.String()); err != nil {
			return err
		}
	}

	if n.Mycelium != nil {
		if err := n.Mycelium.Challenge(w); err != nil {
			return err
		}
	}

	return nil
}

// ZMachine reservation data
type ZMachine struct {
	// Flist of the zmachine, must be a valid url to an flist.
	FList string `json:"flist"`
	// Network configuration for machine network
	Network MachineNetwork `json:"network"`
	// Size of zmachine disk
	Size gridtypes.Unit `json:"size"`
	// ComputeCapacity configuration for machine cpu+memory
	ComputeCapacity MachineCapacity `json:"compute_capacity"`
	// Mounts configure mounts/disks attachments to this machine
	Mounts []MachineMount `json:"mounts"`

	// following items are only available in container mode. if FList is for a container
	// not a VM.

	// Entrypoint entrypoint of the container, if not set the configured one from the flist
	// is going to be used
	Entrypoint string `json:"entrypoint"`
	// Env variables available for a container
	Env map[string]string `json:"env"`
	// Corex works in container mode which forces replace the
	// entrypoing of the container to use `corex`
	Corex bool `json:"corex"`

	// GPU attached to the VM
	// the list of the GPUs ids must:
	// - Exist, obviously
	// - Not used by other VMs
	// - Only possible on `dedicated` nodes
	GPU []GPU `json:"gpu,omitempty"`
}

func (m *ZMachine) MinRootSize() gridtypes.Unit {
	// sru = (cpu * mem_in_gb) / 8
	// each 1 SRU is 50GB of storage
	cu := gridtypes.Unit(m.ComputeCapacity.CPU) * m.ComputeCapacity.Memory / (8 * gridtypes.Gigabyte)

	if cu == 0 {
		return 500 * gridtypes.Megabyte
	}

	return 2 * gridtypes.Gigabyte
}

func (m *ZMachine) RootSize() gridtypes.Unit {
	min := m.MinRootSize()
	if m.Size > min {
		return m.Size
	}

	return min
}

// Valid implementation
func (v ZMachine) Valid(getter gridtypes.WorkloadGetter) error {
	if len(v.Network.Interfaces) != 1 {
		return fmt.Errorf("only one network private network is supported at the moment")
	}

	for _, inf := range v.Network.Interfaces {
		if inf.IP.To4() == nil && inf.IP.To16() == nil {
			return fmt.Errorf("invalid IP")
		}
	}
	if v.ComputeCapacity.CPU == 0 {
		return fmt.Errorf("cpu capacity can't be 0")
	}
	if v.ComputeCapacity.Memory < 250*gridtypes.Megabyte {
		return fmt.Errorf("mem capacity can't be less that 250M")
	}
	minRoot := v.MinRootSize()
	if v.Size != 0 && v.Size < minRoot {
		return fmt.Errorf("disk size can't be less that %d. Set to 0 for minimum", minRoot)
	}
	if !v.Network.PublicIP.IsEmpty() {
		wl, err := getter.Get(v.Network.PublicIP)
		if err != nil {
			return fmt.Errorf("public ip is not found")
		}

		if wl.Type != PublicIPv4Type && wl.Type != PublicIPType {
			return errors.Wrapf(err, "workload of name '%s' is not a public ip", v.Network.PublicIP)
		}

		// also we need to make sure this public ip is not used by other vms in the same
		// deployment.
		allVMs := getter.ByType(ZMachineType)
		count := 0
		for _, vm := range allVMs {
			var data ZMachine
			if err := json.Unmarshal(vm.Data, &data); err != nil {
				return err
			}
			// we can only check the name because unfortunately we don't know
			// `this` workload ID at this level. may be can b added later.
			// for now, we can just count the number of this public ip workload
			// name was referenced in all VMs and fail if it's more than one.
			if data.Network.PublicIP == v.Network.PublicIP {
				count += 1
			}
		}
		if count > 1 {
			return fmt.Errorf("public ip is assigned to multiple vms")
		}
	}

	for _, ifc := range v.Network.Interfaces {
		if ifc.Network == "ygg" || ifc.Network == "pub" || ifc.Network == "mycelium" { //reserved temporary
			return fmt.Errorf("'%s' is reserved network name", ifc.Network)
		}
	}

	mycelium := v.Network.Mycelium
	if mycelium != nil {
		if len(mycelium.Seed) != MyceliumIPSeedLen {
			return fmt.Errorf("invalid mycelium seed length expected 6 bytes")
		}
	}

	return nil
}

// Capacity implementation
func (v ZMachine) Capacity() (gridtypes.Capacity, error) {
	return gridtypes.Capacity{
		CRU: uint64(v.ComputeCapacity.CPU),
		MRU: v.ComputeCapacity.Memory,
		SRU: v.RootSize(),
	}, nil
}

// Challenge creates signature challenge
func (v ZMachine) Challenge(b io.Writer) error {
	if _, err := fmt.Fprintf(b, "%s", v.FList); err != nil {
		return err
	}

	if err := v.Network.Challenge(b); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(b, "%d", v.Size); err != nil {
		return err
	}

	if err := v.ComputeCapacity.Challenge(b); err != nil {
		return err
	}

	for _, mnt := range v.Mounts {
		if err := mnt.Challenge(b); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(b, "%s", v.Entrypoint); err != nil {
		return err
	}

	encodeEnv := func(w io.Writer, env map[string]string) error {
		keys := make([]string, 0, len(env))
		for k := range env {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			if _, err := fmt.Fprintf(w, "%s=%s", k, env[k]); err != nil {
				return err
			}
		}

		return nil
	}

	if err := encodeEnv(b, v.Env); err != nil {
		return err
	}

	for _, gpu := range v.GPU {
		if _, err := fmt.Fprintf(b, "%s", gpu); err != nil {
			return err
		}
	}

	return nil
}

// ZMachineResult result returned by VM reservation
type ZMachineResult struct {
	ID          string `json:"id"`
	IP          string `json:"ip"`
	PlanetaryIP string `json:"planetary_ip"`
	MyceliumIP  string `json:"mycelium_ip"`
	ConsoleURL  string `json:"console_url"`
}

func (r *ZMachineResult) UnmarshalJSON(data []byte) error {
	var deprecated struct {
		ID          string `json:"id"`
		IP          string `json:"ip"`
		YggIP       string `json:"ygg_ip"`
		PlanetaryIP string `json:"planetary_ip"`
		MyceliumIP  string `json:"mycelium_ip"`
		ConsoleURL  string `json:"console_url"`
	}

	if err := json.Unmarshal(data, &deprecated); err != nil {
		return err
	}

	r.ID = deprecated.ID
	r.IP = deprecated.IP
	r.PlanetaryIP = deprecated.PlanetaryIP
	if deprecated.YggIP != "" {
		r.PlanetaryIP = deprecated.YggIP
	}
	r.MyceliumIP = deprecated.MyceliumIP
	r.ConsoleURL = deprecated.ConsoleURL

	return nil
}
