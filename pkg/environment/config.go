package environment

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	defaultHttpTimeout = 10 * time.Second
	cacheDuration      = 6 * time.Hour
)

// cachedConfig holds a cached configuration with its timestamp
type cachedConfig struct {
	config    Config
	timestamp time.Time
}

// configCache holds the cached configuration
var (
	configCache *cachedConfig
	cacheMutex  sync.RWMutex
)

// Config is configuration set by the organization
type Config struct {
	Yggdrasil struct {
		Peers []string `json:"peers"`
	} `json:"yggdrasil"`
	Mycelium struct {
		Peers []string `json:"peers"`
	} `json:"mycelium"`
	Users struct {
		Authorized []string `json:"authorized"`
	} `json:"users"`
	AdminTwins     []uint32 `json:"admin_twins"` // list of twin IDs allowed to access debug endpoints.
	RolloutUpgrade struct {
		TestFarms []uint32 `json:"test_farms"`
	} `json:"rollout_upgrade"`
	SubstrateURL  []string `json:"substrate_urls"`
	RelaysURLs    []string `json:"relays_urls"`
	ActivationURL []string `json:"activation_urls"`
	GraphQL       []string `json:"graphql_urls"`
	KycURL        string   `json:"kyc_url"`
	RegistrarURL  string   `json:"registrar_url"`
	BinRepo       string   `json:"bin_repo"`
	GeoipURLs     []string `json:"geoip_urls"`

	HubURL   []string `json:"hub_urls"`
	V4HubURL []string `json:"v4hub_urls"`

	// we should not be supporting flist url or hub storage from zos-config until we can update them on runtime
	FlistURL     string `json:"flist_url"`
	V4FlistURL   string `json:"v4_flist_url"`
	HubStorage   string `json:"hub_storage"`
	V4HubStorage string `json:"v4_hub_storage"`
}

// Merge, updates current config with cfg merging and override config
// based on some update rules.
func (c *Config) Merge(cfg Config) {
	c.Yggdrasil.Peers = uniqueStr(append(c.Yggdrasil.Peers, cfg.Yggdrasil.Peers...))
	// sort peers for easier comparison
	sort.Strings(c.Yggdrasil.Peers)
}

// GetConfig returns extend config for current run mode
func GetConfig() (base Config, err error) {
	env, err := Get()
	if err != nil {
		return
	}
	return GetConfigForMode(env.RunningMode)
}

// GetConfig returns extend config for specific run mode
func GetConfigForMode(mode RunMode) (Config, error) {
	httpClient := retryablehttp.NewClient()
	httpClient.HTTPClient.Timeout = defaultHttpTimeout
	httpClient.RetryMax = 5

	return getConfig(mode, baseExtendedURL, httpClient.StandardClient())
}

func uniqueStr(slice []string) []string {
	keys := make(map[string]struct{})
	list := slice[:0]
	for _, entry := range slice {
		if _, exists := keys[entry]; !exists {
			keys[entry] = struct{}{}
			list = append(list, entry)
		}
	}
	return list
}

func getConfig(run RunMode, url string, httpClient *http.Client) (ext Config, err error) {
	// Check cache first
	cacheMutex.RLock()
	// Return cached config if it exists and is not expired
	if configCache != nil && time.Since(configCache.timestamp) < cacheDuration {
		log.Debug().Msg("getting zos config from cache")
		config := configCache.config
		cacheMutex.RUnlock()
		return config, nil
	}
	cacheMutex.RUnlock()
	log.Debug().Msg("zos config cache expired fetching from github")
	// Fetch new config from URL
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	u := url + fmt.Sprintf("%s.json", run)

	response, err := httpClient.Get(u)
	if err != nil {
		// If URL is not responding and we have expired cache, use it
		if configCache != nil {
			log.Warn().Err(err).Msg("failed to fetch config from github, using expired cache data")
			return configCache.config, nil
		}
		return ext, err
	}

	defer func() {
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
	}()

	if response.StatusCode != http.StatusOK {
		// If URL returns error and we have expired cache, use it
		if configCache != nil {
			log.Warn().Str("status", response.Status).Msg("github config returned error, using expired cache data")
			return configCache.config, nil
		}
		return ext, fmt.Errorf("failed to get extended config: %s", response.Status)
	}

	if err := json.NewDecoder(response.Body).Decode(&ext); err != nil {
		// If decode fails and we have expired cache, use it
		if configCache != nil {
			log.Warn().Err(err).Msg("failed to decode config from github, using expired cache data")
			return configCache.config, nil
		}
		return ext, errors.Wrap(err, "failed to decode extended settings")
	}

	// Cache the new config
	cacheMutex.Lock()
	configCache = &cachedConfig{
		config:    ext,
		timestamp: time.Now(),
	}
	cacheMutex.Unlock()

	return
}
