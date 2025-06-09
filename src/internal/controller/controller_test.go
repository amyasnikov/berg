package controller

import (
	"errors"
	"testing"

	"github.com/amyasnikov/berg/internal/dto"
	"github.com/google/uuid"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/config/oc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/anypb"
)

// Helper function to create a simple VPN route path for testing
func createTestVPNPath() *api.Path {
	rd, _ := anypb.New(&api.RouteDistinguisherTwoOctetASN{
		Admin:    65000,
		Assigned: 100,
	})
	route := &api.LabeledVPNIPAddressPrefix{
		Rd:        rd,
		Prefix:    "10.0.0.0",
		PrefixLen: 24,
		Labels:    []uint32{1000},
	}
	nlri, _ := anypb.New(route)

	return &api.Path{
		Nlri: nlri,
		Pattrs: []*anypb.Any{
			func() *anypb.Any {
				attr, _ := anypb.New(&api.MpReachNLRIAttribute{
					NextHops: []string{"192.168.1.1"},
				})
				return attr
			}(),
		},
	}
}

// Helper function to create a simple EVPN route path for testing
func createTestEVPNPath() *api.Path {
	rd, _ := anypb.New(&api.RouteDistinguisherTwoOctetASN{
		Admin:    65000,
		Assigned: 100,
	})
	route := &api.EVPNIPPrefixRoute{
		Rd:          rd,
		Esi:         &api.EthernetSegmentIdentifier{},
		EthernetTag: 0,
		IpPrefix:    "10.0.0.0",
		IpPrefixLen: 24,
		GwAddress:   "192.168.1.1",
		Label:       1000,
	}
	nlri, _ := anypb.New(route)

	// Create route target extended community
	rtComm, _ := anypb.New(&api.TwoOctetAsSpecificExtended{
		SubType:    2, // Route Target
		Asn:        65000,
		LocalAdmin: 100,
	})
	extCommAttr, _ := anypb.New(&api.ExtendedCommunitiesAttribute{
		Communities: []*anypb.Any{rtComm},
	})

	return &api.Path{
		Nlri: nlri,
		Pattrs: []*anypb.Any{
			extCommAttr,
		},
	}
}

// Mock for evpnInjector interface
type mockEvpnInjector struct {
	mock.Mock
}

func (m *mockEvpnInjector) AddType5Route(route dto.Evpn5Route) (uuid.UUID, error) {
	args := m.Called(route)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *mockEvpnInjector) DelRoute(routeId uuid.UUID) error {
	args := m.Called(routeId)
	return args.Error(0)
}

// Mock for vpnInjector interface
type mockVpnInjector struct {
	mock.Mock
}

func (m *mockVpnInjector) AddRoute(route dto.VPNRoute) (uuid.UUID, error) {
	args := m.Called(route)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *mockVpnInjector) DelRoute(routeId uuid.UUID) error {
	args := m.Called(routeId)
	return args.Error(0)
}

