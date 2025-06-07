package injector

import (
	"errors"
	"testing"

	"github.com/amyasnikov/berg/internal/dto"
	"github.com/google/uuid"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
)

// Mock implementation is in mock_test.go

func TestVpnInjector_AddRoute_Ok(t *testing.T) {
	m := new(mockBgpServer)
	injector := NewVPNv4Injector(m)

	// Create path attribute (e.g., LOCAL_PREF)
	localPref := &api.LocalPrefAttribute{LocalPref: 100}
	pathAttrAny, err := anypb.New(localPref)
	require.NoError(t, err)

	// Test data
	route := dto.VPNRoute{
		Rd:           "65000:1",
		RouteTargets: []string{"65000:100", "65000:200"},
		Prefix:       "192.168.1.0",
		Prefixlen:    24,
		PathAttrs:    []*anypb.Any{pathAttrAny},
	}
	respUuid := uuid.New()

	// Verify that the correct request is passed
	m.On("AddPath", mock.Anything, mock.MatchedBy(func(req *api.AddPathRequest) bool {
		// Check that it's VPN family
		if req.Path.Family.Afi != api.Family_AFI_IP || req.Path.Family.Safi != api.Family_SAFI_MPLS_VPN {
			return false
		}

		// Check that there are path attributes (1 user provided + extended communities + next hop = 3)
		if len(req.Path.Pattrs) != 3 {
			return false
		}

		// Check LOCAL_PREF attribute (first one)
		localPrefAttr := &api.LocalPrefAttribute{}
		if err := req.Path.Pattrs[0].UnmarshalTo(localPrefAttr); err != nil {
			return false
		}
		if localPrefAttr.LocalPref != 100 {
			return false
		}

		// Check ExtendedCommunitiesAttribute (second one)
		extCommAttr := &api.ExtendedCommunitiesAttribute{}
		if err := req.Path.Pattrs[1].UnmarshalTo(extCommAttr); err != nil {
			return false
		}
		// Should have 2 communities: 65000:100 and 65000:200
		if len(extCommAttr.Communities) != 2 {
			return false
		}

		// Check NextHopAttribute (third one)
		nhAttr := &api.NextHopAttribute{}
		if err := req.Path.Pattrs[2].UnmarshalTo(nhAttr); err != nil {
			return false
		}
		if nhAttr.NextHop != "0.0.0.0" {
			return false
		}

		// Check NLRI
		nlri := &api.LabeledVPNIPAddressPrefix{}
		if err := req.Path.Nlri.UnmarshalTo(nlri); err != nil {
			return false
		}
		if nlri.Prefix != "192.168.1.0" || nlri.PrefixLen != 24 {
			return false
		}
		// Check labels (should contain single 0 label)
		if len(nlri.Labels) != 1 || nlri.Labels[0] != 0 {
			return false
		}

		return true
	})).Return(&api.AddPathResponse{Uuid: respUuid[:]}, nil)

	id, err := injector.AddRoute(route)
	require.NoError(t, err)
	require.Equal(t, respUuid, id)
	m.AssertExpectations(t)
}

func TestVpnInjector_AddRoute_InvalidRd(t *testing.T) {
	m := new(mockBgpServer)
	injector := NewVPNv4Injector(m)

	// Test data with invalid RD
	route := dto.VPNRoute{
		Rd:           "invalid-rd",
		RouteTargets: []string{"65000:100"},
		Prefix:       "192.168.1.0",
		Prefixlen:    24,
		PathAttrs:    []*anypb.Any{},
	}

	id, err := injector.AddRoute(route)
	require.Error(t, err)
	require.Equal(t, uuid.Nil, id)
	// No mock expectations should be called since RD parsing fails first
}

func TestVpnInjector_AddRoute_InvalidRouteTarget(t *testing.T) {
	m := new(mockBgpServer)
	injector := NewVPNv4Injector(m)

	// Test data with invalid route target
	route := dto.VPNRoute{
		Rd:           "65000:1",
		RouteTargets: []string{"invalid-rt"},
		Prefix:       "192.168.1.0",
		Prefixlen:    24,
		PathAttrs:    []*anypb.Any{},
	}

	id, err := injector.AddRoute(route)
	require.Error(t, err)
	require.Equal(t, uuid.Nil, id)
}

func TestVpnInjector_AddRoute_BgpServerError(t *testing.T) {
	m := new(mockBgpServer)
	injector := NewVPNv4Injector(m)

	// Test data
	route := dto.VPNRoute{
		Rd:           "65000:1",
		RouteTargets: []string{"65000:100"},
		Prefix:       "192.168.1.0",
		Prefixlen:    24,
		PathAttrs:    []*anypb.Any{},
	}

	// Mock AddPath to return an error
	m.On("AddPath", mock.Anything, mock.Anything).Return((*api.AddPathResponse)(nil), errors.New("bgp server error"))

	id, err := injector.AddRoute(route)
	require.Error(t, err)
	require.Equal(t, uuid.Nil, id)
	require.Contains(t, err.Error(), "bgp server error")
	m.AssertExpectations(t)
}

func TestVpnInjector_DelRoute_Ok(t *testing.T) {
	m := new(mockBgpServer)
	injector := NewVPNv4Injector(m)

	id := uuid.New()
	binUuid, _ := id.MarshalBinary()

	// Verify that the correct request is passed
	m.On("DeletePath", mock.Anything, mock.MatchedBy(func(req *api.DeletePathRequest) bool {
		// Check UUID matches
		if string(req.Uuid) != string(binUuid) {
			return false
		}

		// Check family is VPN
		if req.Family.Afi != api.Family_AFI_IP || req.Family.Safi != api.Family_SAFI_MPLS_VPN {
			return false
		}

		// Check path UUID matches
		if string(req.Path.Uuid) != string(binUuid) {
			return false
		}

		// Check path family is VPN
		if req.Path.Family.Afi != api.Family_AFI_IP || req.Path.Family.Safi != api.Family_SAFI_MPLS_VPN {
			return false
		}

		// Check NLRI is VPN route
		vpnRoute := &api.LabeledVPNIPAddressPrefix{}
		if err := req.Path.Nlri.UnmarshalTo(vpnRoute); err != nil {
			return false
		}

		// Check path attributes (should have NextHop)
		if len(req.Path.Pattrs) != 1 {
			return false
		}

		nhAttr := &api.NextHopAttribute{}
		if err := req.Path.Pattrs[0].UnmarshalTo(nhAttr); err != nil {
			return false
		}
		if nhAttr.NextHop != "0.0.0.0" {
			return false
		}

		return true
	})).Return(nil)

	err := injector.DelRoute(id)
	require.NoError(t, err)
	m.AssertExpectations(t)
}

func TestVpnInjector_DelRoute_Error(t *testing.T) {
	m := new(mockBgpServer)
	injector := NewVPNv4Injector(m)

	id := uuid.New()
	m.On("DeletePath", mock.Anything, mock.Anything).Return(errors.New("delete failed"))

	err := injector.DelRoute(id)
	require.Error(t, err)
	require.Contains(t, err.Error(), "delete failed")
	m.AssertExpectations(t)
}

func TestNewVPNv4Injector(t *testing.T) {
	m := new(mockBgpServer)
	injector := NewVPNv4Injector(m)

	require.NotNil(t, injector)
	require.Equal(t, api.Family_AFI_IP, injector.afi)
}
