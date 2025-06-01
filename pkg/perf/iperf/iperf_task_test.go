package iperf

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/zosbase/pkg/perf/graphql"
	"go.uber.org/mock/gomock"
)

func TestIperfTest_Run_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGraphQL := NewMockGraphQLClient(ctrl)
	mockExec := NewMockExecWrapper(ctrl)
	mockCmd := NewMockExecCmd(ctrl)

	task := &IperfTest{
		graphqlClient: mockGraphQL,
		execWrapper:   mockExec,
	}

	testNodes := []graphql.Node{
		{
			NodeID: 123,
			PublicConfig: graphql.PublicConfig{
				Ipv4: "192.168.1.100/24",
				Ipv6: "2001:db8::1/64",
			},
		},
	}

	mockGraphQL.EXPECT().
		GetUpNodes(gomock.Any(), 0, uint32(1), uint32(0), true, true).
		Return([]graphql.Node{testNodes[0]}, nil)

	mockGraphQL.EXPECT().
		GetUpNodes(gomock.Any(), 12, uint32(0), uint32(1), true, true).
		Return([]graphql.Node{}, nil)

	mockExec.EXPECT().
		LookPath("iperf").
		Return("/usr/bin/iperf", nil)

	tcpOutput := createMockIperfOutput(false, 1000000, 2000000)
	udpOutput := createMockIperfOutput(true, 500000, 800000)

	tcpOutputBytes, _ := json.Marshal(tcpOutput)
	udpOutputBytes, _ := json.Marshal(udpOutput)

	mockExec.EXPECT().
		CommandContext(gomock.Any(), "iperf", gomock.Any()).
		Return(mockCmd).
		Times(4)

	mockCmd.EXPECT().CombinedOutput().Return(tcpOutputBytes, nil)
	mockCmd.EXPECT().CombinedOutput().Return(tcpOutputBytes, nil)
	mockCmd.EXPECT().CombinedOutput().Return(udpOutputBytes, nil)
	mockCmd.EXPECT().CombinedOutput().Return(udpOutputBytes, nil)

	ctx := context.Background()
	result, err := task.Run(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	results, ok := result.([]IperfResult)
	assert.True(t, ok)
	assert.Len(t, results, 4)

	firstResult := results[0]
	assert.Equal(t, uint32(123), firstResult.NodeID)
	assert.Equal(t, "192.168.1.100", firstResult.NodeIpv4)
	assert.Equal(t, "tcp", firstResult.TestType)
	assert.Equal(t, float64(1000000), firstResult.UploadSpeed)
	assert.Equal(t, float64(2000000), firstResult.DownloadSpeed)
}

func TestIperfTest_Run_GraphQLError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGraphQL := NewMockGraphQLClient(ctrl)

	// Create test task with injected dependencies
	task := &IperfTest{
		graphqlClient: mockGraphQL,
	}

	// Mock GraphQL error
	mockGraphQL.EXPECT().
		GetUpNodes(gomock.Any(), 0, uint32(1), uint32(0), true, true).
		Return(nil, errors.New("graphql connection failed"))

	// Execute the test
	ctx := context.Background()
	result, err := task.Run(ctx)

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to list freefarm nodes from graphql")
}

func TestIperfTest_Run_IperfNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGraphQL := NewMockGraphQLClient(ctrl)
	mockExec := NewMockExecWrapper(ctrl)

	task := &IperfTest{
		graphqlClient: mockGraphQL,
		execWrapper:   mockExec,
	}

	mockGraphQL.EXPECT().
		GetUpNodes(gomock.Any(), 0, uint32(1), uint32(0), true, true).
		Return([]graphql.Node{}, nil)

	mockGraphQL.EXPECT().
		GetUpNodes(gomock.Any(), 12, uint32(0), uint32(1), true, true).
		Return([]graphql.Node{}, nil)

	mockExec.EXPECT().
		LookPath("iperf").
		Return("", errors.New("executable file not found in $PATH"))

	ctx := context.Background()
	result, err := task.Run(ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestIperfTest_Run_InvalidIPAddress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGraphQL := NewMockGraphQLClient(ctrl)
	mockExec := NewMockExecWrapper(ctrl)

	task := &IperfTest{
		graphqlClient: mockGraphQL,
		execWrapper:   mockExec,
	}

	testNodes := []graphql.Node{
		{
			NodeID: 123,
			PublicConfig: graphql.PublicConfig{
				Ipv4: "invalid-ip",
				Ipv6: "invalid-ipv6",
			},
		},
	}

	mockGraphQL.EXPECT().
		GetUpNodes(gomock.Any(), 0, uint32(1), uint32(0), true, true).
		Return(testNodes, nil)

	mockGraphQL.EXPECT().
		GetUpNodes(gomock.Any(), 12, uint32(0), uint32(1), true, true).
		Return([]graphql.Node{}, nil)

	mockExec.EXPECT().
		LookPath("iperf").
		Return("/usr/bin/iperf", nil)

	ctx := context.Background()
	result, err := task.Run(ctx)

	assert.NoError(t, err)
	results, ok := result.([]IperfResult)
	assert.True(t, ok)
	assert.Len(t, results, 0)
}

func TestNewTask(t *testing.T) {
	task := NewTask()
	assert.NotNil(t, task)
	assert.Equal(t, "iperf", task.ID())
}

// Helper function to create mock iperf output
func createMockIperfOutput(isUDP bool, uploadSpeed, downloadSpeed float64) iperfCommandOutput {
	output := iperfCommandOutput{
		End: End{
			SumSent: Sum{
				BitsPerSecond: uploadSpeed,
			},
			SumReceived: Sum{
				BitsPerSecond: downloadSpeed,
			},
			CPUUtilizationPercent: CPUUtilizationPercent{
				HostTotal:   10.5,
				RemoteTotal: 8.2,
			},
		},
	}

	if isUDP {
		output.End.Streams = []EndStream{
			{
				UDP: UDPSum{
					BitsPerSecond: downloadSpeed,
				},
			},
		}
	}

	return output
}
