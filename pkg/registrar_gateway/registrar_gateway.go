package registrargw

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zosbase/pkg"
	"github.com/threefoldtech/zosbase/pkg/environment"
	"github.com/threefoldtech/zosbase/pkg/gridtypes"
)

type registrarGateway struct {
	baseURL    string
	sub        *substrate.Substrate
	httpClient *http.Client
	mu         sync.Mutex
	identity   gridtypes.Identity
	nodeID     uint64
}

var ErrorRecordNotFound = errors.New("could not fine the reqested record")

func NewRegistrarGateway(nodeID uint64, manager substrate.Manager, identity substrate.Identity) (pkg.RegistrarGateway, error) {
	client := http.DefaultClient
	env := environment.MustGet()
	sub, err := manager.Substrate()
	if err != nil {
		return &registrarGateway{}, err
	}

	gw := &registrarGateway{
		sub:        sub,
		httpClient: client,
		baseURL:    env.RegistrarURL,
		mu:         sync.Mutex{},
		identity:   identity,
	}
	return gw, nil
}

func (r *registrarGateway) GetZosVersion() (string, error) {
	log.Debug().Str("method", "GetZosVersion").Msg("method called")

	url := fmt.Sprintf("%s/v1/nodes/%d/version", r.baseURL, r.nodeID)
	resp, err := r.httpClient.Get(url)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", err
	}

	defer resp.Body.Close()

	var version gridtypes.Versioned
	err = json.NewDecoder(resp.Body).Decode(&version)

	return version.Version, err
}

func (r *registrarGateway) CreateNode(node gridtypes.Node) (uint32, error) {
	log.Debug().
		Str("method", "CreateNode").
		Uint32("twin id", uint32(node.TwinID)).
		Uint32("farm id", uint32(node.FarmID)).
		Msg("method called")

	r.mu.Lock()
	defer r.mu.Unlock()

	url := fmt.Sprintf("%s/v1/nodes", r.baseURL)

	var body bytes.Buffer
	_, err := r.httpClient.Post(url, "application/json", &body)
	if err != nil {
		return 0, err
	}

	return r.GetNodeByTwinID(uint32(node.TwinID))
}

func (g *registrarGateway) CreateTwin(relay string, pk []byte) (uint32, error) {
	log.Debug().Str("method", "CreateTwin").Str("relay", relay).Str("pk", hex.EncodeToString(pk)).Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.CreateTwin(g.identity, relay, pk)
}

func (g *registrarGateway) EnsureAccount(activationURL []string, termsAndConditionsLink string, termsAndConditionsHash string) (info substrate.AccountInfo, err error) {
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

func (g *registrarGateway) GetContract(id uint64) (result substrate.Contract, serr pkg.SubstrateError) {
	log.Trace().Str("method", "GetContract").Uint64("id", id).Msg("method called")
	contract, err := g.sub.GetContract(id)

	serr = buildSubstrateError(err)
	if err != nil {
		return
	}
	return *contract, serr
}

func (g *registrarGateway) GetContractIDByNameRegistration(name string) (result uint64, serr pkg.SubstrateError) {
	log.Trace().Str("method", "GetContractIDByNameRegistration").Str("name", name).Msg("method called")
	contractID, err := g.sub.GetContractIDByNameRegistration(name)

	serr = buildSubstrateError(err)
	return contractID, serr
}

func (r *registrarGateway) GetFarm(id uint32) (farm gridtypes.Farm, err error) {
	log.Trace().Str("method", "GetFarm").Uint32("id", id).Msg("method called")

	url := fmt.Sprintf("%s/v1/farms/%d", r.baseURL, id)

	resp, err := r.httpClient.Get(url)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		return
	}

	if resp.StatusCode == http.StatusNotFound {
		return farm, ErrorRecordNotFound
	}

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&farm)
	if err != nil {
		return
	}

	return
}

func (r *registrarGateway) GetNode(id uint32) (node gridtypes.Node, err error) {
	log.Trace().Str("method", "GetNode").Uint32("id", id).Msg("method called")
	url := fmt.Sprintf("%s/v1/nodes/%d", r.baseURL, id)

	resp, err := r.httpClient.Get(url)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		return
	}

	if resp.StatusCode == http.StatusNotFound {
		return node, ErrorRecordNotFound
	}

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&node)
	if err != nil {
		return
	}

	return node, err
}

// support is not added yet
func (r *registrarGateway) GetNodeByTwinID(twin uint32) (result uint32, err error) {
	log.Trace().Str("method", "GetNodeByTwinID").Uint32("twin", twin).Msg("method called")

	url := fmt.Sprintf("%s/v1/nodes", r.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}

	q := req.URL.Query()
	q.Add("twin_id", fmt.Sprint(twin))
	req.URL.RawQuery = q.Encode()

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		return
	}

	if resp.StatusCode == http.StatusNotFound {
		return result, ErrorRecordNotFound
	}

	defer resp.Body.Close()

	var nodes []gridtypes.Node
	err = json.NewDecoder(resp.Body).Decode(&nodes)
	if err != nil {
		return
	}
	if len(nodes) == 0 {
		return 0, fmt.Errorf("failed to get node with twin id %d", twin)
	}

	return uint32(nodes[0].NodeID), nil
}

