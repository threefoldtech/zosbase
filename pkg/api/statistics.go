package api

import (
	"context"

	"github.com/threefoldtech/zosbase/pkg"
)

func (a *API) Statistics(ctx context.Context) (pkg.Counters, error) {
	return a.statisticsStub.GetCounters(ctx)
}

func (a *API) GpuList(ctx context.Context) ([]pkg.GPUInfo, error) {
	return a.statisticsStub.ListGPUs(ctx)
}
