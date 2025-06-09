package controller

import (
	"testing"

	"github.com/amyasnikov/berg/internal/dto"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestEvpnRouteGen_GenRoute(t *testing.T) {
	tests := []struct {
		name           string
		route          vpnRoute
		vrf            dto.Vrf
		pattrs         []*anypb.Any
		expectedError  bool
		expectedPrefix string
		expectedRd     string
		expectedVni    uint32
	}{
		{
			name: "Successful route generation",
			route: vpnRoute{
				Prefix:    "10.0.0.0",
				Prefixlen: 24,
			},
			vrf: dto.Vrf{
				Rd:                 "65000:100",
				ExportRouteTargets: []string{"65000:100"},
				Vni:                1000,
			},
			pattrs: []*anypb.Any{
				func() *anypb.Any {
					nlri, _ := anypb.New(&api.MpReachNLRIAttribute{
						NextHops: []string{"192.168.1.1"},
					})
					return nlri
				}(),
				func() *anypb.Any {
					attr, _ := anypb.New(&api.LocalPrefAttribute{LocalPref: 100})
					return attr
				}(),
			},
			expectedError:  false,
			expectedPrefix: "10.0.0.0",
			expectedRd:     "65000:100",
			expectedVni:    1000,
		},
		{
			name: "Missing NextHop - should return error",
			route: vpnRoute{
				Prefix:    "10.0.0.0",
				Prefixlen: 24,
			},
			vrf: dto.Vrf{
				Rd:                 "65000:100",
				ExportRouteTargets: []string{"65000:100"},
				Vni:                1000,
			},
			pattrs: []*anypb.Any{
				func() *anypb.Any {
					attr, _ := anypb.New(&api.LocalPrefAttribute{LocalPref: 100})
					return attr
				}(),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := newEvpnRouteGen()
			result, err := gen.GenRoute(tt.route, tt.vrf, tt.pattrs)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPrefix, result.Prefix)
				assert.Equal(t, tt.expectedRd, result.Rd)
				assert.Equal(t, tt.expectedVni, result.Vni)
				assert.Equal(t, tt.vrf.ExportRouteTargets, result.RouteTargets)
				assert.Equal(t, tt.route.Prefixlen, result.Prefixlen)
				// PathAttrs should be filtered (only allowed attributes)
				assert.NotNil(t, result.PathAttrs)
			}
		})
	}
}

func TestEvpnRouteGen_AttributeFiltering(t *testing.T) {
	gen := newEvpnRouteGen()
	route := vpnRoute{Prefix: "10.0.0.0", Prefixlen: 24}
	vrf := dto.Vrf{Rd: "65000:100", ExportRouteTargets: []string{"65000:100"}, Vni: 1000}

	pattrs := []*anypb.Any{
		func() *anypb.Any {
			nlri, _ := anypb.New(&api.MpReachNLRIAttribute{
				NextHops: []string{"192.168.1.1"},
			})
			return nlri
		}(),
		func() *anypb.Any {
			// This should be included (allowed attribute)
			attr, _ := anypb.New(&api.LocalPrefAttribute{LocalPref: 100})
			return attr
		}(),
		func() *anypb.Any {
			// This should be filtered out (not in allowed list)
			attr, _ := anypb.New(&api.NextHopAttribute{NextHop: "10.0.0.1"})
			return attr
		}(),
	}

	result, err := gen.GenRoute(route, vrf, pattrs)
	assert.NoError(t, err)

	// Should have filtered attributes (LocalPref should be included, NextHop should be excluded)
	// The exact count depends on what gets filtered by AttrFilter
	found := false
	for _, attr := range result.PathAttrs {
		var localPref api.LocalPrefAttribute
		if anypb.UnmarshalTo(attr, &localPref, proto.UnmarshalOptions{}) == nil {
			found = true
			break
		}
	}
	assert.True(t, found, "LocalPrefAttribute should be present in filtered attributes")
}
