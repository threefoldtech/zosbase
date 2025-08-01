package zosapi

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosbase/pkg/capacity"
	"github.com/threefoldtech/zosbase/pkg/diagnostics"
	"github.com/threefoldtech/zosbase/pkg/environment"
	"github.com/threefoldtech/zosbase/pkg/stubs"
)

const (
	cacheDefaultExpiration = 24 * time.Hour
	cacheDefaultCleanup    = 24 * time.Hour
)

type ZosAPI struct {
	oracle                 *capacity.ResourceOracle
	versionMonitorStub     *stubs.VersionMonitorStub
	systemMonitorStub      *stubs.SystemMonitorStub
	provisionStub          *stubs.ProvisionStub
	networkerStub          *stubs.NetworkerStub
	statisticsStub         *stubs.StatisticsStub
	storageStub            *stubs.StorageModuleStub
	performanceMonitorStub *stubs.PerformanceMonitorStub
	diagnosticsManager     *diagnostics.DiagnosticsManager
	farmerID               uint32
	inMemCache             *cache.Cache
}

func NewZosAPI(manager substrate.Manager, client zbus.Client, msgBrokerCon string) (ZosAPI, error) {
	sub, err := manager.Substrate()
	if err != nil {
		return ZosAPI{}, err
	}
	defer sub.Close()
	diagnosticsManager, err := diagnostics.NewDiagnosticsManager(msgBrokerCon, client)
	if err != nil {
		return ZosAPI{}, err
	}
	storageModuleStub := stubs.NewStorageModuleStub(client)
	api := ZosAPI{
		oracle:                 capacity.NewResourceOracle(storageModuleStub),
		versionMonitorStub:     stubs.NewVersionMonitorStub(client),
		systemMonitorStub:      stubs.NewSystemMonitorStub(client),
		provisionStub:          stubs.NewProvisionStub(client),
		networkerStub:          stubs.NewNetworkerStub(client),
		statisticsStub:         stubs.NewStatisticsStub(client),
		storageStub:            storageModuleStub,
		performanceMonitorStub: stubs.NewPerformanceMonitorStub(client),
		diagnosticsManager:     diagnosticsManager,
	}
	exp := backoff.NewExponentialBackOff()
	exp.MaxInterval = 2 * time.Second
	exp.InitialInterval = 500 * time.Millisecond
	exp.MaxElapsedTime = 5 * time.Second
	var farm substrate.Farm
	err = backoff.Retry(func() error {
		id := uint32(environment.MustGet().FarmID)
		retryfarm, retryErr := sub.GetFarm(id)
		if retryErr != nil {
			log.Debug().Err(retryErr).Uint32("id", id).Msg("GetFarm failed, retrying")
			return retryErr
		}
		farm = *retryfarm
		return nil
	}, exp)
	if err != nil {
		return ZosAPI{}, fmt.Errorf("failed to get farm: %w", err)
	}

	var farmer substrate.Twin
	err = backoff.Retry(func() error {
		id := uint32(farm.TwinID)
		retryfarmer, retryErr := sub.GetTwin(id)
		if retryErr != nil {
			log.Debug().Err(retryErr).Uint32("id", id).Msg("GetTwin failed, retrying")
			return retryErr
		}
		farmer = *retryfarmer
		return nil
	}, exp)
	if err != nil {
		return ZosAPI{}, err
	}

	api.farmerID = uint32(farmer.ID)
	api.inMemCache = cache.New(cacheDefaultExpiration, cacheDefaultCleanup)
	return api, nil
}
