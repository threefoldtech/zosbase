package healthcheck

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosbase/pkg/app"
	"github.com/threefoldtech/zosbase/pkg/environment"
)

const defaultRequestTimeout = 5 * time.Second

func networkCheck(ctx context.Context) []error {
	env := environment.MustGet()
	servicesUrl := []string{env.FlistURL}
	servicesUrl = append(servicesUrl, env.SubstrateURL...)
	servicesUrl = append(servicesUrl, env.GraphQL...)
	servicesUrl = append(servicesUrl, env.ActivationURL...)

	errCh := make(chan error, len(servicesUrl)+1) // +1 for relay URL
	var wg sync.WaitGroup

	// check all services
	for _, serviceUrl := range servicesUrl {
		wg.Add(1)
		go func(serviceUrl string) {
			defer wg.Done()
			if err := checkService(ctx, serviceUrl); err != nil {
				errCh <- err
			}
		}(serviceUrl)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := verifyAtLeastOne(ctx, env.RelayURL); err != nil {
			errCh <- err
		}
	}()

	go func() {
		wg.Wait()
		close(errCh)
	}()

	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	if len(errors) == 0 {
		if err := app.DeleteFlag(app.NotReachable); err != nil {
			log.Error().Err(err).Msg("failed to delete readonly flag")
		}
	}

	return errors
}

func verifyAtLeastOne(ctx context.Context, services []string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	var atLeastOne bool
	for _, serviceUrl := range services {
		if err := checkService(ctx, serviceUrl); err == nil {
			atLeastOne = true
			break
		}
	}

	if !atLeastOne {
		return fmt.Errorf("no realy URL is reachable")
	}

	return nil
}

func checkService(ctx context.Context, serviceUrl string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	address := parseUrl(serviceUrl)
	if address == "" {
		return fmt.Errorf("invalid URL format: %s", serviceUrl)
	}

	err := isReachable(ctx, address)
	if err != nil {
		if err := app.SetFlag(app.NotReachable); err != nil {
			log.Error().Err(err).Msg("failed to set not reachable flag")
		}
		return fmt.Errorf("%s is not reachable: %w", serviceUrl, err)
	}

	return nil
}

func parseUrl(serviceUrl string) string {
	u, err := url.Parse(serviceUrl)
	if err != nil {
		return ""
	}

	port := ":80"
	if u.Scheme == "https" || u.Scheme == "wss" {
		port = ":443"
	}

	if u.Port() == "" {
		u.Host += port
	}

	return u.Host
}

func isReachable(ctx context.Context, address string) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	return nil
}
