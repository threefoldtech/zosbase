package substrategw

import (
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zosbase/pkg"
)

type substrateGateway struct {
	sub      *substrate.Substrate
	mu       sync.Mutex
	identity substrate.Identity
}

func NewSubstrateGateway(manager substrate.Manager, identity substrate.Identity) (pkg.SubstrateGateway, error) {
	sub, err := manager.Substrate()
	if err != nil {
		return nil, err
	}
	gw := &substrateGateway{
		sub:      sub,
		mu:       sync.Mutex{},
		identity: identity,
	}
	return gw, nil
}

func (g *substrateGateway) GetZosVersion() (string, error) {
	log.Debug().Str("method", "GetZosVersion").Msg("method called")

	return g.sub.GetZosVersion()
}

func (g *substrateGateway) CreateNode(node substrate.Node) (uint32, error) {
	log.Debug().
		Str("method", "CreateNode").
		Uint32("twin id", uint32(node.TwinID)).
		Uint32("farm id", uint32(node.FarmID)).
		Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.CreateNode(g.identity, node)
}

func (g *substrateGateway) CreateTwin(relay string, pk []byte) (uint32, error) {
	log.Debug().Str("method", "CreateTwin").Str("relay", relay).Str("pk", hex.EncodeToString(pk)).Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.CreateTwin(g.identity, relay, pk)
}

func (g *substrateGateway) EnsureAccount(activationURL []string, termsAndConditionsLink string, termsAndConditionsHash string) (info substrate.AccountInfo, err error) {
	log.Debug().
		Str("method", "EnsureAccount").
		Strs("activation url", activationURL).
		Str("terms and conditions link", termsAndConditionsLink).
		Str("terms and conditions hash", termsAndConditionsHash).
		Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, url := range activationURL {
		info, err = g.sub.EnsureAccount(g.identity, url, termsAndConditionsLink, termsAndConditionsHash)
		// check other activationURL only if EnsureAccount failed with ActivationServiceError
		if err == nil || !errors.As(err, &substrate.ActivationServiceError{}) {
			return
		}
		log.Debug().Str("activation url", url).Err(err).Msg("failed to EnsureAccount with ActivationServiceError")
	}
	return
}

func (g *substrateGateway) GetContract(id uint64) (result substrate.Contract, serr pkg.SubstrateError) {
	log.Trace().Str("method", "GetContract").Uint64("id", id).Msg("method called")
	contract, err := g.sub.GetContract(id)

	serr = buildSubstrateError(err)
	if err != nil {
		return
	}
	return *contract, serr
}

func (g *substrateGateway) GetContractIDByNameRegistration(name string) (result uint64, serr pkg.SubstrateError) {
	log.Trace().Str("method", "GetContractIDByNameRegistration").Str("name", name).Msg("method called")
	contractID, err := g.sub.GetContractIDByNameRegistration(name)

	serr = buildSubstrateError(err)
	return contractID, serr
}

func (g *substrateGateway) GetFarm(id uint32) (result substrate.Farm, err error) {
	log.Trace().Str("method", "GetFarm").Uint32("id", id).Msg("method called")
	farm, err := g.sub.GetFarm(id)
	if err != nil {
		return
	}
	return *farm, err
}

func (g *substrateGateway) GetNode(id uint32) (result substrate.Node, err error) {
	log.Trace().Str("method", "GetNode").Uint32("id", id).Msg("method called")
	node, err := g.sub.GetNode(id)
	if err != nil {
		return
	}
	return *node, err
}

func (g *substrateGateway) GetNodeByTwinID(twin uint32) (result uint32, serr pkg.SubstrateError) {
	log.Trace().Str("method", "GetNodeByTwinID").Uint32("twin", twin).Msg("method called")
	nodeID, err := g.sub.GetNodeByTwinID(twin)

	serr = buildSubstrateError(err)
	return nodeID, serr
}

func (g *substrateGateway) GetNodeContracts(node uint32) ([]types.U64, error) {
	log.Trace().Str("method", "GetNodeContracts").Uint32("node", node).Msg("method called")
	return g.sub.GetNodeContracts(node)
}

func (g *substrateGateway) GetNodeRentContract(node uint32) (result uint64, serr pkg.SubstrateError) {
	log.Trace().Str("method", "GetNodeRentContract").Uint32("node", node).Msg("method called")
	contractID, err := g.sub.GetNodeRentContract(node)

	serr = buildSubstrateError(err)
	return contractID, serr
}

