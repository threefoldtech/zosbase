package zosapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/threefoldtech/zosbase/pkg/gridtypes"
)

type debugDeploymentsListItem struct {
	TwinID     uint32                     `json:"twin_id"`
	ContractID uint64                     `json:"contract_id"`
	Workloads  []debugDeploymentsWorkload `json:"workloads"`
}

type debugDeploymentsWorkload struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	State string `json:"state"`
}

type debugWorkloadTransaction struct {
	Seq     int                   `json:"seq"`
	Type    string                `json:"type"`
	Name    string                `json:"name"`
	Created gridtypes.Timestamp   `json:"created"`
	State   gridtypes.ResultState `json:"state"`
	Message string                `json:"message"`
}

func (g *ZosAPI) debugDeploymentsListHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var args struct {
		TwinID uint32 `json:"twin_id"`
	}
	if len(payload) != 0 {
		// optional filter
		_ = json.Unmarshal(payload, &args)
	}

	twins := []uint32{args.TwinID}
	if args.TwinID == 0 {
		var err error
		twins, err = g.provisionStub.ListTwins(ctx)
		if err != nil {
			return nil, err
		}
	}

	items := make([]debugDeploymentsListItem, 0)
	for _, twin := range twins {
		deployments, err := g.provisionStub.List(ctx, twin)
		if err != nil {
			return nil, err
		}

		for _, deployment := range deployments {
			workloads := make([]debugDeploymentsWorkload, 0, len(deployment.Workloads))
			for _, wl := range deployment.Workloads {
				workloads = append(workloads, debugDeploymentsWorkload{
					Type:  string(wl.Type),
					Name:  string(wl.Name),
					State: string(wl.Result.State),
				})
			}

			items = append(items, debugDeploymentsListItem{
				TwinID:     deployment.TwinID,
				ContractID: deployment.ContractID,
				Workloads:  workloads,
			})
		}
	}

	return struct {
		Items []debugDeploymentsListItem `json:"items"`
	}{Items: items}, nil
}

func (g *ZosAPI) debugDeploymentGetHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var args struct {
		TwinID      uint32 `json:"twin_id"`
		ContractID  uint64 `json:"contract_id"`
		WithHistory bool   `json:"withhistory"`
	}
	if err := json.Unmarshal(payload, &args); err != nil {
		return nil, err
	}
	if args.TwinID == 0 {
		return nil, fmt.Errorf("twin_id is required")
	}
	if args.ContractID == 0 {
		return nil, fmt.Errorf("contract_id is required")
	}

	deployment, err := g.provisionStub.Get(ctx, args.TwinID, args.ContractID)
	if err != nil {
		return nil, err
	}

	if !args.WithHistory {
		return struct {
			Deployment gridtypes.Deployment `json:"deployment"`
		}{Deployment: deployment}, nil
	}

	history, err := g.provisionStub.Changes(ctx, args.TwinID, args.ContractID)
	if err != nil {
		return nil, err
	}

	transactions := make([]debugWorkloadTransaction, 0, len(history))
	for idx, wl := range history {
		transactions = append(transactions, debugWorkloadTransaction{
			Seq:     idx + 1,
			Type:    string(wl.Type),
			Name:    string(wl.Name),
			Created: wl.Result.Created,
			State:   wl.Result.State,
			Message: wl.Result.Error,
		})
	}

	return struct {
		Deployment gridtypes.Deployment       `json:"deployment"`
		History    []debugWorkloadTransaction `json:"history"`
	}{
		Deployment: deployment,
		History:    transactions,
	}, nil
}
