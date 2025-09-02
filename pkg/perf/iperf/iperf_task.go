package iperf

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/showwin/speedtest-go/speedtest"
	"github.com/threefoldtech/zosbase/pkg/environment"
	"github.com/threefoldtech/zosbase/pkg/network/iperf"
	"github.com/threefoldtech/zosbase/pkg/perf"
	execwrapper "github.com/threefoldtech/zosbase/pkg/perf/exec_wrapper"
	"github.com/threefoldtech/zosbase/pkg/perf/graphql"
)

const (
	maxRetries      = 3
	initialInterval = 5 * time.Minute
	maxInterval     = 20 * time.Minute
	maxElapsedTime  = time.Duration(maxRetries) * maxInterval
	iperfTimeout    = 30 * time.Second

	errServerBusy = "the server is busy running a test. try again later"
)

// IperfTest for iperf tcp/udp tests
type IperfTest struct {
	// Optional dependencies for testing
	graphqlClient GraphQLClient
	execWrapper   execwrapper.ExecWrapper
}

// IperfResult for iperf test results
type IperfResult struct {
	UploadSpeed   float64               `json:"upload_speed"`   // in bit/sec
	DownloadSpeed float64               `json:"download_speed"` // in bit/sec
	NodeID        uint32                `json:"node_id"`
	NodeIpv4      string                `json:"node_ip"`
	TestType      string                `json:"test_type"`
	Error         string                `json:"error"`
	CpuReport     CPUUtilizationPercent `json:"cpu_report"`
}

// NewTask creates a new iperf test
func NewTask() perf.Task {
	// because go-iperf left tmp directories with perf binary in it each time
	// the task had run
	matches, _ := filepath.Glob("/tmp/goiperf*")
	for _, match := range matches {
		os.RemoveAll(match)
	}
	return &IperfTest{}
}

// ID returns the ID of the tcp task
func (t *IperfTest) ID() string {
	return "iperf"
}

// Cron returns the schedule for the tcp task
func (t *IperfTest) Cron() string {
	return "0 0 */6 * * *"
}

// Description returns the task description
func (t *IperfTest) Description() string {
	return "Test public nodes network performance with both UDP and TCP over IPv4 and IPv6"
}

// Jitter returns the max number of seconds the job can sleep before actual execution.
func (t *IperfTest) Jitter() uint32 {
	return 20 * 60
}

// Run runs the tcp test and returns the result
func (t *IperfTest) Run(ctx context.Context) (interface{}, error) {
	var g GraphQLClient
	var err error

	if t.graphqlClient != nil {
		g = t.graphqlClient
	} else {
		env := environment.MustGet()
		graphqlClient, err := graphql.NewGraphQl(env.GraphQL...)
		if err != nil {
			return nil, err
		}
		g = &graphqlClient
	}

	// get public up nodes
	freeFarmNodes, err := g.GetUpNodes(ctx, 0, 1, 0, true, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list freefarm nodes from graphql")
	}

	nodes, err := g.GetUpNodes(ctx, 12, 0, 1, true, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list random nodes from graphql")
	}

	nodes = append(nodes, freeFarmNodes...)

	if t.execWrapper != nil {
		execWrap := t.execWrapper
		_, err = execWrap.LookPath("iperf")
		if err != nil {
			return nil, err
		}
	} else {
		_, err = exec.LookPath("iperf")
		if err != nil {
			return nil, err
		}
	}

	var results []IperfResult
	for _, node := range nodes {
		clientIP, _, err := net.ParseCIDR(node.PublicConfig.Ipv4)
		if err != nil {
			log.Error().Err(err).Msg("failed to parse ipv4 address")
			continue
		}

		clientIPv6, _, err := net.ParseCIDR(node.PublicConfig.Ipv6)
		if err != nil {
			log.Error().Err(err).Msg("failed to parse ipv6 address")
			continue
		}

		// TCP
		res := t.runIperfTest(ctx, clientIP.String(), true)
		res.NodeID = node.NodeID
		results = append(results, res)

		res = t.runIperfTest(ctx, clientIPv6.String(), true)
		res.NodeID = node.NodeID
		results = append(results, res)

		// UDP
		res = t.runIperfTest(ctx, clientIP.String(), false)
		res.NodeID = node.NodeID
		results = append(results, res)

		res = t.runIperfTest(ctx, clientIPv6.String(), false)
		res.NodeID = node.NodeID
		results = append(results, res)
	}

	return results, nil
}

