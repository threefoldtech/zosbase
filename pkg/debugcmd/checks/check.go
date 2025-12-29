package checks

import (
	"context"

	"github.com/threefoldtech/zosbase/pkg/gridtypes"
	"github.com/threefoldtech/zosbase/pkg/gridtypes/zos"
)

type HealthCheck struct {
	Name     string                 `json:"name"`
	OK       bool                   `json:"ok"`
	Message  string                 `json:"message,omitempty"`
	Evidence map[string]interface{} `json:"evidence,omitempty"`
}

type CheckData struct {
	Twin     uint32
	Contract uint64
	Workload gridtypes.Workload
	VM       func(ctx context.Context, id string) bool
	Network  func(ctx context.Context, id zos.NetID) string
}

type NetworkCheck func(ctx context.Context, data *CheckData) HealthCheck

var NetworkChecks = []NetworkCheck{
	CheckNetworkConfig,
	CheckNetworkNamespace,
	CheckNetworkInterfaces,
	CheckNetworkBridge,
	CheckNetworkMycelium,
}

type VMCheck func(ctx context.Context, data *CheckData) HealthCheck

var VMChecks = []VMCheck{
	CheckVMConfig,
	CheckVMVMD,
	CheckVMProcess,
	CheckVMDisks,
	CheckVMVirtioFS,
}

type SystemCheck func(ctx context.Context, data *SystemProbeData) HealthCheck

var SystemProbeCheck = []SystemCheck{
	CheckSystemProbe,
}
