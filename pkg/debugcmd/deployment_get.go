package debugcmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/threefoldtech/zosbase/pkg/gridtypes"
)

type DeploymentGetRequest struct {
	TwinID      uint32 `json:"twin_id"`
	ContractID  uint64 `json:"contract_id"`
	WithHistory bool   `json:"withhistory"`
}

type WorkloadTransaction struct {
	Seq     int                   `json:"seq"`
	Type    string                `json:"type"`
	Name    string                `json:"name"`
	Created gridtypes.Timestamp   `json:"created"`
	State   gridtypes.ResultState `json:"state"`
	Message string                `json:"message"`
}

type DeploymentGetResponse struct {
	Deployment gridtypes.Deployment  `json:"deployment"`
	History    []WorkloadTransaction `json:"history,omitempty"`
}

func ParseDeploymentGetRequest(payload []byte) (DeploymentGetRequest, error) {
	var req DeploymentGetRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return req, err
	}
	return req, nil
}

func DeploymentGet(ctx context.Context, deps Deps, req DeploymentGetRequest) (DeploymentGetResponse, error) {
	if req.TwinID == 0 {
		return DeploymentGetResponse{}, fmt.Errorf("twin_id is required")
	}
	if req.ContractID == 0 {
		return DeploymentGetResponse{}, fmt.Errorf("contract_id is required")
	}

	deployment, err := deps.Provision.Get(ctx, req.TwinID, req.ContractID)
	if err != nil {
		return DeploymentGetResponse{}, err
	}
	if !req.WithHistory {
		return DeploymentGetResponse{Deployment: deployment}, nil
	}

	history, err := deps.Provision.Changes(ctx, req.TwinID, req.ContractID)
	if err != nil {
		return DeploymentGetResponse{}, err
	}

	transactions := make([]WorkloadTransaction, 0, len(history))
	for idx, wl := range history {
		transactions = append(transactions, WorkloadTransaction{
			Seq:     idx + 1,
			Type:    string(wl.Type),
			Name:    string(wl.Name),
			Created: wl.Result.Created,
			State:   wl.Result.State,
			Message: wl.Result.Error,
		})
	}

	return DeploymentGetResponse{Deployment: deployment, History: transactions}, nil
}
