package api

import (
	"context"

	"github.com/threefoldtech/zosbase/pkg"
)

func (a *API) StoragePoolsHandler(ctx context.Context) ([]pkg.PoolMetrics, error) {
	return a.storageStub.Metrics(ctx)
}
