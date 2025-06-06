package diagnostics

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosbase/pkg/utils"
)

const (
	callTimeout    = 3 * time.Second
	testNetworkKey = "perf.healthcheck"
)

// Modules is all the registered modules on zbus
var Modules = []string{
	"storage",
	"node",
	"identityd",
	"vmd",
	"flist",
	"network",
	"container",
	"provision",
	"gateway",
	"qsfsd",
}

// ModuleStatus represents the status of a module or shows if error
type ModuleStatus struct {
	// Status holds the status of the module
	Status zbus.Status `json:"status,omitempty"`
	// Err contains any error related to the module
	Err error `json:"error,omitempty"`
}

// Diagnostics show the health of zbus modules
type Diagnostics struct {
	// SystemStatusOk is the overall system status
	SystemStatusOk bool `json:"system_status_ok"`
	// ZosModules is a list of modules with their objects and workers
	ZosModules map[string]ModuleStatus `json:"modules"`
	// Healthy is the state of the node health check
	Healthy bool `json:"healthy"`
}

// RedisPooler is a redis interface to be used in mocking redis database
type RedisPooler interface {
	Get() redis.Conn
}

type DiagnosticsManager struct {
	redisPool  RedisPooler
	zbusClient zbus.Client
}

// NewDiagnosticsManager creates a new diagnostics manager
func NewDiagnosticsManager(
	msgBrokerCon string,
	busClient zbus.Client,
) (*DiagnosticsManager, error) {
	pool, err := utils.NewRedisPool(msgBrokerCon)
	if err != nil {
		return nil, err
	}
	return &DiagnosticsManager{
		redisPool:  pool,
		zbusClient: busClient,
	}, nil
}

// GetSystemDiagnostics returns the health of the system
func (m *DiagnosticsManager) GetSystemDiagnostics(ctx context.Context) (Diagnostics, error) {
	results := Diagnostics{
		SystemStatusOk: true,
		ZosModules:     make(map[string]ModuleStatus),
	}

	var wg sync.WaitGroup
	var mut sync.Mutex
	var hasError bool

	for _, module := range Modules {
		wg.Add(1)
		go func(module string) {
			defer wg.Done()
			report := m.getModuleStatus(ctx, module)

			mut.Lock()
			defer mut.Unlock()

			results.ZosModules[module] = report

			if report.Err != nil {
				hasError = true
			}
		}(module)

	}

	wg.Wait()

	results.SystemStatusOk = !hasError
	results.Healthy = m.isHealthy()

	return results, nil
}

func (m *DiagnosticsManager) getModuleStatus(ctx context.Context, module string) ModuleStatus {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	status, err := m.zbusClient.Status(ctx, module)
	return ModuleStatus{
		Status: status,
		Err:    err,
	}
}

func (m *DiagnosticsManager) isHealthy() bool {
	conn := m.redisPool.Get()
	defer conn.Close()

	data, err := redis.Bytes(conn.Do("GET", testNetworkKey))
	if err != nil || data == nil {
		return false
	}

	var result struct {
		Result map[string][]string `json:"result"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return false
	}

	for _, errors := range result.Result {
		if len(errors) > 0 {
			return false
		}
	}

	return true
}
