package controller


import (
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/proto"
)


type AttrFilter struct {
	includeAttrs []proto.Message
}

func (f *AttrFilter) Filter(attrs []*anypb.Any) []*anypb.Any {
	result := make([]*anypb.Any, 0, len(attrs))
	for _, attr := range attrs {
		var toInclude bool
		for _, includeAttr := range f.includeAttrs {
			if attr.MessageIs(includeAttr) {
				toInclude = true
				break
			}
		}
		if toInclude {
			result = append(result, attr)
		}
	}
	return result
}
