package ndmz

import (
	"context"
	"time"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosbase/pkg/network/dhcp"
	"github.com/threefoldtech/zosbase/pkg/network/ifaceutil"
	"github.com/threefoldtech/zosbase/pkg/network/namespace"
	"github.com/threefoldtech/zosbase/pkg/zinit"
	"github.com/vishvananda/netlink"
)

// DHCPMon monitor a network interface status and force
// renew of DHCP lease if needed
type DHCPMon struct {
	z       *zinit.Client
	service dhcp.ClientService
}

// NewDHCPMon create a new DHCPMon object managing interface iface
// namespace is then network namespace name to use. it can be empty.
func NewDHCPMon(iface, namespace string, z *zinit.Client) *DHCPMon {
	service := dhcp.NewService(iface, namespace, z)
	return &DHCPMon{
		z:       z,
		service: service,
	}
}

// Start creates a zinit service for a DHCP client and start monitoring it
// this method is blocking, start is in a goroutine if needed.
// cancel the context to start it.
func (d *DHCPMon) Start(ctx context.Context) error {

	if err := d.startZinit(); err != nil {
		return err
	}
	defer func() {
		if err := d.stopZinit(); err != nil {
			log.Error().Err(err).Msgf("error stopping %s zinit service", d.service.Name)
		}
	}()

	t := time.NewTicker(time.Minute)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-t.C:
			has, err := hasDefaultRoute(d.service.Iface, d.service.Namespace)
			if err != nil {
				log.Error().Str("iface", d.service.Iface).Err(err).Msg("error checking default gateway")
				continue
			}

			if !has {
				log.Info().Msg("ndmz default route missing, waking up dhcpcd")
				if err := d.wakeUp(); err != nil {
					log.Error().Err(err).Msg("error while sending signal to service ")
				}
			}
		}
	}
}

// wakeUp sends a signal to the dhcpcd daemon to force a release of the DHCP lease
func (d *DHCPMon) wakeUp() error {
	err := d.z.Kill(d.service.Name, zinit.SIGUSR1)
	if err != nil {
		log.Error().Err(err).Msg("error while sending signal to service ")
	}
	return err
}

// hasDefaultRoute checks if the network interface iface has a default route configured
// if netNS is not empty, switch to the network namespace named netNS before checking the routes
func hasDefaultRoute(iface, netNS string) (bool, error) {
	var hasDefault bool
	do := func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(iface)
		if err != nil {
			return err
		}
		hasDefault, _, err = ifaceutil.HasDefaultGW(link, netlink.FAMILY_V4)
		return err
	}

	var oerr error
	if netNS != "" {
		n, err := namespace.GetByName(netNS)
		if err != nil {
			return false, err
		}
		oerr = n.Do(do)
	} else {
		oerr = do(nil)
	}
	return hasDefault, oerr
}

func (d *DHCPMon) startZinit() error {
	if err := d.service.DestroyOlderService(); err != nil {
		return err
	}

	status, err := d.z.Status(d.service.Name)
	if err != nil && err != zinit.ErrUnknownService {
		log.Error().Err(err).Msgf("error checking zinit service %s status", d.service.Name)
		return err
	}

	if status.State.Exited() {
		log.Info().Msgf("zinit service %s already exists but is stopped, starting it", d.service.Name)
		return d.service.Start()
	} else if status.State.Is(zinit.ServiceStateRunning) {
		return nil
	}

	return d.service.Create()
}

// Stop stops a zinit background process
func (d *DHCPMon) stopZinit() error {
	err := d.z.StopWait(time.Second*10, d.service.Name)
	if err != nil {
		return errors.Wrapf(err, "failed to stop zinit service %s", d.service.Name)
	}
	return d.z.Forget(d.service.Name)
}
