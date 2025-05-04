package injector

import (
	"fmt"

	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/packet/bgp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func parseRD(rdStr string) (*anypb.Any, error) {
	rd, err := bgp.ParseRouteDistinguisher(rdStr)
	if err != nil {
		return &anypb.Any{}, err
	}
	switch v := rd.(type) {
	case *bgp.RouteDistinguisherTwoOctetAS:
		return anypb.New(&api.RouteDistinguisherTwoOctetASN{
			Admin:    uint32(v.Admin),
			Assigned: v.Assigned,
		})
	case *bgp.RouteDistinguisherIPAddressAS:
		return anypb.New(&api.RouteDistinguisherIPAddress{
			Admin:    v.Admin.String(),
			Assigned: uint32(v.Assigned),
		})
	case *bgp.RouteDistinguisherFourOctetAS:
		return anypb.New(&api.RouteDistinguisherFourOctetASN{
			Admin:    v.Admin,
			Assigned: uint32(v.Assigned),
		})
	default:
		return &anypb.Any{}, fmt.Errorf("unsupported RD type %T", v)
	}
}

func mustParseRD(rdStr string) *anypb.Any {
	rd, err := parseRD(rdStr)
	if err != nil {
		panic(err)
	}
	return rd
}

func parseRT(rtStr string) (*anypb.Any, error) {
	raw, err := bgp.ParseRouteTarget(rtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid route-target %q: %w", rtStr, err)
	}

	var msg interface{}
	switch v := raw.(type) {
	case *bgp.TwoOctetAsSpecificExtended:
		msg = &api.TwoOctetAsSpecificExtended{
			IsTransitive: true,
			SubType:      uint32(v.SubType),
			Asn:          uint32(v.AS),
			LocalAdmin:   v.LocalAdmin,
		}
	case *bgp.IPv4AddressSpecificExtended:
		msg = &api.IPv4AddressSpecificExtended{
			IsTransitive: true,
			SubType:      uint32(v.SubType),
			Address:      v.IPv4.String(),
			LocalAdmin:   uint32(v.LocalAdmin),
		}
	case *bgp.FourOctetAsSpecificExtended:
		msg = &api.FourOctetAsSpecificExtended{
			IsTransitive: true,
			SubType:      uint32(v.SubType),
			Asn:          v.AS,
			LocalAdmin:   uint32(v.LocalAdmin),
		}
	default:
		return nil, fmt.Errorf("unsupported RT type %T", v)
	}

	anyRT, err := anypb.New(msg.(*anypb.Any))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RT %q: %w", rtStr, err)
	}
	return anyRT, nil
}

func mustAny(m proto.Message) *anypb.Any {
	a, err := anypb.New(m)
	if err != nil {
		panic(err)
	}
	return a
}