func TestVPNv4Controller_HandleUpdate(t *testing.T) {
	tests := []struct {
		name             string
		path             *api.Path
		vrfExists        bool
		genRouteError    bool
		injectError      bool
		expectedError    bool
		shouldInject     bool
		hasExistingRoute bool
	}{
		{
			name:          "Successful update",
			path:          createTestVPNPath(),
			vrfExists:     true,
			expectedError: false,
			shouldInject:  true,
		},
		{
			name: "VRF not found - should not error but not inject",
			path: func() *api.Path {
				path := createTestVPNPath()
				path.Pattrs = []*anypb.Any{} // No attributes
				return path
			}(),
			vrfExists:     false,
			expectedError: false,
			shouldInject:  false,
		},
		{
			name:          "Injection error",
			path:          createTestVPNPath(),
			vrfExists:     true,
			injectError:   true,
			expectedError: true,
			shouldInject:  true,
		},
		{
			name:             "Update existing route",
			path:             createTestVPNPath(),
			vrfExists:        true,
			expectedError:    false,
			shouldInject:     true,
			hasExistingRoute: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInjector := &mockEvpnInjector{}

			vrfCfg := []oc.VrfConfig{}
			if tt.vrfExists {
				vrfCfg = append(vrfCfg, oc.VrfConfig{
					Name: "test-vrf",
					Rd:   "65000:100",
					Id:   1000,
				})
			}

			controller := NewVPNv4Controller(mockInjector, vrfCfg)

			if tt.hasExistingRoute {
				// Pre-populate with existing route (must match all fields from createTestVPNPath)
				existingRoute := vpnRoute{Rd: "65000:100", Prefix: "10.0.0.0", Prefixlen: 24, Label: 1000}
				existingUuid := uuid.New()
				controller.redistributedEvpn.Store(existingRoute, existingUuid)

				// Expect deletion of existing route
				mockInjector.On("DelRoute", existingUuid).Return(nil)
			}

			if tt.shouldInject {
				newUuid := uuid.New()
				if tt.injectError {
					mockInjector.On("AddType5Route", mock.MatchedBy(func(route dto.Evpn5Route) bool {
						// Validate route parameters
						return route.Rd == "65000:100" &&
							route.Prefix == "10.0.0.0" &&
							route.Prefixlen == 24 &&
							route.Gateway == "192.168.1.1" &&
							route.Vni == 1000 &&
							len(route.RouteTargets) == 0 // Empty since VRF has no explicit route targets
					})).Return(uuid.Nil, errors.New("injection failed"))
				} else {
					mockInjector.On("AddType5Route", mock.MatchedBy(func(route dto.Evpn5Route) bool {
						// Validate route parameters
						return route.Rd == "65000:100" &&
							route.Prefix == "10.0.0.0" &&
							route.Prefixlen == 24 &&
							route.Gateway == "192.168.1.1" &&
							route.Vni == 1000 &&
							len(route.RouteTargets) == 0 // Empty since VRF has no explicit route targets
					})).Return(newUuid, nil)
				}
			}

			err := controller.HandleUpdate(tt.path)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockInjector.AssertExpectations(t)
		})
	}
}

func TestVPNv4Controller_HandleWithdraw(t *testing.T) {
	tests := []struct {
		name          string
		path          *api.Path
		hasRoute      bool
		deleteError   bool
		expectedError bool
	}{
		{
			name:          "Successful withdrawal",
			path:          createTestVPNPath(),
			hasRoute:      true,
			expectedError: false,
		},
		{
			name:          "Route not found - should not error",
			path:          createTestVPNPath(),
			hasRoute:      false,
			expectedError: false,
		},
		{
			name:          "Delete error",
			path:          createTestVPNPath(),
			hasRoute:      true,
			deleteError:   true,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInjector := &mockEvpnInjector{}
			controller := NewVPNv4Controller(mockInjector, []oc.VrfConfig{})

			routeUuid := uuid.New()
			if tt.hasRoute {
				route := vpnRoute{Rd: "65000:100", Prefix: "10.0.0.0", Prefixlen: 24, Label: 1000}
				controller.redistributedEvpn.Store(route, routeUuid)

				if tt.deleteError {
					mockInjector.On("DelRoute", routeUuid).Return(errors.New("delete failed"))
				} else {
					mockInjector.On("DelRoute", routeUuid).Return(nil)
				}
			}

			err := controller.HandleWithdraw(tt.path)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify route was deleted from storage if it existed
			if tt.hasRoute && !tt.deleteError {
				route := vpnRoute{Rd: "65000:100", Prefix: "10.0.0.0", Prefixlen: 24, Label: 1000}
				_, exists := controller.redistributedEvpn.Load(route)
				assert.False(t, exists)
			}

			mockInjector.AssertExpectations(t)
		})
	}
}

