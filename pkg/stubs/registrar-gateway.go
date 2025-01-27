// GENERATED CODE
// --------------
// please do not edit manually instead use the "zbusc" to regenerate

package stubs

import (
	"context"
	types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	tfchainclientgo "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zosbase/pkg"
	gridtypes "github.com/threefoldtech/zosbase/pkg/gridtypes"
	"time"
)

type RegistrarGatewayStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewRegistrarGatewayStub(client zbus.Client) *RegistrarGatewayStub {
	return &RegistrarGatewayStub{
		client: client,
		module: "registrar-gateway",
		object: zbus.ObjectID{
			Name:    "registrar-gateway",
			Version: "0.0.1",
		},
	}
}

func (s *RegistrarGatewayStub) CreateNode(ctx context.Context, arg0 gridtypes.Node) (ret0 uint32, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "CreateNode", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) CreateTwin(ctx context.Context, arg0 string, arg1 []uint8) (ret0 uint32, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "CreateTwin", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) EnsureAccount(ctx context.Context, arg0 []string, arg1 string, arg2 string) (ret0 tfchainclientgo.AccountInfo, ret1 error) {
	args := []interface{}{arg0, arg1, arg2}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "EnsureAccount", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) GetContract(ctx context.Context, arg0 uint64) (ret0 tfchainclientgo.Contract, ret1 pkg.SubstrateError) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetContract", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	loader := zbus.Loader{
		&ret0,
		&ret1,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) GetContractIDByNameRegistration(ctx context.Context, arg0 string) (ret0 uint64, ret1 pkg.SubstrateError) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetContractIDByNameRegistration", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	loader := zbus.Loader{
		&ret0,
		&ret1,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) GetFarm(ctx context.Context, arg0 uint32) (ret0 gridtypes.Farm, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetFarm", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) GetNode(ctx context.Context, arg0 uint32) (ret0 gridtypes.Node, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetNode", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) GetNodeByTwinID(ctx context.Context, arg0 uint32) (ret0 uint32, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetNodeByTwinID", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) GetNodeContracts(ctx context.Context, arg0 uint32) (ret0 []types.U64, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetNodeContracts", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) GetNodeRentContract(ctx context.Context, arg0 uint32) (ret0 uint64, ret1 pkg.SubstrateError) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetNodeRentContract", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	loader := zbus.Loader{
		&ret0,
		&ret1,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) GetNodes(ctx context.Context, arg0 uint32) (ret0 []uint32, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetNodes", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) GetPowerTarget(ctx context.Context) (ret0 tfchainclientgo.NodePower, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetPowerTarget", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) GetTime(ctx context.Context) (ret0 time.Time, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetTime", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) GetTwin(ctx context.Context, arg0 uint32) (ret0 tfchainclientgo.Twin, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetTwin", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) GetTwinByPubKey(ctx context.Context, arg0 []uint8) (ret0 uint32, ret1 pkg.SubstrateError) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetTwinByPubKey", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	loader := zbus.Loader{
		&ret0,
		&ret1,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) GetZosVersion(ctx context.Context) (ret0 string, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetZosVersion", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) Report(ctx context.Context, arg0 []tfchainclientgo.NruConsumption) (ret0 types.Hash, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Report", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) SetContractConsumption(ctx context.Context, arg0 ...tfchainclientgo.ContractResources) (ret0 error) {
	args := []interface{}{}
	for _, argv := range arg0 {
		args = append(args, argv)
	}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "SetContractConsumption", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret0 = result.CallError()
	loader := zbus.Loader{}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) SetNodePowerState(ctx context.Context, arg0 bool) (ret0 types.Hash, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "SetNodePowerState", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) UpdateNode(ctx context.Context, arg0 gridtypes.Node) (ret0 uint32, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "UpdateNode", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *RegistrarGatewayStub) UpdateNodeUptimeV2(ctx context.Context, arg0 uint64, arg1 uint64) (ret0 types.Hash, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "UpdateNodeUptimeV2", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}
