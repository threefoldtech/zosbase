package api

import (
	"context"
	"fmt"

	"github.com/patrickmn/go-cache"
	"github.com/threefoldtech/zosbase/pkg/geoip"
)

const (
	locationCacheKey = "location"
)

func (a *API) LocationGet(ctx context.Context) (geoip.Location, error) {
	if loc, found := a.inMemCache.Get(locationCacheKey); found {
		if loc, ok := loc.(geoip.Location); ok {
			return loc, nil
		}

		return geoip.Location{}, fmt.Errorf("failed to convert cached location")
	}

	loc, err := geoip.Fetch()
	if err != nil {
		return geoip.Location{}, err
	}
	a.inMemCache.Set(locationCacheKey, loc, cache.DefaultExpiration)

	return loc, nil
}
