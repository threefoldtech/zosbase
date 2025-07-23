package publicip

import (
	"net"
	"testing"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/vishvananda/netlink"
)

type MockMacvlan struct {
	mock.Mock
}

func (m *MockMacvlan) Attrs() *netlink.LinkAttrs {
	args := m.Called()
	return args.Get(0).(*netlink.LinkAttrs)
}

func (m *MockMacvlan) Type() string {
	args := m.Called()
	return args.String(0)
}

// MockMacvlanInterface is a mock implementation of MacvlanInterface
type MockMacvlanInterface struct {
	mock.Mock
}

func (m *MockMacvlanInterface) GetByName(name string) (*netlink.Macvlan, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*netlink.Macvlan), args.Error(1)
}

func (m *MockMacvlanInterface) Install(link *netlink.Macvlan, hw net.HardwareAddr, ips []*net.IPNet, routes []*netlink.Route, netns ns.NetNS) error {
	args := m.Called(link, hw, ips, routes, netns)
	return args.Error(0)
}

func TestGetRealPublicIP_NetworkAccess(t *testing.T) {
	ip, err := getRealPublicIP()
	assert.NoError(t, err)
	assert.NotNil(t, ip)
	assert.True(t, ip.To4() != nil || ip.To16() != nil)
}

func TestGetPublicIPFromSTUN(t *testing.T) {
	tests := []struct {
		name       string
		stunServer string
		expectErr  bool
	}{
		{
			name:       "valid STUN server",
			stunServer: "stun:stun.l.google.com:19302",
			expectErr:  false,
		},
		{
			name:       "invalid STUN server URL",
			stunServer: "invalid-url",
			expectErr:  true,
		},
		{
			name:       "unreachable STUN server",
			stunServer: "stun:nonexistent.server.com:19302",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, err := getPublicIPFromSTUN(tt.stunServer)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, ip)
				return
			}
			assert.True(t, ip.To4() != nil || ip.To16() != nil)
		})
	}
}

func TestValidateIPs(t *testing.T) {
	pubIp, err := getRealPublicIP()
	assert.NoError(t, err)
	pubIpStr := pubIp.String() + "/24"
	tests := []struct {
		name           string
		publicIPs      []substrate.PublicIP
		mockSetup      func(*MockMacvlanInterface)
		expectError    bool
		expectedReport map[string]IPReport
	}{
		{
			name: "valid IP with matching real IP",
			publicIPs: []substrate.PublicIP{
				{
					IP:         pubIpStr,
					Gateway:    "192.168.1.1",
					ContractID: 0,
				},
			},
			mockSetup: func(mockMacvlan *MockMacvlanInterface) {
				mockMacvlanDevice := &netlink.Macvlan{
					LinkAttrs: netlink.LinkAttrs{
						Name:  testMacvlan,
						Index: 1,
					},
				}
				mockMacvlan.On("GetByName", testMacvlan).Return(mockMacvlanDevice, nil)
				mockMacvlan.On("Install", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			expectError: false,
			expectedReport: map[string]IPReport{
				pubIpStr: {
					State: ValidState,
				},
			},
		},
		{
			name: "IP with contract ID should be skipped",
			publicIPs: []substrate.PublicIP{
				{
					IP:         pubIpStr,
					Gateway:    "192.168.1.1",
					ContractID: 123,
				},
			},
			mockSetup: func(mockMacvlan *MockMacvlanInterface) {
				mockMacvlanDevice := &netlink.Macvlan{
					LinkAttrs: netlink.LinkAttrs{
						Name:  testMacvlan,
						Index: 1,
					},
				}
				mockMacvlan.On("GetByName", testMacvlan).Return(mockMacvlanDevice, nil)
			},
			expectError: false,
			expectedReport: map[string]IPReport{
				pubIpStr: {
					State:  SkippedState,
					Reason: IPIsUsed,
				},
			},
		},
		{
			name: "invalid IP format",
			publicIPs: []substrate.PublicIP{
				{
					IP:         "invalid-ip",
					Gateway:    "192.168.1.1",
					ContractID: 0,
				},
			},
			mockSetup: func(mockMacvlan *MockMacvlanInterface) {
				mockMacvlanDevice := &netlink.Macvlan{
					LinkAttrs: netlink.LinkAttrs{
						Name:  testMacvlan,
						Index: 1,
					},
				}
				mockMacvlan.On("GetByName", testMacvlan).Return(mockMacvlanDevice, nil)
			},
			expectError: false,
			expectedReport: map[string]IPReport{
				"invalid-ip": {
					State:  InvalidState,
					Reason: PublicIPDataInvalid,
				},
			},
		},
		{
			name: "macvlan install failure",
			publicIPs: []substrate.PublicIP{
				{
					IP:         pubIpStr,
					Gateway:    "192.168.1.1",
					ContractID: 0,
				},
			},
			mockSetup: func(mockMacvlan *MockMacvlanInterface) {
				mockMacvlanDevice := &netlink.Macvlan{
					LinkAttrs: netlink.LinkAttrs{
						Name:  testMacvlan,
						Index: 1,
					},
				}
				mockMacvlan.On("GetByName", testMacvlan).Return(mockMacvlanDevice, nil)
				mockMacvlan.On("Install", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
			},
			expectError: false,
			expectedReport: map[string]IPReport{
				pubIpStr: {
					State:  InvalidState,
					Reason: PublicIPDataInvalid,
				},
			},
		},
		{
			name: "failed to get macvlan",
			publicIPs: []substrate.PublicIP{
				{
					IP:         pubIpStr,
					Gateway:    "192.168.1.1",
					ContractID: 0,
				},
			},
			mockSetup: func(mockMacvlan *MockMacvlanInterface) {
				mockMacvlan.On("GetByName", testMacvlan).Return((*netlink.Macvlan)(nil), assert.AnError)
			},
			expectError: true,
		},
		{
			name: "multiple IPs with different states",
			publicIPs: []substrate.PublicIP{
				{
					IP:         pubIpStr,
					Gateway:    "192.168.1.1",
					ContractID: 0,
				},
				{
					IP:         "192.168.1.101/24",
					Gateway:    "192.168.1.1",
					ContractID: 456,
				},
				{
					IP:         "invalid-ip",
					Gateway:    "192.168.1.1",
					ContractID: 0,
				},
			},
			mockSetup: func(mockMacvlan *MockMacvlanInterface) {
				mockMacvlanDevice := &netlink.Macvlan{
					LinkAttrs: netlink.LinkAttrs{
						Name:  testMacvlan,
						Index: 1,
					},
				}
				mockMacvlan.On("GetByName", testMacvlan).Return(mockMacvlanDevice, nil)
				mockMacvlan.On("Install", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			expectError: false,
			expectedReport: map[string]IPReport{
				pubIpStr: {
					State: ValidState,
				},
				"192.168.1.101/24": {
					State:  SkippedState,
					Reason: IPIsUsed,
				},
				"invalid-ip": {
					State:  InvalidState,
					Reason: PublicIPDataInvalid,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMacvlan := new(MockMacvlanInterface)
			tt.mockSetup(mockMacvlan)

			task := &publicIPValidationTask{}
			report, err := task.validateIPs(tt.publicIPs, mockMacvlan)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, report)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedReport, report)
			mockMacvlan.AssertExpectations(t)
		})
	}
}
