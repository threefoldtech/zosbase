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

// Mock interfaces for testing
type MockNetNS struct {
	mock.Mock
}

func (m *MockNetNS) Do(toRun func(ns.NetNS) error) error {
	args := m.Called(toRun)
	if toRun != nil {
		// Execute the function to test the code path
		return toRun(m)
	}
	return args.Error(0)
}

func (m *MockNetNS) Set() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockNetNS) Path() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockNetNS) Fd() uintptr {
	args := m.Called()
	return args.Get(0).(uintptr)
}

func (m *MockNetNS) Close() error {
	args := m.Called()
	return args.Error(0)
}

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

func TestGetIPWithRoute(t *testing.T) {
	tests := []struct {
		name      string
		publicIP  substrate.PublicIP
		expectErr bool
	}{
		{
			name: "valid IPv4 public IP",
			publicIP: substrate.PublicIP{
				IP:      "192.168.1.100/24",
				Gateway: "192.168.1.1",
			},
			expectErr: false,
		},
		{
			name: "valid IPv6 public IP",
			publicIP: substrate.PublicIP{
				IP:      "2001:db8::1/64",
				Gateway: "2001:db8::ffff",
			},
			expectErr: false,
		},
		{
			name: "invalid IP format",
			publicIP: substrate.PublicIP{
				IP:      "invalid-ip",
				Gateway: "192.168.1.1",
			},
			expectErr: true,
		},
		{
			name: "invalid gateway",
			publicIP: substrate.PublicIP{
				IP:      "192.168.1.100/24",
				Gateway: "invalid-gateway",
			},
			expectErr: true,
		},
		{
			name: "empty IP",
			publicIP: substrate.PublicIP{
				IP:      "",
				Gateway: "192.168.1.1",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, ipNets, routes, err := getIPWithRoute(tt.publicIP)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, ip)
				assert.Nil(t, ipNets)
				assert.Nil(t, routes)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, ip)
			assert.Len(t, ipNets, 1)
			assert.Len(t, routes, 1)

			expectedIP, _, _ := net.ParseCIDR(tt.publicIP.IP)
			assert.True(t, ip.Equal(expectedIP))

			expectedGW := net.ParseIP(tt.publicIP.Gateway)
			assert.True(t, routes[0].Gw.Equal(expectedGW))
		})
	}
}

func TestGetRealPublicIP_NetworkAccess(t *testing.T) {
	ip, err := getRealPublicIP()
	assert.NoError(t, err)
	assert.NotNil(t, ip)
	assert.True(t, ip.To4() != nil || ip.To16() != nil)
}

func TestGetPublicIPFromSTUN_InvalidServer(t *testing.T) {
	tests := []struct {
		name       string
		stunServer string
		expectErr  bool
	}{
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
			} else {
				assert.NotNil(t, ip)
			}
		})
	}
}

func TestValidateIPs(t *testing.T) {
	tests := []struct {
		name        string
		publicIPs   []substrate.PublicIP
		mockSetup   func(*MockMacvlanInterface)
		expectError bool
	}{
		{
			name: "valid IP with matching real IP",
			publicIPs: []substrate.PublicIP{
				{
					IP:         "192.168.1.100/24",
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
		},
		{
			name: "IP with contract ID should be skipped",
			publicIPs: []substrate.PublicIP{
				{
					IP:         "192.168.1.100/24",
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
		},
		{
			name: "macvlan install failure",
			publicIPs: []substrate.PublicIP{
				{
					IP:         "192.168.1.100/24",
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
		},
		{
			name: "failed to get macvlan",
			publicIPs: []substrate.PublicIP{
				{
					IP:         "192.168.1.100/24",
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
					IP:         "192.168.1.100/24",
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

			mockMacvlan.AssertExpectations(t)
		})
	}
}
