package api

import (
	"context"

	"github.com/threefoldtech/zosbase/pkg"
)

func (a *API) PerfSpeed(ctx context.Context) (pkg.IperfTaskResult, error) {
	return a.performanceMonitorStub.GetIperfTaskResult(ctx)
}

func (a *API) PerfHealth(ctx context.Context) (pkg.HealthTaskResult, error) {
	return a.performanceMonitorStub.GetHealthTaskResult(ctx)
}

func (a *API) PerfPublicIp(ctx context.Context) (pkg.PublicIpTaskResult, error) {
	return a.performanceMonitorStub.GetPublicIpTaskResult(ctx)
}

func (a *API) PerfBenchmark(ctx context.Context) (pkg.CpuBenchTaskResult, error) {
	return a.performanceMonitorStub.GetCpuBenchTaskResult(ctx)
}

func (a *API) PerfAll(ctx context.Context) (pkg.AllTaskResult, error) {
	return a.performanceMonitorStub.GetAllTaskResult(ctx)
}
