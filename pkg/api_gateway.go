package pkg

import (
	"context"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

//go:generate zbusc -module api-gateway -version 0.0.1 -name api-gateway -package stubs github.com/threefoldtech/zosbase/pkg+SubstrateGateway stubs/api_gateway_stub.go

type SubstrateGateway interface {
	UpdateSubstrateGatewayConnection(ctx context.Context, manager substrate.Manager) (err error)
	CreateNode(ctx context.Context, node substrate.Node) (uint32, error)
	CreateTwin(ctx context.Context, relay string, pk []byte) (uint32, error)
	EnsureAccount(ctx context.Context, activationURL []string, termsAndConditionsLink string, termsAndConditionsHash string) (info substrate.AccountInfo, err error)
	GetContract(ctx context.Context, id uint64) (substrate.Contract, SubstrateError)
	GetContractIDByNameRegistration(ctx context.Context, name string) (uint64, SubstrateError)
	GetFarm(ctx context.Context, id uint32) (substrate.Farm, error)
	GetNode(ctx context.Context, id uint32) (substrate.Node, error)
	GetNodeByTwinID(ctx context.Context, twin uint32) (uint32, SubstrateError)
	GetNodeContracts(ctx context.Context, node uint32) ([]types.U64, error)
	GetNodeRentContract(ctx context.Context, node uint32) (uint64, SubstrateError)
	GetNodes(ctx context.Context, farmID uint32) ([]uint32, error)
	GetPowerTarget(ctx context.Context, nodeID uint32) (power substrate.NodePower, err error)
	GetTwin(ctx context.Context, id uint32) (substrate.Twin, error)
	GetTwinByPubKey(ctx context.Context, pk []byte) (uint32, SubstrateError)
	Report(ctx context.Context, consumptions []substrate.NruConsumption) (types.Hash, error)
	SetContractConsumption(ctx context.Context, resources ...substrate.ContractResources) error
	SetNodePowerState(ctx context.Context, up bool) (hash types.Hash, err error)
	UpdateNode(ctx context.Context, node substrate.Node) (uint32, error)
	UpdateNodeUptimeV2(ctx context.Context, uptime uint64, timestampHint uint64) (hash types.Hash, err error)
	GetTime(ctx context.Context) (time.Time, error)
	GetZosVersion(ctx context.Context) (string, error)
}

type SubstrateError struct {
	Err  error
	Code int
}

func (e *SubstrateError) IsError() bool {
	return e.Code != CodeNoError
}

func (e *SubstrateError) IsCode(codes ...int) bool {
	for _, code := range codes {
		if code == e.Code {
			return true
		}
	}
	return false
}

const (
	CodeGenericError = iota - 1
	CodeNoError
	CodeNotFound
	CodeBurnTransactionNotFound
	CodeRefundTransactionNotFound
	CodeFailedToDecode
	CodeInvalidVersion
	CodeUnknownVersion
	CodeIsUsurped
	CodeAccountNotFound
	CodeDepositFeeNotFound
	CodeMintTransactionNotFound
)
