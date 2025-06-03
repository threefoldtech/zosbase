package api

import (
	"context"
	"errors"

	"github.com/threefoldtech/tfgrid-sdk-go/messenger"
	"github.com/threefoldtech/zosbase/pkg/gridtypes"
)

func (a *API) DeploymentDeployHandler(ctx context.Context, deployment gridtypes.Deployment) error {
	twinID, ok := ctx.Value(messenger.TwinIdContextKey).(uint32)
	if !ok {
		return errors.New("could not get twin_id from context")
	}
	return a.provisionStub.CreateOrUpdate(ctx, twinID, deployment, false)
}

func (a *API) DeploymentUpdateHandler(ctx context.Context, deployment gridtypes.Deployment) error {
	twinID, ok := ctx.Value(messenger.TwinIdContextKey).(uint32)
	if !ok {
		return errors.New("could not get twin_id from context")
	}
	return a.provisionStub.CreateOrUpdate(ctx, twinID, deployment, true)
}

func (a *API) DeploymentDeleteHandler(ctx context.Context, contractID uint64) error {
	return errors.New("deletion over the api is disabled, please cancel your contract instead")
}

func (a *API) DeploymentGetHandler(ctx context.Context, contractID uint64) (gridtypes.Deployment, error) {
	twinID, ok := ctx.Value(messenger.TwinIdContextKey).(uint32)
	if !ok {
		return gridtypes.Deployment{}, errors.New("could not get twin_id from context")
	}
	return a.provisionStub.Get(ctx, twinID, contractID)
}

func (a *API) DeploymentListHandler(ctx context.Context) ([]gridtypes.Deployment, error) {
	twinID, ok := ctx.Value(messenger.TwinIdContextKey).(uint32)
	if !ok {
		return nil, errors.New("could not get twin_id from context")
	}
	return a.provisionStub.List(ctx, twinID)
}

func (a *API) DeploymentChangesHandler(ctx context.Context, contractID uint64) ([]gridtypes.Workload, error) {
	twinID, ok := ctx.Value(messenger.TwinIdContextKey).(uint32)
	if !ok {
		return nil, errors.New("could not get twin_id from context")
	}
	return a.provisionStub.Changes(ctx, twinID, contractID)
}
