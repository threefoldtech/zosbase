package zosapi

import (
	"context"

	"github.com/threefoldtech/zosbase/pkg/debugcmd"
)

func (g *ZosAPI) debugDeploymentsListHandler(ctx context.Context, payload []byte) (interface{}, error) {
	req, err := debugcmd.ParseDeploymentsListRequest(payload)
	if err != nil {
		return nil, err
	}
	return debugcmd.DeploymentsList(ctx, g.debugDeps(), req)
}

func (g *ZosAPI) debugDeploymentGetHandler(ctx context.Context, payload []byte) (interface{}, error) {
	req, err := debugcmd.ParseDeploymentGetRequest(payload)
	if err != nil {
		return nil, err
	}
	return debugcmd.DeploymentGet(ctx, g.debugDeps(), req)
}

func (g *ZosAPI) debugVMInfoHandler(ctx context.Context, payload []byte) (interface{}, error) {
	req, err := debugcmd.ParseVMInfoRequest(payload)
	if err != nil {
		return nil, err
	}
	return debugcmd.VMInfo(ctx, g.debugDeps(), req)
}

func (g *ZosAPI) debugProvisioningHealthHandler(ctx context.Context, payload []byte) (interface{}, error) {
	req, err := debugcmd.ParseProvisioningHealthRequest(payload)
	if err != nil {
		return nil, err
	}
	return debugcmd.ProvisioningHealth(ctx, g.debugDeps(), req)
}

func (g *ZosAPI) debugDeps() debugcmd.Deps {
	return debugcmd.Deps{
		Provision: g.provisionStub,
		VM:        g.vmStub,
		Network:   g.networkerStub,
	}
}
