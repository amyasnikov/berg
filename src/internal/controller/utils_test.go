package controller

import (
	"fmt"
	"testing"

	api "github.com/osrg/gobgp/v3/api"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestExtractRouteTargets(t *testing.T) {
	tests := []struct {
		name     string
		pattrs   []*anypb.Any
		expected []string
	}{
		{
			name:     "No path attributes",
			pattrs:   []*anypb.Any{},
			expected: []string{},
		},
		{
			name: "No extended communities",
			pattrs: []*anypb.Any{
				func() *anypb.Any {
					attr, _ := anypb.New(&api.LocalPrefAttribute{LocalPref: 100})
					return attr
				}(),
			},
			expected: []string{},
		},
		{
			name: "Extended communities with TwoOctetAsSpecific route target",
			pattrs: []*anypb.Any{
				func() *anypb.Any {
					rt, _ := anypb.New(&api.TwoOctetAsSpecificExtended{
						IsTransitive: true,
						SubType:      2, // Route Target
						Asn:          65000,
						LocalAdmin:   100,
					})
					extComm, _ := anypb.New(&api.ExtendedCommunitiesAttribute{
						Communities: []*anypb.Any{rt},
					})
					return extComm
				}(),
			},
			expected: []string{"65000:100"},
		},
		{
			name: "Extended communities with IPv4AddressSpecific route target",
			pattrs: []*anypb.Any{
				func() *anypb.Any {
					rt, _ := anypb.New(&api.IPv4AddressSpecificExtended{
						IsTransitive: true,
						SubType:      2, // Route Target
						Address:      "192.168.1.1",
						LocalAdmin:   200,
					})
					extComm, _ := anypb.New(&api.ExtendedCommunitiesAttribute{
						Communities: []*anypb.Any{rt},
					})
					return extComm
				}(),
			},
			expected: []string{"192.168.1.1:200"},
		},
		{
			name: "Extended communities with FourOctetAsSpecific route target",
			pattrs: []*anypb.Any{
				func() *anypb.Any {
					rt, _ := anypb.New(&api.FourOctetAsSpecificExtended{
						IsTransitive: true,
						SubType:      2, // Route Target
						Asn:          4200000000,
						LocalAdmin:   300,
					})
					extComm, _ := anypb.New(&api.ExtendedCommunitiesAttribute{
						Communities: []*anypb.Any{rt},
					})
					return extComm
				}(),
			},
			expected: []string{"4200000000:300"},
		},
		{
			name: "Multiple route targets",
			pattrs: []*anypb.Any{
				func() *anypb.Any {
					rt1, _ := anypb.New(&api.TwoOctetAsSpecificExtended{
						IsTransitive: true,
						SubType:      2,
						Asn:          65000,
						LocalAdmin:   100,
					})
					rt2, _ := anypb.New(&api.TwoOctetAsSpecificExtended{
						IsTransitive: true,
						SubType:      2,
						Asn:          65001,
						LocalAdmin:   200,
					})
					extComm, _ := anypb.New(&api.ExtendedCommunitiesAttribute{
						Communities: []*anypb.Any{rt1, rt2},
					})
					return extComm
				}(),
			},
			expected: []string{"65000:100", "65001:200"},
		},
		{
			name: "Non-route-target extended communities (SubType != 2)",
			pattrs: []*anypb.Any{
				func() *anypb.Any {
					nonRT, _ := anypb.New(&api.TwoOctetAsSpecificExtended{
						IsTransitive: true,
						SubType:      1, // Not a Route Target
						Asn:          65000,
						LocalAdmin:   100,
					})
					extComm, _ := anypb.New(&api.ExtendedCommunitiesAttribute{
						Communities: []*anypb.Any{nonRT},
					})
					return extComm
				}(),
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRouteTargets(tt.pattrs)
			assert.Equal(t, tt.expected, result)
		})
	}
}

type mockRoute struct {
	name string
}

func (m mockRoute) String() string {
	return m.name
}

func TestFindNextHop(t *testing.T) {
	tests := []struct {
		name        string
		route       fmt.Stringer
		pattrs      []*anypb.Any
		expected    string
		expectedErr string
	}{
		{
			name:        "No path attributes",
			route:       mockRoute{"test-route"},
			pattrs:      []*anypb.Any{},
			expected:    "",
			expectedErr: "no nexthop was found for route test-route",
		},
		{
			name:  "No MpReachNLRIAttribute",
			route: mockRoute{"test-route"},
			pattrs: []*anypb.Any{
				func() *anypb.Any {
					attr, _ := anypb.New(&api.LocalPrefAttribute{LocalPref: 100})
					return attr
				}(),
			},
			expected:    "",
			expectedErr: "no nexthop was found for route test-route",
		},
		{
			name:  "MpReachNLRIAttribute with zero NextHops",
			route: mockRoute{"test-route"},
			pattrs: []*anypb.Any{
				func() *anypb.Any {
					nlri, _ := anypb.New(&api.MpReachNLRIAttribute{
						NextHops: []string{},
					})
					return nlri
				}(),
			},
			expected:    "",
			expectedErr: "found 0 NextHops for route test-route, while 1 was expected",
		},
		{
			name:  "MpReachNLRIAttribute with one NextHop",
			route: mockRoute{"test-route"},
			pattrs: []*anypb.Any{
				func() *anypb.Any {
					nlri, _ := anypb.New(&api.MpReachNLRIAttribute{
						NextHops: []string{"192.168.1.1"},
					})
					return nlri
				}(),
			},
			expected:    "192.168.1.1",
			expectedErr: "",
		},
		{
			name:  "MpReachNLRIAttribute with multiple NextHops",
			route: mockRoute{"test-route"},
			pattrs: []*anypb.Any{
				func() *anypb.Any {
					nlri, _ := anypb.New(&api.MpReachNLRIAttribute{
						NextHops: []string{"192.168.1.1", "192.168.1.2"},
					})
					return nlri
				}(),
			},
			expected:    "",
			expectedErr: "found 2 NextHops for route test-route, while 1 was expected",
		},
		{
			name:  "Mixed attributes with valid MpReachNLRIAttribute",
			route: mockRoute{"test-route"},
			pattrs: []*anypb.Any{
				func() *anypb.Any {
					attr, _ := anypb.New(&api.LocalPrefAttribute{LocalPref: 100})
					return attr
				}(),
				func() *anypb.Any {
					nlri, _ := anypb.New(&api.MpReachNLRIAttribute{
						NextHops: []string{"10.0.0.1"},
					})
					return nlri
				}(),
			},
			expected:    "10.0.0.1",
			expectedErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := findNextHop(tt.route, tt.pattrs)

			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr, err.Error())
				assert.Equal(t, "", result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
