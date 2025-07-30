package api

import (
	"errors"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosbase/pkg/capacity"
	"github.com/threefoldtech/zosbase/pkg/diagnostics"
	"github.com/threefoldtech/zosbase/pkg/stubs"
)

const (
	cacheDefaultExpiration = 24 * time.Hour
	cacheDefaultCleanup    = 24 * time.Hour
	lightMode              = "light"
)

var (
	ErrNotSupportedInLightMode = errors.New("method is not supported in light mode")
)

type API struct {
	mode                   string
	oracle                 *capacity.ResourceOracle
	versionMonitorStub     *stubs.VersionMonitorStub
	systemMonitorStub      *stubs.SystemMonitorStub
	provisionStub          *stubs.ProvisionStub
	networkerStub          *stubs.NetworkerStub
	networkerLightStub     *stubs.NetworkerLightStub
	statisticsStub         *stubs.StatisticsStub
	storageStub            *stubs.StorageModuleStub
	performanceMonitorStub *stubs.PerformanceMonitorStub
	diagnosticsManager     *diagnostics.DiagnosticsManager
	inMemCache             *cache.Cache
}

func NewAPI(client zbus.Client, msgBrokerCon string, mode string) (*API, error) {
	diagnosticsManager, err := diagnostics.NewDiagnosticsManager(msgBrokerCon, client)
	if err != nil {
		return nil, err
	}

	storageModuleStub := stubs.NewStorageModuleStub(client)

	api := &API{
		mode:                   mode,
		storageStub:            storageModuleStub,
		diagnosticsManager:     diagnosticsManager,
		oracle:                 capacity.NewResourceOracle(storageModuleStub),
		versionMonitorStub:     stubs.NewVersionMonitorStub(client),
		systemMonitorStub:      stubs.NewSystemMonitorStub(client),
		provisionStub:          stubs.NewProvisionStub(client),
		statisticsStub:         stubs.NewStatisticsStub(client),
		performanceMonitorStub: stubs.NewPerformanceMonitorStub(client),
	}

	if api.isLightMode() {
		api.networkerLightStub = stubs.NewNetworkerLightStub(client)
	} else {
		api.networkerStub = stubs.NewNetworkerStub(client)
	}

	api.inMemCache = cache.New(cacheDefaultExpiration, cacheDefaultCleanup)
	return api, nil
}

func (a *API) isLightMode() bool {
	return a.mode == lightMode
}
