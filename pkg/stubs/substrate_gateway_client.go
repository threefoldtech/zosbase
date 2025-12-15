package stubs

import (
	"context"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosbase/pkg"
)

// SubstrateGatewayClient wraps the generated SubstrateGatewayStub to provide
// a cleaner API with single context parameter per method.
// This wrapper internally passes the context twice to the stub (once for RPC transport,
// once for trace ID propagation).
type SubstrateGatewayClient struct {
	stub *SubstrateGatewayStub
}

// NewSubstrateGatewayClient creates a new client wrapper
func NewSubstrateGatewayClient(client zbus.Client) *SubstrateGatewayClient {
	return &SubstrateGatewayClient{
		stub: NewSubstrateGatewayStub(client),
	}
}

// UpdateSubstrateGatewayConnection allows modules to update substrate manager
func (c *SubstrateGatewayClient) UpdateSubstrateGatewayConnection(ctx context.Context, manager substrate.Manager) error {
	return c.stub.UpdateSubstrateGatewayConnection(ctx, ctx, manager)
}

// CreateNode creates a new node on the substrate chain
func (c *SubstrateGatewayClient) CreateNode(ctx context.Context, node substrate.Node) (uint32, error) {
	return c.stub.CreateNode(ctx, ctx, node)
}

// CreateTwin creates a new twin on the substrate chain
func (c *SubstrateGatewayClient) CreateTwin(ctx context.Context, relay string, pk []byte) (uint32, error) {
	return c.stub.CreateTwin(ctx, ctx, relay, pk)
}

// EnsureAccount ensures the account exists and is activated
func (c *SubstrateGatewayClient) EnsureAccount(ctx context.Context, activationURL []string, termsAndConditionsLink string, termsAndConditionsHash string) (substrate.AccountInfo, error) {
	return c.stub.EnsureAccount(ctx, ctx, activationURL, termsAndConditionsLink, termsAndConditionsHash)
}

// GetContract retrieves a contract by ID
func (c *SubstrateGatewayClient) GetContract(ctx context.Context, id uint64) (substrate.Contract, pkg.SubstrateError) {
	return c.stub.GetContract(ctx, ctx, id)
}

// GetContractIDByNameRegistration retrieves contract ID by name
func (c *SubstrateGatewayClient) GetContractIDByNameRegistration(ctx context.Context, name string) (uint64, pkg.SubstrateError) {
	return c.stub.GetContractIDByNameRegistration(ctx, ctx, name)
}

// GetFarm retrieves a farm by ID
func (c *SubstrateGatewayClient) GetFarm(ctx context.Context, id uint32) (substrate.Farm, error) {
	return c.stub.GetFarm(ctx, ctx, id)
}

// GetNode retrieves a node by ID
func (c *SubstrateGatewayClient) GetNode(ctx context.Context, id uint32) (substrate.Node, error) {
	return c.stub.GetNode(ctx, ctx, id)
}

// GetNodeByTwinID retrieves node ID by twin ID
func (c *SubstrateGatewayClient) GetNodeByTwinID(ctx context.Context, twin uint32) (uint32, pkg.SubstrateError) {
	return c.stub.GetNodeByTwinID(ctx, ctx, twin)
}

// GetNodeContracts retrieves all contracts for a node
func (c *SubstrateGatewayClient) GetNodeContracts(ctx context.Context, node uint32) ([]types.U64, error) {
	return c.stub.GetNodeContracts(ctx, ctx, node)
}

// GetNodeRentContract retrieves the rent contract for a node
func (c *SubstrateGatewayClient) GetNodeRentContract(ctx context.Context, node uint32) (uint64, pkg.SubstrateError) {
	return c.stub.GetNodeRentContract(ctx, ctx, node)
}

// GetNodes retrieves all nodes for a farm
func (c *SubstrateGatewayClient) GetNodes(ctx context.Context, farmID uint32) ([]uint32, error) {
	return c.stub.GetNodes(ctx, ctx, farmID)
}

// GetPowerTarget retrieves the power target for a node
func (c *SubstrateGatewayClient) GetPowerTarget(ctx context.Context, nodeID uint32) (substrate.NodePower, error) {
	return c.stub.GetPowerTarget(ctx, ctx, nodeID)
}

// GetTwin retrieves a twin by ID
func (c *SubstrateGatewayClient) GetTwin(ctx context.Context, id uint32) (substrate.Twin, error) {
	return c.stub.GetTwin(ctx, ctx, id)
}

// GetTwinByPubKey retrieves twin ID by public key
func (c *SubstrateGatewayClient) GetTwinByPubKey(ctx context.Context, pk []byte) (uint32, pkg.SubstrateError) {
	return c.stub.GetTwinByPubKey(ctx, ctx, pk)
}

// Report submits resource consumption reports
func (c *SubstrateGatewayClient) Report(ctx context.Context, consumptions []substrate.NruConsumption) (types.Hash, error) {
	return c.stub.Report(ctx, ctx, consumptions)
}

// SetContractConsumption sets consumption for contracts
func (c *SubstrateGatewayClient) SetContractConsumption(ctx context.Context, resources ...substrate.ContractResources) error {
	return c.stub.SetContractConsumption(ctx, ctx, resources...)
}

// SetNodePowerState sets the power state of a node
func (c *SubstrateGatewayClient) SetNodePowerState(ctx context.Context, up bool) (types.Hash, error) {
	return c.stub.SetNodePowerState(ctx, ctx, up)
}

// UpdateNode updates node information
func (c *SubstrateGatewayClient) UpdateNode(ctx context.Context, node substrate.Node) (uint32, error) {
	return c.stub.UpdateNode(ctx, ctx, node)
}

// UpdateNodeUptimeV2 updates node uptime
func (c *SubstrateGatewayClient) UpdateNodeUptimeV2(ctx context.Context, uptime uint64, timestampHint uint64) (types.Hash, error) {
	return c.stub.UpdateNodeUptimeV2(ctx, ctx, uptime, timestampHint)
}

// GetTime retrieves the current chain time
func (c *SubstrateGatewayClient) GetTime(ctx context.Context) (time.Time, error) {
	return c.stub.GetTime(ctx, ctx)
}

// GetZosVersion retrieves the ZOS version
func (c *SubstrateGatewayClient) GetZosVersion(ctx context.Context) (string, error) {
	return c.stub.GetZosVersion(ctx, ctx)
}

// Verify that SubstrateGatewayClient implements the SubstrateGateway interface
var _ pkg.SubstrateGateway = (*SubstrateGatewayClient)(nil)
