package perf

import (
	"encoding/json"
	"fmt"

	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zosbase/pkg"
)

const (
	moduleName           = "perf"
	healthCheckTaskName  = "healthcheck"
	iperfTaskName        = "iperf"
	publicIpTaskName     = "public-ip-validation"
	cpuBenchmarkTaskName = "cpu-benchmark"
)

var (
	ErrResultNotFound = errors.New("result not found")
)

func generatePerfKey(taskName string) string {
	return fmt.Sprintf("%s.%s", moduleName, taskName)
}

func (pm *PerformanceMonitor) exists(key string) (bool, error) {
	conn := pm.pool.Get()
	defer conn.Close()
	key = generatePerfKey(key)

	ok, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return false, errors.Wrapf(err, "error checking if key %s exists", key)
	}
	return ok, nil
}

func (pm *PerformanceMonitor) get(key string) ([]byte, error) {
	conn := pm.pool.Get()
	defer conn.Close()
	key = generatePerfKey(key)

	data, err := redis.Bytes(conn.Do("GET", key))
	if err != nil {
		if err == redis.ErrNil {
			return nil, ErrResultNotFound
		}
		return nil, errors.Wrap(err, "failed to get the result")
	}

	if data == nil {
		return nil, ErrResultNotFound
	}

	return data, nil
}

func (pm *PerformanceMonitor) set(result pkg.TaskResult) error {
	conn := pm.pool.Get()
	defer conn.Close()
	key := generatePerfKey(result.Name)

	data, err := json.Marshal(result)
	if err != nil {
		return errors.Wrap(err, "failed to marshal data to JSON")
	}

	_, err = conn.Do("SET", key, data)
	return err
}