func TestVPNv4Controller_ReloadConfig(t *testing.T) {
	tests := []struct {
		name         string
		initialVrfs  []oc.VrfConfig
		diff         dto.VrfDiff
		hasRoutes    bool
		expectedVrfs int
	}{
		{
			name: "Add new VRFs",
			initialVrfs: []oc.VrfConfig{
				{Name: "vrf1", Rd: "65000:100", Id: 1000},
			},
			diff: dto.VrfDiff{
				Created: []oc.VrfConfig{
					{
						Name:         "vrf2",
						Rd:           "65000:200",
						Id:           2000,
						ImportRtList: []string{"65000:200"},
						ExportRtList: []string{"65000:200"},
					},
				},
			},
			expectedVrfs: 2,
		},
		{
			name: "Delete VRFs and clean up routes",
			initialVrfs: []oc.VrfConfig{
				{Name: "vrf1", Rd: "65000:100", Id: 1000},
				{Name: "vrf2", Rd: "65000:200", Id: 2000},
			},
			diff: dto.VrfDiff{
				Deleted: []oc.VrfConfig{
					{Rd: "65000:200"},
				},
			},
			hasRoutes:    true,
			expectedVrfs: 1,
		},
		{
			name: "Mixed create and delete",
			initialVrfs: []oc.VrfConfig{
				{Name: "vrf1", Rd: "65000:100", Id: 1000},
			},
			diff: dto.VrfDiff{
				Created: []oc.VrfConfig{
					{Name: "vrf2", Rd: "65000:200", Id: 2000},
				},
				Deleted: []oc.VrfConfig{
					{Rd: "65000:100"},
				},
			},
			expectedVrfs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInjector := &mockEvpnInjector{}
			controller := NewVPNv4Controller(mockInjector, tt.initialVrfs)

			if tt.hasRoutes {
				// Add some routes that should be cleaned up
				for _, vrf := range tt.diff.Deleted {
					route := vpnRoute{Rd: vrf.Rd, Prefix: "10.0.0.0", Prefixlen: 24, Label: 1000}
					routeUuid := uuid.New()
					controller.redistributedEvpn.Store(route, routeUuid)
					mockInjector.On("DelRoute", routeUuid).Return(nil)
				}
			}

			err := controller.ReloadConfig(tt.diff)
			assert.NoError(t, err)

			// Verify VRF count
			actualCount := 0
			controller.rdVrfMap.Range(func(key string, value dto.Vrf) bool {
				actualCount++
				return true
			})
			assert.Equal(t, tt.expectedVrfs, actualCount)

			// Verify created VRFs exist
			for _, vrf := range tt.diff.Created {
				_, exists := controller.rdVrfMap.Load(vrf.Rd)
				assert.True(t, exists)
			}

			// Verify deleted VRFs don't exist
			for _, vrf := range tt.diff.Deleted {
				_, exists := controller.rdVrfMap.Load(vrf.Rd)
				assert.False(t, exists)
			}

			mockInjector.AssertExpectations(t)
		})
	}
}

func TestVPNv4Controller_DeleteStaleRoutes(t *testing.T) {
	mockInjector := &mockEvpnInjector{}
	controller := NewVPNv4Controller(mockInjector, []oc.VrfConfig{})

	// Add some routes
	route1 := vpnRoute{Rd: "65000:100", Prefix: "10.0.0.0", Prefixlen: 24, Label: 1000}
	route2 := vpnRoute{Rd: "65000:200", Prefix: "10.1.0.0", Prefixlen: 24, Label: 2000}
	uuid1 := uuid.New()
	uuid2 := uuid.New()

	controller.redistributedEvpn.Store(route1, uuid1)
	controller.redistributedEvpn.Store(route2, uuid2)

	// Only route1 (with RD "65000:100") should be deleted
	mockInjector.On("DelRoute", uuid1).Return(nil)

	deletedRd := []string{"65000:100"}
	err := controller.deleteStaleRoutes(deletedRd)

	assert.NoError(t, err)

	// Verify route1 was deleted but route2 remains
	_, exists1 := controller.redistributedEvpn.Load(route1)
	assert.False(t, exists1, "Route1 should be deleted")

	_, exists2 := controller.redistributedEvpn.Load(route2)
	assert.True(t, exists2, "Route2 should remain")

	mockInjector.AssertExpectations(t)
}

