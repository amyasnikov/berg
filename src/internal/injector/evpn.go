package injector

import (
	"context"

	"github.com/amyasnikov/berg/internal/dto"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	api "github.com/osrg/gobgp/v3/api"
	"google.golang.org/protobuf/types/known/anypb"
	"github.com/amyasnikov/berg/internal/utils"

)

type EvpnInjector struct {
	s bgpServer
}

func NewEvpnInjector(s bgpServer) *EvpnInjector {
	return &EvpnInjector{s: s}
}

func (c *EvpnInjector) AddType5Route(route dto.Evpn5Route) (uuid.UUID, error) {
	rd, err := utils.RdToApi(route.Rd)
	if err != nil {
		return uuid.Nil, err
	}

	nlri, _ := anypb.New(&api.EVPNIPPrefixRoute{
		Rd:          rd,
		Esi:         &api.EthernetSegmentIdentifier{},
		EthernetTag: 0,
		IpPrefix:    route.Prefix,
		IpPrefixLen: route.Prefixlen,
		GwAddress:   route.Gateway,
		Label:       route.Vni,
	})
	extcomms := make([]*anypb.Any, 0, len(route.RouteTargets)+1)
	var merr error
	for _, rtString := range route.RouteTargets {
		rt, err := utils.RtToApi(rtString)
		multierror.Append(merr, err)
		extcomms = append(extcomms, rt)
	}
	if merr != nil {
		return uuid.Nil, merr
	}
	encap, _ := anypb.New(&api.EncapExtended{TunnelType: 8}) // VXLAN encap
	extcommAttr, _ := anypb.New(&api.ExtendedCommunitiesAttribute{
		Communities: append(extcomms, encap),
	})
	nh , _ := anypb.New(&api.NextHopAttribute{NextHop: "0.0.0.0"})
	pattrs := append(route.PathAttrs, extcommAttr, nh)
	req := &api.AddPathRequest{
		Path: &api.Path{
			Family: &api.Family{
				Afi:  api.Family_AFI_L2VPN,
				Safi: api.Family_SAFI_EVPN,
			},
			Nlri:   nlri,
			Pattrs: pattrs,
		},
	}
	resp, err := c.s.AddPath(context.TODO(), req)
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.FromBytes(resp.Uuid)
}

func (c *EvpnInjector) DelRoute(uuid uuid.UUID) error {
	family := &api.Family{
		Afi:  api.Family_AFI_L2VPN,
		Safi: api.Family_SAFI_EVPN,
	}
	return delRoute(c.s, uuid, family)
}
