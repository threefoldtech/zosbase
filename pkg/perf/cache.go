package perf

import (
	"encoding/json"

	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zosbase/pkg"
)

func (pm *PerformanceMonitor) GetIperfTaskResult() (pkg.IperfTaskResult, error) {
	report, err := pm.get(iperfTaskName)
	if err != nil {
		return pkg.IperfTaskResult{}, errors.Wrap(err, "failed to get iperf task result")
	}

	var result pkg.IperfTaskResult
	if err := json.Unmarshal(report, &result); err != nil {
		return pkg.IperfTaskResult{}, errors.Wrap(err, "failed to unmarshal iperf task result")
	}

	return result, nil
}

func (pm *PerformanceMonitor) GetHealthTaskResult() (pkg.HealthTaskResult, error) {
	report, err := pm.get(healthCheckTaskName)
	if err != nil {
		return pkg.HealthTaskResult{}, errors.Wrap(err, "failed to get health check task result")
	}

	var result pkg.HealthTaskResult
	if err := json.Unmarshal(report, &result); err != nil {
		return pkg.HealthTaskResult{}, errors.Wrap(err, "failed to unmarshal health check task result")
	}
	return result, nil
}

func (pm *PerformanceMonitor) GetPublicIpTaskResult() (pkg.PublicIpTaskResult, error) {
	report, err := pm.get(publicIpTaskName)
	if err != nil {
		return pkg.PublicIpTaskResult{}, errors.Wrap(err, "failed to get public IP task result")
	}

	var result pkg.PublicIpTaskResult
	if err := json.Unmarshal(report, &result); err != nil {
		return pkg.PublicIpTaskResult{}, errors.Wrap(err, "failed to unmarshal public IP task result")
	}
	return result, nil
}

func (pm *PerformanceMonitor) GetCpuBenchTaskResult() (pkg.CpuBenchTaskResult, error) {
	var result pkg.CpuBenchTaskResult
	report, err := pm.get(cpuBenchmarkTaskName)
	if err != nil {
		return pkg.CpuBenchTaskResult{}, errors.Wrap(err, "failed to get CPU benchmark task result")
	}

	if err := json.Unmarshal(report, &result); err != nil {
		return pkg.CpuBenchTaskResult{}, errors.Wrap(err, "failed to unmarshal CPU benchmark task result")
	}
	return result, nil
}

func (pm *PerformanceMonitor) GetAllTaskResult() (pkg.AllTaskResult, error) {
	var results pkg.AllTaskResult

	cpuResult, err := pm.GetCpuBenchTaskResult()
	if err != nil {
		return pkg.AllTaskResult{}, errors.Wrap(err, "failed to get CPU benchmark result")
	}
	results.CpuBenchmark = cpuResult

	healthResult, err := pm.GetHealthTaskResult()
	if err != nil {
		return pkg.AllTaskResult{}, errors.Wrap(err, "failed to get health check result")
	}
	results.HealthCheck = healthResult

	iperfResult, err := pm.GetIperfTaskResult()
	if err != nil {
		return pkg.AllTaskResult{}, errors.Wrap(err, "failed to get iperf result")
	}
	results.Iperf = iperfResult

	publicIpResult, err := pm.GetPublicIpTaskResult()
	if err != nil {
		return pkg.AllTaskResult{}, errors.Wrap(err, "failed to get public IP result")
	}
	results.PublicIp = publicIpResult

	return results, nil
}

// DEPRECATED

// get directly gets result for some key
func get(conn redis.Conn, key string) (pkg.TaskResult, error) {
	var res pkg.TaskResult

	data, err := conn.Do("GET", key)
	if err != nil {
		return res, errors.Wrap(err, "failed to get the result")
	}

	if data == nil {
		return res, ErrResultNotFound
	}

	err = json.Unmarshal(data.([]byte), &res)
	if err != nil {
		return res, errors.Wrap(err, "failed to unmarshal data from json")
	}

	return res, nil
}

// Get gets data from redis
func (pm *PerformanceMonitor) Get(taskName string) (pkg.TaskResult, error) {
	conn := pm.pool.Get()
	defer conn.Close()
	return get(conn, generatePerfKey(taskName))
}

// GetAll gets the results for all the tests with moduleName as prefix
func (pm *PerformanceMonitor) GetAll() ([]pkg.TaskResult, error) {
	var res []pkg.TaskResult

	conn := pm.pool.Get()
	defer conn.Close()

	var keys []string

	cursor := 0
	for {
		values, err := redis.Values(conn.Do("SCAN", cursor, "MATCH", generatePerfKey("*")))
		if err != nil {
			return nil, err
		}

		_, err = redis.Scan(values, &cursor, &keys)
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			result, err := get(conn, key)
			if err != nil {
				continue
			}
			res = append(res, result)
		}

		if cursor == 0 {
			break
		}

	}
	return res, nil
}
