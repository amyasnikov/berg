package controller

import (
	"github.com/amyasnikov/berg/internal/dto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/proto"
	api "github.com/osrg/gobgp/v3/api"

)


type evpnRouteGen struct {
	attrFilter *AttrFilter
}

func newEvpnRouteGen() *evpnRouteGen {
	allowedAttrs := []proto.Message{
		&api.CommunitiesAttribute{},
		&api.As4PathAttribute{},
		&api.As4AggregatorAttribute{},
		&api.OriginAttribute{},
		&api.MultiExitDiscAttribute{},
		&api.LocalPrefAttribute{},
		&api.AtomicAggregateAttribute{},
		&api.OriginatorIdAttribute{},
		&api.ClusterListAttribute{},
		&api.AsPathAttribute{},
		&api.AggregatorAttribute{},
		&api.AigpAttribute{},
		&api.LargeCommunitiesAttribute{},
		&api.MpReachNLRIAttribute{},
		&api.MpUnreachNLRIAttribute{},
		&api.TunnelEncapAttribute{},
		&api.PmsiTunnelAttribute{},
	}
	return &evpnRouteGen{
		attrFilter: &AttrFilter{includeAttrs: allowedAttrs},
	}

}

func (g *evpnRouteGen) GenRoute(route ipv4Route, vrf dto.Vrf, pattrs []*anypb.Any) (er dto.Evpn5Route) {
	er.Rd = vrf.Rd
	er.RouteTargets = vrf.ExportRouteTargets
	er.Prefix = route.Prefix
	er.Prefixlen = route.Prefixlen
	er.Gateway = g.mustFindNextHop(pattrs)
	er.Vni = vrf.Vni
	er.PathAttrs = g.attrFilter.Filter(pattrs)
	return
}

func (g *evpnRouteGen) mustFindNextHop(pattrs []*anypb.Any) string {
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


func newIPv4RouteGen() *ipv4RouteGen{
	allowedAttrs := []proto.Message{
		&api.CommunitiesAttribute{},
		&api.As4PathAttribute{},
		&api.As4AggregatorAttribute{},
		&api.OriginAttribute{},
		&api.MultiExitDiscAttribute{},
		&api.LocalPrefAttribute{},
		&api.AtomicAggregateAttribute{},
		&api.OriginatorIdAttribute{},
		&api.ClusterListAttribute{},
		&api.AsPathAttribute{},
		&api.AggregatorAttribute{},
		&api.AigpAttribute{},
		&api.LargeCommunitiesAttribute{},
		&api.NextHopAttribute{},
	}
	return &ipv4RouteGen{
		attrFilter: &AttrFilter{includeAttrs: allowedAttrs},
	}
}

func (g *ipv4RouteGen) genRoute (route evpnRoute, pattrs []*anypb.Any, vrf string) (ir dto.IPv4Route) {
	ir.PathAttrs = g.attrFilter.Filter(pattrs)
	ir.Prefix = route.Prefix
	ir.Prefixlen = route.Prefixlen
	ir.Vrf = vrf
	return
}
