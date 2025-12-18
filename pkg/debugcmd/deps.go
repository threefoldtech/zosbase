package debugcmd

import (
	"context"

	"github.com/threefoldtech/zosbase/pkg"
	"github.com/threefoldtech/zosbase/pkg/gridtypes"
	"github.com/threefoldtech/zosbase/pkg/gridtypes/zos"
)

// Provision is the subset of the provision zbus interface used by debug commands.
type Provision interface {
	ListTwins(ctx context.Context) ([]uint32, error)
	List(ctx context.Context, twin uint32) ([]gridtypes.Deployment, error)
	Get(ctx context.Context, twin uint32, contract uint64) (gridtypes.Deployment, error)
	Changes(ctx context.Context, twin uint32, contract uint64) ([]gridtypes.Workload, error)
}

// VM is the subset of the vmd zbus interface used by debug commands.
type VM interface {
	Exists(ctx context.Context, id string) bool
	Inspect(ctx context.Context, id string) (pkg.VMInfo, error)
	Logs(ctx context.Context, id string) (string, error)
	LogsFull(ctx context.Context, id string) (string, error)
}

// Network is the subset of the network zbus interface used by debug commands.
type Network interface {
	Namespace(ctx context.Context, id zos.NetID) string
}

type Deps struct {
	Provision Provision
	VM        VM
	Network   Network
}
