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

func TestEvpnInjector_AddType5Route_Ok(t *testing.T) {
	m := new(mockBgpServer)
	injector := NewEvpnInjector(m)

	// Create path attribute (e.g., LOCAL_PREF)
	localPref := &api.LocalPrefAttribute{LocalPref: 100}
	pathAttrAny, err := anypb.New(localPref)
	require.NoError(t, err)

	// Test data
	route := dto.Evpn5Route{
		Rd:           "65000:1",
		RouteTargets: []string{"65000:100"},
		Prefix:       "10.0.0.0",
		Prefixlen:    24,
		Gateway:      "10.0.0.1",
		Vni:          1000,
		PathAttrs:    []*anypb.Any{pathAttrAny},
	}
	respUuid := uuid.New()

	// Verify that the correct request is passed
	m.On("AddPath", mock.Anything, mock.MatchedBy(func(req *api.AddPathRequest) bool {
		// Check that it's EVPN family
		if req.Path.Family.Afi != api.Family_AFI_L2VPN || req.Path.Family.Safi != api.Family_SAFI_EVPN {
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
		// Should have 2 communities: route target (65000:100) + encap (VXLAN)
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

		return true
	})).Return(&api.AddPathResponse{Uuid: respUuid[:]}, nil)

	id, err := injector.AddType5Route(route)
	require.NoError(t, err)
	require.Equal(t, respUuid, id)
	m.AssertExpectations(t)
}

func TestEvpnInjector_AddType5Route_Error(t *testing.T) {
	m := new(mockBgpServer)
	injector := NewEvpnInjector(m)

	// Create path attribute (e.g., LOCAL_PREF)
	localPref := &api.LocalPrefAttribute{LocalPref: 100}
	pathAttrAny, err := anypb.New(localPref)
	require.NoError(t, err)

	// Test data
	route := dto.Evpn5Route{
		Rd:           "65000:1",
		RouteTargets: []string{"65000:100"},
		Prefix:       "10.0.0.0",
		Prefixlen:    24,
		Gateway:      "10.0.0.1",
		Vni:          1000,
		PathAttrs:    []*anypb.Any{pathAttrAny},
	}

	// Mock AddPath to return an error
	m.On("AddPath", mock.Anything, mock.Anything).Return((*api.AddPathResponse)(nil), errors.New("bgp server error"))

	id, err := injector.AddType5Route(route)
	require.Error(t, err)
	require.Equal(t, uuid.Nil, id)
	require.Contains(t, err.Error(), "bgp server error")
	m.AssertExpectations(t)
}

func TestEvpnInjector_DelRoute_Ok(t *testing.T) {
	m := new(mockBgpServer)
	injector := NewEvpnInjector(m)

	id := uuid.New()
	binUuid, _ := id.MarshalBinary()

	// Verify that the correct request is passed
	m.On("DeletePath", mock.Anything, mock.MatchedBy(func(req *api.DeletePathRequest) bool {
		// Check UUID matches
		if string(req.Uuid) != string(binUuid) {
			return false
		}

		// Check family is EVPN
		if req.Family.Afi != api.Family_AFI_L2VPN || req.Family.Safi != api.Family_SAFI_EVPN {
			return false
		}

		// Check path UUID matches
		if string(req.Path.Uuid) != string(binUuid) {
			return false
		}

		// Check path family is EVPN
		if req.Path.Family.Afi != api.Family_AFI_L2VPN || req.Path.Family.Safi != api.Family_SAFI_EVPN {
			return false
		}

		// Check NLRI is EVPN IP Prefix Route
		evpnRoute := &api.EVPNIPPrefixRoute{}
		if err := req.Path.Nlri.UnmarshalTo(evpnRoute); err != nil {
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

func TestEvpnInjector_DelRoute_Error(t *testing.T) {
	m := new(mockBgpServer)
	injector := NewEvpnInjector(m)

	id := uuid.New()
	m.On("DeletePath", mock.Anything, mock.Anything).Return(errors.New("fail"))

	err := injector.DelRoute(id)
	require.Error(t, err)
	require.Contains(t, err.Error(), "fail")
	m.AssertExpectations(t)
}