func (g *substrateGateway) GetNodes(farmID uint32) ([]uint32, error) {
	log.Trace().Str("method", "GetNodes").Uint32("farm id", farmID).Msg("method called")
	return g.sub.GetNodes(farmID)
}

func (g *substrateGateway) GetPowerTarget(nodeID uint32) (power substrate.NodePower, err error) {
	log.Trace().Str("method", "GetPowerTarget").Uint32("node id", nodeID).Msg("method called")
	return g.sub.GetPowerTarget(nodeID)
}

func (g *substrateGateway) GetTwin(id uint32) (result substrate.Twin, err error) {
	log.Trace().Str("method", "GetTwin").Uint32("id", id).Msg("method called")
	twin, err := g.sub.GetTwin(id)
	if err != nil {
		return
	}
	return *twin, err
}

func (g *substrateGateway) GetTwinByPubKey(pk []byte) (result uint32, serr pkg.SubstrateError) {
	log.Trace().Str("method", "GetTwinByPubKey").Str("pk", hex.EncodeToString(pk)).Msg("method called")
	twinID, err := g.sub.GetTwinByPubKey(pk)

	serr = buildSubstrateError(err)
	return twinID, serr
}

func (g *substrateGateway) Report(consumptions []substrate.NruConsumption) (types.Hash, error) {
	contractIDs := make([]uint64, 0, len(consumptions))
	for _, v := range consumptions {
		contractIDs = append(contractIDs, uint64(v.ContractID))
	}
	log.Debug().Str("method", "Report").Uints64("contract ids", contractIDs).Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.Report(g.identity, consumptions)
}

func (g *substrateGateway) SetContractConsumption(resources ...substrate.ContractResources) error {
	contractIDs := make([]uint64, 0, len(resources))
	for _, v := range resources {
		contractIDs = append(contractIDs, uint64(v.ContractID))
	}
	log.Debug().Str("method", "SetContractConsumption").Uints64("contract ids", contractIDs).Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.SetContractConsumption(g.identity, resources...)
}

func (g *substrateGateway) SetNodePowerState(up bool) (hash types.Hash, err error) {
	log.Debug().Str("method", "SetNodePowerState").Bool("up", up).Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.SetNodePowerState(g.identity, up)
}

func (g *substrateGateway) UpdateNode(node substrate.Node) (uint32, error) {
	log.Debug().Str("method", "UpdateNode").Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.UpdateNode(g.identity, node)
}

func (g *substrateGateway) UpdateNodeUptimeV2(uptime uint64, timestampHint uint64) (hash types.Hash, err error) {
	log.Debug().
		Str("method", "UpdateNodeUptimeV2").
		Uint64("uptime", uptime).
		Uint64("timestamp hint", timestampHint).
		Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.UpdateNodeUptimeV2(g.identity, uptime, timestampHint)
}
func (g *substrateGateway) GetTime() (time.Time, error) {
	log.Trace().Str("method", "Time").Msg("method called")

	return g.sub.Time()
}

func buildSubstrateError(err error) (serr pkg.SubstrateError) {
	if err == nil {
		return
	}

	serr.Err = err
	serr.Code = pkg.CodeGenericError

	if errors.Is(err, substrate.ErrNotFound) {
		serr.Code = pkg.CodeNotFound
	} else if errors.Is(err, substrate.ErrBurnTransactionNotFound) {
		serr.Code = pkg.CodeBurnTransactionNotFound
	} else if errors.Is(err, substrate.ErrRefundTransactionNotFound) {
		serr.Code = pkg.CodeRefundTransactionNotFound
	} else if errors.Is(err, substrate.ErrFailedToDecode) {
		serr.Code = pkg.CodeFailedToDecode
	} else if errors.Is(err, substrate.ErrInvalidVersion) {
		serr.Code = pkg.CodeInvalidVersion
	} else if errors.Is(err, substrate.ErrUnknownVersion) {
		serr.Code = pkg.CodeUnknownVersion
	} else if errors.Is(err, substrate.ErrIsUsurped) {
		serr.Code = pkg.CodeIsUsurped
	} else if errors.Is(err, substrate.ErrAccountNotFound) {
		serr.Code = pkg.CodeAccountNotFound
	} else if errors.Is(err, substrate.ErrDepositFeeNotFound) {
		serr.Code = pkg.CodeDepositFeeNotFound
	} else if errors.Is(err, substrate.ErrMintTransactionNotFound) {
		serr.Code = pkg.CodeMintTransactionNotFound
	}
	return
}
