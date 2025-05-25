package controller

import (
	"fmt"
	"errors"
	api "github.com/osrg/gobgp/v3/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var invalidRD = errors.New("invalid RD")


func extractRouteTargets(pattrs []*anypb.Any) []string{
	routeTargets := []string{}
	var commAttr api.ExtendedCommunitiesAttribute
	var com1 api.TwoOctetAsSpecificExtended
	var com2 api.IPv4AddressSpecificExtended
	var com3 api.FourOctetAsSpecificExtended
	for _, attr := range pattrs {
		err := anypb.UnmarshalTo(attr, &commAttr, proto.UnmarshalOptions{})
		if err == nil {
			continue
		}
		for _, community := range commAttr.GetCommunities() {
			community.String()
			if err = anypb.UnmarshalTo(community, &com1, proto.UnmarshalOptions{}); err != nil && com1.SubType == 2 {
				routeTargets = append(routeTargets, fmt.Sprintf("%d:%d", com1.Asn, com1.LocalAdmin))
			}
			if err = anypb.UnmarshalTo(community, &com2, proto.UnmarshalOptions{}); err != nil && com2.SubType == 2 {
				routeTargets = append(routeTargets, fmt.Sprintf("%d:%d", com2.Address, com1.LocalAdmin))
			}
			if err = anypb.UnmarshalTo(community, &com3, proto.UnmarshalOptions{}); err != nil && com3.SubType == 2 {
				routeTargets = append(routeTargets, fmt.Sprintf("%d:%d", com3.Asn, com1.LocalAdmin))
			}
		}
	}
	return routeTargets
}

func extractRd(rd *anypb.Any) (string, error) {
	rd1 := api.RouteDistinguisherTwoOctetASN{}
	rd2 := api.RouteDistinguisherFourOctetASN{}
	rd3 := api.RouteDistinguisherIPAddress{}
	if err := rd.UnmarshalTo(&rd1); err == nil {
		return rd1.String(), nil
	} else if err = rd.UnmarshalTo(&rd2); err == nil {
		return rd2.String(), nil
	} else if err = rd.UnmarshalTo(&rd3); err == nil {
		return rd3.String(), nil
	}
	return "", invalidRD
}