func (g *registrarGateway) GetNodeContracts(node uint32) ([]types.U64, error) {
	log.Trace().Str("method", "GetNodeContracts").Uint32("node", node).Msg("method called")
	return g.sub.GetNodeContracts(node)
}

func (g *registrarGateway) GetNodeRentContract(node uint32) (result uint64, serr pkg.SubstrateError) {
	log.Trace().Str("method", "GetNodeRentContract").Uint32("node", node).Msg("method called")
	contractID, err := g.sub.GetNodeRentContract(node)

	serr = buildSubstrateError(err)
	return contractID, serr
}

func (r *registrarGateway) GetNodes(farmID uint32) (nodeIDs []uint32, err error) {
	log.Trace().Str("method", "GetNodes").Uint32("farm id", farmID).Msg("method called")

	url := fmt.Sprintf("%s/v1/nodes", r.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}

	q := req.URL.Query()
	q.Add("farm_id", fmt.Sprint(farmID))
	req.URL.RawQuery = q.Encode()

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var nodes []gridtypes.Node
	err = json.NewDecoder(resp.Body).Decode(&nodes)
	if err != nil {
		return
	}

	for _, node := range nodes {
		nodeIDs = append(nodeIDs, uint32(node.NodeID))
	}

	return nodeIDs, nil
}

func (g *registrarGateway) GetPowerTarget() (power substrate.NodePower, err error) {
	log.Trace().Str("method", "GetPowerTarget").Uint32("node id", uint32(g.nodeID)).Msg("method called")
	return g.sub.GetPowerTarget(uint32(g.nodeID))
}

func (g *registrarGateway) GetTwin(id uint32) (result substrate.Twin, err error) {
	log.Trace().Str("method", "GetTwin").Uint32("id", id).Msg("method called")
	twin, err := g.sub.GetTwin(id)
	if err != nil {
		return
	}
	return *twin, err
}

func (g *registrarGateway) GetTwinByPubKey(pk []byte) (result uint32, serr pkg.SubstrateError) {
	log.Trace().Str("method", "GetTwinByPubKey").Str("pk", hex.EncodeToString(pk)).Msg("method called")
	twinID, err := g.sub.GetTwinByPubKey(pk)

	serr = buildSubstrateError(err)
	return twinID, serr
}

func (r *registrarGateway) Report(consumptions []substrate.NruConsumption) (types.Hash, error) {
	contractIDs := make([]uint64, 0, len(consumptions))
	for _, v := range consumptions {
		contractIDs = append(contractIDs, uint64(v.ContractID))
	}

	log.Debug().Str("method", "Report").Uints64("contract ids", contractIDs).Msg("method called")
	r.mu.Lock()
	defer r.mu.Unlock()

	url := fmt.Sprintf("%s/v1/nodes/%d/consumption", r.baseURL, r.nodeID)

	var body bytes.Buffer
	_, err := r.httpClient.Post(url, "application/json", &body)
	if err != nil {
		return types.Hash{}, err
	}

	// I need to know what is hash to be able to respond with it
	return r.sub.Report(r.identity, consumptions)
}

func (g *registrarGateway) SetContractConsumption(resources ...substrate.ContractResources) error {
	contractIDs := make([]uint64, 0, len(resources))
	for _, v := range resources {
		contractIDs = append(contractIDs, uint64(v.ContractID))
	}
	log.Debug().Str("method", "SetContractConsumption").Uints64("contract ids", contractIDs).Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.SetContractConsumption(g.identity, resources...)
}

func (g *registrarGateway) SetNodePowerState(up bool) (hash types.Hash, err error) {
	log.Debug().Str("method", "SetNodePowerState").Bool("up", up).Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.SetNodePowerState(g.identity, up)
}

func (r *registrarGateway) UpdateNode(node gridtypes.Node) (uint32, error) {
	log.Debug().Str("method", "UpdateNode").Msg("method called")
	r.mu.Lock()
	defer r.mu.Unlock()

	// change this on supporting update node
	url := fmt.Sprintf("%s/v1/nodes/%d", r.baseURL, node.NodeID)

	var body bytes.Buffer
	_, err := r.httpClient.Post(url, "application/json", &body)
	if err != nil {
		return 0, err
	}

	return r.GetNodeByTwinID(uint32(node.TwinID))
}

func (r *registrarGateway) UpdateNodeUptimeV2(uptime uint64, timestampHint uint64) (hash types.Hash, err error) {
	log.Debug().
		Str("method", "UpdateNodeUptimeV2").
		Uint64("uptime", uptime).
		Uint64("timestamp hint", timestampHint).
		Msg("method called")
	r.mu.Lock()
	defer r.mu.Unlock()

	url := fmt.Sprintf("%s/v1/nodes/%d/uptime", r.baseURL, r.nodeID)

	var body bytes.Buffer
	_, err = r.httpClient.Post(url, "application/json", &body)
	if err != nil {
		return
	}

	// I need to know what is hash to be able to respond with it
	return r.sub.UpdateNodeUptimeV2(r.identity, uptime, timestampHint)
}

func (g *registrarGateway) GetTime() (time.Time, error) {
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