func TestEvpnController_HandleUpdate(t *testing.T) {
	tests := []struct {
		name             string
		path             *api.Path
		hasMatchingRT    bool
		genRouteError    bool
		injectError      bool
		expectedError    bool
		shouldInject     bool
		hasExistingRoute bool
	}{
		{
			name:          "Successful update",
			path:          createTestEVPNPath(),
			hasMatchingRT: true,
			expectedError: false,
			shouldInject:  true,
		},
		{
			name:          "No matching route targets - should not error but not inject",
			path:          createTestEVPNPath(),
			hasMatchingRT: false,
			expectedError: false,
			shouldInject:  false,
		},
		{
			name:          "Injection error",
			path:          createTestEVPNPath(),
			hasMatchingRT: true,
			injectError:   true,
			expectedError: true,
			shouldInject:  true,
		},
		{
			name:             "Update existing route",
			path:             createTestEVPNPath(),
			hasMatchingRT:    true,
			expectedError:    false,
			shouldInject:     true,
			hasExistingRoute: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInjector := &mockVpnInjector{}

			vrfCfg := []oc.VrfConfig{}
			if tt.hasMatchingRT {
				vrfCfg = append(vrfCfg, oc.VrfConfig{
					Name:         "test-vrf",
					Rd:           "65000:100",
					Id:           1000,
					ImportRtList: []string{"65000:100"},
				})
			}

			// Create a mock function for listEvpnRoutes
			listEvpnRoutes := func() <-chan EvpnRouteWithPattrs {
				ch := make(chan EvpnRouteWithPattrs)
				close(ch)
				return ch
			}

			controller := NewEvpnController(mockInjector, vrfCfg, listEvpnRoutes)

			if tt.hasExistingRoute {
				// Pre-populate with existing route (must match all fields from createTestEVPNPath)
				existingRoute := evpnRoute{
					Rd:          "65000:100",
					Prefix:      "10.0.0.0",
					Prefixlen:   24,
					Gateway:     "192.168.1.1",
					Label:       1000,
					EthernetTag: 0,
					Esi:         "",
				}
				existingUuid := uuid.New()
				controller.redistributedStorage.Store(existingRoute, []string{"65000:100"}, existingUuid)

				// Expect deletion of existing route
				mockInjector.On("DelRoute", existingUuid).Return(nil)
			}

			if tt.shouldInject {
				newUuid := uuid.New()
				if tt.injectError {
					mockInjector.On("AddRoute", mock.MatchedBy(func(route dto.VPNRoute) bool {
						// Validate route parameters
						return route.Rd == "65000:100" &&
							route.Prefix == "10.0.0.0" &&
							route.Prefixlen == 24 &&
							len(route.RouteTargets) == 1 &&
							route.RouteTargets[0] == "65000:100"
					})).Return(uuid.Nil, errors.New("injection failed"))
				} else {
					mockInjector.On("AddRoute", mock.MatchedBy(func(route dto.VPNRoute) bool {
						// Validate route parameters
						return route.Rd == "65000:100" &&
							route.Prefix == "10.0.0.0" &&
							route.Prefixlen == 24 &&
							len(route.RouteTargets) == 1 &&
							route.RouteTargets[0] == "65000:100"
					})).Return(newUuid, nil)
				}
			}

			err := controller.HandleUpdate(tt.path)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockInjector.AssertExpectations(t)
		})
	}
}

