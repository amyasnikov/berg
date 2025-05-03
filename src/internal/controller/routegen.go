package controller

import (
	"github.com/amyasnikov/gober/internal/dto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/proto"
	api "github.com/osrg/gobgp/v3/api"

)


type EvpnRouteGen struct {
	attrFilter *AttrFilter
}

func (g *EvpnRouteGen) GenRoute (route ipv4Route, vrf dto.Vrf, pattrs []*anypb.Any) (er dto.Evpn5Route) {
	er.Rd = vrf.Rd
	er.RouteTargets = vrf.ExportRouteTargets
	er.Prefix = route.Prefix
	er.Prefixlen = route.Prefixlen
	er.Gateway = g.mustFindNextHop(pattrs)
	er.Vni = vrf.Vni
	er.PathAttrs = g.attrFilter.Filter(pattrs)
	return
}

func (g *EvpnRouteGen) mustFindNextHop(pattrs []*anypb.Any) string {
	var nh api.NextHopAttribute
	for _, attr := range pattrs {
		if err := anypb.UnmarshalTo(attr, &nh, proto.UnmarshalOptions{}); err == nil {
			return nh.GetNextHop()
		}
	}
	panic("no nexthop found")
}


type ipv4RouteGen struct {
	attrFilter *AttrFilter
}


func (g *ipv4RouteGen) genRoute (route evpnRoute, pattrs []*anypb.Any) (ir dto.IPv4Route) {
	return
}
