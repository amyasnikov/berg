package utils

import (
	"testing"

	api "github.com/osrg/gobgp/v3/api"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestRdToString(t *testing.T) {
	tests := []struct {
		name        string
		rd          *anypb.Any
		expected    string
		expectError bool
	}{
		{
			name: "TwoOctetASN",
			rd: func() *anypb.Any {
				rd, _ := anypb.New(&api.RouteDistinguisherTwoOctetASN{
					Admin:    100,
					Assigned: 200,
				})
				return rd
			}(),
			expected:    "100:200",
			expectError: false,
		},
		{
			name: "FourOctetASN",
			rd: func() *anypb.Any {
				rd, _ := anypb.New(&api.RouteDistinguisherFourOctetASN{
					Admin:    65536,
					Assigned: 300,
				})
				return rd
			}(),
			expected:    "65536:300",
			expectError: false,
		},
		{
			name: "IPAddress",
			rd: func() *anypb.Any {
				rd, _ := anypb.New(&api.RouteDistinguisherIPAddress{
					Admin:    "192.168.1.1",
					Assigned: 400,
				})
				return rd
			}(),
			expected:    "192.168.1.1:400",
			expectError: false,
		},
		{
			name: "Invalid RD",
			rd: func() *anypb.Any {
				rd, _ := anypb.New(&api.Family{})
				return rd
			}(),
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RdToString(tt.rd)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, InvalidRD, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}
