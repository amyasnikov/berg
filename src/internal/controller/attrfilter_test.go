package controller

import (
	"testing"

	api "github.com/osrg/gobgp/v3/api"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestAttrFilter_Filter(t *testing.T) {
	tests := []struct {
		name         string
		includeAttrs []proto.Message
		inputAttrs   []*anypb.Any
		expected     int // number of expected attributes in result
	}{
		{
			name:         "Empty filter - no attributes included",
			includeAttrs: []proto.Message{},
			inputAttrs: []*anypb.Any{
				func() *anypb.Any {
					attr, _ := anypb.New(&api.LocalPrefAttribute{LocalPref: 100})
					return attr
				}(),
			},
			expected: 0,
		},
		{
			name:         "Filter matches single attribute",
			includeAttrs: []proto.Message{&api.LocalPrefAttribute{}},
			inputAttrs: []*anypb.Any{
				func() *anypb.Any {
					attr, _ := anypb.New(&api.LocalPrefAttribute{LocalPref: 100})
					return attr
				}(),
			},
			expected: 1,
		},
		{
			name:         "Filter excludes non-matching attributes",
			includeAttrs: []proto.Message{&api.LocalPrefAttribute{}},
			inputAttrs: []*anypb.Any{
				func() *anypb.Any {
					attr, _ := anypb.New(&api.MultiExitDiscAttribute{Med: 50})
					return attr
				}(),
			},
			expected: 0,
		},
		{
			name:         "Filter with multiple include types",
			includeAttrs: []proto.Message{&api.LocalPrefAttribute{}, &api.MultiExitDiscAttribute{}},
			inputAttrs: []*anypb.Any{
				func() *anypb.Any {
					attr, _ := anypb.New(&api.LocalPrefAttribute{LocalPref: 100})
					return attr
				}(),
				func() *anypb.Any {
					attr, _ := anypb.New(&api.MultiExitDiscAttribute{Med: 50})
					return attr
				}(),
				func() *anypb.Any {
					attr, _ := anypb.New(&api.AsPathAttribute{})
					return attr
				}(),
			},
			expected: 2, // LocalPref and Med should be included, AsPath excluded
		},
		{
			name:         "Empty input attributes",
			includeAttrs: []proto.Message{&api.LocalPrefAttribute{}},
			inputAttrs:   []*anypb.Any{},
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &AttrFilter{includeAttrs: tt.includeAttrs}
			result := filter.Filter(tt.inputAttrs)
			assert.Equal(t, tt.expected, len(result))
		})
	}
}
