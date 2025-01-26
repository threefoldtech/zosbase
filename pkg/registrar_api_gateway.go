package pkg

import (
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zosbase/pkg/gridtypes"
)

//go:generate zbusc -module registrar-gateway -version 0.0.1 -name registrar-gateway -package stubs github.com/threefoldtech/zosbase/pkg+RegistrarGateway stubs/registrar-gateway.go

type RegistrarGateway interface {
	CreateNode(node gridtypes.Node) (uint32, error)
	CreateTwin(relay string, pk []byte) (uint32, error)
	EnsureAccount(activationURL []string, termsAndConditionsLink string, termsAndConditionsHash string) (info substrate.AccountInfo, err error)
	GetContract(id uint64) (substrate.Contract, SubstrateError)
	GetContractIDByNameRegistration(name string) (uint64, SubstrateError)
	GetFarm(id uint32) (gridtypes.Farm, error)
	GetNode(id uint32) (gridtypes.Node, error)
	GetNodeByTwinID(twin uint32) (uint32, error)
	GetNodeContracts(node uint32) ([]types.U64, error)
	GetNodeRentContract(node uint32) (uint64, SubstrateError)
	GetNodes(farmID uint32) ([]uint32, error)
	GetPowerTarget() (power substrate.NodePower, err error)
	GetTwin(id uint32) (substrate.Twin, error)
	GetTwinByPubKey(pk []byte) (uint32, SubstrateError)
	Report(consumptions []substrate.NruConsumption) (types.Hash, error)
	SetContractConsumption(resources ...substrate.ContractResources) error
	SetNodePowerState(up bool) (hash types.Hash, err error)
	UpdateNode(node gridtypes.Node) (uint32, error)
	UpdateNodeUptimeV2(timestampHint uint64) (hash types.Hash, err error)
	GetTime() (time.Time, error)
	GetZosVersion() (string, error)
}
