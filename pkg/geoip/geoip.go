package geoip

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosbase/pkg/environment"
)

// Location holds the result of a geoip request
type Location struct {
	Longitude   float64 `json:"longitude"`
	Latitude    float64 `json:"latitude"`
	Continent   string  `json:"continent"`
	Country     string  `json:"country_name"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city_name"`
}

var defaultHTTPClient = retryablehttp.NewClient()

// Fetch retrieves the location of the system calling this function
func Fetch() (Location, error) {
	env := environment.MustGet()
	for _, url := range env.GeoipURLs {
		l, err := getLocation(url)
		if err != nil {
			log.Err(err).Str("url", url).Msg("failed to fetch location from geoip service")
			continue
		}

		return l, nil
	}

	return Location{}, errors.New("failed to fetch location information")
}

func getLocation(geoIPService string) (Location, error) {
	l := Location{
		Longitude: 0.0,
		Latitude:  0.0,
		Continent: "Unknown",
		Country:   "Unknown",
		City:      "Unknown",
	}

	defaultHTTPClient.HTTPClient.Timeout = 10 * time.Second
	defaultHTTPClient.RetryMax = 5
	resp, err := defaultHTTPClient.Get(geoIPService)
	if err != nil {
		return l, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return l, errors.New("error fetching location")
	}

	if err := json.NewDecoder(resp.Body).Decode(&l); err != nil {
		return l, err
	}

	return l, nil
}
