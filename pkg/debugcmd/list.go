package debugcmd

import (
	"context"
	"encoding/json"
)

type ListRequest struct {
	TwinID uint32 `json:"twin_id"`
}

type ListWorkload struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	State string `json:"state"`
}

type ListDeployment struct {
	TwinID     uint32         `json:"twin_id"`
	ContractID uint64         `json:"contract_id"`
	Workloads  []ListWorkload `json:"workloads"`
}

type ListResponse struct {
	Deployments []ListDeployment `json:"deployments"`
}

func ParseListRequest(payload []byte) (ListRequest, error) {
	var req ListRequest
	if len(payload) == 0 {
		return req, nil
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return req, err
	}
	return req, nil
}

func List(ctx context.Context, deps Deps, req ListRequest) (ListResponse, error) {
	twins := []uint32{req.TwinID}
	if req.TwinID == 0 {
		var err error
		twins, err = deps.Provision.ListTwins(ctx)
		if err != nil {
			return ListResponse{}, err
		}
	}

	deployments := make([]ListDeployment, 0)
	for _, twin := range twins {
		deploymentList, err := deps.Provision.List(ctx, twin)
		if err != nil {
			return ListResponse{}, err
		}
		for _, d := range deploymentList {
			workloads := make([]ListWorkload, 0, len(d.Workloads))
			for _, wl := range d.Workloads {
				workloads = append(workloads, ListWorkload{
					Type:  string(wl.Type),
					Name:  string(wl.Name),
					State: string(wl.Result.State),
				})
			}
			deployments = append(deployments, ListDeployment{
				TwinID:     d.TwinID,
				ContractID: d.ContractID,
				Workloads:  workloads,
			})
		}
	}

	return ListResponse{Deployments: deployments}, nil
}

