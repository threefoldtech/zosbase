package debugcmd

import (
	"context"
	"encoding/json"
)

type DeploymentsListRequest struct {
	TwinID uint32 `json:"twin_id"`
}

type DeploymentsListWorkload struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	State string `json:"state"`
}

type DeploymentsListItem struct {
	TwinID     uint32                    `json:"twin_id"`
	ContractID uint64                    `json:"contract_id"`
	Workloads  []DeploymentsListWorkload `json:"workloads"`
}

type DeploymentsListResponse struct {
	Items []DeploymentsListItem `json:"items"`
}

func ParseDeploymentsListRequest(payload []byte) (DeploymentsListRequest, error) {
	var req DeploymentsListRequest
	if len(payload) == 0 {
		return req, nil
	}
	// optional payload
	_ = json.Unmarshal(payload, &req)
	return req, nil
}

func DeploymentsList(ctx context.Context, deps Deps, req DeploymentsListRequest) (DeploymentsListResponse, error) {
	twins := []uint32{req.TwinID}
	if req.TwinID == 0 {
		var err error
		twins, err = deps.Provision.ListTwins(ctx)
		if err != nil {
			return DeploymentsListResponse{}, err
		}
	}

	items := make([]DeploymentsListItem, 0)
	for _, twin := range twins {
		deployments, err := deps.Provision.List(ctx, twin)
		if err != nil {
			return DeploymentsListResponse{}, err
		}
		for _, d := range deployments {
			workloads := make([]DeploymentsListWorkload, 0, len(d.Workloads))
			for _, wl := range d.Workloads {
				workloads = append(workloads, DeploymentsListWorkload{
					Type:  string(wl.Type),
					Name:  string(wl.Name),
					State: string(wl.Result.State),
				})
			}
			items = append(items, DeploymentsListItem{
				TwinID:     d.TwinID,
				ContractID: d.ContractID,
				Workloads:  workloads,
			})
		}
	}

	return DeploymentsListResponse{Items: items}, nil
}
