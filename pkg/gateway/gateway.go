package gateway

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/zinit"
	"gopkg.in/yaml.v2"
)

const (
	traefikService = "traefik"
)

type gatewayModule struct {
	cl zbus.Client

	proxyConfigPath string
}

type ProxyConfig struct {
	Http HTTPConfig
}

type HTTPConfig struct {
	Routers  map[string]Router
	Services map[string]Service
}

type Router struct {
	Rule    string
	Service string
}

type Service struct {
	LoadBalancer LoadBalancer
}

type LoadBalancer struct {
	Servers []Server
}

type Server struct {
	Url string
}

func New(ctx context.Context, cl zbus.Client, root string) (pkg.Gateway, error) {
	configPath := filepath.Join(root, "proxy")
	// where should service-restart/node-reboot recovery be handled?
	err := os.MkdirAll(configPath, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make gateway config dir")
	}

	return &gatewayModule{
		cl:              cl,
		proxyConfigPath: configPath,
	}, nil
}

func (g *gatewayModule) isTraefikStarted(z *zinit.Client) (bool, error) {
	traefikStatus, err := z.Status(traefikService)
	if errors.Is(err, zinit.ErrUnknownService) {
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "failed to check traefik status")
	}

	return traefikStatus.State.Is(zinit.ServiceStateRunning), nil
}

// ensureGateway makes sure that gateway infrastructure is in place and
// that it is supported.
func (g *gatewayModule) ensureGateway(ctx context.Context) (string, error) {
	var (
		networker = stubs.NewNetworkerStub(g.cl)
	)
	cfg, err := networker.GetPublicConfig(ctx)
	if err != nil {
		return "", errors.Wrap(err, "gateway is not supported on this node")
	}

	if cfg.Domain == "" {
		return "", errors.Errorf("gateway is not supported. missing domain configuration")
	}

	z, err := zinit.Default()
	if err != nil {
		return "", errors.Wrap(err, "failed to connect to zinit")
	}
	defer z.Close()
	running, err := g.isTraefikStarted(z)
	if err != nil {
		return "", errors.Wrap(err, "failed to check traefik status")
	}

	if running {
		return cfg.Domain, nil
	}

	//other wise we start traefik
	return cfg.Domain, g.startTraefik(z)
}

func (g *gatewayModule) startTraefik(z *zinit.Client) error {
	cmd := fmt.Sprintf("ip netns exec public traefik --log.level=DEBUG --providers.file.directory=%s --providers.file.watch=true", g.proxyConfigPath)
	zinit.AddService(traefikService, zinit.InitService{
		Exec: cmd,
	})
	if err := z.Monitor(traefikService); err != nil {
		return errors.Wrap(err, "couldn't monitor traefik service")
	}
	if err := z.StartWait(time.Second*20, traefikService); err != nil {
		return errors.Wrap(err, "waiting for trafik start timed out")
	}
	return nil
}

func (g *gatewayModule) configPath(name string) string {
	return filepath.Join(g.proxyConfigPath, fmt.Sprintf("%s.yaml", name))
}

func (g *gatewayModule) SetNamedProxy(wlID string, prefix string, backends []string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	domain, err := g.ensureGateway(ctx)
	if err != nil {
		return "", err
	}

	fqdn := fmt.Sprintf("%s.%s", prefix, domain)

	rule := fmt.Sprintf("Host(`%s`) && PathPrefix(`/`)", fqdn)
	servers := make([]Server, len(backends))
	for idx, backend := range backends {
		servers[idx] = Server{
			Url: backend,
		}
	}

	config := ProxyConfig{
		Http: HTTPConfig{
			Routers: map[string]Router{
				wlID: {
					Rule:    rule,
					Service: wlID,
				},
			},
			Services: map[string]Service{
				wlID: {
					LoadBalancer: LoadBalancer{
						Servers: servers,
					},
				},
			},
		},
	}

	yamlString, err := yaml.Marshal(&config)
	if err != nil {
		return "", errors.Wrap(err, "failed to convert config to yaml")
	}
	log.Debug().Str("yaml-config", string(yamlString)).Msg("configuration file")
	if err = os.WriteFile(g.configPath(wlID), yamlString, 0644); err != nil {
		return "", errors.Wrap(err, "couldn't open config file for writing")
	}

	return fqdn, nil
}
func (g *gatewayModule) DeleteNamedProxy(wlID string) error {
	if err := os.Remove(g.configPath(wlID)); err != nil {
		return errors.Wrap(err, "couldn't remove config file")
	}
	return nil
}
