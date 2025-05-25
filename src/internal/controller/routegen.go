package controller

import (
	"github.com/amyasnikov/berg/internal/dto"
	api "github.com/osrg/gobgp/v3/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
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
		&api.TunnelEncapAttribute{},
		&api.PmsiTunnelAttribute{},
	}
	return &evpnRouteGen{
		attrFilter: &AttrFilter{includeAttrs: allowedAttrs},
	}

}

func (g *evpnRouteGen) GenRoute(route vpnRoute, vrf dto.Vrf, pattrs []*anypb.Any) (er dto.Evpn5Route, err error) {
	er.Rd = vrf.Rd
	er.RouteTargets = vrf.ExportRouteTargets
	er.Prefix = route.Prefix
	er.Prefixlen = route.Prefixlen
	er.Gateway, err = findNextHop(route, pattrs)
	if err != nil {
		return dto.Evpn5Route{}, err
	}
	er.Vni = vrf.Vni
	er.PathAttrs = g.attrFilter.Filter(pattrs)
	return
}

type vpnRouteGen struct {
	attrFilter *AttrFilter
}

func newVpnRouteGen() *vpnRouteGen {
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
		&api.TunnelEncapAttribute{},
		&api.PmsiTunnelAttribute{},
	}
	return &vpnRouteGen{
		attrFilter: &AttrFilter{includeAttrs: allowedAttrs},
	}
}

func (g *vpnRouteGen) GenRoute(route evpnRoute, pattrs []*anypb.Any) (r dto.VPNRoute) {
	r.Rd = route.Rd
	r.Prefix = route.Prefix
	r.Prefixlen = route.Prefixlen
	r.PathAttrs = g.attrFilter.Filter(pattrs)
	return
}