func TestEvpnController_HandleWithdraw(t *testing.T) {
	tests := []struct {
		name          string
		path          *api.Path
		hasRoute      bool
		deleteError   bool
		expectedError bool
	}{
		{
			name:          "Successful withdrawal",
			path:          createTestEVPNPath(),
			hasRoute:      true,
			expectedError: false,
		},
		{
			name:          "Route not found - should not error",
			path:          createTestEVPNPath(),
			hasRoute:      false,
			expectedError: false,
		},
		{
			name:          "Delete error",
			path:          createTestEVPNPath(),
			hasRoute:      true,
			deleteError:   true,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInjector := &mockVpnInjector{}

			// Create a mock function for listEvpnRoutes
			listEvpnRoutes := func() <-chan EvpnRouteWithPattrs {
				ch := make(chan EvpnRouteWithPattrs)
				close(ch)
				return ch
			}

			controller := NewEvpnController(mockInjector, []oc.VrfConfig{}, listEvpnRoutes)

			routeUuid := uuid.New()
			if tt.hasRoute {
				route := evpnRoute{
					Rd:          "65000:100",
					Prefix:      "10.0.0.0",
					Prefixlen:   24,
					Gateway:     "192.168.1.1",
					Label:       1000,
					EthernetTag: 0,
					Esi:         "",
				}
				controller.redistributedStorage.Store(route, []string{"65000:100"}, routeUuid)

				if tt.deleteError {
					mockInjector.On("DelRoute", routeUuid).Return(errors.New("delete failed"))
				} else {
					mockInjector.On("DelRoute", routeUuid).Return(nil)
				}
			}

			err := controller.HandleWithdraw(tt.path)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockInjector.AssertExpectations(t)
		})
	}
}

func TestEvpnController_ReloadConfig(t *testing.T) {
	tests := []struct {
		name        string
		initialVrfs []oc.VrfConfig
		diff        dto.VrfDiff
		hasRoutes   bool
		expectedRTs int
	}{
		{
			name: "Add new VRFs",
			initialVrfs: []oc.VrfConfig{
				{Name: "vrf1", Rd: "65000:100", Id: 1000, ImportRtList: []string{"65000:100"}},
			},
			diff: dto.VrfDiff{
				Created: []oc.VrfConfig{
					{Name: "vrf2", Rd: "65000:200", Id: 2000, ImportRtList: []string{"65000:200"}},
				},
			},
			expectedRTs: 2,
		},
		{
			name: "Delete VRFs and clean up routes",
			initialVrfs: []oc.VrfConfig{
				{Name: "vrf1", Rd: "65000:100", Id: 1000, ImportRtList: []string{"65000:100"}},
				{Name: "vrf2", Rd: "65000:200", Id: 2000, ImportRtList: []string{"65000:200"}},
			},
			diff: dto.VrfDiff{
				Deleted: []oc.VrfConfig{
					{ImportRtList: []string{"65000:200"}},
				},
			},
			hasRoutes:   true,
			expectedRTs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInjector := &mockVpnInjector{}

			// Create a mock function for listEvpnRoutes
			listEvpnRoutes := func() <-chan EvpnRouteWithPattrs {
				ch := make(chan EvpnRouteWithPattrs)
				close(ch)
				return ch
			}

			controller := NewEvpnController(mockInjector, tt.initialVrfs, listEvpnRoutes)

			if tt.hasRoutes {
				// Add some routes that should be cleaned up
				for _, vrf := range tt.diff.Deleted {
					for _, rt := range vrf.ImportRtList {
						route := evpnRoute{Rd: "65000:200", Prefix: "10.0.0.0", Prefixlen: 24}
						routeUuid := uuid.New()
						controller.redistributedStorage.Store(route, []string{rt}, routeUuid)
						mockInjector.On("DelRoute", routeUuid).Return(nil)
					}
				}
			}

			err := controller.ReloadConfig(tt.diff)
			assert.NoError(t, err)

			// Verify route target count
			actualRTCount := controller.existingRT.Cardinality()
			assert.Equal(t, tt.expectedRTs, actualRTCount)

			mockInjector.AssertExpectations(t)
		})
	}
}
