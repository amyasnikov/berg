package utils

import (
	"errors"
	"fmt"

	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/packet/bgp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var InvalidRD = errors.New("invalid RD")

func RdToString(rd *anypb.Any) (string, error) {
	rd1 := api.RouteDistinguisherTwoOctetASN{}
	rd2 := api.RouteDistinguisherFourOctetASN{}
	rd3 := api.RouteDistinguisherIPAddress{}
	if err := rd.UnmarshalTo(&rd1); err == nil {
		return fmt.Sprintf("%v:%v", rd1.Admin, rd1.Assigned), nil
	} else if err = rd.UnmarshalTo(&rd2); err == nil {
		return fmt.Sprintf("%v:%v", rd2.Admin, rd2.Assigned), nil
	} else if err = rd.UnmarshalTo(&rd3); err == nil {
		return fmt.Sprintf("%v:%v", rd3.Admin, rd3.Assigned), nil
	}
	return "", InvalidRD
}

func RdToApi(rdStr string) (*anypb.Any, error) {
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

func RtToApi(rtStr string) (*anypb.Any, error) {
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

	anyRT, err := anypb.New(msg.(proto.Message))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RT %q: %w", rtStr, err)
	}
	return anyRT, nil
}
