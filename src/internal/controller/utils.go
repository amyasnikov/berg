package controller

import (
	"fmt"

	"github.com/amyasnikov/berg/internal/dto"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/config/oc"
	"github.com/puzpuzpuz/xsync/v4"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func extractRouteTargets(pattrs []*anypb.Any) []string {
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

func findNextHop(route fmt.Stringer, pattrs []*anypb.Any) (string, error) {
	var nlri api.MpReachNLRIAttribute
	for _, attr := range pattrs {
		if err := anypb.UnmarshalTo(attr, &nlri, proto.UnmarshalOptions{}); err == nil {
			if nhcount := len(nlri.NextHops); nhcount != 1 {
				return "", fmt.Errorf(
					"found %d NextHops for route %s, while 1 was expected", nhcount, route.String(),
				)
			}
			return nlri.NextHops[0], nil
		}
	}
	return "", fmt.Errorf("no nexthop was found for route %s", route.String())
}


func makeRdVrfMap(vrfCfg []oc.VrfConfig) *xsync.Map[string, dto.Vrf] {
	rdVrfMap := xsync.NewMap[string, dto.Vrf]()
	for _, vrf := range vrfCfg {
		vrfDto := dto.Vrf{
			Name:               vrf.Name,
			Rd:                 vrf.Rd,
			ImportRouteTargets: vrf.BothRtList,
			ExportRouteTargets: vrf.BothRtList,
			Vni:                vrf.Id,
		}
		if len(vrf.ImportRtList) > 0 {
			vrfDto.ImportRouteTargets = vrf.ImportRtList
		}
		if len(vrf.ExportRtList) > 0 {
			vrfDto.ExportRouteTargets = vrf.ExportRtList
		}
		rdVrfMap.Store(vrfDto.Rd, vrfDto)
	}
	return rdVrfMap
}