func (t *IperfTest) runIperfTest(ctx context.Context, clientIP string, tcp bool) IperfResult {
	opts := make([]string, 0)
	opts = append(opts,
		"--client", clientIP,
		"--port", fmt.Sprint(iperf.IperfPort),
		"--interval", "20",
		"--bandwidth", "0", // unlimited because udp limit is set to 1M by default
		"-R", // doing the test in reverse gives more accurate results
		"--json",
	)

	if !tcp {
		opts = append(opts, "--length", "16B", "--udp")
	}

	var execWrap execwrapper.ExecWrapper = &execwrapper.RealExecWrapper{}
	if t.execWrapper != nil {
		execWrap = t.execWrapper
	}

	var report iperfCommandOutput
	operation := func() error {
		timeoutCtx, cancel := context.WithTimeout(ctx, iperfTimeout)
		defer cancel()

		res := runIperfCommand(timeoutCtx, opts, execWrap)
		if res.Error == errServerBusy {
			return errors.New(errServerBusy)
		}

		report = res
		return nil
	}

	notify := func(err error, waitTime time.Duration) {
		log.Debug().Err(err).Stringer("retry-in", waitTime).Msg("retrying")
	}

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = initialInterval
	bo.MaxInterval = maxInterval
	bo.MaxElapsedTime = maxElapsedTime

	b := backoff.WithMaxRetries(bo, maxRetries)
	err := backoff.RetryNotify(operation, b, notify)
	if err != nil {
		return IperfResult{}
	}

	proto := "tcp"
	if !tcp {
		proto = "udp"
	}

	iperfResult := IperfResult{
		UploadSpeed:   report.End.SumSent.BitsPerSecond,
		DownloadSpeed: report.End.SumReceived.BitsPerSecond,
		CpuReport:     report.End.CPUUtilizationPercent,
		NodeIpv4:      clientIP,
		TestType:      proto,
		Error:         report.Error,
	}
	if !tcp && len(report.End.Streams) > 0 {
		iperfResult.DownloadSpeed = report.End.Streams[0].UDP.BitsPerSecond
	}
	var speedtestServer *speedtest.Server
	serverList, _ := speedtest.FetchServers()
	servers, err := serverList.FindServer([]int{})
	if err != nil && len(servers) > 0 {
		speedtestServer = servers[0]
		err = speedtestServer.DownloadTest()
		if err != nil {
			log.Error().Err(err).Msg("speedtest download test failed")
		}
		err = speedtestServer.UploadTest()
		if err != nil {
			log.Error().Err(err).Msg("speedtest upload test failed")
		}
		downloadSpeed := float64(speedtestServer.DLSpeed) * 8
		uploadSpeed := float64(speedtestServer.ULSpeed) * 8
		log.Debug().Msgf("speedtest: %f/%f  vs iperf results: %f/%f", downloadSpeed, uploadSpeed, iperfResult.DownloadSpeed, iperfResult.UploadSpeed)
		// take the best result
		if downloadSpeed > iperfResult.DownloadSpeed {
			iperfResult.DownloadSpeed = downloadSpeed
		}
		if uploadSpeed > iperfResult.UploadSpeed {
			iperfResult.UploadSpeed = uploadSpeed
		}

	}

	return iperfResult
}

func runIperfCommand(ctx context.Context, opts []string, execWrap execwrapper.ExecWrapper) iperfCommandOutput {
	output, err := execWrap.CommandContext(ctx, "iperf", opts...).CombinedOutput()
	exitErr := &exec.ExitError{}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Warn().Msg("iperf command timed out for node with public IP: " + opts[1])
		}
		if !errors.As(err, &exitErr) {
			log.Err(err).Msg("failed to run iperf")
		}

		return iperfCommandOutput{}
	}

	var report iperfCommandOutput
	if err := json.Unmarshal(output, &report); err != nil {
		log.Err(err).Msg("failed to parse iperf output")
		return iperfCommandOutput{}
	}

	return report
}
