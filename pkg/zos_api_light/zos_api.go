package zosapi

import (
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
)

type ZosAPI struct {
	oracle                 *capacity.ResourceOracle
	versionMonitorStub     *stubs.VersionMonitorStub
	systemMonitorStub      *stubs.SystemMonitorStub
	provisionStub          *stubs.ProvisionStub
	networkerLightStub     *stubs.NetworkerLightStub
	statisticsStub         *stubs.StatisticsStub
	storageStub            *stubs.StorageModuleStub
	performanceMonitorStub *stubs.PerformanceMonitorStub
	diagnosticsManager     *diagnostics.DiagnosticsManager
	farmerID               uint32
	inMemCache             *cache.Cache
}

func NewZosAPI(client zbus.Client, farmerID uint32, msgBrokerCon string) (ZosAPI, error) {
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
		networkerLightStub:     stubs.NewNetworkerLightStub(client),
		statisticsStub:         stubs.NewStatisticsStub(client),
		storageStub:            storageModuleStub,
		performanceMonitorStub: stubs.NewPerformanceMonitorStub(client),
		diagnosticsManager:     diagnosticsManager,
	}
	api.farmerID = farmerID
	api.inMemCache = cache.New(cacheDefaultExpiration, cacheDefaultCleanup)
	return api, nil
}
